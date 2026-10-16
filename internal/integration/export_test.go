package integration

import (
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_Export_CreateAndPerm 整体验证导出模块（受保护 + AutoRequirePermission）：
// 有 export:list/create/read 的用户可创建任务并查询；无权限用户 → 403。
func TestInteg_Export_CreateAndPerm(t *testing.T) {
	app := testkit.NewTestApp(t)
	grantPermissions(t, app.DB, "u-export", "t-1", "export:list", "export:create", "export:read")
	grantAuth := app.AuthHeader("u-export", "t-1", []string{"member"})
	noAuth := app.AuthHeader("u-none", "t-1", []string{"member"})

	// 创建导出任务
	w := app.Do(t, http.MethodPost, "/api/v1/exports", `{"resource":"user","format":"csv"}`, grantAuth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	id := decodeID(t, w)

	// 列表/详情可查
	w = app.Do(t, http.MethodGet, "/api/v1/exports", "", grantAuth)
	assert.Contains(t, w.Body.String(), id)
	w = app.Do(t, http.MethodGet, "/api/v1/exports/"+id, "", grantAuth)
	assert.Equal(t, http.StatusOK, w.Code)

	// 无 export 权限 → 403
	w = app.Do(t, http.MethodGet, "/api/v1/exports", "", noAuth)
	assert.Equal(t, http.StatusForbidden, w.Code, "无 export:list 应 403")
}
