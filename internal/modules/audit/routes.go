package audit

import (
	"meteorx/internal/modules/audit/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册审计日志模块路由
func RegisterRoutes(r chi.Router, h *handler.AuditHandler) {
	r.Route("/audit", func(r chi.Router) {
		// Dashboard 统计
		r.Get("/stats", h.GetStats)

		// 审计日志列表和详情
		r.Route("/logs", func(r chi.Router) {
			r.Get("/", h.ListLogs)               // 审计日志列表
			r.Get("/export", h.ExportLogs)       // 导出审计日志
			r.Post("/", h.CreateLog)             // 创建审计日志（内部使用）
			r.Get("/{id}", h.GetLog)             // 审计日志详情
			r.Delete("/cleanup", h.CleanupLogs)  // 清理日志
		})
	})
}