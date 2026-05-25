package repository

import (
	"context"

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
}
