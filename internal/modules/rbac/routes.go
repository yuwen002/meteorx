package rbac

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func RegisterRoutes(r chi.Router, db *gorm.DB) {
	r.Route("/rbac", func(r chi.Router) {
		r.Route("/roles", func(r chi.Router) {
			r.Get("/", notImplemented)
			r.Get("/{id}", notImplemented)
			r.Post("/", notImplemented)
			r.Put("/{id}", notImplemented)
			r.Delete("/{id}", notImplemented)
		})

		r.Route("/permissions", func(r chi.Router) {
			r.Get("/", notImplemented)
			r.Get("/{id}", notImplemented)
			r.Post("/", notImplemented)
			r.Put("/{id}", notImplemented)
			r.Delete("/{id}", notImplemented)
		})
	})
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "rbac module is not implemented", http.StatusNotImplemented)
}
