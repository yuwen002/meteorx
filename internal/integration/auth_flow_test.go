package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	rbacModel "meteorx/internal/modules/rbac/model"
	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_AuthFlow 整体验证认证模块核心闭环：注册 -> 登录 -> 当前用户。
// 通过完整路由 + 真实 sqlite + 真实 JWT 驱动，确认注册默认角色、登录签发、身份校验链路可用。
func TestInteg_AuthFlow(t *testing.T) {
	app := testkit.NewTestApp(t)

	// 注册所需的默认角色 tenant_admin（SeedDatabase 仅在正式启动时播种，测试基座不依赖外部 seed 文件）
	require.NoError(t, app.DB.Create(&rbacModel.Role{
		ID: "role-tenant-admin", Name: "租户管理员", Code: "tenant_admin",
		TenantID: "", Scope: "tenant", Status: 1,
	}).Error)

	// 1) 注册
	w := app.Do(t, http.MethodPost, "/api/v1/auth/register", `{
		"tenant_id":"t-1","username":"alice123","password":"pass1234","nickname":"爱丽丝","email":"a@x.com"
	}`, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 2) 登录
	w = app.Do(t, http.MethodPost, "/api/v1/auth/login", `{
		"tenant_id":"t-1","username":"alice123","password":"pass1234"
	}`, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var loginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginResp))
	require.NotEmpty(t, loginResp.Data.Token, "登录应返回 token")

	// 3) 用登录令牌访问受保护资源（当前用户）
	w = app.Do(t, http.MethodGet, "/api/v1/profile", "", "Bearer "+loginResp.Data.Token)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// 4) 带 email_pending 临时令牌访问受保护资源应被门禁拦截（补全邮箱前的临时态不可越权）
	//    用 tokenHelper 签发 emailPending=true 的令牌
	pendingTok, err := app.TokenHelper.GenerateTokenEx("u-pending", "t-1", nil, true)
	require.NoError(t, err)
	w = app.Do(t, http.MethodGet, "/api/v1/profile", "", "Bearer "+pendingTok)
	assert.Equal(t, http.StatusForbidden, w.Code, "email_pending 临时令牌应被门禁拦截")
}
