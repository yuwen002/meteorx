package plan

import (
	"meteorx/internal/modules/plan/handler"

	"github.com/go-chi/chi/v5"
)

// RegisterAdminRoutes 平台管理员套餐管理接口
func RegisterAdminRoutes(r chi.Router, h *handler.PlanHandler) {
	r.Route("/admin/plans", func(r chi.Router) {
		r.Get("/", h.ListPlans)                    // 套餐分页列表
		r.Get("/select", h.ListAllEnabledPlans)    // 启用套餐下拉列表
		r.Post("/", h.CreatePlan)                  // 创建套餐
		r.Put("/{id}/update", h.UpdatePlan)        // 更新套餐
		r.Delete("/{id}/delete", h.DeletePlan)     // 删除套餐
	})

	// 后台查询/分配租户套餐
	r.Route("/admin/tenants-plan", func(r chi.Router) {
		r.Get("/{id}", h.CheckTenantPlan)    // 查询租户当前套餐
		r.Put("/{id}", h.AssignPlan)         // 为租户分配/变更套餐
	})
}

// RegisterPrivateRoutes 租户侧当前套餐查询接口
func RegisterPrivateRoutes(r chi.Router, h *handler.PlanHandler) {
	r.Route("/tenant/current/plan", func(r chi.Router) {
		r.Get("/", h.GetCurrentPlan) // 当前租户套餐与用量
	})
}