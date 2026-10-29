package repository

import (
	"context"

	"meteorx/internal/modules/settings/model"
)

// SettingRepository 平台运行时配置数据访问层接口
type SettingRepository interface {
	// GetAll 获取全部配置行
	GetAll(ctx context.Context) ([]*model.Setting, error)
	// GetByKey 按键查询配置，不存在时返回 nil（非错误）
	GetByKey(ctx context.Context, key string) (*model.Setting, error)
	// Create 新增配置行
	Create(ctx context.Context, setting *model.Setting) error
	// UpdateValue 仅更新配置值与更新时间
	UpdateValue(ctx context.Context, key string, value string) error
	// Upsert 存在则更新值、不存在则按传入元数据创建
	Upsert(ctx context.Context, setting *model.Setting) error
}
