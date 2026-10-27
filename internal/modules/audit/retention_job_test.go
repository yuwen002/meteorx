package audit

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/config"
	"meteorx/internal/middleware"
	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

func newRetentionDB(t *testing.T) *gorm.DB {
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

func seedLog(db *gorm.DB, id string, createdAt time.Time) {
	_ = db.Create(&repository.AuditLogPO{
		ID: id, UserID: "u-1", Module: "user", Action: model.ActionTypeCreate,
		Result: model.ResultSuccess, CreatedAt: createdAt,
	}).Error
}

// TestRetentionJob_DeletesOldKeepsNew days>0 时删除早于保留期的行、保留新行。
func TestRetentionJob_DeletesOldKeepsNew(t *testing.T) {
	db := newRetentionDB(t)
	now := time.Now()
	seedLog(db, "old", now.AddDate(0, 0, -120))
	seedLog(db, "fresh", now)

	job := NewAuditRetentionJob(db, config.Config{Audit: config.AuditConfig{RetentionDays: 90}})
	deleted, err := job.RunOnce(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)

	var left []repository.AuditLogPO
	require.NoError(t, db.Find(&left).Error)
	require.Len(t, left, 1)
	require.Equal(t, "fresh", left[0].ID)
}

// TestRetentionJob_DisabledByDefault days≤0：既不启动后台任务，直接 RunOnce 也不删任何行。
func TestRetentionJob_DisabledByDefault(t *testing.T) {
	db := newRetentionDB(t)
	seedLog(db, "old", time.Now().AddDate(0, 0, -400))

	job := NewAuditRetentionJob(db, config.Config{}) // RetentionDays 默认 0

	// 直接执行也不删
	deleted, err := job.RunOnce(context.Background())
	require.NoError(t, err)
	require.Zero(t, deleted)
	var cnt int64
	require.NoError(t, db.Model(&repository.AuditLogPO{}).Count(&cnt).Error)
	require.EqualValues(t, 1, cnt, "关闭状态下绝不清理")

	// Start 应立即返回、不起 goroutine
	done := make(chan struct{})
	go func() { job.Start(context.Background(), time.Hour); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("days≤0 时 Start 应立即返回")
	}
}

// TestRetentionJob_RecordsSelfAudit 清理成功须补记一条 audit/delete/critical/success 自指审计（system 触发）。
func TestRetentionJob_RecordsSelfAudit(t *testing.T) {
	db := newRetentionDB(t)
	seedLog(db, "old", time.Now().AddDate(0, 0, -120))

	var captured []dto.CreateAuditLogReq
	restore := middleware.SetAuditCaptureForTest(func(req *dto.CreateAuditLogReq) {
		captured = append(captured, *req)
	})
	defer restore()

	job := NewAuditRetentionJob(db, config.Config{Audit: config.AuditConfig{RetentionDays: 90}})
	deleted, err := job.RunOnce(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)

	require.Len(t, captured, 1, "清理成功须记一条自指审计")
	got := captured[0]
	require.Equal(t, "audit", got.Module)
	require.Equal(t, model.ActionTypeDelete, got.Action)
	require.Equal(t, model.RiskCritical, got.RiskLevel)
	require.Equal(t, model.ResultSuccess, got.Result)
	require.Equal(t, "system", got.UserID)
}
