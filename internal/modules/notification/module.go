// Package notification 提供平台通知公告模块
package notification

import (
	"meteorx/internal/middleware"
	"meteorx/internal/modules/notification/handler"
	"meteorx/internal/modules/notification/repository"
	"meteorx/internal/modules/notification/service"
	rbacrepo "meteorx/internal/modules/rbac/repository"
	rbacsvc "meteorx/internal/modules/rbac/service"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// InitModule 初始化通知公告模块
// 挂载在平台管理员路由组下，需要超级管理员权限
func InitModule(r chi.Router, db *gorm.DB) {
	repo := repository.NewAnnouncementRepository(db)
	svc := service.NewAnnouncementService(repo)
	h := handler.NewAnnouncementHandler(svc)

	checker := initPermissionChecker(db)
	RegisterRoutes(r, h, checker)
}

// InitTenantModule 初始化租户端公告模块
// 挂载在租户私有路由组下，需要登录态
func InitTenantModule(r chi.Router, db *gorm.DB) {
	repo := repository.NewAnnouncementRepository(db)
	svc := service.NewAnnouncementService(repo)
	h := handler.NewAnnouncementHandler(svc)

	RegisterTenantRoutes(r, h)
}

// initPermissionChecker 创建权限检查器（复用 RBACService）
func initPermissionChecker(db *gorm.DB) middleware.PermissionChecker {
	roleRepo := rbacrepo.NewRoleRepository(db)
	permRepo := rbacrepo.NewPermissionRepository(db)
	rolePermRepo := rbacrepo.NewRolePermissionRepository(db)
	userRoleRepo := rbacrepo.NewUserRoleRepository(db)
	return rbacsvc.NewRBACService(roleRepo, permRepo, rolePermRepo, userRoleRepo)
}