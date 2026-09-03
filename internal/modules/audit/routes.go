package audit

import (
	"meteorx/internal/middleware"
	"meteorx/internal/modules/audit/handler"
	"meteorx/pkg/iplocation"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册审计日志模块路由
// r: chi 路由器实例
// h: 审计日志处理器实例
// alertH: 告警处理器实例
// sessionH: 会话分析处理器实例
// checker: 权限检查器
// ipLocator: IP地理位置解析器
// 挂载路径前缀: /api/v1/audit
func RegisterRoutes(r chi.Router, h *handler.AuditHandler, alertH *handler.AlertHandler, sessionH *handler.SessionHandler, checker middleware.PermissionChecker, ipLocator iplocation.IPLocator) {
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

		// 告警规则管理（需要权限）
		r.Route("/alert-rules", func(r chi.Router) {
			r.Use(middleware.AutoRequirePermission(checker))
			r.Get("/", alertH.ListRules)           // audit:alert-rule:list
			r.Post("/", alertH.CreateRule)         // audit:alert-rule:create
			r.Put("/{id}", alertH.UpdateRule)      // audit:alert-rule:update
			r.Delete("/{id}", alertH.DeleteRule)   // audit:alert-rule:delete
		})

		// 告警记录查询（需要权限）
		r.Route("/alerts", func(r chi.Router) {
			r.Use(middleware.AutoRequirePermission(checker))
			r.Get("/", alertH.ListAlerts)          // audit:alert:list
		})

		// 会话分析（需要权限）
		r.Route("/sessions", func(r chi.Router) {
			r.Use(middleware.AutoRequirePermission(checker))
			r.Get("/", sessionH.ListSessions)           // audit:session:list
			r.Get("/{id}/logs", sessionH.GetSessionLogs) // audit:session:read
		})
	})
}