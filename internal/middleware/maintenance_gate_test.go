package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/common/contextx"

	"github.com/stretchr/testify/assert"
)

// maintProviderStub MaintenanceStateProvider 测试桩
type maintProviderStub struct {
	enabled bool
	message string
}

func (m maintProviderStub) MaintenanceState(context.Context) (bool, string) {
	return m.enabled, m.message
}

// maintRequest 构造携带角色上下文的请求
func maintRequest(roles []string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	ctx := contextx.SetVars(req.Context(), "t-1", "u-1", roles)
	return req.WithContext(ctx)
}

func runMaintenanceGate(t *testing.T, provider MaintenanceStateProvider, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	h := MaintenanceGate(provider)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestMaintenanceGate(t *testing.T) {
	cases := []struct {
		name     string
		enabled  bool
		message  string
		roles    []string
		wantCode int
		wantBody string
	}{
		{"未开启维护放行", false, "", []string{"member"}, http.StatusOK, ""},
		{"维护中普通用户503", true, "今晚升级", []string{"member"}, http.StatusServiceUnavailable, "今晚升级"},
		{"维护中含超管放行", true, "今晚升级", []string{"superadmin"}, http.StatusOK, ""},
		{"维护中多角色含超管放行", true, "", []string{"member", "superadmin"}, http.StatusOK, ""},
		{"维护中无角色503", true, "", []string{}, http.StatusServiceUnavailable, "系统维护中，请稍后再试"},
		{"空文案回落默认提示", true, "", []string{"tenant_admin"}, http.StatusServiceUnavailable, "系统维护中，请稍后再试"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := runMaintenanceGate(t, maintProviderStub{enabled: c.enabled, message: c.message}, maintRequest(c.roles))
			assert.Equal(t, c.wantCode, w.Code)
			if c.wantBody != "" {
				assert.Contains(t, w.Body.String(), c.wantBody)
			}
		})
	}
}
