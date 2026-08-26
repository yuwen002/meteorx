package repository

import (
	"context"
	userModel "meteorx/internal/modules/user/model"
	"time"

	"meteorx/internal/modules/tenant/model"
)

// TenantRepository 定义了租户数据访问层接口
// 业务层通过此接口与持久化层交互，不依赖具体实现
type TenantRepository interface {
	// GetByID 根据租户 ID 查询租户信息
	GetByID(ctx context.Context, id string) (*model.Tenant, error)
	// Create 创建新的租户记录
	Create(ctx context.Context, tenant *model.Tenant) error
	// GetByDomain 根据域名查询租户信息
	GetByDomain(ctx context.Context, domain string) (*model.Tenant, error)
	// GetByName 根据租户名称查询租户信息
	GetByName(ctx context.Context, name string) (*model.Tenant, error)
	// CreateTenantWithAdmin 在事务中同时创建租户和管理员用户
	CreateTenantWithAdmin(ctx context.Context, tenant *model.Tenant, user *userModel.User) error
	// UpdateStatus 更新租户状态
	UpdateStatus(ctx context.Context, id string, status int) error
	// Update 更新租户信息
	Update(ctx context.Context, id string, tenant *model.Tenant) error
	// Delete 软删除租户
	Delete(ctx context.Context, id string) error
	// HardDelete 物理删除租户（彻底删除，不可恢复）
	HardDelete(ctx context.Context, id string) error
	// FindPage 分页查询租户列表
	FindPage(ctx context.Context, page, pageSize int, name string, status *int) ([]*model.Tenant, int64, error)
	// BatchUpdateStatus 批量更新租户状态，返回受影响行数和失败的ID列表
	BatchUpdateStatus(ctx context.Context, ids []string, status int) (int64, []string, error)
	// BatchDelete 批量软删除租户，返回受影响行数和失败的ID列表
	BatchDelete(ctx context.Context, ids []string) (int64, []string, error)
	// FindDeleted 分页查询已软删除的租户列表
	FindDeleted(ctx context.Context, page, pageSize int, name string) ([]*model.Tenant, int64, error)
	// Restore 恢复已软删除的租户
	Restore(ctx context.Context, id string) error

	// CreateCancelRequest 创建注销申请
	CreateCancelRequest(ctx context.Context, req *model.CancelRequest) error
	// GetCancelRequestByID 根据ID查询注销申请
	GetCancelRequestByID(ctx context.Context, id string) (*model.CancelRequest, error)
	// GetPendingCancelRequestByTenant 查询租户未处理的注销申请
	GetPendingCancelRequestByTenant(ctx context.Context, tenantID string) (*model.CancelRequest, error)
	// UpdateCancelRequest 更新注销申请
	UpdateCancelRequest(ctx context.Context, req *model.CancelRequest) error
	// FindCancelRequests 分页查询注销申请
	FindCancelRequests(ctx context.Context, page, pageSize int, status int, keyword string) ([]*model.CancelRequest, int64, error)
	// FindApprovedDueCancelRequests 查询所有已到期可执行的注销申请
	FindApprovedDueCancelRequests(ctx context.Context, now time.Time) ([]*model.CancelRequest, error)
}