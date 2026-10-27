// Package handler 实现 OAuth 模块的 HTTP 接口处理器，处理授权回调、账号绑定和解绑请求。
package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/middleware"
	auditmodel "meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/oauth/dto"
	"meteorx/internal/modules/oauth/service"
	userdto "meteorx/internal/modules/user/dto"
	userModel "meteorx/internal/modules/user/model"
)

// OAuthService OAuth 服务接口（handler 依赖的最小业务面，便于测试注入桩）
type OAuthService interface {
	GetRedirectURL(ctx context.Context, provider string) (string, string, error)
	Login(ctx context.Context, provider, code, state, tenantID string) (*userModel.User, []string, []string, string, string, bool, error)
	GetTenantList(ctx context.Context) ([]dto.TenantOption, error)
	ListProviders(ctx context.Context) []dto.ProviderMeta
	ListOAuthAccounts(ctx context.Context, userID string) ([]dto.OAuthAccountResp, error)
	UnbindOAuth(ctx context.Context, userID, provider string) error
	BindOAuth(ctx context.Context, userID, provider, code, state string) error
	RefreshToken(ctx context.Context, refreshToken string) (string, string, error)
	CompleteEmail(ctx context.Context, bearerToken, email string) (string, string, error)
}

// OAuthHandler OAuth 处理器
type OAuthHandler struct {
	svc OAuthService
}

// NewOAuthHandler 创建 OAuth 处理器
func NewOAuthHandler(svc OAuthService) *OAuthHandler {
	return &OAuthHandler{svc: svc}
}

// GetRedirectURL 获取第三方 OAuth2 授权跳转地址
// GET /api/v1/auth/oauth/{provider}/redirect
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

// Callback 处理第三方 OAuth2 授权回调，完成登录或注册
// POST /api/v1/auth/oauth/callback
func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	var req dto.OAuthLoginRequest
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	start := time.Now()
	user, _, permCodes, token, refreshToken, isNew, err := h.svc.Login(r.Context(), req.Provider, req.Code, req.State, req.TenantID)

	// OAuth 回调等价于一次登录/注册，属公开入口，成功失败都入审；
	// code/state 为一次性凭据，不写入审计。
	auditReq := middleware.NewAuditLogReq(r)
	auditReq.Module = "auth"
	auditReq.Action = auditmodel.ActionTypeLogin
	auditReq.Resource = r.URL.Path
	auditReq.Tags = fmt.Sprintf(`["oauth","login",%q]`, req.Provider)
	auditReq.Duration = time.Since(start).Milliseconds()

	if err != nil {
		auditReq.StatusCode = http.StatusUnauthorized
		if errors.Is(err, service.ErrOAuthInvalidState) || errors.Is(err, service.ErrOAuthProviderDisabled) {
			auditReq.StatusCode = http.StatusBadRequest
		}
		auditReq.Result = auditmodel.ResultFailure
		auditReq.ErrorMessage = err.Error()
		auditReq.RiskLevel = auditmodel.RiskHigh
		middleware.RecordAuditLog(r.Context(), &auditReq)

		// 按哨兵错误映射固定文案：服务层错误可能携带上游细节，不得原样回显给客户端
		switch {
		case errors.Is(err, service.ErrOAuthInvalidState):
			response.Fail(w, http.StatusBadRequest, "CSRF state 验证失败，请重新发起授权")
		case errors.Is(err, service.ErrOAuthProviderDisabled):
			response.Fail(w, http.StatusBadRequest, "该 OAuth 提供商未启用")
		case errors.Is(err, service.ErrOAuthExchangeFailed):
			response.Fail(w, http.StatusUnauthorized, "授权码交换失败，请重新发起授权")
		case errors.Is(err, service.ErrOAuthUserInfoFailed):
			response.Fail(w, http.StatusUnauthorized, "获取第三方用户信息失败，请稍后重试")
		default:
			response.Fail(w, http.StatusUnauthorized, "OAuth 登录失败，请稍后重试或联系管理员")
		}
		return
	}

	auditReq.UserID = user.ID
	auditReq.Username = user.Username
	auditReq.TenantID = user.TenantID
	auditReq.ResourceID = user.ID
	auditReq.StatusCode = http.StatusOK
	auditReq.Result = auditmodel.ResultSuccess
	auditReq.RiskLevel = auditmodel.RiskMedium
	if isNew {
		// 首次 OAuth 登录即隐式建号，归类为创建以提升风险标注
		auditReq.Action = auditmodel.ActionTypeCreate
		auditReq.Tags = fmt.Sprintf(`["oauth","login","register",%q]`, req.Provider)
	}
	middleware.RecordAuditLog(r.Context(), &auditReq)

	converter := userdto.UserConverter{}
	loginResp := dto.OAuthLoginResponse{
		Token:             token,
		RefreshToken:      refreshToken,
		User:              converter.ToResponse(user),
		Permissions:       permCodes,
		IsNewUser:         isNew,
		NeedCompleteEmail: user.Email == "",
	}

	response.Success(w, loginResp)
}

// ListProviders 获取已启用的第三方登录提供商列表
// GET /api/v1/auth/oauth/providers
func (h *OAuthHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	providers := h.svc.ListProviders(r.Context())
	response.Success(w, map[string][]dto.ProviderMeta{"providers": providers})
}

// CompleteEmail 第三方登录后补全邮箱（需携带登录签发的临时令牌）
// POST /api/v1/auth/oauth/complete-email
func (h *OAuthHandler) CompleteEmail(w http.ResponseWriter, r *http.Request) {
	var req dto.CompleteEmailRequest
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	bearer := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(bearer) > len(prefix) && strings.HasPrefix(bearer, prefix) {
		bearer = bearer[len(prefix):]
	}
	if bearer == "" {
		response.Fail(w, http.StatusUnauthorized, "缺少认证令牌")
		return
	}

	newToken, newRefresh, err := h.svc.CompleteEmail(r.Context(), bearer, req.Email)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(w, dto.RefreshTokenResponse{
		Token:        newToken,
		RefreshToken: newRefresh,
	})
}

// GetTenantList 获取 OAuth 登录可用的租户列表
// GET /api/v1/auth/oauth/tenants
func (h *OAuthHandler) GetTenantList(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.svc.GetTenantList(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取租户列表失败: "+err.Error())
		return
	}

	response.Success(w, dto.TenantListResponse{Tenants: tenants})
}

// ListAccounts 查看当前用户已绑定的第三方账号列表
// GET /api/v1/auth/oauth/accounts
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

// Unbind 解绑指定的第三方账号
// POST /api/v1/auth/oauth/unbind
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

// Bind 绑定新的第三方账号（已登录用户绑定新 provider）
// POST /api/v1/auth/oauth/bind
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
		// 同样按哨兵错误映射固定文案，避免上游细节回显
		switch {
		case errors.Is(err, service.ErrOAuthInvalidState):
			response.Fail(w, http.StatusBadRequest, "CSRF state 验证失败，请重新发起授权")
		case errors.Is(err, service.ErrOAuthAccountAlreadyBound):
			response.Fail(w, http.StatusConflict, "该提供商已绑定到当前账号")
		case errors.Is(err, service.ErrOAuthProviderDisabled):
			response.Fail(w, http.StatusBadRequest, "该 OAuth 提供商未启用")
		case errors.Is(err, service.ErrOAuthExchangeFailed):
			response.Fail(w, http.StatusBadRequest, "授权码交换失败，请重新发起授权")
		case errors.Is(err, service.ErrOAuthUserInfoFailed):
			response.Fail(w, http.StatusBadRequest, "获取第三方用户信息失败，请稍后重试")
		default:
			response.Fail(w, http.StatusInternalServerError, "绑定失败，请稍后重试")
		}
		return
	}

	response.Success(w, nil)
}

// RefreshToken 使用刷新令牌获取新的 access token
// POST /api/v1/auth/oauth/token/refresh
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
