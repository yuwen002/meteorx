package rbac

import (
	"meteorx/internal/modules/rbac/handler"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *handler.RBACHandler) {
	r.Route("/rbac", func(r chi.Router) {
		// 角色管理
		r.Route("/roles", func(r chi.Router) {
			r.Get("/", h.ListRoles)
			r.Post("/", h.CreateRole)
			r.Get("/{id}", h.GetRole)
			r.Put("/{id}", h.UpdateRole)
			r.Delete("/{id}", h.DeleteRole)
			r.Put("/{id}/permissions", h.BindRolePermissions)
			r.Get("/{id}/permissions", h.GetRolePermissions)
		})

		// 权限管理
		r.Route("/permissions", func(r chi.Router) {
			r.Get("/", h.ListPermissions)
			r.Post("/", h.CreatePermission)
			r.Get("/{id}", h.GetPermission)
			r.Put("/{id}", h.UpdatePermission)
			r.Delete("/{id}", h.DeletePermission)
		})
	})
}
