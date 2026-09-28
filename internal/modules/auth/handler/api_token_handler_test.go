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
	"meteorx/internal/modules/auth/dto"
	"meteorx/internal/modules/auth/handler"
	"meteorx/internal/modules/auth/service"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAPITokenService struct {
	createResult *dto.CreateAPITokenResp
	createErr    error

	tokens    []*dto.APITokenResp
	tokensErr error

	revokeErr error
}

func (s *stubAPITokenService) Create(_ context.Context, userID, tenantID string, req dto.CreateAPITokenReq) (*dto.CreateAPITokenResp, error) {
	return s.createResult, s.createErr
}

func (s *stubAPITokenService) ListByUserID(_ context.Context, userID string) ([]*dto.APITokenResp, error) {
	return s.tokens, s.tokensErr
}

func (s *stubAPITokenService) Revoke(_ context.Context, userID, tokenID string) error {
	return s.revokeErr
}

func newAPITokenRouter(stub *stubAPITokenService) http.Handler {
	h := handler.NewAPITokenHandler(stub)
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Post("/revoke", h.Revoke)
	return r
}

func doAPIToken(t *testing.T, router http.Handler, method, path, body, userID string) *httptest.ResponseRecorder {
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

func doAPITokenNoUser(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestCreate_NoUser_Returns401(t *testing.T) {
	stub := &stubAPITokenService{}
	router := newAPITokenRouter(stub)

	w := doAPITokenNoUser(t, router, http.MethodPost, "/", `{"name":"test"}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "未授权")
}

func TestCreate_ValidationFailure_Returns400(t *testing.T) {
	stub := &stubAPITokenService{}
	router := newAPITokenRouter(stub)

	w := doAPIToken(t, router, http.MethodPost, "/", `{"name":""}`, "u-1")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "名称")
}

func TestCreate_Success_ReturnsToken(t *testing.T) {
	now := time.Now()
	stub := &stubAPITokenService{
		createResult: &dto.CreateAPITokenResp{
			ID:        "tok-1",
			Name:      "ci-token",
			Token:     "mxat_abc123",
			CreatedAt: now,
		},
	}
	router := newAPITokenRouter(stub)

	w := doAPIToken(t, router, http.MethodPost, "/", `{"name":"ci-token"}`, "u-1")

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "tok-1", data["id"])
	assert.Equal(t, "mxat_abc123", data["token"])
}

func TestCreate_MaxExceeded_Returns409(t *testing.T) {
	stub := &stubAPITokenService{createErr: service.ErrAPITokenMaxExceeded}
	router := newAPITokenRouter(stub)

	w := doAPIToken(t, router, http.MethodPost, "/", `{"name":"new"}`, "u-1")

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "上限")
}

func TestCreate_NameExists_Returns409(t *testing.T) {
	stub := &stubAPITokenService{createErr: service.ErrAPITokenNameExists}
	router := newAPITokenRouter(stub)

	w := doAPIToken(t, router, http.MethodPost, "/", `{"name":"dup"}`, "u-1")

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "同名")
}

func TestCreate_GenericError_Returns400(t *testing.T) {
	stub := &stubAPITokenService{createErr: errors.New("something went wrong")}
	router := newAPITokenRouter(stub)

	w := doAPIToken(t, router, http.MethodPost, "/", `{"name":"test"}`, "u-1")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestList_NoUser_Returns401(t *testing.T) {
	stub := &stubAPITokenService{}
	router := newAPITokenRouter(stub)

	w := doAPITokenNoUser(t, router, http.MethodGet, "/", "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "未授权")
}

func TestList_Success_ReturnsTokens(t *testing.T) {
	now := time.Now()
	stub := &stubAPITokenService{
		tokens: []*dto.APITokenResp{
			{ID: "tok-1", Name: "ci", CreatedAt: now, Revoked: false},
			{ID: "tok-2", Name: "deploy", CreatedAt: now, Revoked: true},
		},
	}
	router := newAPITokenRouter(stub)

	w := doAPIToken(t, router, http.MethodGet, "/", "", "u-1")

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	data := resp["data"].(map[string]interface{})
	tokens := data["tokens"].([]interface{})
	assert.Len(t, tokens, 2)
}

func TestList_Empty_ReturnsEmptyArray(t *testing.T) {
	stub := &stubAPITokenService{tokens: []*dto.APITokenResp{}}
	router := newAPITokenRouter(stub)

	w := doAPIToken(t, router, http.MethodGet, "/", "", "u-1")

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	data := resp["data"].(map[string]interface{})
	tokens := data["tokens"].([]interface{})
	assert.Len(t, tokens, 0)
}

func TestList_ServiceError_Returns500(t *testing.T) {
	stub := &stubAPITokenService{tokensErr: errors.New("db error")}
	router := newAPITokenRouter(stub)

	w := doAPIToken(t, router, http.MethodGet, "/", "", "u-1")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取令牌列表失败")
}

func TestRevoke_NoUser_Returns401(t *testing.T) {
	stub := &stubAPITokenService{}
	router := newAPITokenRouter(stub)

	w := doAPITokenNoUser(t, router, http.MethodPost, "/revoke", `{"id":"tok-1"}`)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "未授权")
}

func TestRevoke_ValidationFailure_Returns400(t *testing.T) {
	stub := &stubAPITokenService{}
	router := newAPITokenRouter(stub)

	w := doAPIToken(t, router, http.MethodPost, "/revoke", `{"id":""}`, "u-1")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRevoke_Success(t *testing.T) {
	stub := &stubAPITokenService{}
	router := newAPITokenRouter(stub)

	w := doAPIToken(t, router, http.MethodPost, "/revoke", `{"id":"tok-1"}`, "u-1")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRevoke_NotFound_Returns404(t *testing.T) {
	stub := &stubAPITokenService{revokeErr: service.ErrAPITokenNotFound}
	router := newAPITokenRouter(stub)

	w := doAPIToken(t, router, http.MethodPost, "/revoke", `{"id":"tok-99"}`, "u-1")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "令牌不存在")
}

func TestRevoke_AlreadyRevoked_Returns400(t *testing.T) {
	stub := &stubAPITokenService{revokeErr: service.ErrAPITokenRevoked}
	router := newAPITokenRouter(stub)

	w := doAPIToken(t, router, http.MethodPost, "/revoke", `{"id":"tok-1"}`, "u-1")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "已被撤销")
}

func TestRevoke_GenericError_Returns500(t *testing.T) {
	stub := &stubAPITokenService{revokeErr: errors.New("db error")}
	router := newAPITokenRouter(stub)

	w := doAPIToken(t, router, http.MethodPost, "/revoke", `{"id":"tok-1"}`, "u-1")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "撤销令牌失败")
}