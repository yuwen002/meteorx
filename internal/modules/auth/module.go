package auth

import (
	"meteorx/internal/common/jwt"
	"meteorx/internal/config"
	"meteorx/internal/modules/auth/handler"
	"meteorx/internal/modules/auth/service"
	userrepo "meteorx/internal/modules/user/repository"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// InitModule 现在的签名增加了 config.JWTConfig
func InitModule(r chi.Router, db *gorm.DB, cfg config.JWTConfig) {
	// 1. 根据配置创建 JWT 助手（这是关联的关键点）
	tokenHelper := jwt.NewTokenHelper(cfg)

	// 2. 初始化依赖：Repository -> Service -> Handler
	uRepo := userrepo.NewUserRepository(db)
	svc := service.NewAuthService(uRepo, tokenHelper) // 注入 TokenHelper
	h := handler.NewAuthHandler(svc)

	// 3. 注册路由
	RegisterRoutes(r, h)
}
