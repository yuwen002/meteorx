// Package dashboard 提供平台运营数据看板模块
package dashboard

import (
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
func InitModule(r chi.Router, db *gorm.DB) {
	repo := repository.NewDashboardRepository(db)
	svc := service.NewDashboardService(repo)
	h := handler.NewDashboardHandler(svc)

	checker := initPermissionChecker(db)
	RegisterRoutes(r, h, checker)
}

// initPermissionChecker 创建权限检查器（复用 RBACService）
func initPermissionChecker(db *gorm.DB) middleware.PermissionChecker {
	roleRepo := rbacrepo.NewRoleRepository(db)
	permRepo := rbacrepo.NewPermissionRepository(db)
	rolePermRepo := rbacrepo.NewRolePermissionRepository(db)
	userRoleRepo := rbacrepo.NewUserRoleRepository(db)
	return rbacsvc.NewRBACService(roleRepo, permRepo, rolePermRepo, userRoleRepo)
}
