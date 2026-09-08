package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/rbac/handler"

	"github.com/go-chi/chi/v5"
)

// newRBACRouter 注册 RBACHandler 全部路由（方法+路径与 module 装配一致）
func newRBACRouter(stub *stubRBACService) http.Handler {
	h := handler.NewRBACHandler(stub)
	r := chi.NewRouter()

	// 角色
	r.Post("/roles", h.CreateRole)
	r.Get("/roles", h.ListRoles)
	r.Get("/roles/select", h.ListRolesForSelect)
	r.Get("/roles/system-admin", h.ListSystemAdminRoles)
	r.Get("/roles/deleted", h.ListDeletedRoles)
	r.Get("/roles/{id}/detail", h.GetRole)
	r.Put("/roles/{id}/update", h.UpdateRole)
	r.Delete("/roles/{id}/delete", h.DeleteRole)
	r.Delete("/roles/{id}/permanent", h.PermanentDeleteRole)
	r.Post("/roles/{id}/restore", h.RestoreRole)
	r.Put("/roles/{id}/status", h.UpdateRoleStatus)
	r.Put("/roles/batch/status", h.BatchUpdateRoleStatus)
	r.Delete("/roles/batch/permanent", h.BatchPermanentDeleteRoles)
	r.Delete("/roles/batch/delete", h.BatchDeleteRoles)

	// 角色-权限
	r.Put("/roles/{id}/permissions", h.BindRolePermissions)
	r.Get("/roles/{id}/permissions", h.GetRolePermissions)
	r.Delete("/roles/{id}/permissions", h.UnbindRolePermission)
	r.Delete("/roles/{id}/permissions/batch", h.UnbindRolePermissions)
	r.Put("/roles/batch/permissions", h.BatchBindRolesPermissions)
	r.Delete("/roles/batch/permissions", h.BatchUnbindRolesPermissions)
	r.Get("/role-permissions", h.ListRolePermissions)

	// 权限
	r.Post("/permissions", h.CreatePermission)
	r.Get("/permissions", h.ListPermissions)
	r.Get("/permissions/{id}/detail", h.GetPermission)
	r.Put("/permissions/{id}/update", h.UpdatePermission)
	r.Put("/permissions/{id}/status", h.UpdatePermissionStatus)
	r.Delete("/permissions/{id}/delete", h.DeletePermission)
	r.Put("/permissions/batch/status", h.BatchUpdatePermissionStatus)
	r.Delete("/permissions/batch/delete", h.BatchDeletePermissions)

	// 用户-角色
	r.Post("/user-roles/{user_id}/roles", h.AssignUserRoles)
	r.Get("/user-roles/{user_id}/roles", h.GetUserRoles)
	r.Delete("/user-roles/{user_id}/roles/{role_id}", h.RemoveUserRole)
	r.Delete("/user-roles/{user_id}/roles", h.RemoveAllUserRoles)
	r.Get("/user-roles/roles/{role_id}/users", h.GetRoleUsers)
	r.Get("/user-roles", h.ListUserRoles)
	r.Post("/user-roles/batch/assign", h.BatchAssignUserRoles)

	// 统计
	r.Get("/stats", h.GetStats)
	return r
}

// doRBAC 发送携带登录上下文（租户/用户）的请求
func doRBAC(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
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
