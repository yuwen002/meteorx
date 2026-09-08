package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/user/dto"
	"meteorx/internal/modules/user/handler"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUserRouter(stub handler.UserService) http.Handler {
	h := handler.NewUserHandler(stub)
	r := chi.NewRouter()

	// 与真实路由一致的挂载形态（省略权限中间件，其行为由 middleware 测试覆盖）
	r.Route("/users", func(r chi.Router) {
		r.Get("/", h.ListUsers)
		r.Post("/", h.CreateUser)
		r.Get("/deleted", h.ListDeletedUsers)
		r.Get("/{id}/detail", h.GetUser)
		r.Put("/{id}/update", h.UpdateUser)
		r.Put("/{id}/reset-password", h.ResetPassword)
		r.Delete("/{id}/delete", h.DeleteUser)
		r.Put("/{id}/restore", h.RestoreUser)
		r.Delete("/{id}/permanent", h.PermanentDeleteUser)
	})
	r.Route("/profile", func(r chi.Router) {
		r.Get("/stats", h.GetStats)
		r.Get("/", h.GetProfile)
		r.Put("/", h.UpdateProfile)
		r.Put("/password", h.ChangePassword)
	})

	r.Get("/admin/stats", h.GetAllStats)

	r.Route("/admin/users", func(r chi.Router) {
		r.Get("/", h.ListMasterAdmins)
		r.Post("/", h.CreateMasterAdmin)
		r.Get("/deleted", h.ListDeletedMasterAdmins)
		r.Put("/batch/status", h.BatchUpdateMasterAdminStatus)
		r.Delete("/batch/delete", h.BatchDeleteMasterAdmins)
		r.Get("/{id}/detail", h.GetMasterAdmin)
		r.Put("/{id}/update", h.UpdateMasterAdmin)
		r.Put("/{id}/status", h.UpdateMasterAdminStatus)
		r.Delete("/{id}/delete", h.DeleteMasterAdmin)
		r.Put("/{id}/restore", h.RestoreMasterAdmin)
		r.Delete("/{id}/permanent", h.PermanentDeleteMasterAdmin)
	})

	r.Route("/admin/tenant-users", func(r chi.Router) {
		r.Post("/", h.AdminCreateTenantUser)
		r.Get("/all", h.AdminListAllTenantUsers)
		r.Get("/deleted/all", h.AdminListAllDeletedTenantUsers)
		r.Get("/{tenantID}/list", h.AdminListTenantUsers)
		r.Get("/{tenantID}/deleted", h.AdminListDeletedTenantUsers)
		r.Put("/{tenantID}/{userID}/update", h.AdminUpdateTenantUser)
		r.Put("/{tenantID}/{userID}/status", h.AdminUpdateTenantUserStatus)
		r.Put("/{tenantID}/{userID}/reset-password", h.AdminResetTenantUserPassword)
		r.Put("/{tenantID}/{userID}/restore", h.AdminRestoreTenantUser)
		r.Delete("/{tenantID}/{userID}/delete", h.AdminDeleteTenantUser)
		r.Delete("/{tenantID}/{userID}/permanent", h.AdminPermanentDeleteTenantUser)
		r.Put("/{tenantID}/batch/status", h.AdminBatchUpdateTenantUserStatus)
		r.Delete("/{tenantID}/batch/delete", h.AdminBatchDeleteTenantUsers)
	})
	return r
}

// do 发送带登录态（租户/用户上下文）的请求
func do(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req = req.WithContext(contextx.SetVars(req.Context(), "tenant-1", "user-1", []string{"admin"}))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// doAnon 发送匿名请求（无身份上下文，用于 401 分支）
func doAnon(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// doRaw 不经路由直接调用 handler 方法（chi 上下文无 URL 参数，用于空参数分支）
func doRaw(fn http.HandlerFunc) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(contextx.SetVars(req.Context(), "tenant-1", "user-1", []string{"superadmin"}))
	fn(w, req)
	return w
}

// requireOK 断言 200 并把 data 解码到 target
func requireOK(t *testing.T, w *httptest.ResponseRecorder, target any) {
	t.Helper()
	require.Equal(t, http.StatusOK, w.Code, "unexpected status, body=%s", w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), target))
}

func sampleUserResp(id string) *dto.UserResp {
	return &dto.UserResp{ID: id, TenantID: "tenant-1", Username: "alice", Roles: []string{"admin"}, Status: 1}
}

func assertFailCode(t *testing.T, w *httptest.ResponseRecorder, want int, msg string) {
	t.Helper()
	assert.Equal(t, want, w.Code, "unexpected status, body=%s", w.Body.String())
	if msg != "" {
		assert.Contains(t, w.Body.String(), msg)
	}
}
