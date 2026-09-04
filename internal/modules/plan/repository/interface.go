package repository

import (
	"context"

	"meteorx/internal/modules/plan/model"
)

// PlanRepository 套餐数据访问层接口
type PlanRepository interface {
	Create(ctx context.Context, plan *model.Plan) error
	GetByID(ctx context.Context, id string) (*model.Plan, error)
	GetByCode(ctx context.Context, code string) (*model.Plan, error)
	Update(ctx context.Context, id string, plan *model.Plan) error
	Delete(ctx context.Context, id string) error
	FindPage(ctx context.Context, page, pageSize int, keyword string, status *int) ([]*model.Plan, int64, error)
	ListAllEnabled(ctx context.Context) ([]*model.Plan, error)
	CountByID(ctx context.Context, ids []string) (int64, error)
}

// SubscriptionRepository 租户订阅数据访问层接口
type SubscriptionRepository interface {
	// GetActiveByTenant 获取租户当前生效的订阅
	GetActiveByTenant(ctx context.Context, tenantID string) (*model.TenantSubscription, error)
	// Create 创建订阅记录
	Create(ctx context.Context, sub *model.TenantSubscription) error
	// UpdateStatus 更新订阅状态
	UpdateStatus(ctx context.Context, id string, status int) error
	// FindExpiredActive 查询所有已到期但仍处于 active 状态的订阅
	FindExpiredActive(ctx context.Context) ([]*model.TenantSubscription, error)
	// CountByPlan 统计使用某套餐的生效订阅数
	CountByPlan(ctx context.Context, planID string) (int64, error)
	// ListActiveByPlans 按套餐ID批量查询生效订阅
	ListActiveByPlans(ctx context.Context, planIDs []string) (map[string]int64, error)
	// ListActiveByTenants 批量查询多个租户的生效订阅
	ListActiveByTenants(ctx context.Context, tenantIDs []string) ([]*model.TenantSubscription, error)
}

// QuotaVerifier 配额校验接口：供 user 模块注入，避免循环依赖
type QuotaVerifier interface {
	// CheckUserLimit 校验租户当前用户数是否已达套餐上限
	// 返回 (是否超限, 当前用户数, 上限, error)
	CheckUserLimit(ctx context.Context, tenantID string) (bool, int64, int, error)
}

// UserCounter 用户计数接口（仅统计租户用户数量，避免 plan 依赖完整 user 仓库）
type UserCounter interface {
	CountByTenant(ctx context.Context, tenantID string) (int64, error)
}
