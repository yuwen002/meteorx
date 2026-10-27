// Package audit 提供审计日志模块，记录用户操作轨迹、异常检测、告警规则和会话分析。
package audit

import (
	"time"

	"meteorx/internal/config"
	"meteorx/internal/middleware"
	"meteorx/internal/modules/audit/handler"
	"meteorx/internal/modules/audit/repository"
	"meteorx/internal/modules/audit/service"
	rbacrepo "meteorx/internal/modules/rbac/repository"
	rbacsvc "meteorx/internal/modules/rbac/service"
	"meteorx/internal/pkg/emailer"
	"meteorx/pkg/iplocation"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// NewIntegrityGuard 依配置构造审计行级完整性摘要器。
// 引导层用它构造唯一实例，同时注入中间件/批量侧与审计管理侧两个 AuditService，
// 保证两条写入路径算出的 RecordHash 一致（否则完整性校验必坏）。
func NewIntegrityGuard(cfg config.Config) *service.IntegrityGuard {
	return service.NewIntegrityGuard(cfg)
}

// InitModule 初始化审计日志模块
// 挂载在平台管理员路由组下，需要超级管理员权限
// integrity 须与中间件/批量侧共用同一实例，令两条写入路径的行摘要口径一致
func InitModule(r chi.Router, db *gorm.DB, emailCfg *emailer.Emailer, integrity *service.IntegrityGuard) {
	repo := repository.NewAuditLogRepository(db)
	alertRepo := repository.NewAlertRuleRepository(db)
	ipLocator := iplocation.NewHTTPLocator("ip-api", 3*time.Second)

	svc := service.NewAuditService(repo, service.WithIntegrity(integrity))
	alertSvc := service.NewAlertService(alertRepo, emailCfg)
	sessionSvc := service.NewSessionService(repo)
	integritySvc := service.NewIntegrityService(integrity, repository.NewIntegrityRepository(db))
	integrityCoordinator := NewIntegrityCoordinator(integritySvc)

	h := handler.NewAuditHandler(svc)
	alertH := handler.NewAlertHandler(alertSvc)
	sessionH := handler.NewSessionHandler(sessionSvc)
	integrityH := handler.NewIntegrityHandler(integrityCoordinator)

	_ = repository.InitDefaultAlertRules(db)

	checker := initPermissionChecker(db)
	RegisterRoutes(r, h, alertH, sessionH, integrityH, checker, ipLocator)
}

// initPermissionChecker 创建权限检查器（复用 RBACService）
func initPermissionChecker(db *gorm.DB) middleware.PermissionChecker {
	roleRepo := rbacrepo.NewRoleRepository(db)
	permRepo := rbacrepo.NewPermissionRepository(db)
	rolePermRepo := rbacrepo.NewRolePermissionRepository(db)
	userRoleRepo := rbacrepo.NewUserRoleRepository(db)
	return rbacsvc.NewRBACService(roleRepo, permRepo, rolePermRepo, userRoleRepo, nil, nil)
}
