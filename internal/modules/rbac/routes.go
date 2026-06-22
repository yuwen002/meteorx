package rbac

import (
	"meteorx/internal/modules/rbac/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册 RBAC 模块的路由
// 注意：该模块应挂载在 RequiresMasterAdmin 中间件保护的路由组下，
// 仅后台超级管理员可访问，无需额外的逐接口权限校验。
// 参数:
//
//	r: chi.Router 路由实例
//	h: RBACHandler 处理器实例
func RegisterRoutes(r chi.Router, h *handler.RBACHandler) {
	r.Route("/rbac", func(r chi.Router) {
		// 角色管理
		r.Route("/roles", func(r chi.Router) {
			r.Get("/", h.ListRoles)
			r.Post("/", h.CreateRole)
			// 回收站：静态路由必须注册在通配符路由之前
			r.Get("/deleted", h.ListDeletedRoles)
			r.Put("/batch/status", h.BatchUpdateRoleStatus)
			r.Put("/{id}/restore", h.RestoreRole)
			r.Get("/{id}/detail", h.GetRole)
			r.Put("/{id}/update", h.UpdateRole)
			r.Put("/{id}/status", h.UpdateRoleStatus)
			r.Delete("/{id}/delete", h.DeleteRole)
			r.Put("/{id}/permissions", h.BindRolePermissions)
			r.Get("/{id}/permissions", h.GetRolePermissions)
		})

		// 权限管理
		r.Route("/permissions", func(r chi.Router) {
			r.Get("/", h.ListPermissions)
			r.Post("/", h.CreatePermission)
			r.Get("/{id}/detail", h.GetPermission)
			r.Put("/{id}/update", h.UpdatePermission)
			r.Delete("/{id}/delete", h.DeletePermission)
		})
	})
}
