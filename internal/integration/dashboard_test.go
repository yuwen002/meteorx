package integration

import (
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
)

// TestInteg_Dashboard_Superadmin 整体验证数据看板模块（超管网关内）：
// overview/trend/slow-queries 超管可访问，非超管一律 403。
func TestInteg_Dashboard_Superadmin(t *testing.T) {
	app := testkit.NewTestApp(t)
	sa := app.SuperadminHeader("sa", "t-1")
	member := app.AuthHeader("m", "t-1", []string{"member"})

	for _, route := range []string{
		"/api/v1/admin/dashboard/overview",
		"/api/v1/admin/dashboard/trend",
		"/api/v1/admin/dashboard/slow-queries",
	} {
		assert.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, route, "", sa).Code, "超管应可访问: "+route)
		assert.Equal(t, http.StatusForbidden, app.Do(t, http.MethodGet, route, "", member).Code, "非超管应 403: "+route)
	}
}
