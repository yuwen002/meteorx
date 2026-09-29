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