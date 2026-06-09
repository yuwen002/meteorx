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
	// UsernameExists 全局检查用户名是否已存在（跨所有租户）
	UsernameExists(ctx context.Context, username string) (bool, error)
	// ListByTenant 根据租户ID查询用户列表（支持分页）
	ListByTenant(ctx context.Context, tenantID string, page, pageSize int) ([]*model.User, int64, error)
	// ListMasterAdmins 查询所有系统管理员（is_master = true，支持分页和关键词搜索）
	ListMasterAdmins(ctx context.Context, page, pageSize int, keyword string) ([]*model.User, int64, error)
	// Update 更新用户信息
	Update(ctx context.Context, user *model.User) error
	// Delete 删除用户（软删除）
	Delete(ctx context.Context, id string) error
}