// Package repository 平台运行时配置数据访问层，基于 GORM 实现 SettingRepository 接口。
package repository

import (
	"context"
	"errors"
	"time"

	"meteorx/internal/modules/settings/model"

	"gorm.io/gorm"
)

// 确保实现接口
var _ SettingRepository = (*settingRepository)(nil)

// SettingPO 系统配置持久化模型（KV + 元数据，每键一行）。
// 键列命名 config_key：规避 MySQL 保留字 key。
type SettingPO struct {
	Key       string    `gorm:"primaryKey;size:100;column:config_key;comment:'配置键，全局唯一'"`
	Value     string    `gorm:"type:text;comment:'配置值（按 value_type 字符串化）'"`
	ValueType string    `gorm:"size:20;comment:'值类型：bool/int/string/duration'"`
	IsPublic  bool      `gorm:"comment:'是否随公开接口暴露给未登录前端'"`
	CreatedAt time.Time `gorm:"comment:'创建时间'"`
	UpdatedAt time.Time `gorm:"comment:'更新时间'"`
}

// TableName 返回系统配置表名 system_settings。
func (SettingPO) TableName() string { return "system_settings" }

// Models 返回本模块全部待迁移模型，作为 AutoMigrate 的单一模型来源。
func Models() []interface{} {
	return []interface{}{&SettingPO{}}
}

// AutoMigrate 系统配置模块数据库迁移
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(Models()...)
}

// toDomain 将持久化模型转换为领域模型。
func (po SettingPO) toDomain() *model.Setting {
	return &model.Setting{
		Key:       po.Key,
		Value:     po.Value,
		ValueType: po.ValueType,
		IsPublic:  po.IsPublic,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}
}

// settingRepository SettingRepository 的 GORM 实现。
type settingRepository struct {
	db *gorm.DB
}

// NewSettingRepository 创建系统配置仓库实例
func NewSettingRepository(db *gorm.DB) SettingRepository {
	return &settingRepository{db: db}
}

// GetAll 获取全部配置行
func (r *settingRepository) GetAll(ctx context.Context) ([]*model.Setting, error) {
	var pos []SettingPO
	if err := r.db.WithContext(ctx).Find(&pos).Error; err != nil {
		return nil, err
	}
	settings := make([]*model.Setting, 0, len(pos))
	for i := range pos {
		settings = append(settings, pos[i].toDomain())
	}
	return settings, nil
}

// GetByKey 按键查询配置，不存在时返回 nil（非错误），便于上层判断"不存在"
func (r *settingRepository) GetByKey(ctx context.Context, key string) (*model.Setting, error) {
	var po SettingPO
	err := r.db.WithContext(ctx).Where("config_key = ?", key).First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return po.toDomain(), nil
}

// Create 新增配置行
func (r *settingRepository) Create(ctx context.Context, setting *model.Setting) error {
	po := &SettingPO{
		Key:       setting.Key,
		Value:     setting.Value,
		ValueType: setting.ValueType,
		IsPublic:  setting.IsPublic,
	}
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}
	setting.CreatedAt = po.CreatedAt
	setting.UpdatedAt = po.UpdatedAt
	return nil
}

// UpdateValue 仅更新配置值与更新时间
func (r *settingRepository) UpdateValue(ctx context.Context, key string, value string) error {
	return r.db.WithContext(ctx).Model(&SettingPO{}).
		Where("config_key = ?", key).
		Updates(map[string]interface{}{"value": value, "updated_at": time.Now()}).Error
}

// Upsert 存在则仅更新值、不存在则按传入元数据创建，保证配置写入的幂等性。
func (r *settingRepository) Upsert(ctx context.Context, setting *model.Setting) error {
	existing, err := r.GetByKey(ctx, setting.Key)
	if err != nil {
		return err
	}
	if existing == nil {
		return r.Create(ctx, setting)
	}
	return r.UpdateValue(ctx, setting.Key, setting.Value)
}
