package integration

import (
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
)

// TestInteg_GatewayConsistency 整体验证分组二/分组三网关在完整路由上一致生效。
// 覆盖 oauth/user/tenant/plan/rbac/audit/notification/dashboard 八大后台模块的管理端点，
// 任何一处网关错位（如当初 OAuth 公开侧注册表不 reload、或超管端点挂错分组）都会在此暴露：
//   - 受保护路由无 token → 401
//   - 受保护路由需权限但用户无权限 → 403（AutoRequirePermission 真实拦截）
//   - 平台超管端点：非超管 → 403（RequiresMasterAdmin 真实拦截）；超管 → 放行（不 403）
func TestInteg_GatewayConsistency(t *testing.T) {
	app := testkit.NewTestApp(t)

	// 1) 受保护路由无 token → 401
	w := app.Do(t, http.MethodGet, "/api/v1/profile", "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code, "未登录应 401")

	// 2) 受保护路由（需 user:list）对无权限用户 → 403；超管 → 放行（绕过权限）
	memberAuth := app.AuthHeader("member-1", "t-1", []string{"member"})
	w = app.Do(t, http.MethodGet, "/api/v1/users", "", memberAuth)
	assert.Equal(t, http.StatusForbidden, w.Code, "无 user:list 权限应 403")
	saAuth := app.SuperadminHeader("sa-1", "t-1")
	w = app.Do(t, http.MethodGet, "/api/v1/users", "", saAuth)
	assert.NotEqual(t, http.StatusForbidden, w.Code, "超管应绕过权限放行")

	// 3) 平台超管端点：非超管一律 403；超管放行。逐模块扫描。
	adminRoutes := []string{
		"/api/v1/admin/stats",              // user 后台统计
		"/api/v1/admin/tenants",            // tenant 后台
		"/api/v1/admin/plans",              // plan 后台
		"/api/v1/admin/oauth-providers",    // oauth 后台
		"/api/v1/admin/dashboard/overview", // dashboard 后台
		"/api/v1/admin/announcements",      // notification 后台
		"/api/v1/rbac/stats",               // rbac 后台
		"/api/v1/audit/stats",              // audit 后台
	}
	for _, route := range adminRoutes {
		route := route
		t.Run("non-superadmin-403:"+route, func(t *testing.T) {
			w := app.Do(t, http.MethodGet, route, "", memberAuth)
			assert.Equal(t, http.StatusForbidden, w.Code, "非超管应 403: "+route)
		})
		t.Run("superadmin-pass:"+route, func(t *testing.T) {
			w := app.Do(t, http.MethodGet, route, "", saAuth)
			assert.NotEqual(t, http.StatusForbidden, w.Code, "超管应放行: "+route)
		})
	}
}
