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

func newPoolTest(t *testing.T) (*Pool, repository.ExportRepository, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	require.NoError(t, repository.AutoMigrate(db))
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
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	require.NoError(t, repository.AutoMigrate(db))
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
