package audit

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func RegisterRoutes(r chi.Router, db *gorm.DB) {
	r.Route("/audit", func(r chi.Router) {
		r.Get("/logs", notImplemented)
		r.Get("/logs/{id}", notImplemented)
	})
}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "audit module is not implemented", http.StatusNotImplemented)
}
