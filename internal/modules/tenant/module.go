package tenant

import (
	"meteorx/internal/modules/tenant/handler"
	"meteorx/internal/modules/tenant/repository"
	"meteorx/internal/modules/tenant/service"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// 提取公共工厂方法，保持不变
func initHandler(db *gorm.DB) *handler.TenantHandler {
	repo := repository.NewTenantRepository(db)
	svc := service.NewTenantService(repo)
	return handler.NewTenantHandler(svc)
}

// InitPublicModule 1. 完全公开的租户接口入口
func InitPublicModule(r chi.Router, db *gorm.DB) {
	h := initHandler(db)
	RegisterPublicRoutes(r, h) // 👈 扔给 routes.go 去编排路径
}

// InitPrivateModule 2. 租户内部的管理私有接口入口
func InitPrivateModule(r chi.Router, db *gorm.DB) {
	h := initHandler(db)
	RegisterPrivateRoutes(r, h) // 👈 扔给 routes.go 去编排路径
}

// InitAdminModule 3. MaaS 平台超级管理员的控制台接口入口
func InitAdminModule(r chi.Router, db *gorm.DB) {
	h := initHandler(db)
	RegisterAdminRoutes(r, h) // 👈 扔给 routes.go 去编排路径
}
