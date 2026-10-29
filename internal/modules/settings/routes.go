package settings

import (
	"meteorx/internal/middleware"
	"meteorx/internal/modules/settings/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterAdminRoutes 注册平台管理员配置中心路由
// 挂载 AutoRequirePermission：GET → admin:settings:list，PUT → admin:settings:update
func RegisterAdminRoutes(r chi.Router, h *handler.SettingHandler, checker middleware.PermissionChecker) {
	r.Route("/admin/settings", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/", h.ListSettings)   // 配置列表（含元数据与来源）
		r.Put("/", h.UpdateSettings) // 批量更新配置
	})
}

// RegisterPublicRoutes 注册公开配置路由（免鉴权，登录/注册页消费）
func RegisterPublicRoutes(r chi.Router, h *handler.SettingHandler) {
	r.Get("/settings/public", h.GetPublicSettings) // 公开运行时配置
}
