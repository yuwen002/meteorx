package user

import (
	"meteorx/internal/modules/user/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterProfileRoutes 编排当前用户个人信息管理接口（用户登录后可操作自己的信息）
func RegisterProfileRoutes(r chi.Router, h *handler.UserHandler) {
	r.Route("/profile", func(r chi.Router) {
		r.Get("/", h.GetProfile)             // 获取当前用户个人信息
		r.Put("/", h.UpdateProfile)          // 更新当前用户个人信息
		r.Put("/password", h.ChangePassword) // 修改当前用户密码
	})
}

// RegisterRoutes 编排租户用户管理接口（需要登录，且属于当前租户）
func RegisterRoutes(r chi.Router, h *handler.UserHandler) {
	r.Route("/users", func(r chi.Router) {
		r.Get("/", h.ListUsers)                // 获取用户列表
		r.Post("/", h.CreateUser)              // 创建用户
		r.Get("/{id}/detail", h.GetUser)       // 获取用户详情
		r.Put("/{id}/update", h.UpdateUser)    // 更新用户
		r.Delete("/{id}/delete", h.DeleteUser) // 删除用户
	})
}

// RegisterAdminRoutes 编排系统管理员管理接口（仅限平台超级管理员）
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
		r.Get("/{tenantID}/list", h.AdminListTenantUsers)                // 获取指定租户的用户列表
		r.Put("/{tenantID}/{userID}/update", h.AdminUpdateTenantUser)    // 更新指定租户的用户
		r.Delete("/{tenantID}/{userID}/delete", h.AdminDeleteTenantUser) // 删除指定租户的用户
	})
}
