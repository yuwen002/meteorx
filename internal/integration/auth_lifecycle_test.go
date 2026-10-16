package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"meteorx/pkg/totp"
	"meteorx/internal/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInteg_AuthLifecycle 覆盖认证模块全部主要功能，全程纯 HTTP（sqlite + miniredis，无浏览器/无外部服务）：
//  1. 注册 → 登录 → 当前用户
//  2. 登出：令牌加入 Redis 黑名单后立即失效
//  3. 2FA：setup → 用动态口令登录 → disable 后免口令登录
//  4. API Token：创建 → 列表可见 → 作为 Bearer 访问受保护接口 → 撤销后失效
//  5. 邮箱验证 / 找回密码：无 SMTP 配置时优雅降级为 500
//  6. 非法验证/重置令牌被拒
func TestInteg_AuthLifecycle(t *testing.T) {
	app := testkit.NewTestApp(t)
	seedTenantAdminRole(t, app.DB)

	const (
		tenant   = "t-authlife"
		username = "alice123"
		password = "pass1234"
		email    = "alice@example.com"
	)
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/auth/register",
		fmt.Sprintf(`{"tenant_id":%q,"username":%q,"password":%q,"nickname":"爱丽丝","email":%q}`, tenant, username, password, email), "").Code)

	login := func(otp string) (int, string) {
		body := fmt.Sprintf(`{"tenant_id":%q,"username":%q,"password":%q,"otp":%q}`, tenant, username, password, otp)
		w := app.Do(t, http.MethodPost, "/api/v1/auth/login", body, "")
		return w.Code, "Bearer " + decodeToken(t, w)
	}

	// 1) 首次登录（尚未开启 2FA）
	code, token := login("")
	require.Equal(t, http.StatusOK, code, "尚未开启 2FA 应登录成功")
	require.NotEmpty(t, token)

	// 当前用户
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, "/api/v1/profile", "", token).Code)

	// 2) 登出：令牌加入黑名单后应失效（miniredis+go-redis 连接池有极短可见性竞态，轮询容忍）
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/auth/logout", "", token).Code)
	require.Eventually(t, func() bool {
		return app.Do(t, http.MethodGet, "/api/v1/profile", "", token).Code == http.StatusUnauthorized
	}, 2*time.Second, 20*time.Millisecond, "登出后令牌应失效(401)")

	// 重新登录获得新令牌
	_, token = login("")

	// 3) 开启 2FA
	w := app.Do(t, http.MethodPost, "/api/v1/auth/2fa/setup", "", token)
	require.Equal(t, http.StatusOK, w.Code)
	var setup struct {
		Data struct {
			Secret string `json:"secret"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &setup))
	secret := setup.Data.Secret
	require.NotEmpty(t, secret)

	// setup 仅生成密钥，需提交当前口令调用 enable 后才真正生效
	otpSetup, err := totp.Now(secret)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/auth/2fa/enable",
		fmt.Sprintf(`{"code":%q}`, otpSetup), token).Code, "正确口令应激活 2FA")

	// 已开启 2FA 但未提供 otp → 401 且提示 Require2FA
	w = app.Do(t, http.MethodPost, "/api/v1/auth/login",
		fmt.Sprintf(`{"tenant_id":%q,"username":%q,"password":%q}`, tenant, username, password), "")
	require.Equal(t, http.StatusUnauthorized, w.Code)
	var loginErr struct {
		Data struct {
			Require2FA bool `json:"require_2fa"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &loginErr))
	require.True(t, loginErr.Data.Require2FA, "应提示 Require2FA")

	// 提供正确动态口令登录成功
	otp, err := totp.Now(secret)
	require.NoError(t, err)
	code, token = login(otp)
	require.Equal(t, http.StatusOK, code, "提供正确 otp 应登录成功")
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, "/api/v1/profile", "", token).Code)

	// 关闭 2FA（需当前有效口令）
	otp, err = totp.Now(secret)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/auth/2fa/disable",
		fmt.Sprintf(`{"code":%q}`, otp), token).Code)
	// 关闭后无需 otp 即可登录
	code, token = login("")
	require.Equal(t, http.StatusOK, code, "关闭 2FA 后应无需 otp 登录")

	// 4) API Token 生命周期
	w = app.Do(t, http.MethodPost, "/api/v1/auth/tokens", `{"name":"ci-token","expires_in":"720h"}`, token)
	require.Equal(t, http.StatusOK, w.Code)
	var created struct {
		Data struct {
			ID    string `json:"id"`
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.NotEmpty(t, created.Data.Token)

	list := app.Do(t, http.MethodGet, "/api/v1/auth/tokens", "", token)
	require.Equal(t, http.StatusOK, list.Code)
	assert.Contains(t, list.Body.String(), created.Data.ID)

	// API Token（mxat_ 前缀 Bearer）应能访问受保护接口
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodGet, "/api/v1/profile", "", "Bearer "+created.Data.Token).Code,
		"API Token 应能访问受保护接口")

	// 撤销后该 Token 失效
	require.Equal(t, http.StatusOK, app.Do(t, http.MethodPost, "/api/v1/auth/tokens/revoke",
		fmt.Sprintf(`{"id":%q}`, created.Data.ID), token).Code)
	require.Eventually(t, func() bool {
		return app.Do(t, http.MethodGet, "/api/v1/profile", "", "Bearer "+created.Data.Token).Code == http.StatusUnauthorized
	}, 2*time.Second, 20*time.Millisecond, "撤销后 API Token 应失效")

	// 5) 邮箱验证 / 找回密码：无 SMTP 配置时优雅降级为 500
	require.Equal(t, http.StatusInternalServerError, app.Do(t, http.MethodPost, "/api/v1/auth/email/send-verification",
		fmt.Sprintf(`{"email":%q}`, email), token).Code, "未配置邮件服务应 500")
	require.Equal(t, http.StatusInternalServerError, app.Do(t, http.MethodPost, "/api/v1/auth/forgot-password",
		fmt.Sprintf(`{"email":%q}`, email), "").Code, "未配置邮件服务应 500")

	// 6) 非法令牌的验证/重置应被拒（email/verify 需登录，携带 token；reset-password 为公开路由）
	require.Equal(t, http.StatusBadRequest, app.Do(t, http.MethodPost, "/api/v1/auth/email/verify",
		`{"token":"bogus"}`, token).Code)
	require.Equal(t, http.StatusBadRequest, app.Do(t, http.MethodPost, "/api/v1/auth/reset-password",
		`{"token":"bogus","new_password":"newpass123"}`, "").Code)
}
