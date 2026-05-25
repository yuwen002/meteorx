package rbac

import (
	"meteorx/internal/cache"
	"meteorx/internal/database"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r *chi.Mux, db *database.DB, cache *cache.Redis) {
	r.Route("/api/v1/rbac", func(r chi.Router) {
		r.Route("/roles", func(r chi.Router) {
			r.Get("/", nil)        // handler
			r.Get("/{id}", nil)    // handler
			r.Post("/", nil)       // handler
			r.Put("/{id}", nil)    // handler
			r.Delete("/{id}", nil) // handler
		})
		r.Route("/permissions", func(r chi.Router) {
			r.Get("/", nil)        // handler
			r.Get("/{id}", nil)    // handler
			r.Post("/", nil)       // handler
			r.Put("/{id}", nil)    // handler
			r.Delete("/{id}", nil) // handler
		})
	})
}
