package user

import (
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func InitModule(r chi.Router, db *gorm.DB) {
	RegisterRoutes(r, db)
}
