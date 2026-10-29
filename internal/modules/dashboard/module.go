// Package dashboard 提供平台运营数据看板模块
package dashboard

import (
	"meteorx/internal/cache"
	"meteorx/internal/middleware"
	"meteorx/internal/modules/dashboard/handler"
	"meteorx/internal/modules/dashboard/repository"
	"meteorx/internal/modules/dashboard/service"
	rbacrepo "meteorx/internal/modules/rbac/repository"
	rbacsvc "meteorx/internal/modules/rbac/service"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// InitModule 初始化数据看板模块
// 挂载在平台管理员路由组下，需要超级管理员权限
// rdb 用于运营总览聚合结果的短 TTL 缓存，可为 nil（不可用时降级为直接查库）。
func InitModule(r chi.Router, db *gorm.DB, rdb *cache.Redis) {
	repo := repository.NewDashboardRepository(db, repository.WithCache(rdb))
	svc := service.NewDashboardService(repo)
	h := handler.NewDashboardHandler(svc)

	checker := initPermissionChecker(db)
	RegisterRoutes(r, h, checker)
}

// InitTenantModule 初始化租户侧看板模块（仅登录，不挂 RBAC 权限码）。
// 与 task/wiki/message 同款“仅登录 + 以 JWT 租户身份为准”范式，
// 注册在 /api/v1 登录组内、RequiresMasterAdmin 分组之外。
// rdb 用于租户总览聚合结果的短 TTL 缓存，可为 nil（不可用时降级为直接查库）。
func InitTenantModule(r chi.Router, db *gorm.DB, rdb *cache.Redis) {
	repo := repository.NewDashboardRepository(db, repository.WithCache(rdb))
	svc := service.NewDashboardService(repo)
	h := handler.NewDashboardHandler(svc)

	RegisterTenantRoutes(r, h)
}

// initPermissionChecker 创建权限检查器（复用 RBACService）
func initPermissionChecker(db *gorm.DB) middleware.PermissionChecker {
	roleRepo := rbacrepo.NewRoleRepository(db)
	permRepo := rbacrepo.NewPermissionRepository(db)
	rolePermRepo := rbacrepo.NewRolePermissionRepository(db)
	userRoleRepo := rbacrepo.NewUserRoleRepository(db)
	return rbacsvc.NewRBACService(roleRepo, permRepo, rolePermRepo, userRoleRepo, nil, nil)
}
