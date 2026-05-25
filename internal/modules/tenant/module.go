package tenant

import (
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"meteorx/internal/modules/tenant/handler"
	"meteorx/internal/modules/tenant/repository"
	"meteorx/internal/modules/tenant/service"
)

// InitModule 一键初始化模块并注册路由
func InitModule(r chi.Router, db *gorm.DB) {
	repo := repository.NewTenantRepository(db)
	svc := service.NewTenantService(repo)
	h := handler.NewTenantHandler(svc)
	
	RegisterRoutes(r, h)
}
