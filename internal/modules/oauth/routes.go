package oauth

import (
	"meteorx/internal/modules/oauth/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes 挂载 OAuth2 路由
// 挂载路径前缀: /api/v1/auth/oauth
func RegisterRoutes(r chi.Router, h *handler.OAuthHandler) {
	r.Route("/auth/oauth", func(r chi.Router) {
		r.Get("/{provider}/redirect", h.GetRedirectURL)
		r.Post("/callback", h.Callback)
	})
}