// Package handler 提供数据看板 HTTP 处理器
package handler

import (
	"context"
	"log/slog"
	"meteorx/internal/common/response"
	"meteorx/internal/modules/dashboard/dto"
	"net/http"
)

// DashboardService 看板服务接口（handler 依赖的最小业务面，便于测试注入桩）
type DashboardService interface {
	GetOverview(ctx context.Context) (*dto.DashboardOverviewResp, error)
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
