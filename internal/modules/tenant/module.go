package tenant

import (
	"meteorx/internal/modules/tenant/handler"
	planrepo "meteorx/internal/modules/plan/repository"
	plansvc "meteorx/internal/modules/plan/service"
	rbacrepo "meteorx/internal/modules/rbac/repository"
	tenantrepo "meteorx/internal/modules/tenant/repository"
	"meteorx/internal/modules/tenant/service"
	userrepository "meteorx/internal/modules/user/repository"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// 提取公共工厂方法，保持不变
func initHandler(db *gorm.DB) *handler.TenantHandler {
	tenantRepo := tenantrepo.NewTenantRepository(db)
	userRepo := userrepository.NewUserRepository(db)
	roleRepo := rbacrepo.NewRoleRepository(db)
	userRoleRepo := rbacrepo.NewUserRoleRepository(db)
	svc := service.NewTenantService(tenantRepo, userRepo, roleRepo, userRoleRepo)

	// 注入套餐摘要查询器（PlanService 实现了 TenantPlanProvider 接口）
	planRepo := planrepo.NewPlanRepository(db)
	subRepo := planrepo.NewSubscriptionRepository(db)
	planSvc := plansvc.NewPlanService(planRepo, subRepo, userRepo)
	svc.SetPlanProvider(planSvc)

	return handler.NewTenantHandler(svc)
}

// InitPublicModule 1. 完全公开的租户接口入口
func InitPublicModule(r chi.Router, db *gorm.DB) {
	h := initHandler(db)
	RegisterPublicRoutes(r, h) // 👈 扔给 routes.go 去编排路径
}

// InitPrivateModule 2. 租户内部的管理私有接口入口
func InitPrivateModule(r chi.Router, db *gorm.DB) {
	h := initHandler(db)
	RegisterPrivateRoutes(r, h) // 👈 扔给 routes.go 去编排路径
}

// InitAdminModule 3. MaaS 平台超级管理员的控制台接口入口
func InitAdminModule(r chi.Router, db *gorm.DB) {
	h := initHandler(db)
	RegisterAdminRoutes(r, h) // 👈 扔给 routes.go 去编排路径
}