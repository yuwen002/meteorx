package user

import (
	"meteorx/internal/modules/user/handler"
	"meteorx/internal/modules/user/repository"
	"meteorx/internal/modules/user/service"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// 提取公共工厂方法，保持与租户模块结构一致
func initHandler(db *gorm.DB) *handler.UserHandler {
	repo := repository.NewUserRepository(db)
	svc := service.NewUserService(repo)
	return handler.NewUserHandler(svc)
}

// InitModule 用户模块初始化（普通租户用户管理）
func InitModule(r chi.Router, db *gorm.DB) {
	h := initHandler(db)
	RegisterRoutes(r, h)
}

// InitAdminModule 用户模块的系统管理员接口初始化
func InitAdminModule(r chi.Router, db *gorm.DB) {
	h := initHandler(db)
	RegisterAdminRoutes(r, h)
}