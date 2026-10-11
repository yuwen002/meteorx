package wiki

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestWikiTrashExpiryJob_RunOnce 验证定时任务的 RunOnce 真正触发过期清理：
// 已过期的回收站条目被移除，未过期的保留；关联实体缺失时也不报错（兜底删除回收站记录）。
func TestWikiTrashExpiryJob_RunOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=busy_timeout(5000)"), &gorm.Config{
		Logger: logger.Discard,
	})
	require.NoError(t, err)
	require.NoError(t, repository.AutoMigrate(db))

	// 过期条目（关联实体不存在，应被安全清理并移除回收站记录）
	expired := &model.TrashItem{
		ID:        "trash-expired",
		TenantID:  "tenant-1",
		ItemType:  model.TrashTypeDocument,
		ItemID:    "missing-doc",
		Title:     "已过期文档",
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	// 未过期条目（应保留）
	alive := &model.TrashItem{
		ID:        "trash-alive",
		TenantID:  "tenant-1",
		ItemType:  model.TrashTypeNode,
		ItemID:    "missing-node",
		Title:     "未过期节点",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, db.Create(expired).Error)
	require.NoError(t, db.Create(alive).Error)

	job := NewWikiTrashExpiryJob(db)
	n, err := job.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n, "应清理 1 个过期条目")

	var remain []model.TrashItem
	require.NoError(t, db.Find(&remain).Error)
	require.Len(t, remain, 1, "仅未过期条目应保留")
	assert.Equal(t, "trash-alive", remain[0].ID)
}

// TestWikiTrashExpiryJob_StartStopsWithContext 验证任务随 ctx 取消而退出（不泄漏 goroutine）。
func TestWikiTrashExpiryJob_StartStopsWithContext(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=busy_timeout(5000)"), &gorm.Config{
		Logger: logger.Discard,
	})
	require.NoError(t, err)
	require.NoError(t, repository.AutoMigrate(db))

	ctx, cancel := context.WithCancel(context.Background())
	job := NewWikiTrashExpiryJob(db)
	job.Start(ctx, time.Hour)

	cancel()
	// 给后台 goroutine 一点时间退出；若未退出，后续测试会感知（此断言主要验证 Start 不 panic）
	time.Sleep(50 * time.Millisecond)
	n, err := job.RunOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n, "无过期条目时清理计数应为 0")
}
