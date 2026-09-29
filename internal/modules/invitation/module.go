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

func InitModule(r chi.Router, db *gorm.DB, cfg config.Config) {
	h, checker := initHandler(db, cfg)
	RegisterRoutes(r, h, checker)
}

func InitPublicModule(r chi.Router, db *gorm.DB, cfg config.Config) {
	h, _ := initHandler(db, cfg)
	RegisterPublicRoutes(r, h)
}

func RegisterRoutes(r chi.Router, h *handler.InvitationHandler, checker middleware.PermissionChecker) {
	r.Route("/invitations", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Put("/{id}/cancel", h.Cancel)
		r.Put("/{id}/resend", h.Resend)
		r.Delete("/{id}/delete", h.Delete)
	})
}

func RegisterPublicRoutes(r chi.Router, h *handler.InvitationHandler) {
	r.Route("/invitations", func(r chi.Router) {
		r.Post("/accept", h.Accept)
		r.Get("/info", h.GetByToken)
	})
}