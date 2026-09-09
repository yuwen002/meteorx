package handler

import (
	"context"
	"errors"
	"net/http"

	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/oauth/dto"
	"meteorx/internal/modules/oauth/service"
	userdto "meteorx/internal/modules/user/dto"
	userModel "meteorx/internal/modules/user/model"
)

// OAuthService OAuth2 服务接口
type OAuthService interface {
	GetRedirectURL(provider string) (string, error)
	Login(ctx context.Context, provider, code string) (*userModel.User, []string, []string, string, bool, error)
}

// OAuthHandler OAuth2 处理器
type OAuthHandler struct {
	svc OAuthService
}

// NewOAuthHandler 创建 OAuth2 处理器
func NewOAuthHandler(svc OAuthService) *OAuthHandler {
	return &OAuthHandler{svc: svc}
}

// GetRedirectURL 获取 OAuth2 授权跳转链接
// GET /api/v1/auth/oauth/{provider}/redirect
func (h *OAuthHandler) GetRedirectURL(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	if provider == "" {
		response.Fail(w, http.StatusBadRequest, "provider is required")
		return
	}

	redirectURL, err := h.svc.GetRedirectURL(provider)
	if err != nil {
		if errors.Is(err, service.ErrOAuthProviderDisabled) {
			response.Fail(w, http.StatusBadRequest, "该 OAuth 提供商未启用")
			return
		}
		response.Fail(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(w, dto.OAuthRedirectResponse{URL: redirectURL})
}

// Callback OAuth2 登录回调
// POST /api/v1/auth/oauth/callback
func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	var req dto.OAuthLoginRequest
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	user, _, permCodes, token, isNew, err := h.svc.Login(r.Context(), req.Provider, req.Code)
	if err != nil {
		response.Fail(w, http.StatusUnauthorized, "OAuth 登录失败: "+err.Error())
		return
	}

	converter := userdto.UserConverter{}
	loginResp := dto.OAuthLoginResponse{
		Token:       token,
		User:        converter.ToResponse(user),
		Permissions: permCodes,
		IsNewUser:   isNew,
	}

	response.Success(w, loginResp)
}