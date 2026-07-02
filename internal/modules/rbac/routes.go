package rbac

import (
	"meteorx/internal/middleware"
	"meteorx/internal/modules/rbac/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册 RBAC 模块的路由
// 使用 AutoRequirePermission 中间件自动推导权限码：
//   GET  /rbac/roles              → rbac:role:list
//   POST /rbac/roles              → rbac:role:create
//   ...等
//
// 说明：该路由组应挂载在登录验证中间件之后，超级管理员（superadmin）自动放行，
// 其他管理员通过数据库中配置的权限进行细粒度控制。
// 参数:
//   r: chi.Router 路由实例
//   h: RBACHandler 处理器实例
//   checker: PermissionChecker 权限检查器（通常就是 RBACService 本身）
func RegisterRoutes(r chi.Router, h *handler.RBACHandler, checker middleware.PermissionChecker) {
	r.Route("/rbac", func(r chi.Router) {
		// Dashboard 统计接口：无需细粒度权限校验（只需登录）
		r.Get("/stats", h.GetStats)

		// 需要细粒度权限校验的路由组
		r.Group(func(r chi.Router) {
			r.Use(middleware.AutoRequirePermission(checker))

			// 角色管理
		r.Route("/roles", func(r chi.Router) {
			r.Get("/", h.ListRoles)                          // rbac:role:list
			r.Get("/select", h.ListRolesForSelect)           // rbac:role:list_select（下拉列表，不分页）
			r.Post("/", h.CreateRole)                        // rbac:role:create
			r.Get("/deleted", h.ListDeletedRoles)           // rbac:role:list_deleted
			r.Put("/batch/status", h.BatchUpdateRoleStatus)  // rbac:role:batch_status
			r.Delete("/batch/delete", h.BatchDeleteRoles)    // rbac:role:batch_delete
			r.Put("/batch/permissions", h.BatchBindRolesPermissions)     // rbac:role:batch_bind
			r.Delete("/batch/permissions", h.BatchUnbindRolesPermissions) // rbac:role:batch_unbind
			r.Put("/{id}/restore", h.RestoreRole)            // rbac:role:restore
			r.Get("/{id}/detail", h.GetRole)                 // rbac:role:read
			r.Put("/{id}/update", h.UpdateRole)              // rbac:role:update
			r.Put("/{id}/status", h.UpdateRoleStatus)        // rbac:role:status
			r.Delete("/{id}/delete", h.DeleteRole)           // rbac:role:delete
			r.Put("/{id}/permissions", h.BindRolePermissions)            // rbac:role:bind_perm
			r.Get("/{id}/permissions", h.GetRolePermissions)            // rbac:role:get_perms
			r.Delete("/{id}/permissions", h.UnbindRolePermission)       // rbac:role:unbind_perm
			r.Delete("/{id}/permissions/batch", h.UnbindRolePermissions) // rbac:role:batch_unbind_perm
		})

			// 权限管理
		r.Route("/permissions", func(r chi.Router) {
			r.Get("/", h.ListPermissions)                          // rbac:perm:list
			r.Post("/", h.CreatePermission)                        // rbac:perm:create
			r.Put("/batch/status", h.BatchUpdatePermissionStatus)  // rbac:perm:batch_status
			r.Delete("/batch/delete", h.BatchDeletePermissions)    // rbac:perm:batch_delete
			r.Get("/{id}/detail", h.GetPermission)                 // rbac:perm:read
			r.Put("/{id}/update", h.UpdatePermission)              // rbac:perm:update
			r.Put("/{id}/status", h.UpdatePermissionStatus)        // rbac:perm:status
			r.Delete("/{id}/delete", h.DeletePermission)           // rbac:perm:delete
		})

			// 角色权限关系管理
			r.Route("/role-permissions", func(r chi.Router) {
				r.Get("/", h.ListRolePermissions)                      // rbac:role_perm:list
			})

			// 用户角色管理
			r.Route("/user-roles", func(r chi.Router) {
				r.Get("/", h.ListUserRoles)                             // rbac:user_role:list
				r.Post("/batch/assign", h.BatchAssignUserRoles)         // rbac:user_role:batch_assign
				r.Route("/{user_id}/roles", func(r chi.Router) {
					r.Post("/", h.AssignUserRoles)                       // rbac:user_role:assign
					r.Get("/", h.GetUserRoles)                           // rbac:user_role:get_roles
					r.Delete("/", h.RemoveAllUserRoles)                  // rbac:user_role:remove_all
					r.Delete("/{role_id}", h.RemoveUserRole)             // rbac:user_role:remove_one
				})
				r.Route("/roles/{role_id}/users", func(r chi.Router) {
					r.Get("/", h.GetRoleUsers)                            // rbac:user_role:get_users
				})
			})
		})
	})
}