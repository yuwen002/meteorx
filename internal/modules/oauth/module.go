package oauth

import (
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"meteorx/internal/common/jwt"
	"meteorx/internal/config"
	"meteorx/internal/modules/oauth/handler"
	"meteorx/internal/modules/oauth/service"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	userrepo "meteorx/internal/modules/user/repository"
)

// InitModule 初始化 OAuth2 模块
func InitModule(r chi.Router, db *gorm.DB, cfg config.Config, tokenHelper *jwt.TokenHelper) {
	// 初始化依赖
	uRepo := userrepo.NewUserRepository(db)
	rRepo := rbacRepo.NewRoleRepository(db)
	urRepo := rbacRepo.NewUserRoleRepository(db)
	rpRepo := rbacRepo.NewRolePermissionRepository(db)

	svc := service.NewOAuthService(cfg.OAuth, uRepo, rRepo, urRepo, rpRepo, tokenHelper)
	h := handler.NewOAuthHandler(svc)

	// 注册路由
	RegisterRoutes(r, h)
}