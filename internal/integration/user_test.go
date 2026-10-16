package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_User_ProfileAndAdmin 整体验证用户模块：注册→登录→当前用户资料；
// 后台用户管理仅超管可访问（普通租户用户越权 403）。
func TestInteg_User_ProfileAndAdmin(t *testing.T) {
	app := testkit.NewTestApp(t)
	seedTenantAdminRole(t, app.DB)

	// 注册租户管理员
	w := app.Do(t, http.MethodPost, "/api/v1/auth/register", `{
		"tenant_id":"t-u1","username":"bob1234","password":"pass1234","nickname":"鲍勃","email":"b@x.com"
	}`, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 登录拿令牌
	w = app.Do(t, http.MethodPost, "/api/v1/auth/login", `{
		"tenant_id":"t-u1","username":"bob1234","password":"pass1234"
	}`, "")
	require.Equal(t, http.StatusOK, w.Code)
	var login struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &login))
	require.NotEmpty(t, login.Data.Token)
	userAuth := "Bearer " + login.Data.Token

	// 当前用户资料 → 200
	w = app.Do(t, http.MethodGet, "/api/v1/profile", "", userAuth)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 后台用户管理：超管可列，普通租户用户越权 403
	sa := app.SuperadminHeader("sa", "t-1")
	w = app.Do(t, http.MethodGet, "/api/v1/admin/users", "", sa)
	assert.Equal(t, http.StatusOK, w.Code)
	w = app.Do(t, http.MethodGet, "/api/v1/admin/users", "", userAuth)
	assert.Equal(t, http.StatusForbidden, w.Code, "普通租户用户不应访问后台用户管理")
}
