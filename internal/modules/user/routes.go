package user

import (
	"meteorx/internal/modules/user/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 编排租户用户管理接口（需要登录，且属于当前租户）
func RegisterRoutes(r chi.Router, h *handler.UserHandler) {
	r.Route("/users", func(r chi.Router) {
		r.Get("/", h.ListUsers)         // 获取用户列表
		r.Post("/", h.CreateUser)       // 创建用户
		r.Get("/{id}", h.GetUser)       // 获取用户详情
		r.Put("/{id}", h.UpdateUser)    // 更新用户
		r.Delete("/{id}", h.DeleteUser) // 删除用户
	})
}

// RegisterAdminRoutes 编排系统管理员管理接口（仅限平台超级管理员）
func RegisterAdminRoutes(r chi.Router, h *handler.UserHandler) {
	r.Route("/admin/users", func(r chi.Router) {
		r.Get("/", h.ListMasterAdmins)         // 获取系统管理员列表
		r.Post("/", h.CreateMasterAdmin)       // 创建系统管理员
		r.Get("/{id}", h.GetMasterAdmin)       // 获取系统管理员详情
		r.Put("/{id}", h.UpdateMasterAdmin)    // 更新系统管理员
		r.Delete("/{id}", h.DeleteMasterAdmin) // 删除系统管理员
	})
	
	// 系统管理员跨租户用户管理
	r.Route("/admin/tenant-users", func(r chi.Router) {
		r.Post("/", h.AdminCreateTenantUser)        // 为指定租户创建用户
		r.Get("/{tenantID}", h.AdminListTenantUsers)      // 获取指定租户的用户列表
		r.Put("/{tenantID}/{userID}", h.AdminUpdateTenantUser)   // 更新指定租户的用户
		r.Delete("/{tenantID}/{userID}", h.AdminDeleteTenantUser) // 删除指定租户的用户
	})
}