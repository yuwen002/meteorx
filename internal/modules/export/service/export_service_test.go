package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/export/dto"
	"meteorx/internal/modules/export/model"
	"meteorx/internal/modules/export/repository"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

func newService(t *testing.T, retentionDays int) (*ExportService, *gorm.DB) {
	t.Helper()
	// 具名 shared-cache 内存库：裸 ":memory:" 下 sqlite 连接池每新一条连接都是全新空库，
	// AutoMigrate 的表只对首条连接可见，并发访问会随机出现 no such table。
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, repository.AutoMigrate(db))
	repo := repository.NewExportRepository(db)
	// Dir 用 t.TempDir()：用例结束自动回收，避免固定目录在多次运行间残留干扰“目录当占位文件”类断言
	svc := NewExportService(repo, Config{Dir: t.TempDir(), RetentionDays: retentionDays, WorkerCount: 1, BatchSize: 100})
	return svc, db
}

// newServiceWithLimit 构造指定队列上限的服务实例（验证 export.max_active_jobs 可配置性）
func newServiceWithLimit(t *testing.T, maxActiveJobs int) *ExportService {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, repository.AutoMigrate(db))
	return NewExportService(repository.NewExportRepository(db), Config{
		Dir: t.TempDir(), RetentionDays: 7,
		WorkerCount: 1, BatchSize: 100, MaxActiveJobs: maxActiveJobs,
	})
}

func TestService_CreateExport_Valid(t *testing.T) {
	svc, _ := newService(t, 7)
	ctx := context.Background()

	job, err := svc.CreateExport(ctx, "t1", "u1", dto.CreateExportReq{Resource: model.ResourceUser, Filters: map[string]any{"status": "1"}})
	require.NoError(t, err)
	require.NotEmpty(t, job.ID)
	require.Equal(t, model.ExportStatusPending, job.Status)
	require.Equal(t, "t1", job.TenantID)
	require.Equal(t, "u1", job.UserID)
	require.True(t, job.ExpiresAt.After(time.Now()))
}

func TestService_CreateExport_InvalidResource(t *testing.T) {
	svc, _ := newService(t, 7)
	_, err := svc.CreateExport(context.Background(), "t1", "u1", dto.CreateExportReq{Resource: "unknown"})
	require.ErrorIs(t, err, ErrInvalidResource)
}

func TestService_CreateExport_Format(t *testing.T) {
	svc, _ := newService(t, 7)
	ctx := context.Background()

	// 显式指定 xlsx
	job, err := svc.CreateExport(ctx, "t1", "u1", dto.CreateExportReq{Resource: model.ResourceUser, Format: model.FormatXLSX})
	require.NoError(t, err)
	require.Equal(t, model.FormatXLSX, job.Format)

	// 不传格式 -> 默认 csv
	job2, err := svc.CreateExport(ctx, "t1", "u1", dto.CreateExportReq{Resource: model.ResourceTask})
	require.NoError(t, err)
	require.Equal(t, model.FormatCSV, job2.Format)

	// 非法格式 -> 回落默认 csv（保证任务可创建）
	job3, err := svc.CreateExport(ctx, "t1", "u1", dto.CreateExportReq{Resource: model.ResourceTask, Format: "doc"})
	require.NoError(t, err)
	require.Equal(t, model.FormatCSV, job3.Format)
}

func TestService_CreateExport_IDFilterNormalized(t *testing.T) {
	svc, _ := newService(t, 7)
	ctx := context.Background()

	job, err := svc.CreateExport(ctx, "t1", "u1", dto.CreateExportReq{
		Resource: model.ResourceUser,
		Filters:  map[string]any{"ids": []any{"u1", " u2 ", "u1", ""}},
	})
	require.NoError(t, err)
	// 剔空、去重、去首尾空格后落库，避免 IN 条件里带无意义值
	require.JSONEq(t, `{"ids":["u1","u2"]}`, job.FiltersJSON)
}

func TestService_CreateExport_IDFilterRejected(t *testing.T) {
	svc, _ := newService(t, 7)
	ctx := context.Background()

	invalid := []map[string]any{
		{"ids": "u1"},                      // 非数组
		{"ids": []any{"u1", 2}},            // 含非字符串元素
		{"ids": []any{}},                   // 空数组
		{"ids": []any{"", "  "}},           // 剔空后为空
		{"ids": manyIDs(MaxFilterIDs + 1)}, // 超上限
	}
	for _, filters := range invalid {
		_, err := svc.CreateExport(ctx, "t1", "u1", dto.CreateExportReq{Resource: model.ResourceUser, Filters: filters})
		require.ErrorIs(t, err, ErrInvalidFilters, "filters=%v", filters)
	}

	// 恰好等于上限时允许
	_, err := svc.CreateExport(ctx, "t1", "u1", dto.CreateExportReq{
		Resource: model.ResourceUser,
		Filters:  map[string]any{"ids": manyIDs(MaxFilterIDs)},
	})
	require.NoError(t, err)
}

func manyIDs(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("u%d", i)
	}
	return ids
}

func TestService_CreateExport_QueueFull(t *testing.T) {
	svc, _ := newService(t, 7)
	ctx := context.Background()

	// 先达到上限：连续创建 MaxActiveJobsPerTenant 个 pending 任务
	for i := 0; i < MaxActiveJobsPerTenant; i++ {
		_, err := svc.CreateExport(ctx, "t1", "u1", dto.CreateExportReq{Resource: model.ResourceUser})
		require.NoError(t, err)
	}

	// 第 11 个被拒绝：队列已满
	_, err := svc.CreateExport(ctx, "t1", "u1", dto.CreateExportReq{Resource: model.ResourceUser})
	require.ErrorIs(t, err, ErrExportQueueFull)

	// 上限按租户隔离：其他租户不受影响
	_, err = svc.CreateExport(ctx, "t2", "u2", dto.CreateExportReq{Resource: model.ResourceUser})
	require.NoError(t, err)
}

func TestService_CreateExport_CustomQueueLimit(t *testing.T) {
	// 配置覆盖：上限为 2 时第 3 个被拒，第 3 个之前的正常
	svc := newServiceWithLimit(t, 2)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		_, err := svc.CreateExport(ctx, "t1", "u1", dto.CreateExportReq{Resource: model.ResourceUser})
		require.NoError(t, err)
	}
	_, err := svc.CreateExport(ctx, "t1", "u1", dto.CreateExportReq{Resource: model.ResourceUser})
	require.ErrorIs(t, err, ErrExportQueueFull)

	// 零值回落默认上限：同为默认 10 个
	svc0 := newServiceWithLimit(t, 0)
	for i := 0; i < MaxActiveJobsPerTenant; i++ {
		_, err := svc0.CreateExport(ctx, "t9", "u9", dto.CreateExportReq{Resource: model.ResourceUser})
		require.NoError(t, err)
	}
	_, err = svc0.CreateExport(ctx, "t9", "u9", dto.CreateExportReq{Resource: model.ResourceUser})
	require.ErrorIs(t, err, ErrExportQueueFull)
}

func TestService_GetByID_Ownership(t *testing.T) {
	svc, _ := newService(t, 7)
	ctx := context.Background()
	job, err := svc.CreateExport(ctx, "t1", "u1", dto.CreateExportReq{Resource: model.ResourceTask})
	require.NoError(t, err)

	got, err := svc.GetByID(ctx, job.ID, "t1", "u1", false)
	require.NoError(t, err)
	require.Equal(t, job.ID, got.ID)

	_, err = svc.GetByID(ctx, job.ID, "t1", "other", false)
	require.ErrorIs(t, err, ErrExportForbidden)

	_, err = svc.GetByID(ctx, job.ID, "tX", "uX", false)
	require.ErrorIs(t, err, ErrExportForbidden)

	gotMaster, err := svc.GetByID(ctx, job.ID, "tX", "root", true)
	require.NoError(t, err)
	require.Equal(t, job.ID, gotMaster.ID)

	_, err = svc.GetByID(ctx, "nonexistent", "t1", "u1", false)
	require.ErrorIs(t, err, ErrExportNotFound)
}

func TestService_List(t *testing.T) {
	svc, _ := newService(t, 7)
	ctx := context.Background()
	_, err := svc.CreateExport(ctx, "t1", "u1", dto.CreateExportReq{Resource: model.ResourceUser})
	require.NoError(t, err)
	_, err = svc.CreateExport(ctx, "t1", "u1", dto.CreateExportReq{Resource: model.ResourceTask})
	require.NoError(t, err)

	items, total, err := svc.List(ctx, "t1", "u1", dto.ListExportReq{Page: 1, PageSize: 20}, false)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 2)

	// 系统管理员为其它租户创建的任务归属目标租户：
	// 按当前租户（SYSTEM_ROOT）严格过滤时看不到，master 视角不限租户则能看到
	_, err = svc.CreateExport(ctx, "t2", "root", dto.CreateExportReq{Resource: model.ResourceUser})
	require.NoError(t, err)
	_, total, err = svc.List(ctx, contextx.SystemTenantID, "root", dto.ListExportReq{Page: 1, PageSize: 20}, false)
	require.NoError(t, err)
	require.Equal(t, int64(0), total)

	items, total, err = svc.List(ctx, contextx.SystemTenantID, "root", dto.ListExportReq{Page: 1, PageSize: 20}, true)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "t2", items[0].TenantID)
}

func TestService_CleanupExpired(t *testing.T) {
	svc, db := newService(t, 7)
	ctx := context.Background()
	dir := svc.Config().Dir
	require.NoError(t, os.MkdirAll(dir, 0o755))

	filePath := filepath.Join(dir, "old.csv")
	require.NoError(t, os.WriteFile(filePath, []byte("a,b\n"), 0o640))

	repo := repository.NewExportRepository(db)
	expired := &model.ExportJob{
		ID:        "expired1",
		TenantID:  "t1",
		UserID:    "u1",
		Resource:  model.ResourceUser,
		Status:    model.ExportStatusCompleted,
		FilePath:  filePath,
		CreatedAt: time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	require.NoError(t, repo.Create(ctx, expired))

	fresh := &model.ExportJob{
		ID:        "fresh1",
		TenantID:  "t1",
		UserID:    "u1",
		Resource:  model.ResourceUser,
		Status:    model.ExportStatusCompleted,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.Create(ctx, fresh))

	n, err := svc.CleanupExpired(ctx, time.Now(), 100)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	_, statErr := os.Stat(filePath)
	require.True(t, os.IsNotExist(statErr), "过期文件应被删除")
	_, err = repo.GetByID(ctx, "expired1")
	require.ErrorIs(t, err, repository.ErrExportJobNotFound)
	_, err = repo.GetByID(ctx, "fresh1")
	require.NoError(t, err, "未过期任务应保留")
}

// newExpiredJob 写入一条已过期的完成任务记录，FilePath 由调用方指定
func newExpiredJob(t *testing.T, repo repository.ExportRepository, id, filePath string) {
	t.Helper()
	require.NoError(t, repo.Create(context.Background(), &model.ExportJob{
		ID: id, TenantID: "t1", UserID: "u1",
		Resource:  model.ResourceUser,
		Status:    model.ExportStatusCompleted,
		FilePath:  filePath,
		CreatedAt: time.Now().Add(-48 * time.Hour),
		ExpiresAt: time.Now().Add(-time.Hour),
	}))
}

// TestService_CleanupExpired_KeepsRowWhenFileUndeletable 文件删不掉时必须保留记录。
// 若照旧删除记录，该文件将不再被任何记录引用，而过期清理只扫数据库行，
// 于是形成永久孤儿文件（磁盘单调增长）。Windows 上导出文件正被下载或
// 被安全软件持有句柄时 os.Remove 会失败，属现实场景。
// 用“非空目录”作为FilePath复现失败：Linux/Windows 均报错且不是 ErrNotExist。
func TestService_CleanupExpired_KeepsRowWhenFileUndeletable(t *testing.T) {
	svc, _ := newService(t, 7)
	ctx := context.Background()
	repo := svc.Repo()

	dir := svc.Config().Dir
	require.NoError(t, os.MkdirAll(dir, 0o755))
	blocked := filepath.Join(dir, "blocked.csv")
	require.NoError(t, os.Mkdir(blocked, 0o755))
	inner := filepath.Join(blocked, "hold")
	require.NoError(t, os.WriteFile(inner, []byte("x"), 0o640))

	newExpiredJob(t, repo, "locked1", blocked)

	n, err := svc.CleanupExpired(ctx, time.Now(), 100)
	require.NoError(t, err)
	require.Equal(t, int64(0), n, "文件未删除成功时不应计入清理数")

	// 关键断言：记录必须还在，否则文件就永久失联了
	got, err := repo.GetByID(ctx, "locked1")
	require.NoError(t, err, "删不掉文件时应保留记录，否则形成永久孤儿文件")
	require.Equal(t, blocked, got.FilePath)
	_, err = os.Stat(blocked)
	require.NoError(t, err, "目录本体不应被误删")

	// 障碍排除后，下一轮应能补删，证明保留记录不是为了永久堵死
	require.NoError(t, os.Remove(inner))
	n2, err := svc.CleanupExpired(ctx, time.Now(), 100)
	require.NoError(t, err)
	require.Equal(t, int64(1), n2)
	_, err = repo.GetByID(ctx, "locked1")
	require.ErrorIs(t, err, repository.ErrExportJobNotFound)
	_, statErr := os.Stat(blocked)
	require.True(t, os.IsNotExist(statErr), "障碍排除后文件应被补删")
}

// TestService_CleanupExpired_MissingFileStillDeletesRow 文件本就不存在（已被人工清走、
// 存储迁移等）时不能卡住，应照常删除记录，避免记录无限堆积占住批次窗口。
func TestService_CleanupExpired_MissingFileStillDeletesRow(t *testing.T) {
	svc, _ := newService(t, 7)
	ctx := context.Background()

	missing := filepath.Join(svc.Config().Dir, "already-gone.csv")
	newExpiredJob(t, svc.Repo(), "gone1", missing)

	n, err := svc.CleanupExpired(ctx, time.Now(), 100)
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	_, err = svc.Repo().GetByID(ctx, "gone1")
	require.ErrorIs(t, err, repository.ErrExportJobNotFound)
}

// TestService_CleanupExpired_Boundary 过期判定以 expires_at < now 为准，
// 恰好到期这一时刻的记录本轮不动，下一轮才被清理（防止边界抖动误删）。
func TestService_CleanupExpired_Boundary(t *testing.T) {
	svc, _ := newService(t, 7)
	ctx := context.Background()

	// 用秒级对齐的 now，避开驱动存储精度带来的边界抖动
	now := time.Now().Truncate(time.Second)
	exact := now.Add(-time.Hour) // 已过期
	require.NoError(t, svc.Repo().Create(ctx, &model.ExportJob{
		ID: "exact1", TenantID: "t1", UserID: "u1", Resource: model.ResourceUser,
		Status: model.ExportStatusCompleted, CreatedAt: exact, ExpiresAt: now,
	}))

	n, err := svc.CleanupExpired(ctx, now, 100)
	require.NoError(t, err)
	require.Equal(t, int64(0), n, "expires_at 恰等于 now 的任务本轮不应被清理")

	// 时间推进一秒后被清理
	n2, err := svc.CleanupExpired(ctx, now.Add(time.Second), 100)
	require.NoError(t, err)
	require.Equal(t, int64(1), n2)
}

// ---------- 用户主动删除（需与过期清理同样防孤儿文件） ----------

// newCompletedJob 写入一条带文件的已完成任务
func newCompletedJob(t *testing.T, svc *ExportService, id, filePath string) {
	t.Helper()
	require.NoError(t, svc.Repo().Create(context.Background(), &model.ExportJob{
		ID: id, TenantID: "t1", UserID: "u1",
		Resource:  model.ResourceUser,
		Status:    model.ExportStatusCompleted,
		FilePath:  filePath,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}))
}

// TestService_Delete_RemovesFileAndRow 删除任务应同时干掉文件与记录。
func TestService_Delete_RemovesFileAndRow(t *testing.T) {
	svc, _ := newService(t, 7)
	ctx := context.Background()
	require.NoError(t, os.MkdirAll(svc.Config().Dir, 0o755))
	path := filepath.Join(svc.Config().Dir, "del.csv")
	require.NoError(t, os.WriteFile(path, []byte("a,b\n"), 0o640))
	newCompletedJob(t, svc, "del1", path)

	job, err := svc.Delete(ctx, "del1", "t1", "u1", false)
	require.NoError(t, err)
	require.Equal(t, "del1", job.ID)

	_, statErr := os.Stat(path)
	require.True(t, os.IsNotExist(statErr), "文件应被删除")
	_, err = svc.Repo().GetByID(ctx, "del1")
	require.ErrorIs(t, err, repository.ErrExportJobNotFound)
}

// TestService_Delete_KeepsRowWhenFileInUse 文件删不掉时必须报错并保留记录。
// 旧实现先删记录、再在 handler 里 `_ = os.Remove(...)` 丢弃错误，
// 一旦文件被下载句柄占用或权限不足，记录已没、文件永遗留在磁盘。
func TestService_Delete_KeepsRowWhenFileInUse(t *testing.T) {
	svc, _ := newService(t, 7)
	ctx := context.Background()
	require.NoError(t, os.MkdirAll(svc.Config().Dir, 0o755))

	blocked := filepath.Join(svc.Config().Dir, "busy.csv")
	require.NoError(t, os.Mkdir(blocked, 0o755))
	inner := filepath.Join(blocked, "hold")
	require.NoError(t, os.WriteFile(inner, []byte("x"), 0o640))
	newCompletedJob(t, svc, "busy1", blocked)

	_, err := svc.Delete(ctx, "busy1", "t1", "u1", false)
	require.ErrorIs(t, err, ErrExportFileInUse)

	_, statErr := os.Stat(blocked)
	require.NoError(t, statErr, "删不掉时不应误删文件")
	got, err := svc.Repo().GetByID(ctx, "busy1")
	require.NoError(t, err, "报错时必须保留记录，否则文件永久失联")
	require.Equal(t, blocked, got.FilePath)

	// 障碍排除后可正常删除
	require.NoError(t, os.Remove(inner))
	_, err = svc.Delete(ctx, "busy1", "t1", "u1", false)
	require.NoError(t, err)
	_, err = svc.Repo().GetByID(ctx, "busy1")
	require.ErrorIs(t, err, repository.ErrExportJobNotFound)
}

// TestService_Delete_MissingFileStillDeletesRow 文件已不在时删除流程不应被卡住。
func TestService_Delete_MissingFileStillDeletesRow(t *testing.T) {
	svc, _ := newService(t, 7)
	ctx := context.Background()
	newCompletedJob(t, svc, "gone2", filepath.Join(svc.Config().Dir, "not-here.csv"))

	_, err := svc.Delete(ctx, "gone2", "t1", "u1", false)
	require.NoError(t, err, "文件不存在应视为删除成功")
	_, err = svc.Repo().GetByID(ctx, "gone2")
	require.ErrorIs(t, err, repository.ErrExportJobNotFound)
}

// TestService_Delete_ForbiddenKeepsFile 无权删除时不能预先动文件。
func TestService_Delete_ForbiddenKeepsFile(t *testing.T) {
	svc, _ := newService(t, 7)
	ctx := context.Background()
	require.NoError(t, os.MkdirAll(svc.Config().Dir, 0o755))
	path := filepath.Join(svc.Config().Dir, "other.csv")
	require.NoError(t, os.WriteFile(path, []byte("a\n"), 0o640))
	newCompletedJob(t, svc, "own1", path)

	_, err := svc.Delete(ctx, "own1", "t1", "someone-else", false)
	require.ErrorIs(t, err, ErrExportForbidden)
	_, statErr := os.Stat(path)
	require.NoError(t, statErr, "鉴权失败时文件必须完好")
	_, err = svc.Repo().GetByID(ctx, "own1")
	require.NoError(t, err, "鉴权失败时记录必须保留")
}
