package user

import (
	"meteorx/internal/modules/auth/handler"
	"meteorx/internal/modules/auth/service"
	"meteorx/internal/modules/user/repository"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type Module struct {
	Repo repository.UserRepository
}

func InitModule(r chi.Router, db *gorm.DB) {
	// 1. Auth 模块在初始化时，自己去建 User 的仓库实现
	// 这种方式最简单，避免了在 bootstrap 里传一大堆 Repo
	uRepo := userrepo.NewUserRepository(db)

	svc := service.NewAuthService(uRepo)
	h := handler.NewAuthHandler(svc)

	RegisterRoutes(r, h)
}
