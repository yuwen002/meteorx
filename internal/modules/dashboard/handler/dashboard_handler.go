// Package handler 提供数据看板 HTTP 处理器
package handler

import (
	"meteorx/internal/common/response"
	"meteorx/internal/modules/dashboard/service"
	"net/http"
)

// DashboardHandler 数据看板处理器
type DashboardHandler struct {
	svc *service.DashboardService
}

// NewDashboardHandler 创建数据看板处理器
func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

// GetOverview 获取运营数据总览
// GET /api/v1/admin/dashboard/overview
func (h *DashboardHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.svc.GetOverview(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取运营数据总览失败")
		return
	}

	response.Success(w, overview)
}
