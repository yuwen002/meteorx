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
	r.Route("/dashboard", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/overview", h.GetOverview) // admin:dashboard:list
	})
}
