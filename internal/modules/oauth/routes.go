package oauth

import (
	"meteorx/internal/modules/oauth/handler"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *handler.OAuthHandler) {
	r.Route("/auth/oauth", func(r chi.Router) {
		r.Get("/{provider}/redirect", h.GetRedirectURL)
		r.Post("/callback", h.Callback)
		r.Get("/tenants", h.GetTenantList)
		r.Post("/token/refresh", h.RefreshToken)
	})
}

func RegisterProtectedRoutes(r chi.Router, h *handler.OAuthHandler) {
	r.Route("/auth/oauth", func(r chi.Router) {
		r.Get("/accounts", h.ListAccounts)
		r.Post("/unbind", h.Unbind)
		r.Post("/bind", h.Bind)
	})
}