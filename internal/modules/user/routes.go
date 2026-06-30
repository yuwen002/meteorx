package user

import (
	"meteorx/internal/middleware"
	"meteorx/internal/modules/user/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterProfileRoutes 编排当前用户个人信息管理接口（用户登录后可操作自己的信息）
func RegisterProfileRoutes(r chi.Router, h *handler.UserHandler) {
	r.Route("/profile", func(r chi.Router) {
		r.Get("/stats", h.GetStats) // Dashboard: 用户总数
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
		r.Get("/", h.ListUsers)                                // → 自动需要 user:list
		r.Post("/", h.CreateUser)                               // → 自动需要 user:create
		r.Get("/{id}/detail", h.GetUser)                        // → 自动需要 user:read
		r.Put("/{id}/update", h.UpdateUser)                     // → 自动需要 user:update
		r.Delete("/{id}/delete", h.DeleteUser)                  // → 自动需要 user:delete
	})
}

// RegisterAdminRoutes 编排系统管理员管理接口（仅限平台超级管理员）
// 超级管理员已通过外层 RequiresMasterAdmin 中间件放行，这里不再重复配置权限校验
func RegisterAdminRoutes(r chi.Router, h *handler.UserHandler) {
	r.Route("/admin/users", func(r chi.Router) {
		r.Get("/", h.ListMasterAdmins)                         // 获取系统管理员列表
		r.Post("/", h.CreateMasterAdmin)                       // 创建系统管理员
		r.Get("/deleted", h.ListDeletedMasterAdmins)           // 回收站：获取已删除的系统管理员列表
		r.Put("/{id}/restore", h.RestoreMasterAdmin)           // 回收站：恢复已删除的系统管理员
		r.Put("/batch/status", h.BatchUpdateMasterAdminStatus) // 批量更新系统管理员状态
		r.Delete("/batch/delete", h.BatchDeleteMasterAdmins)   // 批量删除系统管理员
		r.Get("/{id}/detail", h.GetMasterAdmin)                // 获取系统管理员详情
		r.Put("/{id}/update", h.UpdateMasterAdmin)             // 更新系统管理员
		r.Put("/{id}/status", h.UpdateMasterAdminStatus)       // 更新系统管理员状态
		r.Delete("/{id}/delete", h.DeleteMasterAdmin)          // 删除系统管理员
	})

	// 系统管理员跨租户用户管理
	r.Route("/admin/tenant-users", func(r chi.Router) {
		r.Post("/", h.AdminCreateTenantUser)                             // 为指定租户创建用户
		r.Get("/all", h.AdminListAllTenantUsers)                         // 获取所有租户用户列表（不包括系统管理员）
		r.Get("/deleted/all", h.AdminListAllDeletedTenantUsers)          // 回收站：获取所有租户的已删除用户列表
		r.Get("/{tenantID}/list", h.AdminListTenantUsers)                // 获取指定租户的用户列表
		r.Get("/{tenantID}/deleted", h.AdminListDeletedTenantUsers)      // 回收站：获取指定租户的已删除用户列表
		r.Put("/{tenantID}/{userID}/update", h.AdminUpdateTenantUser)    // 更新指定租户的用户
		r.Put("/{tenantID}/{userID}/status", h.AdminUpdateTenantUserStatus) // 更新指定租户的用户状态
		r.Put("/{tenantID}/{userID}/restore", h.AdminRestoreTenantUser)  // 恢复已删除的租户用户
		r.Delete("/{tenantID}/{userID}/delete", h.AdminDeleteTenantUser) // 删除指定租户的用户
		r.Put("/{tenantID}/batch/status", h.AdminBatchUpdateTenantUserStatus) // 批量更新指定租户的用户状态
		r.Delete("/{tenantID}/batch/delete", h.AdminBatchDeleteTenantUsers)  // 批量删除指定租户的用户
	})
}