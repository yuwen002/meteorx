// Package task 任务管理模块入口。
// 提供个人待办与租户团队协作任务的创建、查询、更新、完成/重开、删除与统计能力。
// 路由挂载于受保护分组（需登录），访问控制以数据归属为准。
package task

import (
	"meteorx/internal/modules/task/handler"
	"meteorx/internal/modules/task/repository"
	"meteorx/internal/modules/task/service"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// InitModule 装配任务依赖并注册需登录的任务路由。
func InitModule(r chi.Router, db *gorm.DB) {
	repo := repository.NewTaskRepository(db)
	svc := service.NewTaskService(repo)
	h := handler.NewTaskHandler(svc)
	RegisterRoutes(r, h)
}

// RegisterRoutes 注册任务路由（需登录）。
// 静态段（/stats、/deleted、/batch/*）必须在 /{id} 之前注册，避免路径参数吞掉静态段。
func RegisterRoutes(r chi.Router, h *handler.TaskHandler) {
	r.Route("/tasks", func(r chi.Router) {
		r.Get("/stats", h.Stats)
		r.Get("/deleted", h.ListTrash)
		r.Post("/batch/complete", h.BatchComplete)
		r.Post("/batch/delete", h.BatchDelete)
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Put("/{id}/restore", h.Restore)
		r.Delete("/{id}/permanent", h.PermanentDelete)
		r.Put("/{id}/complete", h.Complete)
		r.Put("/{id}/reopen", h.Reopen)
	})
}
