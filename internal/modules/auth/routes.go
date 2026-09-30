package auth

import (
	"meteorx/internal/modules/auth/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 注册公开认证路由（无需登录）：注册、登录、登出、忘记密码与重置密码。
func RegisterRoutes(r chi.Router, h *handler.AuthHandler) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/logout", h.Logout)
		r.Post("/forgot-password", h.ForgotPassword)
		r.Post("/reset-password", h.ResetPassword)
	})
}

// RegisterEmailVerificationRoutes 注册邮箱验证路由（需登录）。
// 与公开路由分离，防止未授权用户滥用验证邮件发送。
func RegisterEmailVerificationRoutes(r chi.Router, h *handler.AuthHandler) {
	r.Route("/auth/email", func(r chi.Router) {
		r.Post("/send-verification", h.SendEmailVerification)
		r.Post("/verify", h.VerifyEmail)
	})
}

// RegisterAPITokenRoutes 注册 API Token 管理路由（需登录）：列表/创建/撤销。
func RegisterAPITokenRoutes(r chi.Router, h *handler.APITokenHandler) {
	r.Route("/auth/tokens", func(r chi.Router) {
		r.Get("/", h.List)          // 获取 API Token 列表
		r.Post("/", h.Create)       // 创建 API Token
		r.Post("/revoke", h.Revoke) // 撤销 API Token
	})
}
