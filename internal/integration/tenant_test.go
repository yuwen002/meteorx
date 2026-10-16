package integration

import (
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_Tenant_AdminCRUDAndGate 整体验证租户模块：
// 后台建租户闭环、非超管建租户被 403、租户状态门禁（禁用租户成员业务请求一律 403）。
func TestInteg_Tenant_AdminCRUDAndGate(t *testing.T) {
	app := testkit.NewTestApp(t)
	seedTenantAdminRole(t, app.DB)
	sa := app.SuperadminHeader("sa", "t-1")
	member := app.AuthHeader("m", "t-1", []string{"member"})

	// 超管创建启用态租户 A（含初始管理员）
	tenantA := ensureTenant(t, app, "租户A", "acme-a", sa)
	require.NotEmpty(t, tenantA)

	w := app.Do(t, http.MethodGet, "/api/v1/admin/tenants", "", sa)
	assert.Contains(t, w.Body.String(), tenantA)

	// 非超管创建租户 → 403
	w = app.Do(t, http.MethodPost, "/api/v1/admin/tenants", `{
		"name":"X","domain":"x-invalid","status":1,
		"admin_user":{"username":"xadm1234","password":"secret123","nickname":"X","email":"x@x.com"}
	}`, member)
	assert.Equal(t, http.StatusForbidden, w.Code, "非超管创建租户应 403")

	// 创建租户 B 后立即禁用，验证状态门禁拦截其成员业务请求
	tenantB := ensureTenant(t, app, "租户B", "acme-b", sa)
	w = app.Do(t, http.MethodPut, "/api/v1/admin/tenants/"+tenantB+"/status", `{"status":0}`, sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	disabledAuth := app.AuthHeader("anyone", tenantB, []string{"member"})
	w = app.Do(t, http.MethodGet, "/api/v1/profile", "", disabledAuth)
	assert.Equal(t, http.StatusForbidden, w.Code, "禁用租户成员应被状态门禁拦截")

	// 启用态租户 A 的成员请求应通过门禁（非 403）
	enabledAuth := app.AuthHeader("anyone", tenantA, []string{"member"})
	w = app.Do(t, http.MethodGet, "/api/v1/profile", "", enabledAuth)
	assert.NotEqual(t, http.StatusForbidden, w.Code, "启用租户应通过状态门禁")
}
