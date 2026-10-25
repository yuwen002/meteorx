package repository

import (
	"context"
	"meteorx/internal/modules/message/model"
	"time"
)

// ListFilter 收件箱列表过滤条件。
type ListFilter struct {
	UserID     string // 收件人（必填）
	Category   string // 分类，空表示不限
	UnreadOnly bool   // 仅看未读
	Keyword    string // 标题关键字
	Page       int
	PageSize   int
}

// MessageRepository 站内消息仓储。
// 所有查询都以「收件人」为第一过滤维度：站内消息天然按人隔离，
// 调用方不需要也不应该依赖租户过滤来隔离已读状态。
type MessageRepository interface {
	// BatchCreate 批量写入消息（单事件多收件人），空切片直接返回 nil。
	BatchCreate(ctx context.Context, msgs []*model.Message) error
	// ListByUser 按过滤条件分页查询收件人的消息，返回列表与总数。
	ListByUser(ctx context.Context, f ListFilter) ([]*model.Message, int64, error)
	// CountUnread 统计收件人未读数；category 为空表示全部。
	CountUnread(ctx context.Context, userID, category string) (int64, error)
	// CountUnreadByUsers 按收件人分组统计未读数（用于落库后推送未读角标）。
	CountUnreadByUsers(ctx context.Context, userIDs []string) (map[string]int64, error)
	// CountUnreadByCategory 返回收件人各分类未读数。
	CountUnreadByCategory(ctx context.Context, userID string) (map[string]int64, error)
	// MarkRead 将指定消息标记为已读，不存在或不属于该用户时返回 ErrMessageNotFound。
	MarkRead(ctx context.Context, userID, id string) error
	// MarkAllRead 将收件人的未读消息全部标记已读，返回影响行数。
	MarkAllRead(ctx context.Context, userID, category string) (int64, error)
	// Delete 删除收件人的单条消息（物理删除：消息无回收站语义）。
	Delete(ctx context.Context, userID, id string) error
	// DeleteOlderThan 删除创建时间早于 cut 的消息，返回删除行数。
	DeleteOlderThan(ctx context.Context, cut time.Time) (int64, error)
}

// UserDirectory 按租户展开收件人的最小依赖面。
// 声明为接口以便测试注入桩，避免服务层直接依赖 user 模块。
type UserDirectory interface {
	// ListActiveUserIDs 返回租户内启用状态的用户 ID（上限 limit）。
	ListActiveUserIDs(ctx context.Context, tenantID string, limit int) ([]string, error)
}

// PreferenceRepository 用户消息偏好仓储。
// 偏好按「用户 × 分类」存：没有记录即默认接收，只有关闭接收才建行。
type PreferenceRepository interface {
	// MutedCategories 返回收件人关闭接收的分类集合。
	MutedCategories(ctx context.Context, userID string) (map[string]bool, error)
	// Upsert 覆盖写入偏好（幂等，同一分类只保留一行）。
	Upsert(ctx context.Context, userID string, prefs []model.CategoryPreference) error
	// MutedUserIDs 从给定收件人中筛出关闭了该分类的用户（批量判定，避免 N+1）。
	MutedUserIDs(ctx context.Context, category string, userIDs []string) (map[string]struct{}, error)
}
