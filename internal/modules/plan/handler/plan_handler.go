// Package handler 提供套餐管理 HTTP 处理器
package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/plan/dto"
	"meteorx/internal/modules/plan/service"
	"meteorx/pkg/pagination"
)

// PlanService 套餐服务接口（handler 依赖的最小业务面；*service.PlanService 完整实现）
type PlanService interface {
	CreatePlan(ctx context.Context, req dto.CreatePlanReq) (*dto.PlanResp, error)
	UpdatePlan(ctx context.Context, id string, req dto.UpdatePlanReq) (*dto.PlanResp, error)
	ListPlans(ctx context.Context, page, pageSize int, keyword string, status *int) ([]*dto.PlanResp, int64, error)
	ListEnabledPlans(ctx context.Context) ([]*dto.PlanResp, error)
	DeletePlan(ctx context.Context, id string) error
	AssignPlan(ctx context.Context, tenantID string, req dto.AssignPlanReq) error
	GetCurrentPlan(ctx context.Context, tenantID string) (*dto.CurrentPlanResp, error)
}

// PlanHandler 套餐管理处理器
type PlanHandler struct {
	svc PlanService
}

// NewPlanHandler 创建套餐管理处理器
func NewPlanHandler(svc PlanService) *PlanHandler {
	return &PlanHandler{svc: svc}
}

// CreatePlan 后台创建套餐
// POST /api/v1/admin/plans
func (h *PlanHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePlanReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	plan, err := h.svc.CreatePlan(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPlanCodeConflict):
			response.Fail(w, http.StatusConflict, "套餐编码已存在")
		case errors.Is(err, service.ErrUserLimitExceeded):
			response.Fail(w, http.StatusBadRequest, err.Error())
		default:
			response.Fail(w, http.StatusInternalServerError, "创建套餐失败")
		}
		return
	}
	response.Success(w, plan)
}

// UpdatePlan 后台更新套餐
// PUT /api/v1/admin/plans/{id}/update
func (h *PlanHandler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "套餐ID不能为空")
		return
	}

	var req dto.UpdatePlanReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	plan, err := h.svc.UpdatePlan(r.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPlanNotFound):
			response.Fail(w, http.StatusNotFound, "套餐不存在")
		default:
			response.Fail(w, http.StatusInternalServerError, "更新套餐失败")
		}
		return
	}
	response.Success(w, plan)
}

// ListPlans 后台获取套餐列表（分页+搜索）
// GET /api/v1/admin/plans
func (h *PlanHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	keyword := r.URL.Query().Get("keyword")
	statusStr := r.URL.Query().Get("status")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	var status *int
	if statusStr != "" {
		s, err := strconv.Atoi(statusStr)
		if err == nil && (s == 0 || s == 1) {
			status = &s
		}
	}

	plans, total, err := h.svc.ListPlans(r.Context(), pg.Page, pg.PageSize, keyword, status)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "查询套餐列表失败")
		return
	}

	result := pagination.NewPaginatedResult(plans, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// ListAllEnabledPlans 获取启用中的套餐下拉列表
// GET /api/v1/admin/plans/select
func (h *PlanHandler) ListAllEnabledPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.svc.ListEnabledPlans(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "查询套餐列表失败")
		return
	}
	response.Success(w, plans)
}

// DeletePlan 后台删除套餐
// DELETE /api/v1/admin/plans/{id}/delete
func (h *PlanHandler) DeletePlan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "套餐ID不能为空")
		return
	}

	err := h.svc.DeletePlan(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPlanNotFound):
			response.Fail(w, http.StatusNotFound, "套餐不存在")
		case errors.Is(err, service.ErrPlanInUse):
			response.Fail(w, http.StatusConflict, "套餐正在被租户使用，无法删除")
		default:
			response.Fail(w, http.StatusInternalServerError, "删除套餐失败")
		}
		return
	}
	response.Success(w, nil)
}

// AssignPlan 后台为租户分配套餐
// PUT /api/v1/admin/tenants/{id}/plan
func (h *PlanHandler) AssignPlan(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "id")
	if tenantID == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID不能为空")
		return
	}

	var req dto.AssignPlanReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	err := h.svc.AssignPlan(r.Context(), tenantID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPlanNotFound):
			response.Fail(w, http.StatusNotFound, "套餐不存在")
		case errors.Is(err, service.ErrExpiredAtInPast):
			response.Fail(w, http.StatusBadRequest, "到期时间不能早于当前时间")
		default:
			response.Fail(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	response.Success(w, nil)
}

// GetCurrentPlan 获取当前租户套餐与用量
// GET /api/v1/tenant/current/plan
func (h *PlanHandler) GetCurrentPlan(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}

	plan, err := h.svc.GetCurrentPlan(r.Context(), tenantID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSubscriptionEmpty):
			response.Fail(w, http.StatusNotFound, "当前租户尚未开通套餐")
		default:
			response.Fail(w, http.StatusInternalServerError, "查询套餐信息失败")
		}
		return
	}
	response.Success(w, plan)
}

// CheckTenantPlan 后台查询指定租户当前套餐（含到期状态）
// GET /api/v1/admin/tenants/{id}/plan
func (h *PlanHandler) CheckTenantPlan(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "id")
	if tenantID == "" {
		response.Fail(w, http.StatusBadRequest, "租户ID不能为空")
		return
	}

	plan, err := h.svc.GetCurrentPlan(r.Context(), tenantID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSubscriptionEmpty):
			response.Success(w, nil)
		default:
			response.Fail(w, http.StatusInternalServerError, "查询套餐信息失败")
		}
		return
	}
	response.Success(w, plan)
}
