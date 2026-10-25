package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"meteorx/internal/modules/message/model"
	"meteorx/pkg/idgen"
)

// UserMessagePreferencePO 用户消息偏好数据库模型。
// 只为「关闭接收」的分类建行：缺行即默认接收，
// 这样新分类上线时不必给存量用户补行，也不会出现「无偏好」与「全部接收」两种语义。
type UserMessagePreferencePO struct {
	ID        string    `gorm:"primaryKey;size:26;comment:偏好ID"`
	UserID    string    `gorm:"size:26;uniqueIndex:idx_user_message_prefs,priority:1;comment:用户ID"`
	Category  string    `gorm:"size:32;uniqueIndex:idx_user_message_prefs,priority:2;comment:分类"`
	Muted     bool      `gorm:"not null;default:false;comment:是否关闭接收"`
	CreatedAt time.Time `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;comment:更新时间"`
}

// TableName 返回用户消息偏好表名 user_message_preferences。
func (UserMessagePreferencePO) TableName() string { return "user_message_preferences" }

type preferenceRepository struct {
	db *gorm.DB
}

// NewPreferenceRepository 创建用户消息偏好仓储。
func NewPreferenceRepository(db *gorm.DB) PreferenceRepository {
	return &preferenceRepository{db: db}
}

// MutedCategories 返回收件人关闭接收的分类集合。
// 只认 muted = true 的行：若有调用方把开关改回「接收」而留下 false 行，
// 语义上也应当是「未关闭」，不能因为存在记录就当成已关闭。
func (r *preferenceRepository) MutedCategories(ctx context.Context, userID string) (map[string]bool, error) {
	out := make(map[string]bool)
	if userID == "" {
		return out, nil
	}
	var rows []UserMessagePreferencePO
	if err := r.db.WithContext(ctx).Model(&UserMessagePreferencePO{}).
		Select("category").
		Where("user_id = ? AND muted = ?", userID, true).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, po := range rows {
		out[po.Category] = true
	}
	return out, nil
}

// MutedUserIDs 从给定收件人中筛出关闭了该分类的用户。
// 一次查询覆盖整批收件人：站内信落库本就是批量写，逐人判定会把写放大成 N+1。
func (r *preferenceRepository) MutedUserIDs(ctx context.Context, category string, userIDs []string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	if category == "" || len(userIDs) == 0 {
		return out, nil
	}
	var rows []UserMessagePreferencePO
	if err := r.db.WithContext(ctx).Model(&UserMessagePreferencePO{}).
		Select("user_id").
		Where("category = ? AND muted = ? AND user_id IN ?", category, true, userIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, po := range rows {
		out[po.UserID] = struct{}{}
	}
	return out, nil
}

// Upsert 覆盖写入偏好（幂等）。
// 唯一索引 (user_id, category) 命中时只更新 muted，因此重复提交同一份偏好不会堆行。
// 入参里同一分类重复出现时行为未定义（MySQL 顺序更新 / SQLite 报错），
// 由服务层先行拒绝重复分类。
func (r *preferenceRepository) Upsert(ctx context.Context, userID string, prefs []model.CategoryPreference) error {
	if userID == "" || len(prefs) == 0 {
		return nil
	}
	now := time.Now()
	pos := make([]*UserMessagePreferencePO, len(prefs))
	for i, p := range prefs {
		pos[i] = &UserMessagePreferencePO{
			ID:        idgen.New(),
			UserID:    userID,
			Category:  p.Category,
			Muted:     p.Muted,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "category"}},
			DoUpdates: clause.AssignmentColumns([]string{"muted", "updated_at"}),
		}).
		Create(&pos).Error
}
