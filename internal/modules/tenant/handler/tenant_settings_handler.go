// Package handler 提供租户设置管理 HTTP 处理器
package handler

import (
	"net/http"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/tenant/dto"
	"meteorx/internal/modules/tenant/service"
)

// TenantSettingsHandler 租户设置处理器
type TenantSettingsHandler struct {
	svc *service.TenantSettingsService
}

// NewTenantSettingsHandler 创建租户设置处理器
func NewTenantSettingsHandler(svc *service.TenantSettingsService) *TenantSettingsHandler {
	return &TenantSettingsHandler{svc: svc}
}

// GetSettings 获取当前租户设置
// GET /api/v1/tenant-settings
func (h *TenantSettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}

	settings, err := h.svc.GetSettings(r.Context(), tenantID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取租户设置失败")
		return
	}

	response.Success(w, settings)
}

// UpdateSettings 更新当前租户设置
// PUT /api/v1/tenant-settings
func (h *TenantSettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到租户信息")
		return
	}

	var req dto.UpdateTenantSettingsReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	settings, err := h.svc.UpdateSettings(r.Context(), tenantID, req)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "更新租户设置失败")
		return
	}

	response.Success(w, settings)
}