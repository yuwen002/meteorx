package rbac

import (
	"meteorx/internal/modules/rbac/handler"
	"meteorx/internal/modules/rbac/repository"
	"meteorx/internal/modules/rbac/service"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func InitModule(r chi.Router, db *gorm.DB) {
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	rolePermRepo := repository.NewRolePermissionRepository(db)

	svc := service.NewRBACService(roleRepo, permRepo, rolePermRepo)
	h := handler.NewRBACHandler(svc)

	RegisterRoutes(r, h)
}
