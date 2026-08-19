package plan

import (
	"meteorx/internal/middleware"
	"meteorx/internal/modules/plan/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterAdminRoutes 平台管理员套餐管理接口
// 挂载 AutoRequirePermission 中间件做细粒度权限校验
func RegisterAdminRoutes(r chi.Router, h *handler.PlanHandler, checker middleware.PermissionChecker) {
	r.Route("/admin/plans", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/", h.ListPlans)                    // → admin:plan:list
		r.Get("/select", h.ListAllEnabledPlans)    // → admin:plan:list
		r.Post("/", h.CreatePlan)                  // → admin:plan:create
		r.Put("/{id}/update", h.UpdatePlan)        // → admin:plan:update
		r.Delete("/{id}/delete", h.DeletePlan)     // → admin:plan:delete
	})

	// 后台查询/分配租户套餐
	r.Route("/admin/tenants-plan", func(r chi.Router) {
		r.Use(middleware.AutoRequirePermission(checker))
		r.Get("/{id}", h.CheckTenantPlan)    // → admin:plan:list (读取)
		r.Put("/{id}", h.AssignPlan)         // → admin:plan:assign
	})
}

// RegisterPrivateRoutes 租户侧当前套餐查询接口
func RegisterPrivateRoutes(r chi.Router, h *handler.PlanHandler) {
	r.Route("/tenant/current/plan", func(r chi.Router) {
		r.Get("/", h.GetCurrentPlan) // 当前租户套餐与用量
	})
}