package audit

import (
	"meteorx/internal/cache"
	"meteorx/internal/database"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r *chi.Mux, db *database.DB, cache *cache.Redis) {
	r.Route("/api/v1/audit", func(r chi.Router) {
		r.Get("/logs", nil)      // handler
		r.Get("/logs/{id}", nil) // handler
	})
}
