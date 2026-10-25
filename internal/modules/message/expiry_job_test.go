package message

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/config"
	messagerepo "meteorx/internal/modules/message/repository"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

func newMessageModuleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, Migrate(db))
	return db
}

func TestExpiryJob_RunOnceDeletesExpired(t *testing.T) {
	db := newMessageModuleTestDB(t)
	now := time.Now()

	require.NoError(t, db.Create(&messagerepo.UserMessagePO{
		ID: "old", TenantID: "t1", UserID: "u1", Category: "system",
		Title: "旧消息", CreatedAt: now.AddDate(0, 0, -120),
	}).Error)
	require.NoError(t, db.Create(&messagerepo.UserMessagePO{
		ID: "fresh", TenantID: "t1", UserID: "u1", Category: "system",
		Title: "新消息", CreatedAt: now,
	}).Error)

	job := NewMessageExpiryJob(db, config.Config{Message: config.MessageConfig{RetentionDays: 90}})
	n, err := job.RunOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	var left []messagerepo.UserMessagePO
	require.NoError(t, db.Find(&left).Error)
	require.Len(t, left, 1)
	require.Equal(t, "fresh", left[0].ID)
}

// TestExpiryJob_DefaultRetention 未配置 retention_days（0）时按 90 天保留。
func TestExpiryJob_DefaultRetention(t *testing.T) {
	db := newMessageModuleTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&messagerepo.UserMessagePO{
		ID: "m", TenantID: "t1", UserID: "u1", Category: "system",
		Title: "近 89 天", CreatedAt: now.AddDate(0, 0, -89),
	}).Error)

	job := NewMessageExpiryJob(db, config.Config{})
	n, err := job.RunOnce(context.Background())
	require.NoError(t, err)
	require.Zero(t, n, "89 天前的消息未超过默认 90 天保留期")
}

// TestExpiryJob_DisabledWhenNegative Start 在保留期为负值时不应启动后台清理。
func TestExpiryJob_DisabledWhenNegative(t *testing.T) {
	db := newMessageModuleTestDB(t)
	job := NewMessageExpiryJob(db, config.Config{Message: config.MessageConfig{RetentionDays: -1}})
	// 不启动 goroutine：调用应立即返回（此处只验证不阻塞、不 panic）
	done := make(chan struct{})
	go func() {
		job.Start(context.Background(), time.Hour)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("负数保留期时 Start 应立即返回")
	}
}
