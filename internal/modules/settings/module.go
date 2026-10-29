// Package settings 平台运行时配置中心：DB 驱动的系统级 KV 配置，
// 支持管理后台在线修改（免重启生效）、公开侧只读消费与类型化内部读取。
package settings

import (
	"meteorx/internal/cache"
	"meteorx/internal/middleware"
	rbacrepo "meteorx/internal/modules/rbac/repository"
	rbacsvc "meteorx/internal/modules/rbac/service"
	"meteorx/internal/modules/settings/handler"
	"meteorx/internal/modules/settings/repository"
	"meteorx/internal/modules/settings/service"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// NewSettingService 创建配置服务实例（供跨模块消费者注入：注册开关、维护模式门禁等）
func NewSettingService(db *gorm.DB, rdb *cache.Redis) *service.SettingService {
	return service.NewSettingService(repository.NewSettingRepository(db), rdb)
}

// initPermissionChecker 创建权限检查器（复用 RBACService，与 plan 等模块同款）
func initPermissionChecker(db *gorm.DB) middleware.PermissionChecker {
	roleRepo := rbacrepo.NewRoleRepository(db)
	permRepo := rbacrepo.NewPermissionRepository(db)
	rolePermRepo := rbacrepo.NewRolePermissionRepository(db)
	userRoleRepo := rbacrepo.NewUserRoleRepository(db)
	return rbacsvc.NewRBACService(roleRepo, permRepo, rolePermRepo, userRoleRepo, nil, nil)
}

// InitAdminModule 平台管理员配置中心接口（仅平台超管可操作）
func InitAdminModule(r chi.Router, db *gorm.DB, rdb *cache.Redis) {
	h := handler.NewSettingHandler(NewSettingService(db, rdb))
	checker := initPermissionChecker(db)
	RegisterAdminRoutes(r, h, checker)
}

// InitPublicModule 公开运行时配置接口（免鉴权，登录/注册页消费）
func InitPublicModule(r chi.Router, db *gorm.DB, rdb *cache.Redis) {
	h := handler.NewSettingHandler(NewSettingService(db, rdb))
	RegisterPublicRoutes(r, h)
}
