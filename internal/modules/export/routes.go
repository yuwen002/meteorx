package export

import (
	"meteorx/internal/middleware"
	exportHandler "meteorx/internal/modules/export/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册受保护的导出路由，全部走自动权限推导。
func RegisterRoutes(r chi.Router, h *exportHandler.ExportHandler, checker middleware.PermissionChecker) {
	r.Route("/exports", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Get("/{id}/download", h.Download)
		r.Delete("/{id}", h.Delete)
	})
}
