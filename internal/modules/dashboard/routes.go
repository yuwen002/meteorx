package dashboard

import (
	"meteorx/internal/middleware"
	"meteorx/internal/modules/dashboard/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册数据看板模块路由
// r: chi 路由器实例
// h: 数据看板处理器实例
// checker: 权限检查器
// 挂载路径前缀: /api/v1/admin/dashboard
func RegisterRoutes(r chi.Router, h *handler.DashboardHandler, checker middleware.PermissionChecker) {
	r.Route("/admin/dashboard", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/overview", h.GetOverview)                      // admin:dashboard:list
		r.Get("/trend", h.GetTrend)                            // admin:dashboard:list
		r.Get("/tenants-health", h.GetTenantHealth)            // admin:dashboard:list
		r.Get("/tenants-health/{id}", h.GetTenantHealthDetail) // admin:dashboard:list
		r.Get("/slow-queries", h.GetSlowQueries)               // admin:dashboard:list
		r.Post("/slow-queries/reset", h.ResetSlowQueries)      // admin:dashboard:create
	})
}

// RegisterTenantRoutes 注册租户侧看板路由（仅登录，不挂权限码）。
// 与 task/wiki/message 一致，以当前 JWT 租户身份为访问边界。
// 挂载路径前缀: /api/v1/dashboard
func RegisterTenantRoutes(r chi.Router, h *handler.DashboardHandler) {
	r.Route("/dashboard", func(r chi.Router) {
		r.Get("/tenant-overview", h.GetTenantOverview) // 当前租户总览（成员/任务/Wiki/存储）
	})
}
