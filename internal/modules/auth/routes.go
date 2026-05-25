package auth

import (
	"meteorx/internal/modules/auth/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 挂载认证相关路由
func RegisterRoutes(r chi.Router, h *handler.AuthHandler) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
	})
}
