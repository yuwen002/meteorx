// Package handler 平台运行时配置 HTTP 处理器（管理侧 + 公开侧）。
package handler

import (
	"context"
	"errors"
	"net/http"

	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/settings/dto"
	"meteorx/internal/modules/settings/service"
)

// SettingService 配置服务接口（handler 依赖的最小业务面；*service.SettingService 完整实现）
type SettingService interface {
	ListAll(ctx context.Context) ([]*dto.SettingView, error)
	GetPublicMap(ctx context.Context) (map[string]interface{}, error)
	UpdateBulk(ctx context.Context, items []dto.SettingUpdateItem) ([]*dto.SettingView, error)
}

// SettingHandler 平台运行时配置处理器
type SettingHandler struct {
	svc SettingService
}

// NewSettingHandler 创建配置处理器
func NewSettingHandler(svc SettingService) *SettingHandler {
	return &SettingHandler{svc: svc}
}

// ListSettings 管理视角列出全部配置（含元数据、默认值与来源）
// GET /api/v1/admin/settings → 权限码 admin:settings:list
func (h *SettingHandler) ListSettings(w http.ResponseWriter, r *http.Request) {
	views, err := h.svc.ListAll(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "查询系统配置失败")
		return
	}
	response.Success(w, views)
}

// UpdateSettings 批量更新系统配置
// PUT /api/v1/admin/settings → 权限码 admin:settings:update
func (h *SettingHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req dto.UpdateSettingsReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	views, err := h.svc.UpdateBulk(r.Context(), req.Settings)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUnknownSettingKey):
			response.Fail(w, http.StatusBadRequest, "存在未登记的配置键: "+err.Error())
		case errors.Is(err, service.ErrInvalidSettingValue):
			response.Fail(w, http.StatusBadRequest, "存在不匹配类型的配置值: "+err.Error())
		default:
			response.Fail(w, http.StatusInternalServerError, "更新系统配置失败")
		}
		return
	}
	response.Success(w, views)
}

// GetPublicSettings 公开配置（免鉴权）：仅返回 IsPublic 键，登录/注册页消费
// GET /api/v1/settings/public
func (h *SettingHandler) GetPublicSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.svc.GetPublicMap(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取公开配置失败")
		return
	}
	response.Success(w, settings)
}
