package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"meteorx/internal/modules/export/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// newExportTestDB 创建内存 SQLite 并迁移 export_jobs 表
func newExportTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	require.NoError(t, AutoMigrate(db))
	return db
}

func newJob(id, status string, expiresAt time.Time) *model.ExportJob {
	return &model.ExportJob{
		ID:        id,
		TenantID:  "t1",
		UserID:    "u1",
		Resource:  model.ResourceUser,
		Status:    status,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}
}

func TestExportRepo_Create_GetByID(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newJob("job1", model.ExportStatusPending, time.Now().Add(time.Hour))))

	got, err := repo.GetByID(ctx, "job1")
	require.NoError(t, err)
	require.Equal(t, "job1", got.ID)
	require.Equal(t, model.ExportStatusPending, got.Status)

	_, err = repo.GetByID(ctx, "missing")
	require.True(t, errors.Is(err, ErrExportJobNotFound))
}

func TestExportRepo_MarkRunning_Preemption(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, newJob("job1", model.ExportStatusPending, time.Now().Add(time.Hour))))

	ok, err := repo.MarkRunning(ctx, "job1", time.Now())
	require.NoError(t, err)
	require.True(t, ok, "首次抢占 pending→running 应成功")

	ok, err = repo.MarkRunning(ctx, "job1", time.Now())
	require.NoError(t, err)
	require.False(t, ok, "已 running 的任务不应被重复抢占")
}

func TestExportRepo_MarkCompleted_Failed(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, newJob("job1", model.ExportStatusPending, time.Now().Add(time.Hour))))
	_, err := repo.MarkRunning(ctx, "job1", time.Now())
	require.NoError(t, err)

	require.NoError(t, repo.MarkCompleted(ctx, "job1", "/tmp/job1.csv", 42, time.Now()))
	got, err := repo.GetByID(ctx, "job1")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusCompleted, got.Status)
	require.Equal(t, int64(42), got.Rows)
	require.Equal(t, "/tmp/job1.csv", got.FilePath)

	require.NoError(t, repo.Create(ctx, newJob("job2", model.ExportStatusPending, time.Now().Add(time.Hour))))
	require.NoError(t, repo.MarkFailed(ctx, "job2", "boom", time.Now()))
	got2, err := repo.GetByID(ctx, "job2")
	require.NoError(t, err)
	require.Equal(t, model.ExportStatusFailed, got2.Status)
	require.Equal(t, "boom", got2.ErrorMsg)
}

func TestExportRepo_ListPending(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, newJob("p1", model.ExportStatusPending, time.Now().Add(time.Hour))))
	require.NoError(t, repo.Create(ctx, newJob("p2", model.ExportStatusPending, time.Now().Add(time.Hour))))
	require.NoError(t, repo.Create(ctx, newJob("c1", model.ExportStatusCompleted, time.Now().Add(time.Hour))))

	list, err := repo.ListPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, list, 2)
}

func TestExportRepo_ListExpired(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	now := time.Now()
	require.NoError(t, repo.Create(ctx, newJob("exp", model.ExportStatusCompleted, now.Add(-time.Hour))))
	require.NoError(t, repo.Create(ctx, newJob("fresh", model.ExportStatusCompleted, now.Add(time.Hour))))

	list, err := repo.ListExpired(ctx, now, 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "exp", list[0].ID)
}

func TestExportRepo_ListByUser_Ownership(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, newJob("j1", model.ExportStatusPending, time.Now().Add(time.Hour))))
	other := newJob("j2", model.ExportStatusPending, time.Now().Add(time.Hour))
	other.UserID = "u2"
	require.NoError(t, repo.Create(ctx, other))

	items, total, err := repo.ListByUser(ctx, "t1", "u1", 1, 20, "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "j1", items[0].ID)
}

func TestExportRepo_Delete(t *testing.T) {
	db := newExportTestDB(t)
	repo := NewExportRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, newJob("job1", model.ExportStatusPending, time.Now().Add(time.Hour))))
	require.NoError(t, repo.Delete(ctx, "job1"))

	_, err := repo.GetByID(ctx, "job1")
	require.True(t, errors.Is(err, ErrExportJobNotFound))
}
