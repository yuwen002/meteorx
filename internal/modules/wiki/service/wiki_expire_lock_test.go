package service

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/cache"
	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"
	dbpkg "meteorx/internal/pkg/db"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupExpireLockTest 构建独立内存 SQLite（含一条已过期回收站条目）与 miniredis 实例，
// 返回按生产装配方式注入分布式锁 Redis 的 wikiService。
// 条目为 Node 类型且指向不存在的节点：purge 路径容忍缺失实体，可稳定验证“清理成功”计数。
func setupExpireLockTest(t *testing.T, itemID string) (WikiService, *gorm.DB, *miniredis.Miniredis, *cache.Redis) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(uniqueMemoryDSN()), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	require.NoError(t, repository.AutoMigrate(db))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	item := &model.TrashItem{
		ID:        itemID,
		TenantID:  "tenant-1",
		ItemType:  model.TrashTypeNode,
		ItemID:    itemID + "-node",
		SpaceID:   "space-1",
		Title:     "过期节点",
		DeletedBy: "u-1",
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	require.NoError(t, db.Create(item).Error)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb, err := cache.NewRedis(mr.Addr(), "", 0)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rdb.Close() })

	svc := NewWikiService(repository.NewWikiRepository(db), dbpkg.NewTxManager(db), WithExpireCacheLock(rdb))
	return svc, db, mr, rdb
}

// TestExpireTrashItems_DistributedLockAcquireRelease Redis 可用时应抢占分布式锁，
// 清理完成后锁必须被释放（否则后续轮次将被永久阻塞至 TTL 到期）。
func TestExpireTrashItems_DistributedLockAcquireRelease(t *testing.T) {
	svc, db, mr, _ := setupExpireLockTest(t, "trash-lock-1")

	purged, err := svc.ExpireTrashItems(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, purged)
	assert.Zero(t, countTrash(t, db, "trash-lock-1"))

	assert.False(t, mr.Exists(expireLockKey), "清理完成后分布式锁应已释放")
}

// TestExpireTrashItems_LockHeldByOtherInstance 锁被其他实例（副本）持有时，
// 本次调用应直接跳过：不产生任何清理，也不得误删他人的锁。
func TestExpireTrashItems_LockHeldByOtherInstance(t *testing.T) {
	svc, db, mr, _ := setupExpireLockTest(t, "trash-lock-2")
	mr.Set(expireLockKey, "other-instance-token")

	purged, err := svc.ExpireTrashItems(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, purged, "未抢到锁时应跳过清理")

	var count int64
	require.NoError(t, db.Model(&model.TrashItem{}).Where("id = ?", "trash-lock-2").Count(&count).Error)
	assert.Equal(t, int64(1), count, "过期条目应保留，等待持锁实例或下一轮清理")
	lockVal, err := mr.Get(expireLockKey)
	require.NoError(t, err)
	assert.Equal(t, "other-instance-token", lockVal, "不得误删其他实例持有的锁")
}

// TestExpireTrashItems_NoCacheFallback 未注入 Redis（单副本/降级场景）时行为不变：
// 仅依赖进程内互斥即可完成清理。
func TestExpireTrashItems_NoCacheFallback(t *testing.T) {
	svc, db, _, _ := setupExpireLockTest(t, "trash-lock-3")
	svcNoCache := NewWikiService(repository.NewWikiRepository(db), dbpkg.NewTxManager(db))

	purged, err := svcNoCache.ExpireTrashItems(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, purged)
	assert.Zero(t, countTrash(t, db, "trash-lock-3"))
	_ = svc
}

// countTrash 统计指定 ID 的回收站条目剩余数。
func countTrash(t *testing.T, db *gorm.DB, id string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&model.TrashItem{}).Where("id = ?", id).Count(&count).Error)
	return count
}
