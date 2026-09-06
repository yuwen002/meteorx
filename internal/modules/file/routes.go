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

// RegisterRoutes 注册文件管理模块路由，自动组装依赖与权限检查器
func RegisterRoutes(r chi.Router, db *gorm.DB, cfg *config.Config) {
	// 初始化依赖
	fileRepo := filerepo.NewFileRepository(db)
	fileStorage := storage.NewStorage(cfg.File, cfg.JWT.Secret)
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
		r.Use(middleware.AutoRequirePermission(checker))

		r.Post("/upload", fileHandler.Upload)                    // 上传文件
		r.Get("/", fileHandler.ListByTenant)                     // 租户文件列表
		r.Get("/my", fileHandler.ListByUser)                     // 我的文件列表
		r.Get("/{id}", fileHandler.GetByID)                      // 文件详情
		r.Get("/{id}/download", fileHandler.Download)            // 下载文件
		r.Put("/{id}", fileHandler.Update)                       // 更新文件信息
		r.Delete("/{id}", fileHandler.Delete)                    // 删除文件
		r.Post("/batch/delete", fileHandler.BatchDelete)         // 批量删除文件
		r.Get("/deleted", fileHandler.GetDeletedList)            // 回收站列表
		r.Put("/{id}/restore", fileHandler.Restore)              // 从回收站恢复
		r.Delete("/{id}/permanent", fileHandler.PermanentDelete) // 永久删除
	})
}
