package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/jwt"
	"meteorx/internal/config"
	"meteorx/internal/modules/oauth/handler"
	oauthRepo "meteorx/internal/modules/oauth/repository"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	tenantModel "meteorx/internal/modules/tenant/model"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	userRepo "meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const integSecretKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=" // base64(32 bytes)

// fakeChecker 模拟「平台超级管理员」：任何权限码都放行。
type fakeChecker struct{}

func (fakeChecker) GetUserPermissionCodes(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

// withSuperadmin 在请求上下文注入 superadmin 角色，使 AutoRequirePermission 走超管旁路。
func withSuperadmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := contextx.SetVars(r.Context(), "t-1", "u-admin", []string{"superadmin"})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newIntegDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, userRepo.AutoMigrate(db))
	require.NoError(t, tenantRepo.AutoMigrate(db))
	require.NoError(t, rbacRepo.AutoMigrate(db))
	require.NoError(t, oauthRepo.Migrate(db))

	// 登录闭环需要一个可用租户
	require.NoError(t, tenantRepo.NewTenantRepository(db).Create(context.Background(), &tenantModel.Tenant{
		ID: "t-1", Name: "默认租户", Status: 1,
	}))

	t.Cleanup(func() {
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// newIntegApp 用真实仓储 + 真实加解密 + 真实 JWT 装配一个 OAuth 应用：
// 同一个 *OAuthService 实例同时挂载「公开路由」与「管理路由」，共享同一份注册表，
// 以此验证后台写入后公开侧（登录页/登录流程）能即时反映（热重载）。
func newIntegApp(t *testing.T) (*chi.Mux, *gorm.DB, *jwt.TokenHelper, oauthRepo.OAuthProviderRepository) {
	t.Helper()
	db := newIntegDB(t)

	uRepo := userRepo.NewUserRepository(db)
	rRepo := rbacRepo.NewRoleRepository(db)
	urRepo := rbacRepo.NewUserRoleRepository(db)
	rpRepo := rbacRepo.NewRolePermissionRepository(db)
	tRepo := tenantRepo.NewTenantRepository(db)
	accRepo := oauthRepo.NewOAuthAccountRepository(db)
	providerRepo := oauthRepo.NewOAuthProviderRepository(db)

	tokenHelper := jwt.NewTokenHelper(config.JWTConfig{Secret: "test-jwt-secret-which-is-long-enough-32"})

	// 使用包内 newOAuthService（含 providerRepo + cipher），公开侧与绑定侧均会定时 reload
	svc := newOAuthService(config.OAuthConfig{SecretKey: integSecretKey}, uRepo, rRepo, urRepo, rpRepo, tRepo, accRepo, providerRepo, tokenHelper, nil)

	// 启动与线上一致的注册表刷新（公开侧 + 绑定侧），但本测试在写入后同步调用 ReloadProviders 立即生效
	startProviderReloadLoop(context.Background(), svc)

	r := chi.NewRouter()
	r.Use(withSuperadmin)
	r.Route("/api/v1", func(r chi.Router) {
		RegisterRoutes(r, handler.NewOAuthHandler(svc))
		RegisterAdminRoutes(r, handler.NewOAuthProviderAdminHandler(svc), fakeChecker{})
	})

	return r, db, tokenHelper, providerRepo
}

func integDo(t *testing.T, r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func integDecodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(w.Body)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m), "response body: %s", string(raw))
	return m
}

// TestInteg_AdminCRUDAndHotReload 真实 HTTP 走通后台 CRUD，并验证写入后公开登录侧即时反映。
func TestInteg_AdminCRUDAndHotReload(t *testing.T) {
	r, _, _, providerRepo := newIntegApp(t)

	// 1) 新建微信配置（含 client_secret）
	w := integDo(t, r, http.MethodPost, "/api/v1/admin/oauth-providers", `{
		"kind":"wechat","code":"wechat","display_name":"微信","enabled":true,
		"client_id":"wx-app-id","client_secret":"alpha-secret-OLD","redirect_url":"http://localhost:5173/auth/wechat/callback"
	}`)
	require.Equal(t, http.StatusOK, w.Code, "create wechat: %s", w.Body.String())

	// 落库为密文，绝不明文
	cnt, err := providerRepo.Count(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt)
	stored, err := providerRepo.GetByCode(context.Background(), "wechat")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(stored.ClientSecret, crypto.CipherPrefix))
	assert.NotContains(t, stored.ClientSecret, "alpha-secret-OLD")

	// 2) 后台列表：secret 仅脱敏返回
	w = integDo(t, r, http.MethodGet, "/api/v1/admin/oauth-providers", "")
	require.Equal(t, http.StatusOK, w.Code)
	body := integDecodeBody(t, w)
	providers := body["data"].(map[string]any)["providers"].([]any)
	require.Len(t, providers, 1)
	p0 := providers[0].(map[string]any)
	assert.Equal(t, "wechat", p0["code"])
	assert.True(t, p0["has_secret"].(bool))
	mask := p0["client_secret_mask"].(string)
	assert.True(t, strings.HasSuffix(mask, "OLD"), "脱敏应保留末尾 4 位")
	assert.NotContains(t, mask, "alpha-secret-OLD")

	// 3) 热重载：公开登录侧（GET /auth/oauth/providers）应已包含 wechat（共享注册表）
	w = integDo(t, r, http.MethodGet, "/api/v1/auth/oauth/providers", "")
	require.Equal(t, http.StatusOK, w.Code)
	pub := integDecodeBody(t, w)
	pubProviders := pub["data"].(map[string]any)["providers"].([]any)
	assert.Equal(t, []any{map[string]any{"id": "wechat", "label": "微信"}}, castProviderMetas(t, pubProviders))

	// 4) 停用后公开侧不再出现
	w = integDo(t, r, http.MethodPut, "/api/v1/admin/oauth-providers/wechat/status", `{"enabled":false}`)
	require.Equal(t, http.StatusOK, w.Code)
	w = integDo(t, r, http.MethodGet, "/api/v1/auth/oauth/providers", "")
	pub = integDecodeBody(t, w)
	pubProviders = pub["data"].(map[string]any)["providers"].([]any)
	assert.Empty(t, pubProviders, "停用后登录侧应移除该 provider")

	// 5) 更新并轮换 client_secret（留空保持 / 填新值替换）
	w = integDo(t, r, http.MethodPut, "/api/v1/admin/oauth-providers/wechat", `{
		"kind":"wechat","code":"wechat","client_id":"wx-app-id","client_secret":"beta-secret-NEW","redirect_url":"http://localhost:5173/auth/wechat/callback"
	}`)
	require.Equal(t, http.StatusOK, w.Code)
	w = integDo(t, r, http.MethodGet, "/api/v1/admin/oauth-providers", "")
	p0 = integDecodeBody(t, w)["data"].(map[string]any)["providers"].([]any)[0].(map[string]any)
	assert.True(t, strings.HasSuffix(p0["client_secret_mask"].(string), "NEW"), "应已轮换为 NEW")

	// 6) 删除后彻底消失
	w = integDo(t, r, http.MethodDelete, "/api/v1/admin/oauth-providers/wechat", "")
	require.Equal(t, http.StatusOK, w.Code)
	w = integDo(t, r, http.MethodGet, "/api/v1/admin/oauth-providers", "")
	providers = integDecodeBody(t, w)["data"].(map[string]any)["providers"].([]any)
	assert.Empty(t, providers)
}

func castProviderMetas(t *testing.T, in []any) []any {
	t.Helper()
	out := make([]any, 0, len(in))
	for _, item := range in {
		m := item.(map[string]any)
		out = append(out, map[string]any{"id": m["id"], "label": m["label"]})
	}
	return out
}

// mockIdP 起一个伪 OAuth2 提供方：authorize/token/userinfo。
// returnEmail=false 时 userinfo 不含 email，用于验证 NeedCompleteEmail 分支。
func mockIdP(t *testing.T, returnEmail bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/token"):
			_, _ = w.Write([]byte(`{"access_token":"mock-access-token","token_type":"Bearer"}`))
		case strings.HasPrefix(r.URL.Path, "/userinfo"):
			if returnEmail {
				_, _ = w.Write([]byte(`{"sub":"mock-user-123","email":"mock@example.com","name":"Mock User"}`))
			} else {
				_, _ = w.Write([]byte(`{"sub":"mock-user-123","name":"Mock User"}`))
			}
		default:
			_, _ = w.Write([]byte(`<html>authorize</html>`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestInteg_MockIdPLoginFlow 用伪 IdP 真实跑通「后台配置 → 获取跳转地址 → 回调换 token → 用户落库/登录」。
func TestInteg_MockIdPLoginFlow(t *testing.T) {
	r, db, tokenHelper, _ := newIntegApp(t)

	// 含邮箱的 IdP：登录后不应要求补全邮箱
	withEmail := mockIdP(t, true)
	// 不含邮箱的 IdP：登录后应要求补全邮箱
	noEmail := mockIdP(t, false)

	// 1) 后台配置两个 OIDC provider，指向各自伪 IdP
	createOIDCMock(t, r, "oidc_mock", withEmail.URL)
	createOIDCMock(t, r, "oidc_noemail", noEmail.URL)

	// 2) 公开登录侧应已热重载出现这两个 provider
	w := integDo(t, r, http.MethodGet, "/api/v1/auth/oauth/providers", "")
	pubProviders := integDecodeBody(t, w)["data"].(map[string]any)["providers"].([]any)
	ids := make([]string, 0, len(pubProviders))
	for _, p := range pubProviders {
		ids = append(ids, p.(map[string]any)["id"].(string))
	}
	assert.ElementsMatch(t, []string{"oidc_mock", "oidc_noemail"}, ids)

	// 3) 获取跳转地址：应指向伪 IdP 的 authorize 端点
	w = integDo(t, r, http.MethodGet, "/api/v1/auth/oauth/oidc_mock/redirect", "")
	require.Equal(t, http.StatusOK, w.Code)
	redirectURL := integDecodeBody(t, w)["data"].(map[string]any)["url"].(string)
	assert.Contains(t, redirectURL, strings.TrimPrefix(withEmail.URL, "http://"))

	// 4) 回调换 token（含邮箱）：200 + 真实 JWT + 无需补全邮箱
	w = integDo(t, r, http.MethodPost, "/api/v1/auth/oauth/callback", `{
		"provider":"oidc_mock","code":"mock-code","state":"any-state","tenant_id":"t-1"
	}`)
	require.Equal(t, http.StatusOK, w.Code, "callback with email: %s", w.Body.String())
	login := integDecodeBody(t, w)["data"].(map[string]any)
	token := login["token"].(string)
	assert.NotEmpty(t, token)
	assert.False(t, login["need_complete_email"].(bool))

	// 解析 JWT，确认是真实签发的令牌且用户身份正确
	claims, err := tokenHelper.ParseToken(token)
	require.NoError(t, err)
	assert.NotEmpty(t, claims.UserID)

	// 5) 绑定关系应已落库（oauth_accounts）
	accRepo := oauthRepo.NewOAuthAccountRepository(db)
	acc, err := accRepo.GetByProviderAndProviderID(context.Background(), "oidc_mock", "mock-user-123")
	require.NoError(t, err)
	require.NotNil(t, acc)
	assert.Equal(t, claims.UserID, acc.UserID)

	// 6) 回调换 token（无邮箱）：200 + 需要补全邮箱
	w = integDo(t, r, http.MethodPost, "/api/v1/auth/oauth/callback", `{
		"provider":"oidc_noemail","code":"mock-code","state":"any-state","tenant_id":"t-1"
	}`)
	require.Equal(t, http.StatusOK, w.Code, "callback no email: %s", w.Body.String())
	login = integDecodeBody(t, w)["data"].(map[string]any)
	assert.True(t, login["need_complete_email"].(bool))
	assert.NotEmpty(t, login["token"].(string)) // 返回临时令牌，供补全邮箱使用

	// 7) 验证补全邮箱后换发正式令牌（端到端闭环）：complete-email 需 Authorization 头携带临时令牌
	tempToken := login["token"].(string)
	w = doCompleteEmail(t, r, tempToken, "newuser@example.com")
	require.Equal(t, http.StatusOK, w.Code, "complete email: %s", w.Body.String())
	finalToken := integDecodeBody(t, w)["data"].(map[string]any)["token"].(string)
	assert.NotEmpty(t, finalToken)
	finalClaims, err := tokenHelper.ParseToken(finalToken)
	require.NoError(t, err)
	assert.False(t, finalClaims.EmailPending, "补全邮箱后不应仍带 email_pending 声明")
}

func createOIDCMock(t *testing.T, r http.Handler, code, mockURL string) {
	t.Helper()
	body := `{
		"kind":"oidc","code":"` + code + `","display_name":"Mock","enabled":true,
		"client_id":"mock-client","client_secret":"mock-secret",
		"redirect_url":"http://localhost:5173/auth/` + code + `/callback",
		"auth_url":"` + mockURL + `/authorize",
		"token_url":"` + mockURL + `/token",
		"userinfo_url":"` + mockURL + `/userinfo",
		"scopes":["openid","email"]
	}`
	w := integDo(t, r, http.MethodPost, "/api/v1/admin/oauth-providers", body)
	require.Equal(t, http.StatusOK, w.Code, "create %s: %s", code, w.Body.String())
	// 写入后同实例同步 reload，使公开侧立即生效
}

func doCompleteEmail(t *testing.T, r http.Handler, bearer, email string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/complete-email",
		bytes.NewBufferString(`{"email":"`+email+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
