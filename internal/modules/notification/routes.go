package notification

import (
	"meteorx/internal/middleware"
	"meteorx/internal/modules/notification/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册通知公告模块路由
// r: chi 路由器实例
// h: 公告处理器实例
// checker: 权限检查器
// 挂载路径前缀: /api/v1/admin/announcements
func RegisterRoutes(r chi.Router, h *handler.AnnouncementHandler, checker middleware.PermissionChecker) {
	r.Route("/admin/announcements", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/", h.List)                    // admin:announcement:list
		r.Post("/", h.Create)                 // admin:announcement:create
		r.Get("/{id}", h.Get)                 // admin:announcement:read
		r.Put("/{id}", h.Update)              // admin:announcement:update
		r.Put("/{id}/status", h.UpdateStatus) // admin:announcement:status
		r.Delete("/{id}", h.Delete)           // admin:announcement:delete
	})
}

// RegisterTenantRoutes 注册租户端公告路由
// 挂载路径前缀: /api/v1/announcements
func RegisterTenantRoutes(r chi.Router, h *handler.AnnouncementHandler) {
	r.Route("/announcements", func(r chi.Router) {
		r.Get("/", h.ListForTenant)
	})
}
