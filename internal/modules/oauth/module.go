package oauth

import (
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"meteorx/internal/cache"
	"meteorx/internal/common/jwt"
	"meteorx/internal/config"
	"meteorx/internal/modules/oauth/handler"
	"meteorx/internal/modules/oauth/model"
	"meteorx/internal/modules/oauth/repository"
	"meteorx/internal/modules/oauth/service"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	userrepo "meteorx/internal/modules/user/repository"
)

func InitModule(r chi.Router, db *gorm.DB, cfg config.Config, tokenHelper *jwt.TokenHelper, rdb *cache.Redis) {
	db.AutoMigrate(&model.OAuthAccount{})

	uRepo := userrepo.NewUserRepository(db)
	rRepo := rbacRepo.NewRoleRepository(db)
	urRepo := rbacRepo.NewUserRoleRepository(db)
	rpRepo := rbacRepo.NewRolePermissionRepository(db)
	tRepo := tenantRepo.NewTenantRepository(db)
	oauthRepo := repository.NewOAuthAccountRepository(db)

	svc := service.NewOAuthService(cfg.OAuth, uRepo, rRepo, urRepo, rpRepo, tRepo, oauthRepo, tokenHelper, rdb)
	h := handler.NewOAuthHandler(svc)

	RegisterRoutes(r, h)
}

func InitProtectedModule(r chi.Router, db *gorm.DB, cfg config.Config, tokenHelper *jwt.TokenHelper, rdb *cache.Redis) {
	oauthRepo := repository.NewOAuthAccountRepository(db)
	uRepo := userrepo.NewUserRepository(db)
	rRepo := rbacRepo.NewRoleRepository(db)
	urRepo := rbacRepo.NewUserRoleRepository(db)
	rpRepo := rbacRepo.NewRolePermissionRepository(db)
	tRepo := tenantRepo.NewTenantRepository(db)

	svc := service.NewOAuthService(cfg.OAuth, uRepo, rRepo, urRepo, rpRepo, tRepo, oauthRepo, tokenHelper, rdb)
	h := handler.NewOAuthHandler(svc)

	RegisterProtectedRoutes(r, h)
}