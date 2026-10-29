package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"meteorx/internal/modules/settings/dto"
	"meteorx/internal/modules/settings/service"
	"meteorx/internal/testutil"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// stubService SettingService 测试桩
type stubService struct {
	views     []*dto.SettingView
	publicMap map[string]interface{}
	updateOut []*dto.SettingView
	updateErr error
	listErr   error
	publicErr error

	gotUpdateItems []dto.SettingUpdateItem
}

func (s *stubService) ListAll(context.Context) ([]*dto.SettingView, error) {
	return s.views, s.listErr
}

func (s *stubService) GetPublicMap(context.Context) (map[string]interface{}, error) {
	return s.publicMap, s.publicErr
}

func (s *stubService) UpdateBulk(_ context.Context, items []dto.SettingUpdateItem) ([]*dto.SettingView, error) {
	s.gotUpdateItems = items
	return s.updateOut, s.updateErr
}

// singleRoute 组装单路由处理器（权限中间件由 middleware 包派生专项测试单独锁定）
func singleRoute(method, path string, h func(http.ResponseWriter, *http.Request)) http.Handler {
	r := chi.NewRouter()
	switch method {
	case http.MethodGet:
		r.Get(path, h)
	case http.MethodPut:
		r.Put(path, h)
	}
	return r
}

// TestSettingHandler_ListSettings 管理列表 200 + data 透传
func TestSettingHandler_ListSettings(t *testing.T) {
	svc := &stubService{views: []*dto.SettingView{{Key: "a", Value: "1", Source: "db"}}}
	h := NewSettingHandler(svc)

	w := testutil.ExecuteRequest(singleRoute(http.MethodGet, "/", h.ListSettings), http.MethodGet, "/", nil)
	testutil.AssertStatusCode(t, w, http.StatusOK)

	var resp struct {
		Data []*dto.SettingView `json:"data"`
	}
	testutil.ParseResponse(t, w, &resp)
	require.Len(t, resp.Data, 1)
	require.Equal(t, "a", resp.Data[0].Key)
}

// TestSettingHandler_ListSettings_Error 服务错误映射 500
func TestSettingHandler_ListSettings_Error(t *testing.T) {
	svc := &stubService{listErr: errors.New("db down")}
	h := NewSettingHandler(svc)

	w := testutil.ExecuteRequest(singleRoute(http.MethodGet, "/", h.ListSettings), http.MethodGet, "/", nil)
	testutil.AssertStatusCode(t, w, http.StatusInternalServerError)
}

// TestSettingHandler_UpdateSettings 正常批量更新：请求体解析后透传 service，响应返回最新视图
func TestSettingHandler_UpdateSettings(t *testing.T) {
	svc := &stubService{updateOut: []*dto.SettingView{{Key: "registration_enabled", Value: "false"}}}
	h := NewSettingHandler(svc)

	body := testutil.NewJSONBody(t, dto.UpdateSettingsReq{
		Settings: []dto.SettingUpdateItem{{Key: "registration_enabled", Value: "false"}},
	})
	w := testutil.ExecuteRequest(singleRoute(http.MethodPut, "/", h.UpdateSettings), http.MethodPut, "/", body)
	testutil.AssertStatusCode(t, w, http.StatusOK)
	require.Len(t, svc.gotUpdateItems, 1)
	require.Equal(t, "registration_enabled", svc.gotUpdateItems[0].Key)

	var resp struct {
		Data []*dto.SettingView `json:"data"`
	}
	testutil.ParseResponse(t, w, &resp)
	require.Equal(t, "false", resp.Data[0].Value)
}

// TestSettingHandler_UpdateSettings_ValidationError 请求体校验失败（空 settings 数组）→ 400，不调 service
func TestSettingHandler_UpdateSettings_ValidationError(t *testing.T) {
	svc := &stubService{}
	h := NewSettingHandler(svc)

	body := testutil.NewJSONBody(t, dto.UpdateSettingsReq{Settings: []dto.SettingUpdateItem{}})
	w := testutil.ExecuteRequest(singleRoute(http.MethodPut, "/", h.UpdateSettings), http.MethodPut, "/", body)
	testutil.AssertStatusCode(t, w, http.StatusBadRequest)
	require.Nil(t, svc.gotUpdateItems, "校验失败不得触达 service")
}

// TestSettingHandler_UpdateSettings_BadRequest 服务返回键/值类错误 → 400；其他错误 → 500
func TestSettingHandler_UpdateSettings_BadRequest(t *testing.T) {
	hCases := []struct {
		name string
		err  error
		want int
	}{
		{"unknown key", fmt.Errorf("%w: hacker_key", service.ErrUnknownSettingKey), http.StatusBadRequest},
		{"invalid value", fmt.Errorf("%w: maintenance_mode=%q", service.ErrInvalidSettingValue, "abc"), http.StatusBadRequest},
		{"internal", errors.New("db down"), http.StatusInternalServerError},
	}
	for _, c := range hCases {
		t.Run(c.name, func(t *testing.T) {
			svc := &stubService{updateErr: c.err}
			h := NewSettingHandler(svc)
			body := testutil.NewJSONBody(t, dto.UpdateSettingsReq{
				Settings: []dto.SettingUpdateItem{{Key: "k", Value: "v"}},
			})
			w := testutil.ExecuteRequest(singleRoute(http.MethodPut, "/", h.UpdateSettings), http.MethodPut, "/", body)
			testutil.AssertStatusCode(t, w, c.want)
		})
	}
}

// TestSettingHandler_GetPublicSettings 公开接口 200，data 为键→类型化值映射
func TestSettingHandler_GetPublicSettings(t *testing.T) {
	svc := &stubService{publicMap: map[string]interface{}{"registration_enabled": true}}
	h := NewSettingHandler(svc)

	w := testutil.ExecuteRequest(singleRoute(http.MethodGet, "/public", h.GetPublicSettings), http.MethodGet, "/public", nil)
	testutil.AssertStatusCode(t, w, http.StatusOK)

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	testutil.ParseResponse(t, w, &resp)
	require.Equal(t, true, resp.Data["registration_enabled"])
	require.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

// TestSettingHandler_GetPublicSettings_Error 服务错误映射 500
func TestSettingHandler_GetPublicSettings_Error(t *testing.T) {
	svc := &stubService{publicErr: errors.New("db down")}
	h := NewSettingHandler(svc)

	w := testutil.ExecuteRequest(singleRoute(http.MethodGet, "/public", h.GetPublicSettings), http.MethodGet, "/public", nil)
	testutil.AssertStatusCode(t, w, http.StatusInternalServerError)
}
