// Package handler 提供数据看板 HTTP 处理器
package handler

import (
	"context"
	"errors"
	"log/slog"
	"meteorx/internal/common/response"
	"meteorx/internal/modules/dashboard/dto"
	"meteorx/internal/modules/dashboard/model"
	"meteorx/internal/modules/dashboard/repository"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// DashboardService 看板服务接口（handler 依赖的最小业务面，便于测试注入桩）
type DashboardService interface {
	GetOverview(ctx context.Context) (*dto.DashboardOverviewResp, error)
	GetTenantHealth(ctx context.Context, query model.HealthQuery) (*dto.TenantHealthListResp, error)
	GetTenantHealthDetail(ctx context.Context, tenantID string) (*dto.TenantHealthItemResp, error)
}

// DashboardHandler 数据看板处理器
type DashboardHandler struct {
	svc DashboardService
}

// NewDashboardHandler 创建数据看板处理器
func NewDashboardHandler(svc DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// GetOverview 获取运营数据总览
// GET /api/v1/admin/dashboard/overview
func (h *DashboardHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.svc.GetOverview(r.Context())
	if err != nil {
		slog.Error("获取运营数据总览失败", "error", err)
		response.Fail(w, http.StatusInternalServerError, "获取运营数据总览失败: "+err.Error())
		return
	}

	response.Success(w, overview)
}

// GetTenantHealth 获取租户健康度列表
// GET /api/v1/admin/dashboard/tenants-health
func (h *DashboardHandler) GetTenantHealth(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	query := model.HealthQuery{
		Page:     page,
		PageSize: pageSize,
		Keyword:  q.Get("keyword"),
	}

	list, err := h.svc.GetTenantHealth(r.Context(), query)
	if err != nil {
		slog.Error("获取租户健康度失败", "error", err)
		response.Fail(w, http.StatusInternalServerError, "获取租户健康度失败: "+err.Error())
		return
	}

	response.Success(w, list)
}

// GetTenantHealthDetail 获取单个租户健康度详情
// GET /api/v1/admin/dashboard/tenants-health/{id}
func (h *DashboardHandler) GetTenantHealthDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "缺少租户 ID")
		return
	}

	item, err := h.svc.GetTenantHealthDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrTenantNotFound) {
			response.NotFound(w, "租户不存在")
			return
		}
		slog.Error("获取租户健康度详情失败", "tenant_id", id, "error", err)
		response.Fail(w, http.StatusInternalServerError, "获取租户健康度详情失败: "+err.Error())
		return
	}

	response.Success(w, item)
}
