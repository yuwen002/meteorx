// Package handler 提供数据看板 HTTP 处理器
package handler

import (
	"context"
	"log/slog"
	"meteorx/internal/common/response"
	"meteorx/internal/modules/dashboard/dto"
	"meteorx/internal/modules/dashboard/model"
	"net/http"
	"strconv"
)

// DashboardService 看板服务接口（handler 依赖的最小业务面，便于测试注入桩）
type DashboardService interface {
	GetOverview(ctx context.Context) (*dto.DashboardOverviewResp, error)
	GetTenantHealth(ctx context.Context, query model.HealthQuery) (*dto.TenantHealthListResp, error)
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
