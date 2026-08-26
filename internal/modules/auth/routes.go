package auth

import (
	"meteorx/internal/modules/auth/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 挂载认证相关路由（注册、登录、登出、密码找回）
// 挂载路径前缀: /api/v1/auth
func RegisterRoutes(r chi.Router, h *handler.AuthHandler) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)         // 用户注册
		r.Post("/login", h.Login)               // 用户登录
		r.Post("/logout", h.Logout)             // 用户登出
		r.Post("/forgot-password", h.ForgotPassword) // 忘记密码
		r.Post("/reset-password", h.ResetPassword)   // 重置密码
	})
}