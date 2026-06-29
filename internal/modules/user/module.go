package user

import (
	"meteorx/internal/middleware"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	rbacSvc "meteorx/internal/modules/rbac/service"
	"meteorx/internal/modules/tenant/repository"
	"meteorx/internal/modules/user/handler"
	userRepository "meteorx/internal/modules/user/repository"
	"meteorx/internal/modules/user/service"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// newPermissionChecker 创建一个权限检查器（复用 RBACService），供路由层使用
func newPermissionChecker(db *gorm.DB) middleware.PermissionChecker {
	permRepo := rbacRepo.NewPermissionRepository(db)
	roleRepo := rbacRepo.NewRoleRepository(db)
	rolePermRepo := rbacRepo.NewRolePermissionRepository(db)
	userRoleRepo := rbacRepo.NewUserRoleRepository(db)
	return rbacSvc.NewRBACService(roleRepo, permRepo, rolePermRepo, userRoleRepo)
}

// 提取公共工厂方法，保持与租户模块结构一致
func initHandler(db *gorm.DB) (*handler.UserHandler, middleware.PermissionChecker) {
	repo := userRepository.NewUserRepository(db)
	tenantRepo := repository.NewTenantRepository(db)
	roleRepo := rbacRepo.NewRoleRepository(db)
	userRoleRepo := rbacRepo.NewUserRoleRepository(db)
	svc := service.NewUserService(repo, tenantRepo, roleRepo, userRoleRepo)
	checker := newPermissionChecker(db)
	return handler.NewUserHandler(svc), checker
}

// InitModule 用户模块初始化（普通租户用户管理）
func InitModule(r chi.Router, db *gorm.DB) {
	h, checker := initHandler(db)
	RegisterRoutes(r, h, checker)
}

// InitProfileModule 用户个人信息接口初始化（当前用户操作自己的信息）
func InitProfileModule(r chi.Router, db *gorm.DB) {
	h, _ := initHandler(db)
	RegisterProfileRoutes(r, h)
}

// InitAdminModule 用户模块的系统管理员接口初始化
func InitAdminModule(r chi.Router, db *gorm.DB) {
	h, _ := initHandler(db)
	RegisterAdminRoutes(r, h)
}