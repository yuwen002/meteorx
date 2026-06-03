package repository

import (
	"context"
	"meteorx/internal/modules/user/model"
)

type UserRepository interface {
	// Create 创建用户
	Create(ctx context.Context, user *model.User) error
	// GetByUsername 用于登录校验：根据租户ID和用户名查询唯一用户
	GetByUsername(ctx context.Context, tenantID, username string) (*model.User, error)
	// GetByID 根据ID查询用户
	GetByID(ctx context.Context, id string) (*model.User, error)
}
