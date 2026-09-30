// Package auth 提供认证模块，处理登录/登出、Token 签发刷新、API Token 管理及邮箱验证。
package auth

import (
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"meteorx/internal/cache"
	"meteorx/internal/common/jwt"
	"meteorx/internal/config"
	"meteorx/internal/modules/auth/handler"
	"meteorx/internal/modules/auth/model"
	"meteorx/internal/modules/auth/repository"
	"meteorx/internal/modules/auth/service"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	userrepo "meteorx/internal/modules/user/repository"
)

// InitModule 装配认证依赖（仓储/服务/处理器）并注册公开认证路由，返回 AuthHandler 供后续需登录路由复用。
func InitModule(r chi.Router, db *gorm.DB, cfg config.Config, rdb *cache.Redis) *handler.AuthHandler {
	tokenHelper := jwt.NewTokenHelper(cfg.JWT)

	uRepo := userrepo.NewUserRepository(db)
	rRepo := rbacRepo.NewRoleRepository(db)
	urRepo := rbacRepo.NewUserRoleRepository(db)
	rpRepo := rbacRepo.NewRolePermissionRepository(db)
	svc := service.NewAuthService(uRepo, rRepo, urRepo, rpRepo, tokenHelper, rdb, cfg.Security, cfg.Email, cfg.Client)
	h := handler.NewAuthHandler(svc)

	RegisterRoutes(r, h)
	return h
}

// InitAPITokenModule 迁移 API Token 表、装配依赖并注册需登录的 API Token 管理路由。
func InitAPITokenModule(r chi.Router, db *gorm.DB, cfg config.Config, rdb *cache.Redis) {
	db.AutoMigrate(&model.APIToken{})

	apiTokenRepo := repository.NewAPITokenRepository(db)
	uRepo := userrepo.NewUserRepository(db)
	urRepo := rbacRepo.NewUserRoleRepository(db)
	rpRepo := rbacRepo.NewRolePermissionRepository(db)

	svc := service.NewAPITokenService(apiTokenRepo, uRepo, urRepo, rpRepo, rdb, cfg.Auth)
	h := handler.NewAPITokenHandler(svc)

	RegisterAPITokenRoutes(r, h)
}
