package oauth

import (
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
	})
}

// RegisterProtectedRoutes 编排 OAuth2 账号管理接口（需登录，用于查看绑定、解绑、绑定新账号）
func RegisterProtectedRoutes(r chi.Router, h *handler.OAuthHandler) {
	r.Get("/auth/oauth/accounts", h.ListAccounts)  // 查看当前用户已绑定的第三方账号列表
	r.Post("/auth/oauth/unbind", h.Unbind)         // 解绑指定第三方账号
	r.Post("/auth/oauth/bind", h.Bind)             // 绑定新的第三方账号
}