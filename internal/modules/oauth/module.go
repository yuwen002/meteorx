// Package oauth 提供第三方 OAuth2 登录集成，支持 GitHub/Google 等 Provider 授权码流程及账号绑定。
package oauth

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"meteorx/internal/cache"
	"meteorx/internal/common/jwt"
	"meteorx/internal/config"
	"meteorx/internal/middleware"
	"meteorx/internal/modules/oauth/handler"
	"meteorx/internal/modules/oauth/model"
	"meteorx/internal/modules/oauth/repository"
	"meteorx/internal/modules/oauth/service"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	rbacSvc "meteorx/internal/modules/rbac/service"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	userrepo "meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"
	"meteorx/pkg/logger"
)

// InitModule 迁移 OAuth 账号表、装配依赖并注册公开的 OAuth 授权/回调路由。
func InitModule(r chi.Router, db *gorm.DB, cfg config.Config, tokenHelper *jwt.TokenHelper, rdb *cache.Redis) {
	if err := db.AutoMigrate(&model.OAuthAccount{}); err != nil {
		logger.Warnf("[oauth] AutoMigrate OAuthAccount failed: %v", err)
	}
	// oauth_providers 必须用 PO 迁移：领域模型含 []string 字段，GORM 无法直接迁移
	if err := repository.AutoMigrateProvider(db); err != nil {
		logger.Warnf("[oauth] AutoMigrate oauth_providers failed: %v", err)
	}

	uRepo := userrepo.NewUserRepository(db)
	rRepo := rbacRepo.NewRoleRepository(db)
	urRepo := rbacRepo.NewUserRoleRepository(db)
	rpRepo := rbacRepo.NewRolePermissionRepository(db)
	tRepo := tenantRepo.NewTenantRepository(db)
	oauthRepo := repository.NewOAuthAccountRepository(db)
	providerRepo := repository.NewOAuthProviderRepository(db)

	svc := newOAuthService(cfg.OAuth, uRepo, rRepo, urRepo, rpRepo, tRepo, oauthRepo, providerRepo, tokenHelper, rdb)

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

	svc := service.NewOAuthService(cfg.OAuth, uRepo, rRepo, urRepo, rpRepo, tRepo, oauthRepo, tokenHelper, rdb, nil, nil)
	h := handler.NewOAuthHandler(svc)

	RegisterProtectedRoutes(r, h)
}

// InitAdminModule 装配 OAuth 提供商配置管理依赖、注册平台级 CRUD 路由，并启动注册表定时刷新。
// ctx 用于优雅停止后台刷新协程。
func InitAdminModule(ctx context.Context, r chi.Router, db *gorm.DB, cfg config.Config, tokenHelper *jwt.TokenHelper, rdb *cache.Redis) {
	uRepo := userrepo.NewUserRepository(db)
	rRepo := rbacRepo.NewRoleRepository(db)
	urRepo := rbacRepo.NewUserRoleRepository(db)
	rpRepo := rbacRepo.NewRolePermissionRepository(db)
	tRepo := tenantRepo.NewTenantRepository(db)
	oauthRepo := repository.NewOAuthAccountRepository(db)
	providerRepo := repository.NewOAuthProviderRepository(db)

	cipher, err := crypto.NewSecretCipher(cfg.OAuth.SecretKey)
	if err != nil {
		// 主密钥格式错误不应阻断启动，但必须显式告警：此时无法安全保存 client_secret
		logger.Warnf("[oauth] invalid oauth.secret_key, provider secrets cannot be encrypted: %v", err)
	}
	if !cipher.Enabled() {
		logger.Warnf("[oauth] oauth.secret_key 未配置，后台保存的 client_secret 将被拒绝（需配置后才能启用 DB 提供商配置）")
	}

	svc := newOAuthService(cfg.OAuth, uRepo, rRepo, urRepo, rpRepo, tRepo, oauthRepo, providerRepo, tokenHelper, rdb)

	h := handler.NewOAuthProviderAdminHandler(svc)
	RegisterAdminRoutes(r, h, initPermissionChecker(db))

	// 启动时立即加载一次，随后定时刷新（多副本下任一副本写入后其余副本在此间隔内收敛）
	go func() {
		if err := svc.ReloadProviders(ctx); err != nil {
			logger.Warnf("[oauth] initial provider reload failed: %v", err)
		}
		ticker := time.NewTicker(service.ProviderReloadInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := svc.ReloadProviders(ctx); err != nil {
					logger.Warnf("[oauth] periodic provider reload failed: %v", err)
				}
			}
		}
	}()
}

// newOAuthService 装配带 DB 提供商存储的 OAuth 服务（初始化与管理员模块共用）。
func newOAuthService(
	cfg config.OAuthConfig,
	uRepo userrepo.UserRepository,
	rRepo rbacRepo.RoleRepository,
	urRepo rbacRepo.UserRoleRepository,
	rpRepo rbacRepo.RolePermissionRepository,
	tRepo tenantRepo.TenantRepository,
	oauthRepo repository.OAuthAccountRepository,
	providerRepo repository.OAuthProviderRepository,
	tokenHelper *jwt.TokenHelper,
	rdb *cache.Redis,
) *service.OAuthService {
	cipher, err := crypto.NewSecretCipher(cfg.SecretKey)
	if err != nil {
		logger.Warnf("[oauth] invalid oauth.secret_key: %v", err)
	}
	return service.NewOAuthService(cfg, uRepo, rRepo, urRepo, rpRepo, tRepo, oauthRepo, tokenHelper, rdb, providerRepo, cipher)
}

// initPermissionChecker 创建权限检查器（复用 RBACService）
func initPermissionChecker(db *gorm.DB) middleware.PermissionChecker {
	roleRepo := rbacRepo.NewRoleRepository(db)
	permRepo := rbacRepo.NewPermissionRepository(db)
	rolePermRepo := rbacRepo.NewRolePermissionRepository(db)
	userRoleRepo := rbacRepo.NewUserRoleRepository(db)
	return rbacSvc.NewRBACService(roleRepo, permRepo, rolePermRepo, userRoleRepo, nil, nil)
}
