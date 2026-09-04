package user

import (
	"meteorx/internal/middleware"
	"meteorx/internal/modules/user/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterProfileRoutes 编排当前用户个人信息管理接口（用户登录后可操作自己的信息）
func RegisterProfileRoutes(r chi.Router, h *handler.UserHandler) {
	r.Route("/profile", func(r chi.Router) {
		r.Get("/stats", h.GetStats)          // Dashboard: 用户总数
		r.Get("/", h.GetProfile)             // 获取当前用户个人信息
		r.Put("/", h.UpdateProfile)          // 更新当前用户个人信息
		r.Put("/password", h.ChangePassword) // 修改当前用户密码
	})
}

// RegisterRoutes 编排租户用户管理接口（需要登录，且属于当前租户）
// 路由组级使用 AutoRequirePermission 自动推导权限码，
// 避免每个路由手动写权限码字符串
func RegisterRoutes(r chi.Router, h *handler.UserHandler, checker middleware.PermissionChecker) {
	r.Route("/users", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/", h.ListUsers)                            // → 自动需要 user:list
		r.Post("/", h.CreateUser)                          // → 自动需要 user:create
		r.Get("/deleted", h.ListDeletedUsers)              // recycle: list deleted users
		r.Get("/{id}/detail", h.GetUser)                   // → 自动需要 user:read
		r.Put("/{id}/update", h.UpdateUser)                // → 自动需要 user:update
		r.Put("/{id}/reset-password", h.ResetPassword)     // → 自动需要 user:reset_password
		r.Delete("/{id}/delete", h.DeleteUser)             // → 自动需要 user:delete
		r.Put("/{id}/restore", h.RestoreUser)              // recycle: restore user
		r.Delete("/{id}/permanent", h.PermanentDeleteUser) // recycle: permanent delete
	})
}

// RegisterAdminRoutes 编排系统管理员管理接口（仅限平台超级管理员）
// 超级管理员已通过外层 RequiresMasterAdmin 中间件放行，
// 其他管理员通过 AutoRequirePermission 中间件做细粒度权限校验
func RegisterAdminRoutes(r chi.Router, h *handler.UserHandler, checker middleware.PermissionChecker) {
	r.Get("/admin/stats", h.GetAllStats) // Dashboard: 所有用户总数

	r.Route("/admin/users", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/", h.ListMasterAdmins)                            // → admin:master:list
		r.Post("/", h.CreateMasterAdmin)                          // → admin:master:create
		r.Get("/deleted", h.ListDeletedMasterAdmins)              // → admin:master:list_deleted
		r.Put("/{id}/restore", h.RestoreMasterAdmin)              // → admin:master:restore
		r.Delete("/{id}/permanent", h.PermanentDeleteMasterAdmin) // → admin:master:permanent_delete
		r.Put("/batch/status", h.BatchUpdateMasterAdminStatus)    // → admin:master:batch_status
		r.Delete("/batch/delete", h.BatchDeleteMasterAdmins)      // → admin:master:batch_delete
		r.Get("/{id}/detail", h.GetMasterAdmin)                   // → admin:master:read
		r.Put("/{id}/update", h.UpdateMasterAdmin)                // → admin:master:update
		r.Put("/{id}/status", h.UpdateMasterAdminStatus)          // → admin:master:status
		r.Delete("/{id}/delete", h.DeleteMasterAdmin)             // → admin:master:delete
	})

	// 系统管理员跨租户用户管理
	r.Route("/admin/tenant-users", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Post("/", h.AdminCreateTenantUser)                                         // → admin:tenant_user:create
		r.Get("/all", h.AdminListAllTenantUsers)                                     // → admin:tenant_user:list
		r.Get("/deleted/all", h.AdminListAllDeletedTenantUsers)                      // → admin:tenant_user:list_deleted
		r.Get("/{tenantID}/list", h.AdminListTenantUsers)                            // → admin:tenant_user:list
		r.Get("/{tenantID}/deleted", h.AdminListDeletedTenantUsers)                  // → admin:tenant_user:list_deleted
		r.Put("/{tenantID}/{userID}/update", h.AdminUpdateTenantUser)                // → admin:tenant_user:update
		r.Put("/{tenantID}/{userID}/status", h.AdminUpdateTenantUserStatus)          // → admin:tenant_user:status
		r.Put("/{tenantID}/{userID}/reset-password", h.AdminResetTenantUserPassword) // → admin:tenant_user:reset_password
		r.Put("/{tenantID}/{userID}/restore", h.AdminRestoreTenantUser)              // → admin:tenant_user:restore
		r.Delete("/{tenantID}/{userID}/delete", h.AdminDeleteTenantUser)             // → admin:tenant_user:delete
		r.Delete("/{tenantID}/{userID}/permanent", h.AdminPermanentDeleteTenantUser) // → admin:tenant_user:permanent_delete
		r.Put("/{tenantID}/batch/status", h.AdminBatchUpdateTenantUserStatus)        // → admin:tenant_user:batch_status
		r.Delete("/{tenantID}/batch/delete", h.AdminBatchDeleteTenantUsers)          // → admin:tenant_user:batch_delete
	})
}
