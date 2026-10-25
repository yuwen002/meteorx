package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	metermodel "meteorx/internal/modules/message/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// newMessageTestDB 创建内存 SQLite 并迁移 user_messages 表。
// 使用具名 shared-cache DSN：跨连接共享同一内存库，避免连接池新建连接看到空库。
func newMessageTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, AutoMigrate(db))
	return db
}

func newMsg(id, userID, category string, read bool, createdAt time.Time) *metermodel.Message {
	m := &metermodel.Message{
		ID:         id,
		TenantID:   "t1",
		UserID:     userID,
		Category:   category,
		Title:      "标题-" + id,
		Content:    "内容-" + id,
		Link:       "/task",
		Priority:   1,
		SourceType: "task",
		SourceID:   "src-1",
		CreatedAt:  createdAt,
	}
	if read {
		at := createdAt.Add(time.Minute)
		m.ReadAt = &at
	}
	return m
}

func TestMessageRepo_BatchCreate_ListByUser(t *testing.T) {
	db := newMessageTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	require.NoError(t, repo.BatchCreate(ctx, []*metermodel.Message{
		newMsg("m1", "u1", metermodel.CategoryTask, false, now.Add(-3*time.Minute)),
		newMsg("m2", "u1", metermodel.CategoryTask, true, now.Add(-2*time.Minute)),
		newMsg("m3", "u1", metermodel.CategorySystem, false, now.Add(-time.Minute)),
		newMsg("m4", "u2", metermodel.CategoryTask, false, now),
	}))

	// 空列表不报错
	require.NoError(t, repo.BatchCreate(ctx, nil))

	// 只返回本人的消息，按创建时间倒序
	items, total, err := repo.ListByUser(ctx, ListFilter{UserID: "u1", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, items, 3)
	require.Equal(t, "m3", items[0].ID)
	require.Equal(t, "m1", items[2].ID)

	// 分类过滤
	items, total, err = repo.ListByUser(ctx, ListFilter{UserID: "u1", Category: metermodel.CategoryTask, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 2)

	// 仅未读
	items, total, err = repo.ListByUser(ctx, ListFilter{UserID: "u1", UnreadOnly: true, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, items, 2)

	// 关键字命中标题
	items, total, err = repo.ListByUser(ctx, ListFilter{UserID: "u1", Keyword: "m2", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "m2", items[0].ID)
	require.NotNil(t, items[0].ReadAt, "已读时间应随领域模型返回")

	// 分页
	items, _, err = repo.ListByUser(ctx, ListFilter{UserID: "u1", Page: 2, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func TestMessageRepo_UnreadCounts(t *testing.T) {
	db := newMessageTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	require.NoError(t, repo.BatchCreate(ctx, []*metermodel.Message{
		newMsg("m1", "u1", metermodel.CategoryTask, false, now),
		newMsg("m2", "u1", metermodel.CategoryTask, false, now),
		newMsg("m3", "u1", metermodel.CategorySystem, false, now),
		newMsg("m4", "u1", metermodel.CategorySystem, true, now),
		newMsg("m5", "u2", metermodel.CategoryAnnouncement, false, now),
	}))

	n, err := repo.CountUnread(ctx, "u1", "")
	require.NoError(t, err)
	require.Equal(t, int64(3), n)

	n, err = repo.CountUnread(ctx, "u1", metermodel.CategoryTask)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)

	byUser, err := repo.CountUnreadByUsers(ctx, []string{"u1", "u2", "u3"})
	require.NoError(t, err)
	require.Equal(t, int64(3), byUser["u1"])
	require.Equal(t, int64(1), byUser["u2"])
	_, ok := byUser["u3"]
	require.False(t, ok, "无未读的收件人不出现在结果里")

	byUser, err = repo.CountUnreadByUsers(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, byUser)

	byCategory, err := repo.CountUnreadByCategory(ctx, "u1")
	require.NoError(t, err)
	require.Equal(t, int64(2), byCategory[metermodel.CategoryTask])
	require.Equal(t, int64(1), byCategory[metermodel.CategorySystem])
}

// TestMessageRepo_MarkRead_Idempotent 重复标记不报错且保留首次阅读时间。
func TestMessageRepo_MarkRead_Idempotent(t *testing.T) {
	db := newMessageTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.BatchCreate(ctx, []*metermodel.Message{
		newMsg("m1", "u1", metermodel.CategoryTask, false, time.Now()),
	}))

	require.NoError(t, repo.MarkRead(ctx, "u1", "m1"))
	var first UserMessagePO
	require.NoError(t, db.Where("id = ?", "m1").First(&first).Error)
	require.NotNil(t, first.ReadAt)

	// 第二次标记：幂等成功，时间不被刷新
	require.NoError(t, repo.MarkRead(ctx, "u1", "m1"))
	var second UserMessagePO
	require.NoError(t, db.Where("id = ?", "m1").First(&second).Error)
	require.True(t, first.ReadAt.Equal(*second.ReadAt), "重复标记应保留首次阅读时间")
}

// TestMessageRepo_MarkRead_OtherUser 非本人标记返回未找到，不泄露消息是否存在。
func TestMessageRepo_MarkRead_OtherUser(t *testing.T) {
	db := newMessageTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.BatchCreate(ctx, []*metermodel.Message{
		newMsg("m1", "u1", metermodel.CategoryTask, false, time.Now()),
	}))

	err := repo.MarkRead(ctx, "u2", "m1")
	require.True(t, errors.Is(err, ErrMessageNotFound))

	var po UserMessagePO
	require.NoError(t, db.Where("id = ?", "m1").First(&po).Error)
	require.Nil(t, po.ReadAt, "他人标记不得改变已读状态")

	err = repo.MarkRead(ctx, "u1", "missing")
	require.True(t, errors.Is(err, ErrMessageNotFound))
}

func TestMessageRepo_MarkAllRead(t *testing.T) {
	db := newMessageTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)
	require.NoError(t, repo.BatchCreate(ctx, []*metermodel.Message{
		newMsg("m1", "u1", metermodel.CategoryTask, false, now),
		newMsg("m2", "u1", metermodel.CategoryTask, false, now),
		newMsg("m3", "u1", metermodel.CategorySystem, false, now),
		newMsg("m4", "u2", metermodel.CategoryTask, false, now),
	}))

	n, err := repo.MarkAllRead(ctx, "u1", metermodel.CategoryTask)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)

	n, err = repo.CountUnread(ctx, "u1", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), n, "其他分类的未读不受影响")

	n, err = repo.MarkAllRead(ctx, "u2", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
}

func TestMessageRepo_Delete(t *testing.T) {
	db := newMessageTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.BatchCreate(ctx, []*metermodel.Message{
		newMsg("m1", "u1", metermodel.CategoryTask, false, time.Now()),
	}))

	require.True(t, errors.Is(repo.Delete(ctx, "u2", "m1"), ErrMessageNotFound), "他人消息不可删除")
	require.NoError(t, repo.Delete(ctx, "u1", "m1"))

	var n int64
	require.NoError(t, db.Model(&UserMessagePO{}).Where("id = ?", "m1").Count(&n).Error)
	require.Zero(t, n)

	require.True(t, errors.Is(repo.Delete(ctx, "u1", "m1"), ErrMessageNotFound))
}

func TestMessageRepo_DeleteOlderThan(t *testing.T) {
	db := newMessageTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)
	require.NoError(t, repo.BatchCreate(ctx, []*metermodel.Message{
		newMsg("old", "u1", metermodel.CategoryTask, false, now.AddDate(0, 0, -120)),
		newMsg("fresh", "u1", metermodel.CategoryTask, false, now),
	}))

	n, err := repo.DeleteOlderThan(ctx, now.AddDate(0, 0, -90))
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	var left []UserMessagePO
	require.NoError(t, db.Find(&left).Error)
	require.Len(t, left, 1)
	require.Equal(t, "fresh", left[0].ID)
}

func TestUserDirectory_ListActiveUserIDs(t *testing.T) {
	db := newMessageTestDB(t)
	// users 表由 user 模块维护，这里只建最小列以验证查询条件
	require.NoError(t, db.Exec(`CREATE TABLE users (
		id TEXT PRIMARY KEY, tenant_id TEXT, status INTEGER, deleted_at DATETIME, created_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO users (id, tenant_id, status, deleted_at, created_at) VALUES
		('u1', 't1', 1, NULL, '2026-01-01 00:00:00'),
		('u2', 't1', 0, NULL, '2026-01-02 00:00:00'),
		('u3', 't1', 1, '2026-01-03 00:00:00', '2026-01-03 00:00:00'),
		('u4', 't2', 1, NULL, '2026-01-04 00:00:00')`).Error)

	dir := NewUserDirectory(db)
	ids, err := dir.ListActiveUserIDs(context.Background(), "t1", 10)
	require.NoError(t, err)
	require.Equal(t, []string{"u1"}, ids, "仅返回同租户、启用且未软删除的用户")

	ids, err = dir.ListActiveUserIDs(context.Background(), "t1", 0)
	require.NoError(t, err)
	require.Empty(t, ids, "limit<=0 视为无收件人，不展开全量用户")

	ids, err = dir.ListActiveUserIDs(context.Background(), "", 10)
	require.NoError(t, err)
	require.Empty(t, ids)
}

// TestMessageRepo_BatchCreate_DedupeKey 同一收件人 + 同一去重键只落一条，
// 不同收件人各自一条；无键（存 NULL）的消息不受唯一约束影响。
func TestMessageRepo_BatchCreate_DedupeKey(t *testing.T) {
	db := newMessageTestDB(t)
	repo := NewMessageRepository(db)
	ctx := context.Background()
	now := time.Now().Truncate(time.Millisecond)

	withKey := func(id, userID, key string) *metermodel.Message {
		m := newMsg(id, userID, metermodel.CategoryTask, false, now)
		m.DedupeKey = key
		return m
	}

	key := "task:tk1:task_due:20261025"
	require.NoError(t, repo.BatchCreate(ctx, []*metermodel.Message{
		withKey("m1", "u1", key),
		withKey("m2", "u1", key),
	}))
	require.NoError(t, repo.BatchCreate(ctx, []*metermodel.Message{
		withKey("m3", "u2", key),
	}))

	var u1 []UserMessagePO
	require.NoError(t, db.Where("user_id = ?", "u1").Find(&u1).Error)
	require.Len(t, u1, 1, "同一收件人的重复事件只保留一条")
	require.Equal(t, "m1", u1[0].ID, "先到的一条胜出")
	require.NotNil(t, u1[0].DedupeKey)
	require.Equal(t, key, *u1[0].DedupeKey)

	var u2 []UserMessagePO
	require.NoError(t, db.Where("user_id = ?", "u2").Find(&u2).Error)
	require.Len(t, u2, 1, "去重按收件人隔离，不跨人合并")

	// 无去重键：NULL 不受唯一约束，多条都能写入
	require.NoError(t, repo.BatchCreate(ctx, []*metermodel.Message{
		newMsg("n1", "u3", metermodel.CategoryTask, false, now),
		newMsg("n2", "u3", metermodel.CategoryTask, false, now),
	}))
	var u3 []UserMessagePO
	require.NoError(t, db.Where("user_id = ?", "u3").Find(&u3).Error)
	require.Len(t, u3, 2, "不参与去重的消息不应互相冲突")
	for _, po := range u3 {
		require.Nil(t, po.DedupeKey, "空键应存为 NULL")
	}
}
