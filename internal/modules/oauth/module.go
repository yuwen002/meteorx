// Package oauth 提供第三方 OAuth2 登录集成，支持 GitHub/Google 等 Provider 授权码流程及账号绑定。
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

// InitModule 迁移 OAuth 账号表、装配依赖并注册公开的 OAuth 授权/回调路由。
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

// InitProtectedModule 装配 OAuth 依赖并注册需登录的 OAuth 账号绑定/解绑路由。
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
