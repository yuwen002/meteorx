// Package task 任务管理模块入口。
// 提供个人待办与租户团队协作任务的创建、查询、更新、完成/重开、删除与统计能力，
// 并叠加可选的 AI 辅助能力（自然语言解析、建议、扩写、摘要）。
// 路由挂载于受保护分组（需登录），访问控制以数据归属为准。
package task

import (
	"meteorx/internal/ai"
	"meteorx/internal/config"
	"meteorx/internal/modules/task/handler"
	"meteorx/internal/modules/task/repository"
	"meteorx/internal/modules/task/service"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// InitModule 装配任务依赖并注册需登录的任务路由（含可选 AI 辅助端点）。
// aiCfg 控制 AI 叠加层：未启用/未配置凭据时 AI 端点自动降级为 503，不影响任务主流程。
func InitModule(r chi.Router, db *gorm.DB, aiCfg config.AIConfig) {
	repo := repository.NewTaskRepository(db)
	// 注入用户姓名解析器，使任务响应携带创建人/负责人展示名，前端无需二次查询
	svc := service.NewTaskServiceWithNames(repo, repository.NewUserDirectory(db))
	h := handler.NewTaskHandler(svc)

	// AI 辅助叠加层：通用 ai.Client 实现 ai.Provider，注入任务仓储供摘要读取
	aiSvc := service.NewTaskAIService(ai.New(aiCfg), repo)
	aiH := handler.NewTaskAIHandler(aiSvc)

	RegisterRoutes(r, h, aiH)
}

// RegisterRoutes 注册任务路由（需登录）。
// 静态段（/stats、/deleted、/batch/*、/ai/*）必须在 /{id} 之前注册，避免路径参数吞掉静态段。
func RegisterRoutes(r chi.Router, h *handler.TaskHandler, aiH *handler.TaskAIHandler) {
	r.Route("/tasks", func(r chi.Router) {
		// AI 辅助能力（附加只读/草稿端点，不写库）
		r.Get("/ai/capabilities", aiH.Capabilities)
		r.Post("/ai/parse", aiH.Parse)
		r.Post("/ai/suggest", aiH.Suggest)
		r.Post("/ai/expand", aiH.Expand)
		r.Post("/ai/summary", aiH.Summarize)

		r.Get("/stats", h.Stats)
		r.Get("/deleted", h.ListTrash)
		r.Post("/batch/complete", h.BatchComplete)
		r.Post("/batch/delete", h.BatchDelete)
		r.Post("/batch/status", h.BatchStatus)
		r.Post("/batch/assign", h.BatchAssign)
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
