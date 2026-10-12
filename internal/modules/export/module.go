// Package export 数据导出模块入口。
// 提供 CSV 流式异步导出、下载中心、过期清理等能力。
package export

import (
	"meteorx/internal/config"
	"meteorx/internal/middleware"
	"meteorx/internal/modules/export/handler"
	"meteorx/internal/modules/export/repository"
	"meteorx/internal/modules/export/service"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	rbacSvc "meteorx/internal/modules/rbac/service"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// newPermissionChecker 复用 RBAC 服务作为权限校验器
func newPermissionChecker(db *gorm.DB) middleware.PermissionChecker {
	permRepo := rbacRepo.NewPermissionRepository(db)
	roleRepo := rbacRepo.NewRoleRepository(db)
	rolePermRepo := rbacRepo.NewRolePermissionRepository(db)
	userRoleRepo := rbacRepo.NewUserRoleRepository(db)
	return rbacSvc.NewRBACService(roleRepo, permRepo, rolePermRepo, userRoleRepo, nil, nil)
}

// buildServiceConfig 从全局配置构造导出服务配置，未配置时回落默认
func buildServiceConfig(cfg config.Config) service.Config {
	export := cfg.Export
	dir := export.Dir
	if dir == "" {
		dir = "data/exports"
	}
	retention := export.RetentionDays
	if retention <= 0 {
		retention = 7
	}
	workers := export.WorkerCount
	if workers <= 0 {
		workers = 3
	}
	batch := export.BatchSize
	if batch <= 0 {
		batch = 1000
	}
	return service.Config{
		Dir:           dir,
		RetentionDays: retention,
		WorkerCount:   workers,
		BatchSize:     batch,
	}
}

// InitModule 装配依赖并注册受保护路由。
// 目前仅包含 CRUD 端点，worker 池与过期清理任务由 StartExportWorker / StartExportExpiryJob 后续接入。
func InitModule(r chi.Router, db *gorm.DB, cfg config.Config) {
	repo := repository.NewExportRepository(db)
	svc := service.NewExportService(repo, buildServiceConfig(cfg))
	h := handler.NewExportHandler(svc)
	checker := newPermissionChecker(db)
	RegisterRoutes(r, h, checker)
}
