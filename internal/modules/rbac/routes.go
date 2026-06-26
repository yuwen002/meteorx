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
			r.Get("/deleted", h.ListDeletedRoles)
			r.Put("/batch/status", h.BatchUpdateRoleStatus)
			r.Delete("/batch/delete", h.BatchDeleteRoles)
			r.Put("/batch/permissions", h.BatchBindRolesPermissions)
			r.Delete("/batch/permissions", h.BatchUnbindRolesPermissions)
			r.Put("/{id}/restore", h.RestoreRole)
			r.Get("/{id}/detail", h.GetRole)
			r.Put("/{id}/update", h.UpdateRole)
			r.Put("/{id}/status", h.UpdateRoleStatus)
			r.Delete("/{id}/delete", h.DeleteRole)
			r.Put("/{id}/permissions", h.BindRolePermissions)
			r.Get("/{id}/permissions", h.GetRolePermissions)
			r.Delete("/{id}/permissions/{permission_id}", h.UnbindRolePermission)
		})

		// 权限管理
		r.Route("/permissions", func(r chi.Router) {
			r.Get("/", h.ListPermissions)
			r.Post("/", h.CreatePermission)
			r.Put("/batch/status", h.BatchUpdatePermissionStatus)
			r.Delete("/batch/delete", h.BatchDeletePermissions)
			r.Get("/{id}/detail", h.GetPermission)
			r.Put("/{id}/update", h.UpdatePermission)
			r.Put("/{id}/status", h.UpdatePermissionStatus)
			r.Delete("/{id}/delete", h.DeletePermission)
		})

		// 用户角色管理
		r.Route("/user-roles", func(r chi.Router) {
			r.Post("/batch/assign", h.BatchAssignUserRoles)
			r.Route("/{user_id}/roles", func(r chi.Router) {
				r.Post("/", h.AssignUserRoles)
				r.Get("/", h.GetUserRoles)
				r.Delete("/", h.RemoveAllUserRoles)
				r.Delete("/{role_id}", h.RemoveUserRole)
			})
			r.Route("/roles/{role_id}/users", func(r chi.Router) {
				r.Get("/", h.GetRoleUsers)
			})
		})
	})
}
