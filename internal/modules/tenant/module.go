package tenant

import (
	"meteorx/internal/middleware"
	planrepo "meteorx/internal/modules/plan/repository"
	plansvc "meteorx/internal/modules/plan/service"
	rbacrepo "meteorx/internal/modules/rbac/repository"
	rbacsvc "meteorx/internal/modules/rbac/service"
	"meteorx/internal/modules/tenant/handler"
	tenantrepo "meteorx/internal/modules/tenant/repository"
	"meteorx/internal/modules/tenant/service"
	userrepository "meteorx/internal/modules/user/repository"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// NewTenantSettingsHandler 创建租户设置处理器
func NewTenantSettingsHandler(db *gorm.DB) *handler.TenantSettingsHandler {
	repo := tenantrepo.NewTenantSettingsRepository(db)
	svc := service.NewTenantSettingsService(repo)
	return handler.NewTenantSettingsHandler(svc)
}

// 提取公共工厂方法，保持不变
func initHandler(db *gorm.DB) *handler.TenantHandler {
	tenantRepo := tenantrepo.NewTenantRepository(db)
	userRepo := userrepository.NewUserRepository(db)
	roleRepo := rbacrepo.NewRoleRepository(db)
	userRoleRepo := rbacrepo.NewUserRoleRepository(db)
	svc := service.NewTenantService(tenantRepo, userRepo, roleRepo, userRoleRepo)

	// 注入套餐摘要查询器 & 套餐分配器（PlanService 实现了两个接口）
	planRepo := planrepo.NewPlanRepository(db)
	subRepo := planrepo.NewSubscriptionRepository(db)
	planSvc := plansvc.NewPlanService(planRepo, subRepo, userRepo)
	svc.SetPlanProvider(planSvc)
	svc.SetPlanAssignProvider(planSvc)

	// 注入订阅仓库（注销/物理删除时取消生效订阅）
	svc.SetSubscriptionRepository(subRepo)

	return handler.NewTenantHandler(svc)
}

// initPermissionChecker 创建权限检查器（复用 RBACService）
func initPermissionChecker(db *gorm.DB) middleware.PermissionChecker {
	roleRepo := rbacrepo.NewRoleRepository(db)
	permRepo := rbacrepo.NewPermissionRepository(db)
	rolePermRepo := rbacrepo.NewRolePermissionRepository(db)
	userRoleRepo := rbacrepo.NewUserRoleRepository(db)
	return rbacsvc.NewRBACService(roleRepo, permRepo, rolePermRepo, userRoleRepo)
}

// InitPublicModule 1. 完全公开的租户接口入口
func InitPublicModule(r chi.Router, db *gorm.DB) {
	h := initHandler(db)
	RegisterPublicRoutes(r, h)
}

// InitPrivateModule 2. 租户内部的管理私有接口入口
func InitPrivateModule(r chi.Router, db *gorm.DB) {
	h := initHandler(db)
	RegisterPrivateRoutes(r, h)
}

// InitAdminModule 3. MaaS 平台超级管理员的控制台接口入口
func InitAdminModule(r chi.Router, db *gorm.DB) {
	h := initHandler(db)
	checker := initPermissionChecker(db)
	RegisterAdminRoutes(r, h, checker)
}