package plan

import (
	"meteorx/internal/middleware"
	"meteorx/internal/modules/plan/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterAdminRoutes 注册平台管理员套餐管理路由
// 挂载 AutoRequirePermission 中间件做细粒度权限校验
func RegisterAdminRoutes(r chi.Router, h *handler.PlanHandler, checker middleware.PermissionChecker) {
	// 套餐 CRUD
	r.Route("/admin/plans", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/", h.ListPlans)                    // 套餐列表
		r.Get("/select", h.ListAllEnabledPlans)    // 启用套餐下拉列表
		r.Post("/", h.CreatePlan)                  // 创建套餐
		r.Put("/{id}/update", h.UpdatePlan)        // 更新套餐
		r.Delete("/{id}/delete", h.DeletePlan)     // 删除套餐
	})

	// 租户套餐查询与分配
	r.Route("/admin/tenants-plan", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/{id}", h.CheckTenantPlan)    // 查询租户套餐
		r.Put("/{id}", h.AssignPlan)         // 分配套餐
	})
}

// RegisterPrivateRoutes 租户侧当前套餐查询接口
func RegisterPrivateRoutes(r chi.Router, h *handler.PlanHandler) {
	r.Route("/tenant/current/plan", func(r chi.Router) {
		r.Get("/", h.GetCurrentPlan) // 当前租户套餐与用量
	})
}