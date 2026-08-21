package audit

import (
	"meteorx/internal/middleware"
	"meteorx/internal/modules/audit/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册审计日志模块路由
// r: chi 路由器实例
// h: 审计日志处理器实例
// checker: 权限检查器
// 挂载路径前缀: /api/v1/audit
func RegisterRoutes(r chi.Router, h *handler.AuditHandler, checker middleware.PermissionChecker) {
	r.Route("/audit", func(r chi.Router) {
		// Dashboard 统计接口：无需细粒度权限校验（只需登录）
		r.Get("/stats", h.GetStats)
		r.Get("/dashboard", h.GetDashboard)

		// 需要细粒度权限校验的路由组
		r.Route("/logs", func(r chi.Router) {
			r.Use(middleware.AutoRequirePermission(checker))
			r.Get("/", h.ListLogs)               // audit:log:list
			r.Get("/export", h.ExportLogs)       // audit:log:export
			r.Post("/", h.CreateLog)             // audit:log:create
			r.Get("/{id}", h.GetLog)             // audit:log:read
			r.Delete("/cleanup", h.CleanupLogs)  // audit:log:cleanup
		})
	})
}