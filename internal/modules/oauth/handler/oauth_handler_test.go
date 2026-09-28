package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/oauth/dto"
	"meteorx/internal/modules/oauth/handler"
	"meteorx/internal/modules/oauth/service"
	userModel "meteorx/internal/modules/user/model"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubOAuthService struct {
	redirectURL string
	state       string
	redirectErr error

	loginUser        *userModel.User
	loginPermCodes   []string
	loginToken       string
	loginRefreshToken string
	loginIsNew       bool
	loginErr         error

	tenantList    []dto.TenantOption
	tenantListErr error

	accounts    []dto.OAuthAccountResp
	accountsErr error

	unbindErr error

	bindErr error

	refreshToken        string
	refreshRefreshToken string
	refreshErr          error
}

func (s *stubOAuthService) GetRedirectURL(_ context.Context, provider string) (string, string, error) {
	return s.redirectURL, s.state, s.redirectErr
}

func (s *stubOAuthService) Login(_ context.Context, provider, code, state, tenantID string) (*userModel.User, []string, []string, string, string, bool, error) {
	return s.loginUser, nil, s.loginPermCodes, s.loginToken, s.loginRefreshToken, s.loginIsNew, s.loginErr
}

func (s *stubOAuthService) GetTenantList(_ context.Context) ([]dto.TenantOption, error) {
	return s.tenantList, s.tenantListErr
}

func (s *stubOAuthService) ListOAuthAccounts(_ context.Context, userID string) ([]dto.OAuthAccountResp, error) {
	return s.accounts, s.accountsErr
}

func (s *stubOAuthService) UnbindOAuth(_ context.Context, userID, provider string) error {
	return s.unbindErr
}

func (s *stubOAuthService) BindOAuth(_ context.Context, userID, provider, code, state string) error {
	return s.bindErr
}

func (s *stubOAuthService) RefreshToken(_ context.Context, refreshToken string) (string, string, error) {
	return s.refreshToken, s.refreshRefreshToken, s.refreshErr
}

func newOAuthRouter(stub *stubOAuthService) http.Handler {
	h := handler.NewOAuthHandler(stub)
	r := chi.NewRouter()
	r.Get("/{provider}/redirect", h.GetRedirectURL)
	r.Post("/callback", h.Callback)
	r.Get("/tenants", h.GetTenantList)
	r.Post("/token/refresh", h.RefreshToken)
	r.Get("/accounts", h.ListAccounts)
	r.Post("/unbind", h.Unbind)
	r.Post("/bind", h.Bind)
	return r
}

func doOAuth(t *testing.T, router http.Handler, method, path, body string, userID string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	}
	req.Header.Set("Content-Type", "application/json")
	ctx := contextx.SetVars(req.Context(), "t-1", userID, []string{"user"})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func sampleOAuthUser() *userModel.User {
	return &userModel.User{ID: "u-1", Username: "alice", Nickname: "Alice", Email: "alice@example.com", Status: 1}
}

func TestGetRedirectURL_Success(t *testing.T) {
	stub := &stubOAuthService{
		redirectURL: "https://accounts.google.com/o/oauth2/v2/auth?client_id=xxx",
		state:       "random-state-123",
	}
	router := newOAuthRouter(stub)

	w := doOAuth(t, router, http.MethodGet, "/google/redirect", "", "")
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	data := resp["data"].(map[string]interface{})
	assert.Contains(t, data["url"], "accounts.google.com")
	assert.Equal(t, "random-state-123", data["state"])
}

func TestGetRedirectURL_ProviderDisabled(t *testing.T) {
	stub := &stubOAuthService{redirectErr: service.ErrOAuthProviderDisabled}
	router := newOAuthRouter(stub)

	w := doOAuth(t, router, http.MethodGet, "/google/redirect", "", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "未启用")
}

func TestCallback_Success(t *testing.T) {
	stub := &stubOAuthService{
		loginUser:        sampleOAuthUser(),
		loginPermCodes:   []string{"wiki:read"},
		loginToken:       "jwt-token",
		loginRefreshToken: "refresh-token",
		loginIsNew:       false,
	}
	router := newOAuthRouter(stub)

	body := `{"provider":"google","code":"auth-code","state":"csrf-state","tenant_id":"t-1"}`
	w := doOAuth(t, router, http.MethodPost, "/callback", body, "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"token":"jwt-token"`)
	assert.Contains(t, w.Body.String(), `"refresh_token":"refresh-token"`)
	assert.Contains(t, w.Body.String(), `"wiki:read"`)
}

func TestCallback_InvalidState(t *testing.T) {
	stub := &stubOAuthService{loginErr: service.ErrOAuthInvalidState}
	router := newOAuthRouter(stub)

	body := `{"provider":"google","code":"auth-code","state":"bad-state","tenant_id":"t-1"}`
	w := doOAuth(t, router, http.MethodPost, "/callback", body, "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "CSRF state")
}

func TestCallback_EmailRequired(t *testing.T) {
	stub := &stubOAuthService{loginErr: service.ErrOAuthEmailRequired}
	router := newOAuthRouter(stub)

	body := `{"provider":"github","code":"auth-code","state":"csrf-state","tenant_id":"t-1"}`
	w := doOAuth(t, router, http.MethodPost, "/callback", body, "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "邮箱")
}

func TestCallback_ValidationFailure(t *testing.T) {
	stub := &stubOAuthService{}
	router := newOAuthRouter(stub)

	w := doOAuth(t, router, http.MethodPost, "/callback", `{"provider":"google"}`, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCallback_LoginError(t *testing.T) {
	stub := &stubOAuthService{loginErr: errors.New("exchange failed")}
	router := newOAuthRouter(stub)

	body := `{"provider":"google","code":"bad","state":"s","tenant_id":"t"}`
	w := doOAuth(t, router, http.MethodPost, "/callback", body, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetTenantList_Success(t *testing.T) {
	stub := &stubOAuthService{
		tenantList: []dto.TenantOption{
			{ID: "t-1", Name: "Tenant A"},
			{ID: "t-2", Name: "Tenant B"},
		},
	}
	router := newOAuthRouter(stub)

	w := doOAuth(t, router, http.MethodGet, "/tenants", "", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Tenant A")
	assert.Contains(t, w.Body.String(), "Tenant B")
}

func TestGetTenantList_Error(t *testing.T) {
	stub := &stubOAuthService{tenantListErr: errors.New("db error")}
	router := newOAuthRouter(stub)

	w := doOAuth(t, router, http.MethodGet, "/tenants", "", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListAccounts_Success(t *testing.T) {
	stub := &stubOAuthService{
		accounts: []dto.OAuthAccountResp{
			{ID: "oa-1", Provider: "google", Email: "alice@gmail.com", CreatedAt: time.Now()},
		},
	}
	router := newOAuthRouter(stub)

	w := doOAuth(t, router, http.MethodGet, "/accounts", "", "u-1")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "google")
	assert.Contains(t, w.Body.String(), "alice@gmail.com")
}

func TestListAccounts_Unauthorized(t *testing.T) {
	stub := &stubOAuthService{}
	router := newOAuthRouter(stub)

	w := doOAuth(t, router, http.MethodGet, "/accounts", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUnbind_Success(t *testing.T) {
	stub := &stubOAuthService{}
	router := newOAuthRouter(stub)

	body := `{"provider":"google"}`
	w := doOAuth(t, router, http.MethodPost, "/unbind", body, "u-1")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUnbind_NotFound(t *testing.T) {
	stub := &stubOAuthService{unbindErr: service.ErrOAuthAccountNotFound}
	router := newOAuthRouter(stub)

	body := `{"provider":"google"}`
	w := doOAuth(t, router, http.MethodPost, "/unbind", body, "u-1")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUnbind_Unauthorized(t *testing.T) {
	stub := &stubOAuthService{}
	router := newOAuthRouter(stub)

	body := `{"provider":"google"}`
	w := doOAuth(t, router, http.MethodPost, "/unbind", body, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBind_Success(t *testing.T) {
	stub := &stubOAuthService{}
	router := newOAuthRouter(stub)

	body := `{"provider":"github","code":"auth-code","state":"csrf-state"}`
	w := doOAuth(t, router, http.MethodPost, "/bind", body, "u-1")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBind_InvalidState(t *testing.T) {
	stub := &stubOAuthService{bindErr: service.ErrOAuthInvalidState}
	router := newOAuthRouter(stub)

	body := `{"provider":"github","code":"auth-code","state":"bad"}`
	w := doOAuth(t, router, http.MethodPost, "/bind", body, "u-1")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "CSRF state")
}

func TestBind_AlreadyBound(t *testing.T) {
	stub := &stubOAuthService{bindErr: service.ErrOAuthAccountAlreadyBound}
	router := newOAuthRouter(stub)

	body := `{"provider":"github","code":"auth-code","state":"s"}`
	w := doOAuth(t, router, http.MethodPost, "/bind", body, "u-1")
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRefreshToken_Success(t *testing.T) {
	stub := &stubOAuthService{
		refreshToken:        "new-jwt-token",
		refreshRefreshToken: "new-refresh-token",
	}
	router := newOAuthRouter(stub)

	body := `{"refresh_token":"old-refresh-token"}`
	w := doOAuth(t, router, http.MethodPost, "/token/refresh", body, "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "new-jwt-token")
	assert.Contains(t, w.Body.String(), "new-refresh-token")
}

func TestRefreshToken_Invalid(t *testing.T) {
	stub := &stubOAuthService{refreshErr: service.ErrRefreshTokenInvalid}
	router := newOAuthRouter(stub)

	body := `{"refresh_token":"bad-token"}`
	w := doOAuth(t, router, http.MethodPost, "/token/refresh", body, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "刷新令牌")
}

func TestRefreshToken_ValidationFailure(t *testing.T) {
	stub := &stubOAuthService{}
	router := newOAuthRouter(stub)

	w := doOAuth(t, router, http.MethodPost, "/token/refresh", `{}`, "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}