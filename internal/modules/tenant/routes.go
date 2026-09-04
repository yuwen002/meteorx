package tenant

import (
	"meteorx/internal/middleware"
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
		r.Get("/", h.GetCurrentTenant)         // 查看当前租户详情
		r.Put("/", h.UpdateCurrentTenant)      // 修改当前租户信息（如企业名称、Logo）
		r.Get("/status", h.GetInitStatus)      // 查询租户异步初始化/开通状态
		r.Post("/cancel", h.ApplyCancellation) // 租户申请自主注销
	})
}

// RegisterTenantSettingsRoutes 注册租户设置路由
func RegisterTenantSettingsRoutes(r chi.Router, h *handler.TenantSettingsHandler) {
	r.Route("/tenant-settings", func(r chi.Router) {
		r.Get("/", h.GetSettings)
		r.Put("/", h.UpdateSettings)
	})
}

// RegisterAdminRoutes 编排 MaaS 平台超级管理员的控制台接口
// 挂载 AutoRequirePermission 中间件做细粒度权限校验
func RegisterAdminRoutes(r chi.Router, h *handler.TenantHandler, checker middleware.PermissionChecker) {
	r.Route("/admin/tenants", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Post("/", h.AdminCreate)                       // → admin:tenant:create
		r.Get("/", h.List)                               // → admin:tenant:list
		r.Get("/deleted", h.AdminDeletedList)            // → admin:tenant:list_deleted
		r.Put("/batch/status", h.AdminBatchUpdateStatus) // → admin:tenant:batch_status
		r.Delete("/batch", h.AdminBatchDelete)           // → admin:tenant:batch_delete
		r.Put("/{id}/status", h.AdminUpdateStatus)       // → admin:tenant:status
		r.Get("/{id}/detail", h.AdminDetail)             // → admin:tenant:read
		r.Put("/{id}/update", h.AdminUpdate)             // → admin:tenant:update
		r.Delete("/{id}/delete", h.AdminDelete)          // → admin:tenant:delete
		r.Put("/{id}/restore", h.AdminRestore)           // → admin:tenant:restore
		r.Delete("/{id}/hard", h.AdminHardDelete)        // → admin:tenant:hard_delete
		r.Put("/{id}/plan", h.AdminUpdatePlan)           // → admin:tenant:update_plan
	})

	// 注销申请审批
	r.Route("/admin/cancel-requests", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/", h.AdminListCancelRequests)        // → admin:cancel_request:list
		r.Put("/{id}/approve", h.AdminApproveCancel) // → admin:cancel_request:approve
		r.Put("/{id}/reject", h.AdminRejectCancel)   // → admin:cancel_request:reject
	})
}
