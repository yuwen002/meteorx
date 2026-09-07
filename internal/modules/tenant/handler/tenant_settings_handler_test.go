package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/modules/tenant/dto"
	"meteorx/internal/modules/tenant/handler"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// tenantSettingsStub 桩：嵌入接口自动满足 TenantSettingsService
type tenantSettingsStub struct {
	handler.TenantSettingsService

	err    error
	calls  []string
	params map[string][]string
}

func newSettingsStub() *tenantSettingsStub {
	return &tenantSettingsStub{params: map[string][]string{}}
}

func (s *tenantSettingsStub) rec(method string, args ...string) {
	s.calls = append(s.calls, method)
	s.params[method] = args
}

func (s *tenantSettingsStub) failWith(err error) *tenantSettingsStub {
	s.err = err
	return s
}

func (s *tenantSettingsStub) GetSettings(_ context.Context, tenantID string) (*dto.TenantSettingsResp, error) {
	s.rec("GetSettings", tenantID)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.TenantSettingsResp{TenantID: tenantID, WelcomeText: "欢迎使用系统", PrimaryColor: "#1890ff"}, nil
}

func (s *tenantSettingsStub) UpdateSettings(_ context.Context, tenantID string, req dto.UpdateTenantSettingsReq) (*dto.TenantSettingsResp, error) {
	s.rec("UpdateSettings", tenantID, req.Logo, req.Language)
	if s.err != nil {
		return nil, s.err
	}
	return &dto.TenantSettingsResp{TenantID: tenantID, Logo: req.Logo, Language: req.Language}, nil
}

func newSettingsRouter(stub *tenantSettingsStub) http.Handler {
	h := handler.NewTenantSettingsHandler(stub)
	r := chi.NewRouter()
	r.Use(withTenantIdentity)
	r.Get("/tenant-settings", h.GetSettings)
	r.Put("/tenant-settings", h.UpdateSettings)
	return r
}

func doSettings(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
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

func TestTenantSettingsGet_Success(t *testing.T) {
	stub := newSettingsStub()
	router := newSettingsRouter(stub)

	w := doSettings(t, router, http.MethodGet, "/tenant-settings", "")

	assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), `"welcome_text":"欢迎使用系统"`)
	assert.Equal(t, []string{tTenantID}, stub.params["GetSettings"])
}

func TestTenantSettingsGet_ServiceError_Returns500(t *testing.T) {
	stub := newSettingsStub().failWith(assert.AnError)
	router := newSettingsRouter(stub)

	w := doSettings(t, router, http.MethodGet, "/tenant-settings", "")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取租户设置失败")
}

func TestTenantSettingsUpdate_Success(t *testing.T) {
	stub := newSettingsStub()
	router := newSettingsRouter(stub)

	w := doSettings(t, router, http.MethodPut, "/tenant-settings", `{"logo":"/logo.png","language":"zh-CN"}`)

	assert.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), `"language":"zh-CN"`)
	assert.Equal(t, []string{tTenantID, "/logo.png", "zh-CN"}, stub.params["UpdateSettings"])
}

func TestTenantSettingsUpdate_ServiceError_Returns500(t *testing.T) {
	stub := newSettingsStub().failWith(assert.AnError)
	router := newSettingsRouter(stub)

	w := doSettings(t, router, http.MethodPut, "/tenant-settings", `{"logo":"/logo.png"}`)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "更新租户设置失败")
}

// 上下文缺失时返回 401
func TestTenantSettings_MissingTenant_Returns401(t *testing.T) {
	stub := newSettingsStub()
	r := chi.NewRouter() // 无 withTenantIdentity
	h := handler.NewTenantSettingsHandler(stub)
	r.Get("/tenant-settings", h.GetSettings)

	w := doSettings(t, r, http.MethodGet, "/tenant-settings", "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, stub.calls)
}
