// Package repository 提供站内消息的 GORM 实现。
package repository

import (
	"context"
	"errors"
	metermodel "meteorx/internal/modules/message/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrMessageNotFound 消息不存在或不属于当前收件人。
// 用同一条错误覆盖「不存在」与「非本人」两种情形，避免通过报错差异探测他人消息。
var ErrMessageNotFound = errors.New("message not found")

// userStatusEnabled 用户状态：1 正常 / 0 禁用（与 user 模块口径一致）。
const userStatusEnabled = 1

// UserMessagePO 站内消息数据库模型。
type UserMessagePO struct {
	ID         string `gorm:"primaryKey;size:26;comment:消息ID"`
	TenantID   string `gorm:"size:26;index:idx_user_messages_inbox,priority:1;comment:租户ID"`
	UserID     string `gorm:"size:26;index:idx_user_messages_inbox,priority:2;uniqueIndex:idx_user_messages_dedupe,priority:1;comment:收件人ID"`
	Category   string `gorm:"size:32;index;comment:分类"`
	Title      string `gorm:"size:200;comment:标题"`
	Content    string `gorm:"type:text;comment:正文"`
	Link       string `gorm:"size:255;comment:前端跳转路由"`
	Priority   int    `gorm:"default:1;comment:优先级"`
	SourceType string `gorm:"size:32;index;comment:来源类型"`
	SourceID   string `gorm:"size:64;index;comment:来源ID"`
	// DedupeKey 去重键，与 user_id 组成唯一索引。
	// 为空时存 NULL：MySQL / SQLite 的唯一索引都允许多个 NULL，
	// 因此「不参与去重」的消息与 AutoMigrate 补列的历史行都不会被误伤。
	DedupeKey *string    `gorm:"size:191;uniqueIndex:idx_user_messages_dedupe,priority:2;comment:去重键"`
	ReadAt    *time.Time `gorm:"index;comment:已读时间"`
	CreatedAt time.Time  `gorm:"autoCreateTime;index:idx_user_messages_inbox,priority:3;comment:创建时间"`
}

// TableName 返回站内消息表名 user_messages。
func (UserMessagePO) TableName() string { return "user_messages" }

// toDomain 转换为领域模型。
func (po UserMessagePO) toDomain() *metermodel.Message {
	m := &metermodel.Message{
		ID:         po.ID,
		TenantID:   po.TenantID,
		UserID:     po.UserID,
		Category:   po.Category,
		Title:      po.Title,
		Content:    po.Content,
		Link:       po.Link,
		Priority:   po.Priority,
		SourceType: po.SourceType,
		SourceID:   po.SourceID,
		ReadAt:     po.ReadAt,
		CreatedAt:  po.CreatedAt,
	}
	if po.DedupeKey != nil {
		m.DedupeKey = *po.DedupeKey
	}
	return m
}

// nullableKey 把空去重键存为 NULL：唯一索引对 NULL 不做唯一约束，
// 若存空串，同一收件人的多条「不参与去重」消息会互相冲突。
func nullableKey(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// messageFromDomain 从领域模型转换。
func messageFromDomain(m *metermodel.Message) *UserMessagePO {
	return &UserMessagePO{
		ID:         m.ID,
		TenantID:   m.TenantID,
		UserID:     m.UserID,
		Category:   m.Category,
		Title:      m.Title,
		Content:    m.Content,
		Link:       m.Link,
		Priority:   m.Priority,
		SourceType: m.SourceType,
		SourceID:   m.SourceID,
		DedupeKey:  nullableKey(m.DedupeKey),
		ReadAt:     m.ReadAt,
		CreatedAt:  m.CreatedAt,
	}
}

type messageRepository struct {
	db *gorm.DB
}

// NewMessageRepository 创建站内消息仓储。
func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

// Models 返回本模块全部待迁移模型，作为 AutoMigrate 与脏列巡检的单一来源。
func Models() []interface{} {
	return []interface{}{&UserMessagePO{}}
}

// AutoMigrate 自动迁移表结构。
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(Models()...)
}

// BatchCreate 批量写入消息。
// 唯一索引 (user_id, dedupe_key) 命中的行被静默跳过：重试、多副本与重启补偿
// 可能重复投递同一个事件，收件箱只应保留一条。
func (r *messageRepository) BatchCreate(ctx context.Context, msgs []*metermodel.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	pos := make([]*UserMessagePO, len(msgs))
	for i, m := range msgs {
		pos[i] = messageFromDomain(m)
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(pos, 200).Error
}

// ListByUser 分页查询收件人的消息。
func (r *messageRepository) ListByUser(ctx context.Context, f ListFilter) ([]*metermodel.Message, int64, error) {
	db := r.db.WithContext(ctx).Model(&UserMessagePO{}).Where("user_id = ?", f.UserID)
	if f.Category != "" {
		db = db.Where("category = ?", f.Category)
	}
	if f.UnreadOnly {
		db = db.Where("read_at IS NULL")
	}
	if f.Keyword != "" {
		db = db.Where("title LIKE ?", "%"+f.Keyword+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var pos []UserMessagePO
	if err := db.Order("created_at DESC").Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*metermodel.Message, len(pos))
	for i, po := range pos {
		items[i] = po.toDomain()
	}
	return items, total, nil
}

// CountUnread 统计收件人未读数。
func (r *messageRepository) CountUnread(ctx context.Context, userID, category string) (int64, error) {
	db := r.db.WithContext(ctx).Model(&UserMessagePO{}).Where("user_id = ? AND read_at IS NULL", userID)
	if category != "" {
		db = db.Where("category = ?", category)
	}
	var n int64
	err := db.Count(&n).Error
	return n, err
}

// CountUnreadByUsers 按收件人分组统计未读数。
func (r *messageRepository) CountUnreadByUsers(ctx context.Context, userIDs []string) (map[string]int64, error) {
	result := make(map[string]int64, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	type row struct {
		UserID string
		Total  int64
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&UserMessagePO{}).
		Select("user_id, COUNT(*) AS total").
		Where("user_id IN ? AND read_at IS NULL", userIDs).
		Group("user_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, it := range rows {
		result[it.UserID] = it.Total
	}
	return result, nil
}

// CountUnreadByCategory 返回收件人各分类未读数。
func (r *messageRepository) CountUnreadByCategory(ctx context.Context, userID string) (map[string]int64, error) {
	type row struct {
		Category string
		Total    int64
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&UserMessagePO{}).
		Select("category, COUNT(*) AS total").
		Where("user_id = ? AND read_at IS NULL", userID).
		Group("category").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, it := range rows {
		result[it.Category] = it.Total
	}
	return result, nil
}

// MarkRead 将单条消息标记已读。
// 已读的消息不重复更新：WHERE 带 read_at IS NULL，保留首次阅读时间。
func (r *messageRepository) MarkRead(ctx context.Context, userID, id string) error {
	res := r.db.WithContext(ctx).Model(&UserMessagePO{}).
		Where("id = ? AND user_id = ? AND read_at IS NULL", id, userID).
		Update("read_at", time.Now())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// 可能是不存在/非本人，也可能是已读（已读行不会被再次更新）
		var n int64
		if err := r.db.WithContext(ctx).Model(&UserMessagePO{}).
			Where("id = ? AND user_id = ?", id, userID).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return ErrMessageNotFound
		}
	}
	return nil
}

// MarkAllRead 将收件人未读消息全部标记已读。
func (r *messageRepository) MarkAllRead(ctx context.Context, userID, category string) (int64, error) {
	db := r.db.WithContext(ctx).Model(&UserMessagePO{}).
		Where("user_id = ? AND read_at IS NULL", userID)
	if category != "" {
		db = db.Where("category = ?", category)
	}
	res := db.Update("read_at", time.Now())
	return res.RowsAffected, res.Error
}

// Delete 删除收件人的单条消息。
func (r *messageRepository) Delete(ctx context.Context, userID, id string) error {
	res := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&UserMessagePO{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrMessageNotFound
	}
	return nil
}

// DeleteOlderThan 删除创建时间早于 cut 的消息。
func (r *messageRepository) DeleteOlderThan(ctx context.Context, cut time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Where("created_at < ?", cut).Delete(&UserMessagePO{})
	return res.RowsAffected, res.Error
}
