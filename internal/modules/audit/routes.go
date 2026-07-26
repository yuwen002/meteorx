package audit

import (
	"meteorx/internal/modules/audit/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册审计日志模块路由
// r: chi 路由器实例
// h: 审计日志处理器实例
// 挂载路径前缀: /api/v1/audit
func RegisterRoutes(r chi.Router, h *handler.AuditHandler) {
	r.Route("/audit", func(r chi.Router) {
		// GET /api/v1/audit/stats - Dashboard 统计
		r.Get("/stats", h.GetStats)

		// 审计日志列表和详情
		r.Route("/logs", func(r chi.Router) {
			r.Get("/", h.ListLogs)               // audit:log:list（审计日志列表）
			r.Get("/export", h.ExportLogs)       // audit:log:export（导出审计日志）
			r.Post("/", h.CreateLog)             // audit:log:create（创建审计日志，内部使用）
			r.Get("/{id}", h.GetLog)             // audit:log:read（审计日志详情）
			r.Delete("/cleanup", h.CleanupLogs)  // audit:log:cleanup（清理日志）
		})
	})
}