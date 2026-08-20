package file

import (
	"meteorx/internal/config"
	"meteorx/internal/middleware"
	"meteorx/internal/modules/file/handler"
	filerepo "meteorx/internal/modules/file/repository"
	"meteorx/internal/modules/file/service"
	"meteorx/internal/modules/file/storage"
	rbacrepo "meteorx/internal/modules/rbac/repository"
	rbacsvc "meteorx/internal/modules/rbac/service"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// RegisterRoutes 注册文件管理模块路由
func RegisterRoutes(r chi.Router, db *gorm.DB, cfg *config.Config) {
	// 初始化依赖
	fileRepo := filerepo.NewFileRepository(db)
	fileStorage := storage.NewStorage(cfg.File)
	fileSvc := service.NewFileService(fileRepo, fileStorage)
	fileHandler := handler.NewFileHandler(fileSvc, cfg.File)

	// 初始化权限检查器
	roleRepo := rbacrepo.NewRoleRepository(db)
	permRepo := rbacrepo.NewPermissionRepository(db)
	rolePermRepo := rbacrepo.NewRolePermissionRepository(db)
	userRoleRepo := rbacrepo.NewUserRoleRepository(db)
	checker := rbacsvc.NewRBACService(roleRepo, permRepo, rolePermRepo, userRoleRepo)

	// 文件管理路由组
	r.Route("/files", func(r chi.Router) {
		// 需要权限校验的路由
		r.Use(middleware.AutoRequirePermission(checker))

		// 文件上传
		r.Post("/upload", fileHandler.Upload)

		// 文件列表
		r.Get("/", fileHandler.ListByTenant)
		r.Get("/my", fileHandler.ListByUser)

		// 文件详情
		r.Get("/{id}", fileHandler.GetByID)

		// 文件下载
		r.Get("/{id}/download", fileHandler.Download)

		// 文件更新
		r.Put("/{id}", fileHandler.Update)

		// 文件删除
		r.Delete("/{id}", fileHandler.Delete)
		r.Post("/batch/delete", fileHandler.BatchDelete)

		// 回收站
		r.Get("/deleted", fileHandler.GetDeletedList)
		r.Put("/{id}/restore", fileHandler.Restore)
		r.Delete("/{id}/permanent", fileHandler.PermanentDelete)
	})
}