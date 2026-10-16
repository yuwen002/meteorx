package oauth

import (
	"meteorx/internal/middleware"
	"meteorx/internal/modules/oauth/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 编排 OAuth2 第三方登录公开接口（免登录，用于发起授权与回调）
func RegisterRoutes(r chi.Router, h *handler.OAuthHandler) {
	r.Route("/auth/oauth", func(r chi.Router) {
		r.Get("/{provider}/redirect", h.GetRedirectURL) // 获取第三方授权跳转地址
		r.Post("/callback", h.Callback)                 // 第三方授权回调，完成登录或注册
		r.Get("/tenants", h.GetTenantList)              // 回调后查询该 OAuth 账号关联的租户列表
		r.Post("/token/refresh", h.RefreshToken)        // 刷新 OAuth 临时 Token
		r.Get("/providers", h.ListProviders)            // 获取已启用第三方登录提供商列表
		r.Post("/complete-email", h.CompleteEmail)      // 第三方登录后补全邮箱
	})
}

// RegisterProtectedRoutes 编排 OAuth2 账号管理接口（需登录，用于查看绑定、解绑、绑定新账号）
func RegisterProtectedRoutes(r chi.Router, h *handler.OAuthHandler) {
	r.Get("/auth/oauth/accounts", h.ListAccounts)  // 查看当前用户已绑定的第三方账号列表
	r.Post("/auth/oauth/unbind", h.Unbind)         // 解绑指定第三方账号
	r.Post("/auth/oauth/bind", h.Bind)             // 绑定新的第三方账号
}

// RegisterAdminRoutes 编排第三方登录提供商配置管理接口（平台超级管理员）。
// 挂载 AutoRequirePermission 中间件做细粒度权限校验，权限码形如 admin:oauth_provider:list。
func RegisterAdminRoutes(r chi.Router, h *handler.OAuthProviderAdminHandler, checker middleware.PermissionChecker) {
	r.Route("/admin/oauth-providers", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/", h.ListProviders)                   // 配置列表
		r.Post("/", h.CreateProvider)                 // 新建配置
		r.Get("/{code}", h.GetProvider)               // 配置详情
		r.Put("/{code}", h.UpdateProvider)            // 更新配置
		r.Delete("/{code}", h.DeleteProvider)         // 删除配置
		r.Put("/{code}/status", h.SetProviderStatus)  // 切换启用状态
	})
}