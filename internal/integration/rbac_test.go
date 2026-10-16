package integration

import (
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_RBAC_PermissionEnforcement 整体验证权限强制链路：
// 受保护路由 /api/v1/users 需要 user:list，数据库权限码决定放行/拦截（非仅看 JWT 角色）。
func TestInteg_RBAC_PermissionEnforcement(t *testing.T) {
	app := testkit.NewTestApp(t)

	grantPermissionToUser(t, app.DB, "u-grant", "t-1", "user:list")
	grantAuth := app.AuthHeader("u-grant", "t-1", []string{"member"})
	noAuth := app.AuthHeader("u-none", "t-1", []string{"member"})

	w := app.Do(t, http.MethodGet, "/api/v1/users", "", grantAuth)
	assert.Equal(t, http.StatusOK, w.Code, "有 user:list 权限应放行")

	w = app.Do(t, http.MethodGet, "/api/v1/users", "", noAuth)
	assert.Equal(t, http.StatusForbidden, w.Code, "无权限应 403")
}

// TestInteg_RBAC_RoleCRUD 整体验证角色管理的后台闭环与网关：
// 超管可创建并查到角色；非超管创建被 403 拦截。
func TestInteg_RBAC_RoleCRUD(t *testing.T) {
	app := testkit.NewTestApp(t)
	sa := app.SuperadminHeader("sa", "t-1")
	member := app.AuthHeader("m", "t-1", []string{"member"})

	body := `{"name":"集成测试角色","code":"role-integ-1","scope":"tenant","status":1}`
	w := app.Do(t, http.MethodPost, "/api/v1/rbac/roles", body, sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = app.Do(t, http.MethodGet, "/api/v1/rbac/roles", "", sa)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "role-integ-1")

	w = app.Do(t, http.MethodPost, "/api/v1/rbac/roles", body, member)
	assert.Equal(t, http.StatusForbidden, w.Code, "非超管创建角色应 403")
}
