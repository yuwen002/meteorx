// Package repository 定义用户模块的数据访问接口和 GORM 实现。
package repository

import (
	"context"
	"meteorx/internal/modules/user/model"
	"time"
)

// UserListFilter 租户内用户列表查询过滤条件（空零值均表示该维度不设限）。
type UserListFilter struct {
	Keyword     string     // 按用户名/昵称/邮箱模糊搜索
	Status      *int       // 按状态筛选
	RoleID      string     // 按角色筛选：仅返回拥有该角色的用户（user_roles 关联）
	SortBy      string     // 排序列：created_at/updated_at/username/status/email/last_login_at；空=created_at
	Desc        bool       // 排序方向，true 为降序
	CreatedFrom *time.Time // 注册时间下限（含），nil 不限
	CreatedTo   *time.Time // 注册时间上限（含），nil 不限
}

// UserRepository 用户数据访问接口，覆盖用户及系统管理员的增删改查、状态/回收站与批量操作。
type UserRepository interface {
	// Create 创建用户
	Create(ctx context.Context, user *model.User) error
	// GetByUsername 用于登录校验：根据租户ID和用户名查询唯一用户
	GetByUsername(ctx context.Context, tenantID, username string) (*model.User, error)
	// GetByID 根据ID查询用户
	GetByID(ctx context.Context, id string) (*model.User, error)
	// GetByEmail 根据邮箱查询用户
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	// UsernameExists 全局检查用户名是否已存在（跨所有租户）
	UsernameExists(ctx context.Context, username string) (bool, error)
	// ListByTenant 根据租户ID查询用户列表（支持分页和状态筛选）
	ListByTenant(ctx context.Context, tenantID string, page, pageSize int, keyword string, status *int) ([]*model.User, int64, error)
	// ListByTenantFiltered 按多维过滤条件（角色/排序/注册时间区间等）分页查询租户内用户
	ListByTenantFiltered(ctx context.Context, tenantID string, page, pageSize int, f UserListFilter) ([]*model.User, int64, error)
	// ListMasterAdmins 查询所有系统管理员（is_master = true，支持分页和关键词搜索）
	ListMasterAdmins(ctx context.Context, page, pageSize int, keyword string) ([]*model.User, int64, error)
	// ListAllTenantUsers 查询所有租户用户（不包括系统管理员，支持分页和关键词搜索）
	ListAllTenantUsers(ctx context.Context, page, pageSize int, keyword string) ([]*model.User, int64, error)
	// Update 更新用户信息
	Update(ctx context.Context, user *model.User) error
	// Delete 删除用户（软删除）
	Delete(ctx context.Context, id string) error
	// UpdateStatus 更新用户状态
	UpdateStatus(ctx context.Context, id string, status int) error
	// FindDeletedMasterAdmins 查询已删除的系统管理员列表
	FindDeletedMasterAdmins(ctx context.Context, page, pageSize int, keyword string) ([]*model.User, int64, error)
	// RestoreMasterAdmin 恢复已删除的系统管理员
	RestoreMasterAdmin(ctx context.Context, id string) error
	// PermanentDeleteMasterAdmin 永久删除系统管理员（物理删除）
	PermanentDeleteMasterAdmin(ctx context.Context, id string) error
	// BatchUpdateStatus 批量更新系统管理员状态
	BatchUpdateStatus(ctx context.Context, ids []string, status int) (int64, error)
	// BatchDelete 批量删除系统管理员
	BatchDelete(ctx context.Context, ids []string) (int64, error)

	// FindDeletedTenantUsers 查询指定租户的已删除用户列表（回收站）
	FindDeletedTenantUsers(ctx context.Context, tenantID string, page, pageSize int, keyword string) ([]*model.User, int64, error)
	// FindAllDeletedTenantUsers 查询所有租户的已删除用户列表（回收站，排除系统管理员）
	FindAllDeletedTenantUsers(ctx context.Context, page, pageSize int, keyword string) ([]*model.User, int64, error)
	// RestoreTenantUser 恢复已删除的租户用户
	RestoreTenantUser(ctx context.Context, tenantID, userID string) error
	// PermanentDeleteTenantUser 永久删除租户用户（物理删除）
	PermanentDeleteTenantUser(ctx context.Context, tenantID, userID string) error
	// BatchUpdateTenantUserStatus 批量更新租户用户状态（指定租户，排除系统管理员）
	BatchUpdateTenantUserStatus(ctx context.Context, tenantID string, ids []string, status int) (int64, error)
	// BatchDeleteTenantUsers 批量删除租户用户（指定租户）
	BatchDeleteTenantUsers(ctx context.Context, tenantID string, ids []string) (int64, error)

	// CountByTenant 统计指定租户下的用户总数
	CountByTenant(ctx context.Context, tenantID string) (int64, error)
	// CountAllUsers 统计所有用户总数（跨租户）
	CountAllUsers(ctx context.Context) (int64, error)
	// UpdateEmailVerified 更新用户邮箱验证状态
	UpdateEmailVerified(ctx context.Context, userID string, verified bool) error
	// GetByPhone 根据手机号查询用户
	GetByPhone(ctx context.Context, phone string) (*model.User, error)
	// UpdateLastLogin 回写用户最后登录时间
	UpdateLastLogin(ctx context.Context, userID string) error
	// UpdateTOTP 更新用户双因素认证密钥与启用状态（secret 为空且 enabled=false 表示解除绑定）
	UpdateTOTP(ctx context.Context, userID, secret string, enabled bool) error
	// ReplaceRecoveryCodes 用给定哈希集合覆盖用户的全部备份恢复码（先清空再写入）；hashes 为空表示仅清空。
	ReplaceRecoveryCodes(ctx context.Context, userID string, hashes []string) error
	// ConsumeRecoveryCode 校验并核销一条未使用的恢复码：命中则标记核销时间并返回 true，未命中返回 false。
	ConsumeRecoveryCode(ctx context.Context, userID, codeHash string) (bool, error)
	// CountRecoveryCodes 统计用户剩余未使用的备份恢复码数量。
	CountRecoveryCodes(ctx context.Context, userID string) (int64, error)
}
