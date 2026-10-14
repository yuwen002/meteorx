// Package handler 提供数据看板 HTTP 处理器
package handler

import (
	"context"
	"errors"
	"log/slog"
	"meteorx/internal/common/response"
	"meteorx/internal/dbmonitor"
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
	GetTrend(ctx context.Context, days int) (*dto.GrowthTrendResp, error)
}

// 增长趋势查询的天数默认值与上限，与仓库层上限保持一致。
const (
	trendDefaultDays = 30
	trendMaxDays     = 365
)

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

// GetTrend 获取增长趋势时间序列
// GET /api/v1/admin/dashboard/trend?days=N（默认 30，上限 365）
func (h *DashboardHandler) GetTrend(w http.ResponseWriter, r *http.Request) {
	days := trendDefaultDays
	if raw := r.URL.Query().Get("days"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 || n > trendMaxDays {
			response.BadRequest(w, "days 参数不合法，应为 1-365 的整数")
			return
		}
		days = n
	}

	trend, err := h.svc.GetTrend(r.Context(), days)
	if err != nil {
		slog.Error("获取增长趋势失败", "error", err)
		response.Fail(w, http.StatusInternalServerError, "获取增长趋势失败: "+err.Error())
		return
	}
	response.Success(w, trend)
}

// GetSlowQueries 获取慢查询看板快照（统计 + 最近慢查询列表）
// GET /api/v1/admin/dashboard/slow-queries
func (h *DashboardHandler) GetSlowQueries(w http.ResponseWriter, r *http.Request) {
	response.Success(w, dbmonitor.GetSnapshot())
}

// ResetSlowQueries 重置慢查询缓冲区（清空统计与明细）
// POST /api/v1/admin/dashboard/slow-queries/reset
func (h *DashboardHandler) ResetSlowQueries(w http.ResponseWriter, r *http.Request) {
	dbmonitor.Reset()
	response.Success(w, map[string]string{"message": "慢查询统计已重置"})
}
