// Package invitation 租户成员邀请模块入口。
// 提供邀请创建、邮件通知、接受注册、状态管理等功能。
// 路由分为认证路由（需登录+权限）和公开路由（接受邀请页面）两部分。
package invitation

import (
	"meteorx/internal/config"
	"meteorx/internal/middleware"
	"meteorx/internal/modules/invitation/handler"
	"meteorx/internal/modules/invitation/repository"
	"meteorx/internal/modules/invitation/service"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	rbacSvc "meteorx/internal/modules/rbac/service"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	userRepo "meteorx/internal/modules/user/repository"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func newPermissionChecker(db *gorm.DB) middleware.PermissionChecker {
	permRepo := rbacRepo.NewPermissionRepository(db)
	roleRepo := rbacRepo.NewRoleRepository(db)
	rolePermRepo := rbacRepo.NewRolePermissionRepository(db)
	userRoleRepo := rbacRepo.NewUserRoleRepository(db)
	return rbacSvc.NewRBACService(roleRepo, permRepo, rolePermRepo, userRoleRepo, nil, nil)
}

func initHandler(db *gorm.DB, cfg config.Config) (*handler.InvitationHandler, middleware.PermissionChecker) {
	invRepo := repository.NewInvitationRepository(db)
	uRepo := userRepo.NewUserRepository(db)
	tRepo := tenantRepo.NewTenantRepository(db)
	rRepo := rbacRepo.NewRoleRepository(db)
	urRepo := rbacRepo.NewUserRoleRepository(db)

	svc := service.NewInvitationService(invRepo, uRepo, tRepo, rRepo, urRepo, cfg.Email, cfg.Client, cfg.Security)
	h := handler.NewInvitationHandler(svc)
	checker := newPermissionChecker(db)
	return h, checker
}

// InitModule 装配邀请依赖并注册需登录的管理路由。
func InitModule(r chi.Router, db *gorm.DB, cfg config.Config) {
	h, checker := initHandler(db, cfg)
	RegisterRoutes(r, h, checker)
}

// InitPublicModule 装配邀请依赖并注册无需登录的公开路由（接受邀请/查询信息）。
func InitPublicModule(r chi.Router, db *gorm.DB, cfg config.Config) {
	h, _ := initHandler(db, cfg)
	RegisterPublicRoutes(r, h)
}

// RegisterRoutes 注册认证路由（需登录 + 自动权限校验）。
// 包含邀请的 CRUD、取消、重发等管理接口。
func RegisterRoutes(r chi.Router, h *handler.InvitationHandler, checker middleware.PermissionChecker) {
	r.Route("/invitations", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/roles", h.AssignableRoles) // 必须在 /{id} 之前
		r.Put("/{id}/cancel", h.Cancel)
		r.Put("/{id}/resend", h.Resend)
		r.Delete("/{id}/delete", h.Delete)
	})
}

// RegisterPublicRoutes 注册公开路由（无需登录）。
// 包含邀请信息查询和接受邀请接口，供被邀请人从邮件链接访问。
// 注意：直接在父路由上注册，不使用 r.Route 分组，避免与 RegisterRoutes 的 /invitations Mount 冲突。
func RegisterPublicRoutes(r chi.Router, h *handler.InvitationHandler) {
	r.Post("/invitations/accept", h.Accept)
	r.Get("/invitations/info", h.GetByToken)
}
