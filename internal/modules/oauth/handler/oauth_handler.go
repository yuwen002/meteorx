package handler

import (
	"context"
	"errors"
	"net/http"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/oauth/dto"
	"meteorx/internal/modules/oauth/service"
	userdto "meteorx/internal/modules/user/dto"
	userModel "meteorx/internal/modules/user/model"
)

type OAuthService interface {
	GetRedirectURL(ctx context.Context, provider string) (string, string, error)
	Login(ctx context.Context, provider, code, state, tenantID string) (*userModel.User, []string, []string, string, string, bool, error)
	GetTenantList(ctx context.Context) ([]dto.TenantOption, error)
	ListOAuthAccounts(ctx context.Context, userID string) ([]dto.OAuthAccountResp, error)
	UnbindOAuth(ctx context.Context, userID, provider string) error
	BindOAuth(ctx context.Context, userID, provider, code, state string) error
	RefreshToken(ctx context.Context, refreshToken string) (string, string, error)
}

type OAuthHandler struct {
	svc OAuthService
}

func NewOAuthHandler(svc OAuthService) *OAuthHandler {
	return &OAuthHandler{svc: svc}
}

func (h *OAuthHandler) GetRedirectURL(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	if provider == "" {
		response.Fail(w, http.StatusBadRequest, "provider is required")
		return
	}

	redirectURL, state, err := h.svc.GetRedirectURL(r.Context(), provider)
	if err != nil {
		if errors.Is(err, service.ErrOAuthProviderDisabled) {
			response.Fail(w, http.StatusBadRequest, "该 OAuth 提供商未启用")
			return
		}
		response.Fail(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(w, dto.OAuthRedirectResponse{URL: redirectURL, State: state})
}

func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	var req dto.OAuthLoginRequest
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	user, _, permCodes, token, refreshToken, isNew, err := h.svc.Login(r.Context(), req.Provider, req.Code, req.State, req.TenantID)
	if err != nil {
		if errors.Is(err, service.ErrOAuthInvalidState) {
			response.Fail(w, http.StatusBadRequest, "CSRF state 验证失败，请重新发起授权")
			return
		}
		if errors.Is(err, service.ErrOAuthEmailRequired) {
			response.Fail(w, http.StatusBadRequest, "第三方账号未提供邮箱，无法完成登录")
			return
		}
		response.Fail(w, http.StatusUnauthorized, "OAuth 登录失败: "+err.Error())
		return
	}

	converter := userdto.UserConverter{}
	loginResp := dto.OAuthLoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         converter.ToResponse(user),
		Permissions:  permCodes,
		IsNewUser:    isNew,
	}

	response.Success(w, loginResp)
}

func (h *OAuthHandler) GetTenantList(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.svc.GetTenantList(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取租户列表失败: "+err.Error())
		return
	}

	response.Success(w, dto.TenantListResponse{Tenants: tenants})
}

func (h *OAuthHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "未授权")
		return
	}

	accounts, err := h.svc.ListOAuthAccounts(r.Context(), userID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取绑定账号失败: "+err.Error())
		return
	}

	response.Success(w, dto.OAuthAccountListResponse{Accounts: accounts})
}

func (h *OAuthHandler) Unbind(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "未授权")
		return
	}

	var req dto.UnbindOAuthRequest
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if err := h.svc.UnbindOAuth(r.Context(), userID, req.Provider); err != nil {
		if errors.Is(err, service.ErrOAuthAccountNotFound) {
			response.Fail(w, http.StatusNotFound, "未找到该绑定的第三方账号")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "解绑失败: "+err.Error())
		return
	}

	response.Success(w, nil)
}

func (h *OAuthHandler) Bind(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "未授权")
		return
	}

	var req dto.BindOAuthRequest
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if err := h.svc.BindOAuth(r.Context(), userID, req.Provider, req.Code, req.State); err != nil {
		if errors.Is(err, service.ErrOAuthInvalidState) {
			response.Fail(w, http.StatusBadRequest, "CSRF state 验证失败，请重新发起授权")
			return
		}
		if errors.Is(err, service.ErrOAuthAccountAlreadyBound) {
			response.Fail(w, http.StatusConflict, "该提供商已绑定到当前账号")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "绑定失败: "+err.Error())
		return
	}

	response.Success(w, nil)
}

func (h *OAuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshTokenRequest
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	newToken, newRefreshToken, err := h.svc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrRefreshTokenInvalid) {
			response.Fail(w, http.StatusUnauthorized, "刷新令牌无效或已过期")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "刷新令牌失败: "+err.Error())
		return
	}

	response.Success(w, dto.RefreshTokenResponse{
		Token:        newToken,
		RefreshToken: newRefreshToken,
	})
}