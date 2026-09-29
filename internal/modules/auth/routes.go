package auth

import (
	"meteorx/internal/modules/auth/handler"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *handler.AuthHandler) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/logout", h.Logout)
		r.Post("/forgot-password", h.ForgotPassword)
		r.Post("/reset-password", h.ResetPassword)
		r.Post("/email/send-verification", h.SendEmailVerification)
		r.Post("/email/verify", h.VerifyEmail)
	})
}

func RegisterAPITokenRoutes(r chi.Router, h *handler.APITokenHandler) {
	r.Route("/auth/tokens", func(r chi.Router) {
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Post("/revoke", h.Revoke)
	})
}