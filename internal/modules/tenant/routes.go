package tenant

import (
	"meteorx/internal/modules/tenant/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterPublicRoutes 编排完全公开的接口
func RegisterPublicRoutes(r chi.Router, h *handler.TenantHandler) {
	r.Route("/tenants", func(r chi.Router) {
		r.Post("/register", h.Register) // 前端自主开户
	})
}

// RegisterPrivateRoutes 编排需要普通登录 Token 的接口
func RegisterPrivateRoutes(r chi.Router, h *handler.TenantHandler) {
	r.Route("/tenants/current", func(r chi.Router) {
		//r.Get("/", h.GetCurrentTenant)    // 查看当前租户详情
		//r.Put("/", h.UpdateCurrentTenant) // 修改当前租户信息
	})
}

// RegisterAdminRoutes 编排 MaaS 平台超级管理员的控制台接口
func RegisterAdminRoutes(r chi.Router, h *handler.TenantHandler) {
	r.Route("/admin/tenants", func(r chi.Router) {
		r.Post("/", h.AdminCreate)                       // 后台手动新建租户
		r.Get("/", h.List)                               // 后台分页查全盘租户
		r.Get("/deleted", h.AdminDeletedList)            // 回收站：查询已软删除的租户列表
		r.Put("/batch/status", h.AdminBatchUpdateStatus) // 批量启用/禁用租户
		r.Delete("/batch", h.AdminBatchDelete)           // 批量软删除租户
		r.Put("/{id}/status", h.AdminUpdateStatus)       // 后台禁用/启用租户
		r.Get("/{id}", h.AdminDetail)                    // 后台查询租户详情
		r.Put("/{id}", h.AdminUpdate)                    // 后台编辑租户信息
		r.Delete("/{id}", h.AdminDelete)                 // 后台软删除租户
		r.Put("/{id}/restore", h.AdminRestore)           // 回收站：恢复已软删除的租户
	})
}
