package auth

import (
	"meteorx/internal/modules/auth/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 挂载认证相关路由
// r: chi 路由器实例
// h: 认证处理器实例，包含注册、登录、登出等处理方法
// 挂载路径前缀: /api/v1/auth
func RegisterRoutes(r chi.Router, h *handler.AuthHandler) {
	r.Route("/auth", func(r chi.Router) {
		// POST /api/v1/auth/register - 用户注册
		r.Post("/register", h.Register)
		// POST /api/v1/auth/login - 用户登录
		r.Post("/login", h.Login)
		// POST /api/v1/auth/logout - 用户登出
		r.Post("/logout", h.Logout)
		// POST /api/v1/auth/forgot-password - 忘记密码
		r.Post("/forgot-password", h.ForgotPassword)
		// POST /api/v1/auth/reset-password - 重置密码
		r.Post("/reset-password", h.ResetPassword)
	})
}