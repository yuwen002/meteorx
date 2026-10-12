// Package plan 提供套餐管理与订阅模块，支持套餐定义、租户订阅绑定、到期自动禁用和配额校验。
package plan

import (
	"meteorx/internal/middleware"
	"meteorx/internal/modules/plan/handler"
	"meteorx/internal/modules/plan/repository"
	"meteorx/internal/modules/plan/service"
	rbacrepo "meteorx/internal/modules/rbac/repository"
	rbacsvc "meteorx/internal/modules/rbac/service"
	userrepository "meteorx/internal/modules/user/repository"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// 提取公共工厂方法
func initHandler(db *gorm.DB) *handler.PlanHandler {
	planRepo := repository.NewPlanRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)
	userRepo := userrepository.NewUserRepository(db)
	svc := service.NewPlanService(planRepo, subRepo, userRepo)
	return handler.NewPlanHandler(svc)
}

// initPermissionChecker 创建权限检查器（复用 RBACService）
func initPermissionChecker(db *gorm.DB) middleware.PermissionChecker {
	roleRepo := rbacrepo.NewRoleRepository(db)
	permRepo := rbacrepo.NewPermissionRepository(db)
	rolePermRepo := rbacrepo.NewRolePermissionRepository(db)
	userRoleRepo := rbacrepo.NewUserRoleRepository(db)
	return rbacsvc.NewRBACService(roleRepo, permRepo, rolePermRepo, userRoleRepo, nil, nil)
}

// NewQuotaVerifier 创建配额校验器（供 user 模块注入，避免循环依赖）
func NewQuotaVerifier(db *gorm.DB) repository.QuotaVerifier {
	planRepo := repository.NewPlanRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)
	userRepo := userrepository.NewUserRepository(db)
	return service.NewPlanService(planRepo, subRepo, userRepo)
}

// NewQuotaRateProvider 创建租户 API 速率配额提供者（供限流中间件注入）
func NewQuotaRateProvider(db *gorm.DB) middleware.QuotaRateProvider {
	return NewQuotaService(db)
}

// NewQuotaService 创建套餐配额服务（具体类型，同时满足存储/速率等配额查询接口，供 file、wiki 等模块注入）
func NewQuotaService(db *gorm.DB) *service.PlanService {
	planRepo := repository.NewPlanRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)
	userRepo := userrepository.NewUserRepository(db)
	return service.NewPlanService(planRepo, subRepo, userRepo)
}

// InitAdminModule 平台管理员套餐管理接口
func InitAdminModule(r chi.Router, db *gorm.DB) {
	h := initHandler(db)
	checker := initPermissionChecker(db)
	RegisterAdminRoutes(r, h, checker)
}

// InitPrivateModule 租户侧当前套餐查询接口
func InitPrivateModule(r chi.Router, db *gorm.DB) {
	h := initHandler(db)
	RegisterPrivateRoutes(r, h)
}
