package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	require.NoError(t, repository.AutoMigrate(db))
	repo := repository.NewExportRepository(db)
	svc := NewExportService(repo, Config{Dir: filepath.Join(os.TempDir(), "export-test"), RetentionDays: retentionDays, WorkerCount: 1, BatchSize: 100})
	return svc, db
}

// newServiceWithLimit 构造指定队列上限的服务实例（验证 export.max_active_jobs 可配置性）
func newServiceWithLimit(t *testing.T, maxActiveJobs int) *ExportService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	require.NoError(t, repository.AutoMigrate(db))
	return NewExportService(repository.NewExportRepository(db), Config{
		Dir: filepath.Join(os.TempDir(), "export-test"), RetentionDays: 7,
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

	items, total, err := svc.List(ctx, "t1", "u1", dto.ListExportReq{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 2)
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
