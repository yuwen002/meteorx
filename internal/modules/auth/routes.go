package auth

import (
	"meteorx/internal/modules/auth/handler"
	"meteorx/internal/modules/auth/service"
	"meteorx/internal/modules/user/repository"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 挂载认证相关路由
func RegisterRoutes(r chi.Router, userRepo repository.UserRepository) {
	// 1. 初始化 Service 和 Handler
	svc := service.NewAuthService(userRepo)
	h := handler.NewAuthHandler(svc)

	// 2. 定义路由组
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
	})
}
