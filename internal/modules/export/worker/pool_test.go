package worker

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"meteorx/internal/modules/export/model"
	"meteorx/internal/modules/export/repository"
	"meteorx/internal/modules/export/service"
	"meteorx/internal/modules/export/service/exporter"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// fakeExporter 写出固定行数以驱动 worker 流程
type fakeExporter struct {
	resource string
	rows     int64
	err      error
}

func (f *fakeExporter) Resource() string { return f.resource }
func (f *fakeExporter) Stream(_ context.Context, _ string, _ map[string]any, sink exporter.RowSink) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	if err := sink.WriteHeader([]string{"id", "name"}); err != nil {
		return 0, err
	}
	records := make([][]string, 0, f.rows)
	for i := int64(0); i < f.rows; i++ {
		records = append(records, []string{"x", "y"})
	}
	n, err := sink.WriteRows(records)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// newPoolTestDB 创建内存 SQLite 并迁移 export_jobs 表。
// 必须用具名 shared-cache DSN：裸 ":memory:" 下连接池每新开一条连接都是一个全新空库，
// 而执行期心跳协程与主线程断言会并发访问库，随机出现 no such table。
func newPoolTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, repository.AutoMigrate(db))
	return db
}

func newPoolTest(t *testing.T) (*Pool, repository.ExportRepository, string) {
	t.Helper()
	db := newPoolTestDB(t)
	dir := filepath.Join(t.TempDir(), "exports")
	require.NoError(t, os.MkdirAll(dir, 0o750))
	repo := repository.NewExportRepository(db)
	registry := exporter.NewRegistry()
	registry.Register(&fakeExporter{resource: model.ResourceUser, rows: 3})
	pool := NewPool(repo, registry, service.Config{Dir: dir, WorkerCount: 1, BatchSize: 100, RetentionDays: 7})
	return pool, repo, dir
}

func seedPending(t *testing.T, repo repository.ExportRepository, id, resource string) {
	t.Helper()
	require.NoError(t, repo.Create(context.Background(), &model.ExportJob{
		ID: id, TenantID: "t1", UserID: "u1", Resource: resource,
		Status: model.ExportStatusPending, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour),
	}))
}

func TestPool_ProcessJob_Success(t *testing.T) {
	pool, repo, dir := newPoolTest(t)
	ctx := context.Background()
	seedPending(t, repo, "job1", model.ResourceUser)

	pool.processJob(ctx, "job1")

	got, err := repo.GetByID(ctx, "job1")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusCompleted, got.Status)
	require.Equal(t, int64(3), got.Rows)

	data, err := os.ReadFile(filepath.Join(dir, "job1.csv"))
	require.NoError(t, err)
	require.Equal(t, []byte{0xEF, 0xBB, 0xBF}, data[:3], "文件应含 UTF-8 BOM")
}

func TestPool_ProcessJob_Preemption(t *testing.T) {
	pool, repo, _ := newPoolTest(t)
	ctx := context.Background()
	seedPending(t, repo, "job1", model.ResourceUser)

	pool.processJob(ctx, "job1")
	pool.processJob(ctx, "job1") // 第二次应因抢占失败而跳过

	got, err := repo.GetByID(ctx, "job1")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusCompleted, got.Status)
	require.Equal(t, int64(3), got.Rows, "重复处理不应叠加行数")
}

func TestPool_ProcessJob_UnregisteredResource(t *testing.T) {
	pool, repo, _ := newPoolTest(t)
	ctx := context.Background()
	seedPending(t, repo, "job9", model.ResourceTask) // registry 未注册 task

	pool.processJob(ctx, "job9")

	got, err := repo.GetByID(ctx, "job9")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusFailed, got.Status)
	require.NotEmpty(t, got.ErrorMsg)
}

func TestPool_ProcessJob_ExporterError(t *testing.T) {
	db := newPoolTestDB(t)
	dir := t.TempDir()
	repo := repository.NewExportRepository(db)
	registry := exporter.NewRegistry()
	registry.Register(&fakeExporter{resource: model.ResourceUser, err: os.ErrPermission})
	pool := NewPool(repo, registry, service.Config{Dir: dir, WorkerCount: 1})

	seedPending(t, repo, "job1", model.ResourceUser)
	pool.processJob(context.Background(), "job1")

	got, err := repo.GetByID(context.Background(), "job1")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusFailed, got.Status)
	// 半成品文件应被清理
	_, statErr := os.Stat(filepath.Join(dir, "job1.csv"))
	require.True(t, os.IsNotExist(statErr))
}

// ctxExporter 在 Stream 执行途中触发 ctx 取消，忠实复现优雅关闭的时序：
// 抢占（MarkRunning）已成功、但终态回写时 ctx 已被取消。
type ctxExporter struct {
	resource string
	cancel   context.CancelFunc
}

func (c *ctxExporter) Resource() string { return c.resource }
func (c *ctxExporter) Stream(ctx context.Context, _ string, _ map[string]any, _ exporter.RowSink) (int64, error) {
	c.cancel() // 模拟关闭发生在导出途中
	<-ctx.Done()
	return 0, ctx.Err()
}

// TestPool_ProcessJob_CanceledCtxReachesTerminalState 防止优雅关闭泄漏：
// 导出进行中发生关闭（ctx 被取消）时，终态回写必须仍能落库（用脱离取消的上下文），
// 否则任务永久停在 running：poll 只拉 pending 不会重驱，而过期清理要数天后，
// 期间一直占用租户队列额度（CountActiveByTenant 计入 running）。
func TestPool_ProcessJob_CanceledCtxReachesTerminalState(t *testing.T) {
	db := newPoolTestDB(t)
	repo := repository.NewExportRepository(db)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	registry := exporter.NewRegistry()
	registry.Register(&ctxExporter{resource: model.ResourceUser, cancel: cancel})
	pool := NewPool(repo, registry, service.Config{Dir: t.TempDir(), WorkerCount: 1})

	seedPending(t, repo, "job-shutdown", model.ResourceUser)
	pool.processJob(ctx, "job-shutdown")

	// 用独立上下文读取：验证落库结果而非读取能力
	got, err := repo.GetByID(context.Background(), "job-shutdown")
	require.NoError(t, err)
	require.NotEqual(t, model.ExportStatusRunning, got.Status,
		"关闭时终态回写不能因 ctx 取消而丢失，否则任务永久卡在 running 并泄漏队列额度")
	require.Equal(t, model.ExportStatusFailed, got.Status)
}

// blockedExporter 在 Stream 中阻塞直到 release 被关闭，用于稳定观察“任务执行中”这一中间态
// （如心跳续租）；靠 sleep 制造时序窗口会让用例在慢 CI 上抖动。
type blockedExporter struct {
	resource string
	release  <-chan struct{}
}

func (b *blockedExporter) Resource() string { return b.resource }
func (b *blockedExporter) Stream(ctx context.Context, _ string, _ map[string]any, sink exporter.RowSink) (int64, error) {
	select {
	case <-b.release:
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	if err := sink.WriteHeader([]string{"id"}); err != nil {
		return 0, err
	}
	return sink.WriteRows([][]string{{"1"}})
}

// waitRunningLease 等到任务进入 running 并返回当时读到的租约到期时间。
func waitRunningLease(t *testing.T, repo repository.ExportRepository, id string) time.Time {
	t.Helper()
	var lease time.Time
	require.Eventually(t, func() bool {
		got, err := repo.GetByID(context.Background(), id)
		if err != nil || got.Status != model.ExportStatusRunning || got.LeaseExpiresAt == nil {
			return false
		}
		lease = *got.LeaseExpiresAt
		return true
	}, 2*time.Second, 5*time.Millisecond, "任务应被置为 running 并持有租约")
	return lease
}

// TestPool_ProcessJob_HeartbeatKeepsLeaseAlive 长导出执行期间心跳必须不断把租约推后：
// 不续租则超过 TTL 的导出会被回收扫描误判为“执行进程已死”而重驱，
// 导致同一任务被重复执行（重复写文件、重复回写终态）。
func TestPool_ProcessJob_HeartbeatKeepsLeaseAlive(t *testing.T) {
	db := newPoolTestDB(t)
	repo := repository.NewExportRepository(db)
	dir := t.TempDir()
	release := make(chan struct{})
	registry := exporter.NewRegistry()
	registry.Register(&blockedExporter{resource: model.ResourceUser, release: release})
	pool := NewPool(repo, registry, service.Config{Dir: dir, WorkerCount: 1})
	// 缩短到毫秒级：否则需真等 60s 租约与 20s 心跳间隔
	pool.leaseTTL = 200 * time.Millisecond
	pool.heartbeatRate = 20 * time.Millisecond

	seedPending(t, repo, "job-hb", model.ResourceUser)
	done := make(chan struct{})
	go func() {
		defer close(done)
		pool.processJob(context.Background(), "job-hb")
	}()

	first := waitRunningLease(t, repo, "job-hb")
	require.Eventually(t, func() bool {
		got, err := repo.GetByID(context.Background(), "job-hb")
		return err == nil && got.LeaseExpiresAt != nil && got.LeaseExpiresAt.After(first)
	}, 2*time.Second, 5*time.Millisecond,
		"执行期间心跳应持续把租约推后，否则慢导出会被误判为进程已死而被重驱")

	close(release)
	<-done
	got, err := repo.GetByID(context.Background(), "job-hb")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusCompleted, got.Status)
	require.Nil(t, got.LeaseExpiresAt, "进入终态后应释放租约")
	require.Empty(t, got.LeaseToken, "进入终态后令牌应清空，使迟到的旧回写无法命中")
}

// TestPool_Start_RecoversCrashedJob 模拟上一个进程被 kill -9：任务停在 running 且租约早已过期。
// 重启后不需人工清库：poll 先回收（回队 pending）再拉取，使它重跑并产出文件；
// 否则该任务会停在 running 直到数天后被过期清理扫掉，期间一直占住租户队列额度。
func TestPool_Start_RecoversCrashedJob(t *testing.T) {
	pool, repo, dir := newPoolTest(t)
	pool.pollEvery = 20 * time.Millisecond
	// 回收阈值远大于单轮执行耗时，避免正在跑的任务被自己当残留抢回
	pool.leaseTTL = 10 * time.Second
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	now := time.Now()
	require.NoError(t, repo.Create(ctx, &model.ExportJob{
		ID: "job-crash", TenantID: "t1", UserID: "u1", Resource: model.ResourceUser,
		Status: model.ExportStatusPending, CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}))
	// 复现崩溃现场：已抢占、租约在 1 分钟前就到期，且无人会再回写终态
	_, err := repo.MarkRunning(ctx, "job-crash", repository.Claim{
		StartedAt: now.Add(-time.Hour), LeaseExpiresAt: now.Add(-time.Minute), Token: "tok-crash",
	})
	require.NoError(t, err)

	started := make(chan struct{})
	go func() {
		defer close(started)
		pool.Start(ctx)
	}()

	require.Eventually(t, func() bool {
		got, err := repo.GetByID(context.Background(), "job-crash")
		return err == nil && got.Status == model.ExportStatusCompleted
	}, 3*time.Second, 10*time.Millisecond,
		"重启后应自动回收强杀残留的 running 任务并重驱至终态")

	_, err = os.Stat(filepath.Join(dir, "job-crash.csv"))
	require.NoError(t, err, "重驱应真实重新执行并产出文件，而不是只改写状态")
	got, err := repo.GetByID(context.Background(), "job-crash")
	require.NoError(t, err)
	require.Equal(t, 2, got.Attempts, "崩溃那次与重驱各计 1 次执行")

	cancel()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("pool 未在 ctx 取消后退出")
	}
}

// TestTruncateErrMsg 验证长错误按 rune 截断，短错误原样返回，避免内部长错误原样回显前端。
func TestTruncateErrMsg(t *testing.T) {
	short := "简短错误"
	require.Equal(t, short, truncateErrMsg(short))

	// 构造超长含中文错误，验证不会按字节切断导致乱码
	long := ""
	for i := 0; i < 300; i++ {
		long += "错"
	}
	got := truncateErrMsg(long)
	runes := []rune(got)
	require.LessOrEqual(t, len(runes), maxErrMsgLen+3, "截断后长度应受限（含省略号）")
	require.True(t, got[len(got)-3:] == "..." || got == long, "超长应以省略号结尾")
}
