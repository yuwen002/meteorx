package integration

import (
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_OAuth_ProviderAdminAndPublic 整体验证 OAuth 模块：
//   - 前台公开：providers 列表、授权跳转（未配置→400、配置后→200）
//   - 后台配置 CRUD：仅超管可访问（RequiresMasterAdmin 外层 + AutoRequirePermission 内层）
//   - 两层防线：即便授予细粒度权限码，非超管仍被 403 拦截
func TestInteg_OAuth_ProviderAdminAndPublic(t *testing.T) {
	app := testkit.NewTestApp(t)
	sa := app.SuperadminHeader("sa", "t-1")
	member := app.AuthHeader("m", "t-1", []string{"member"})

	// 前台：未登录可查 providers 列表
	w := app.Do(t, http.MethodGet, "/api/v1/auth/oauth/providers", "", "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "providers")

	// 前台：未配置的 provider 跳转 → 失败（400/500，不得 200）
	w = app.Do(t, http.MethodGet, "/api/v1/auth/oauth/google/redirect", "", "")
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusInternalServerError}, w.Code, "未配置 provider 应失败")

	// 后台：非超管访问提供商配置 → 403
	assert.Equal(t, http.StatusForbidden, app.Do(t, http.MethodGet, "/api/v1/admin/oauth-providers", "", member).Code)

	// 两层防线：授予细粒度权限码后，非超管仍被 RequiresMasterAdmin 拦截
	grantPermissionToUser(t, app.DB, "u-oauth", "t-1", "admin:oauth_provider:list")
	granted := app.AuthHeader("u-oauth", "t-1", []string{"member"})
	assert.Equal(t, http.StatusForbidden, app.Do(t, http.MethodGet, "/api/v1/admin/oauth-providers", "", granted).Code,
		"有细粒度权限但非超管仍应 403")

	// 超管：创建提供商配置
	create := `{"kind":"google","code":"google","client_id":"gid-123","client_secret":"gsecret","redirect_url":"https://app.example.com/callback","enabled":true}`
	w = app.Do(t, http.MethodPost, "/api/v1/admin/oauth-providers", create, sa)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 超管：列表/详情/更新/状态/删除闭环
	w = app.Do(t, http.MethodGet, "/api/v1/admin/oauth-providers", "", sa)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "google")

	assert.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, "/api/v1/admin/oauth-providers/google", "", sa).Code)

	upd := `{"kind":"google","code":"google","client_id":"gid-123","redirect_url":"https://app.example.com/cb","enabled":true}`
	assert.Equal(t, http.StatusOK, app.Do(t, http.MethodPut, "/api/v1/admin/oauth-providers/google", upd, sa).Code)

	assert.Equal(t, http.StatusOK, app.Do(t, http.MethodPut, "/api/v1/admin/oauth-providers/google/status",
		`{"enabled":false}`, sa).Code)

	// 删除后详情查不到（非 200）
	assert.Equal(t, http.StatusOK, app.Do(t, http.MethodDelete, "/api/v1/admin/oauth-providers/google", "", sa).Code)
	assert.NotEqual(t, http.StatusOK, app.Do(t, http.MethodGet, "/api/v1/admin/oauth-providers/google", "", sa).Code,
		"删除后详情应查不到")
}
