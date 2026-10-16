package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/modules/oauth/dto"
	"meteorx/internal/modules/oauth/handler"
	"meteorx/internal/modules/oauth/service"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubOAuthProviderAdminService struct {
	listResult []dto.OAuthProviderResp
	listErr    error

	getResult *dto.OAuthProviderResp
	getErr    error

	createErr error
	updateErr error
	deleteErr error
	statusErr error

	lastCreate       dto.SaveOAuthProviderReq
	lastUpdateCode   string
	lastUpdate       dto.SaveOAuthProviderReq
	lastDeleteCode   string
	lastStatusCode   string
	lastStatusEnable bool
}

func (s *stubOAuthProviderAdminService) ListOAuthProviderConfigs(context.Context) ([]dto.OAuthProviderResp, error) {
	return s.listResult, s.listErr
}

func (s *stubOAuthProviderAdminService) GetOAuthProviderConfig(_ context.Context, code string) (*dto.OAuthProviderResp, error) {
	s.lastDeleteCode = code // 复用字段记录查询 code
	return s.getResult, s.getErr
}

func (s *stubOAuthProviderAdminService) CreateOAuthProviderConfig(_ context.Context, req dto.SaveOAuthProviderReq) error {
	s.lastCreate = req
	return s.createErr
}

func (s *stubOAuthProviderAdminService) UpdateOAuthProviderConfig(_ context.Context, code string, req dto.SaveOAuthProviderReq) error {
	s.lastUpdateCode = code
	s.lastUpdate = req
	return s.updateErr
}

func (s *stubOAuthProviderAdminService) DeleteOAuthProviderConfig(_ context.Context, code string) error {
	s.lastDeleteCode = code
	return s.deleteErr
}

func (s *stubOAuthProviderAdminService) SetOAuthProviderConfigStatus(_ context.Context, code string, enabled bool) error {
	s.lastStatusCode = code
	s.lastStatusEnable = enabled
	return s.statusErr
}

func newProviderAdminRouter(stub *stubOAuthProviderAdminService) http.Handler {
	h := handler.NewOAuthProviderAdminHandler(stub)
	r := chi.NewRouter()
	r.Get("/admin/oauth-providers", h.ListProviders)
	r.Post("/admin/oauth-providers", h.CreateProvider)
	r.Get("/admin/oauth-providers/{code}", h.GetProvider)
	r.Put("/admin/oauth-providers/{code}", h.UpdateProvider)
	r.Delete("/admin/oauth-providers/{code}", h.DeleteProvider)
	r.Put("/admin/oauth-providers/{code}/status", h.SetProviderStatus)
	return r
}

func doProviderAdmin(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
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

func TestProviderAdmin_List(t *testing.T) {
	stub := &stubOAuthProviderAdminService{
		listResult: []dto.OAuthProviderResp{
			{Code: "wechat", Kind: "wechat", DisplayName: "微信", Enabled: true, ClientID: "wx", ClientSecretMask: "****cret", HasSecret: true},
		},
	}
	w := doProviderAdmin(t, newProviderAdminRouter(stub), http.MethodGet, "/admin/oauth-providers", "")

	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data struct {
			Providers []dto.OAuthProviderResp `json:"providers"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Data.Providers, 1)
	assert.Equal(t, "wechat", resp.Data.Providers[0].Code)
	assert.True(t, resp.Data.Providers[0].Enabled)
}

func TestProviderAdmin_Create(t *testing.T) {
	stub := &stubOAuthProviderAdminService{}
	w := doProviderAdmin(t, newProviderAdminRouter(stub), http.MethodPost, "/admin/oauth-providers",
		`{"kind":"wechat","code":"wechat","client_id":"wx-id","client_secret":"wx-secret","redirect_url":"https://e/c","enabled":true}`)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "wechat", stub.lastCreate.Kind)
	assert.Equal(t, "wx-id", stub.lastCreate.ClientID)
	assert.True(t, stub.lastCreate.Enabled)
}

func TestProviderAdmin_Create_ValidationFailure(t *testing.T) {
	stub := &stubOAuthProviderAdminService{}
	w := doProviderAdmin(t, newProviderAdminRouter(stub), http.MethodPost, "/admin/oauth-providers", `{"kind":"wechat"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProviderAdmin_Create_InvalidInput(t *testing.T) {
	stub := &stubOAuthProviderAdminService{createErr: service.ErrOAuthProviderInvalid}
	w := doProviderAdmin(t, newProviderAdminRouter(stub), http.MethodPost, "/admin/oauth-providers",
		`{"kind":"wechat","code":"wechat","client_id":"a","redirect_url":"https://e/c"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProviderAdmin_Create_CodeTaken(t *testing.T) {
	stub := &stubOAuthProviderAdminService{createErr: service.ErrOAuthProviderCodeTaken}
	w := doProviderAdmin(t, newProviderAdminRouter(stub), http.MethodPost, "/admin/oauth-providers",
		`{"kind":"wechat","code":"wechat","client_id":"a","redirect_url":"https://e/c"}`)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestProviderAdmin_Create_SecretKeyMissing(t *testing.T) {
	stub := &stubOAuthProviderAdminService{createErr: service.ErrOAuthSecretKeyNotConfigured}
	w := doProviderAdmin(t, newProviderAdminRouter(stub), http.MethodPost, "/admin/oauth-providers",
		`{"kind":"wechat","code":"wechat","client_id":"a","client_secret":"s","redirect_url":"https://e/c"}`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestProviderAdmin_Update(t *testing.T) {
	stub := &stubOAuthProviderAdminService{}
	w := doProviderAdmin(t, newProviderAdminRouter(stub), http.MethodPut, "/admin/oauth-providers/wechat",
		`{"kind":"wechat","code":"wechat","client_id":"new-id","redirect_url":"https://e/c2","enabled":false}`)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "wechat", stub.lastUpdateCode)
	assert.Equal(t, "new-id", stub.lastUpdate.ClientID)
	assert.False(t, stub.lastUpdate.Enabled)
}

func TestProviderAdmin_Update_NotFound(t *testing.T) {
	stub := &stubOAuthProviderAdminService{updateErr: service.ErrOAuthProviderNotFound}
	w := doProviderAdmin(t, newProviderAdminRouter(stub), http.MethodPut, "/admin/oauth-providers/nope",
		`{"kind":"wechat","code":"nope","client_id":"a","redirect_url":"https://e/c"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProviderAdmin_Delete(t *testing.T) {
	stub := &stubOAuthProviderAdminService{}
	w := doProviderAdmin(t, newProviderAdminRouter(stub), http.MethodDelete, "/admin/oauth-providers/wechat", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "wechat", stub.lastDeleteCode)
}

func TestProviderAdmin_Delete_NotFound(t *testing.T) {
	stub := &stubOAuthProviderAdminService{deleteErr: service.ErrOAuthProviderNotFound}
	w := doProviderAdmin(t, newProviderAdminRouter(stub), http.MethodDelete, "/admin/oauth-providers/nope", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProviderAdmin_SetStatus(t *testing.T) {
	stub := &stubOAuthProviderAdminService{}
	w := doProviderAdmin(t, newProviderAdminRouter(stub), http.MethodPut, "/admin/oauth-providers/wechat/status", `{"enabled":false}`)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "wechat", stub.lastStatusCode)
	assert.False(t, stub.lastStatusEnable)
}

func TestProviderAdmin_SetStatus_ValidationFailure(t *testing.T) {
	stub := &stubOAuthProviderAdminService{}
	w := doProviderAdmin(t, newProviderAdminRouter(stub), http.MethodPut, "/admin/oauth-providers/wechat/status", `{}`)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProviderAdmin_Get(t *testing.T) {
	stub := &stubOAuthProviderAdminService{
		getResult: &dto.OAuthProviderResp{Code: "wechat", Kind: "wechat", DisplayName: "微信", Enabled: true},
	}
	w := doProviderAdmin(t, newProviderAdminRouter(stub), http.MethodGet, "/admin/oauth-providers/wechat", "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "wechat")
}

func TestProviderAdmin_Get_NotFound(t *testing.T) {
	stub := &stubOAuthProviderAdminService{getErr: service.ErrOAuthProviderNotFound}
	w := doProviderAdmin(t, newProviderAdminRouter(stub), http.MethodGet, "/admin/oauth-providers/nope", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}
