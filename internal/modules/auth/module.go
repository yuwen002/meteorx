package auth

import (
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"meteorx/internal/cache"
	"meteorx/internal/common/jwt"
	"meteorx/internal/config"
	"meteorx/internal/modules/auth/handler"
	"meteorx/internal/modules/auth/service"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	userrepo "meteorx/internal/modules/user/repository"
)

// InitModule 现在的签名增加了 config.JWTConfig 和 Redis
func InitModule(r chi.Router, db *gorm.DB, cfg config.Config, rdb *cache.Redis) {
	// 1. 根据配置创建 JWT 助手（这是关联的关键点）
	tokenHelper := jwt.NewTokenHelper(cfg.JWT)

	// 2. 初始化依赖：Repository -> Service -> Handler
	uRepo := userrepo.NewUserRepository(db)
	rRepo := rbacRepo.NewRoleRepository(db)
	urRepo := rbacRepo.NewUserRoleRepository(db)
	rpRepo := rbacRepo.NewRolePermissionRepository(db)
	svc := service.NewAuthService(uRepo, rRepo, urRepo, rpRepo, tokenHelper, rdb, cfg.Security)
	h := handler.NewAuthHandler(svc)

	// 3. 注册路由
	RegisterRoutes(r, h)
}