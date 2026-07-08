package audit

import (
	"meteorx/internal/modules/audit/handler"
	"meteorx/internal/modules/audit/repository"
	"meteorx/internal/modules/audit/service"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// InitModule 初始化审计日志模块
// 挂载在平台管理员路由组下，需要超级管理员权限
func InitModule(r chi.Router, db *gorm.DB) {
	repo := repository.NewAuditLogRepository(db)
	svc := service.NewAuditService(repo)
	h := handler.NewAuditHandler(svc)
	RegisterRoutes(r, h)
}