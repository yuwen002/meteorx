// Package handler 实现 OAuth 模块的 HTTP 接口处理器，处理授权回调、账号绑定和解绑请求。
package handler

import (
	"context"
	"errors"
	"net/http"

	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/oauth/dto"
	"meteorx/internal/modules/oauth/service"
)

// OAuthProviderAdminService 平台级提供商配置管理服务接口（handler 依赖的最小业务面）
type OAuthProviderAdminService interface {
	ListOAuthProviderConfigs(ctx context.Context) ([]dto.OAuthProviderResp, error)
	GetOAuthProviderConfig(ctx context.Context, code string) (*dto.OAuthProviderResp, error)
	CreateOAuthProviderConfig(ctx context.Context, req dto.SaveOAuthProviderReq) error
	UpdateOAuthProviderConfig(ctx context.Context, code string, req dto.SaveOAuthProviderReq) error
	DeleteOAuthProviderConfig(ctx context.Context, code string) error
	SetOAuthProviderConfigStatus(ctx context.Context, code string, enabled bool) error
}

// OAuthProviderAdminHandler 第三方登录提供商配置管理处理器（平台超级管理员）
type OAuthProviderAdminHandler struct {
	svc OAuthProviderAdminService
}

// NewOAuthProviderAdminHandler 创建提供商配置管理处理器
func NewOAuthProviderAdminHandler(svc OAuthProviderAdminService) *OAuthProviderAdminHandler {
	return &OAuthProviderAdminHandler{svc: svc}
}

// ListProviders 获取全部提供商配置
// GET /api/v1/admin/oauth-providers
func (h *OAuthProviderAdminHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.svc.ListOAuthProviderConfigs(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取第三方登录配置失败: "+err.Error())
		return
	}
	response.Success(w, dto.OAuthProviderListResponse{Providers: providers})
}

// GetProvider 获取单个提供商配置
// GET /api/v1/admin/oauth-providers/{code}
func (h *OAuthProviderAdminHandler) GetProvider(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		response.Fail(w, http.StatusBadRequest, "provider 标识不能为空")
		return
	}
	provider, err := h.svc.GetOAuthProviderConfig(r.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrOAuthProviderNotFound) {
			response.Fail(w, http.StatusNotFound, "第三方登录配置不存在")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "获取第三方登录配置失败: "+err.Error())
		return
	}
	response.Success(w, provider)
}

// CreateProvider 新建提供商配置
// POST /api/v1/admin/oauth-providers
func (h *OAuthProviderAdminHandler) CreateProvider(w http.ResponseWriter, r *http.Request) {
	var req dto.SaveOAuthProviderReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	if err := h.svc.CreateOAuthProviderConfig(r.Context(), req); err != nil {
		h.failWrite(w, err, "创建第三方登录配置失败")
		return
	}
	response.Success(w, nil)
}

// UpdateProvider 更新提供商配置
// PUT /api/v1/admin/oauth-providers/{code}
func (h *OAuthProviderAdminHandler) UpdateProvider(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		response.Fail(w, http.StatusBadRequest, "provider 标识不能为空")
		return
	}
	var req dto.SaveOAuthProviderReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	if err := h.svc.UpdateOAuthProviderConfig(r.Context(), code, req); err != nil {
		h.failWrite(w, err, "更新第三方登录配置失败")
		return
	}
	response.Success(w, nil)
}

// DeleteProvider 删除提供商配置
// DELETE /api/v1/admin/oauth-providers/{code}
func (h *OAuthProviderAdminHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		response.Fail(w, http.StatusBadRequest, "provider 标识不能为空")
		return
	}
	if err := h.svc.DeleteOAuthProviderConfig(r.Context(), code); err != nil {
		if errors.Is(err, service.ErrOAuthProviderNotFound) {
			response.Fail(w, http.StatusNotFound, "第三方登录配置不存在")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "删除第三方登录配置失败: "+err.Error())
		return
	}
	response.Success(w, nil)
}

// SetProviderStatus 切换提供商启用状态
// PUT /api/v1/admin/oauth-providers/{code}/status
func (h *OAuthProviderAdminHandler) SetProviderStatus(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		response.Fail(w, http.StatusBadRequest, "provider 标识不能为空")
		return
	}
	var req dto.SetOAuthProviderStatusReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	enabled := req.Enabled != nil && *req.Enabled
	if err := h.svc.SetOAuthProviderConfigStatus(r.Context(), code, enabled); err != nil {
		if errors.Is(err, service.ErrOAuthProviderNotFound) {
			response.Fail(w, http.StatusNotFound, "第三方登录配置不存在")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "更新启用状态失败: "+err.Error())
		return
	}
	response.Success(w, nil)
}

// failWrite 统一映射写操作错误：参数非法 400、标识冲突 409、缺主密钥 422，其余 500。
func (h *OAuthProviderAdminHandler) failWrite(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrOAuthProviderInvalid):
		response.Fail(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrOAuthProviderCodeTaken):
		response.Fail(w, http.StatusConflict, "该 provider 标识已存在")
	case errors.Is(err, service.ErrOAuthSecretKeyNotConfigured):
		response.Fail(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, service.ErrOAuthProviderNotFound):
		response.Fail(w, http.StatusNotFound, "第三方登录配置不存在")
	default:
		response.Fail(w, http.StatusInternalServerError, fallback+": "+err.Error())
	}
}
