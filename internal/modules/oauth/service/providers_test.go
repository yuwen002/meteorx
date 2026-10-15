package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProviderRegistry_AllFixedRegistered(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})
	for _, id := range []string{"google", "github", "wechat", "feishu", "dingtalk", "qq", "microsoft"} {
		_, ok := svc.registry.Get(id)
		assert.Truef(t, ok, "fixed provider %s should always be registered", id)
	}
}

func TestWeChatRedirectURL(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{
		WeChat: config.OAuthProviderConfig{Enabled: true, ClientID: "wx-id", RedirectURL: "https://e/c"},
	})
	p, ok := svc.registry.Get("wechat")
	require.True(t, ok)
	u := p.RedirectURL("st")
	assert.Contains(t, u, "open.weixin.qq.com/connect/qrconnect")
	assert.Contains(t, u, "appid=wx-id")
	assert.Contains(t, u, "scope=snsapi_login")
}

func TestFeishuRedirectURL(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{
		Feishu: config.OAuthProviderConfig{Enabled: true, ClientID: "cli", RedirectURL: "https://e/c"},
	})
	p, ok := svc.registry.Get("feishu")
	require.True(t, ok)
	u := p.RedirectURL("st")
	assert.Contains(t, u, "feishu.cn/open-apis/authen/v1/authorize")
	assert.Contains(t, u, "app_id=cli")
}

func TestDingTalkRedirectURL(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{
		DingTalk: config.OAuthProviderConfig{Enabled: true, ClientID: "dt", RedirectURL: "https://e/c"},
	})
	p, ok := svc.registry.Get("dingtalk")
	require.True(t, ok)
	u := p.RedirectURL("st")
	assert.Contains(t, u, "login.dingtalk.com/oauth2/auth")
	assert.Contains(t, u, "client_id=dt")
	assert.Contains(t, u, "scope=openid")
}

func TestQQRedirectURL(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{
		QQ: config.OAuthProviderConfig{Enabled: true, ClientID: "qq-id", RedirectURL: "https://e/c"},
	})
	p, ok := svc.registry.Get("qq")
	require.True(t, ok)
	u := p.RedirectURL("st")
	assert.Contains(t, u, "graph.qq.com/oauth2.0/authorize")
	assert.Contains(t, u, "client_id=qq-id")
}

func TestMicrosoftRedirectURL(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{
		Microsoft: config.MicrosoftOAuthConfig{Enabled: true, ClientID: "ms", ClientSecret: "s", Tenant: "my-tenant", RedirectURL: "https://e/c"},
	})
	p, ok := svc.registry.Get("microsoft")
	require.True(t, ok)
	u := p.RedirectURL("st")
	assert.Contains(t, u, "login.microsoftonline.com/my-tenant/oauth2/v2.0/authorize")
}

func TestExtractQQOpenID(t *testing.T) {
	id, err := extractQQOpenID(`callback( {"client_id":"xxx","openid":"OPENID123"} );`)
	require.NoError(t, err)
	assert.Equal(t, "OPENID123", id)

	_, err = extractQQOpenID(`garbage`)
	assert.Error(t, err)
}

func TestStandardUserInfo(t *testing.T) {
	body := []byte(`{"id":"42","email":"a@b.com","name":"A","picture":"http://p"}`)
	id, email, name, avatar, err := standardUserInfo(body)
	require.NoError(t, err)
	assert.Equal(t, "42", id)
	assert.Equal(t, "a@b.com", email)
	assert.Equal(t, "A", name)
	assert.Equal(t, "http://p", avatar)

	// OIDC sub 优先于 id
	body = []byte(`{"sub":"sub-1","email":"c@d.com"}`)
	id, email, _, _, err = standardUserInfo(body)
	require.NoError(t, err)
	assert.Equal(t, "sub-1", id)
	assert.Equal(t, "c@d.com", email)
}

// TestStandardExchange_ErrorNotLeakBody 令牌端点返回非 JSON（含有效令牌）时，
// 对外错误必须是语义化哨兵错误，不得将上游响应体（可能含 access_token）拼入错误。
func TestStandardExchange_ErrorNotLeakBody(t *testing.T) {
	const leakedToken = "SECRET_ACCESS_TOKEN_SHOULD_NOT_LEAK"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟未按预期返回 JSON，而是 urlencoded 含令牌
		w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
		_, _ = w.Write([]byte("access_token=" + leakedToken + "&scope=read"))
	}))
	defer srv.Close()

	p := &standardOAuthProvider{
		name:         "test",
		enabled:      true,
		clientID:     "cid",
		clientSecret: "secret",
		redirectURL:  "https://example.com/cb",
		tokenURL:     srv.URL,
		userInfoURL:  srv.URL,
		scopes:       []string{"openid"},
	}

	_, err := p.Exchange(context.TODO(), "code123")
	require.Error(t, err)
	// 必须回归到哨兵错误，且错误文本不回显上游原始令牌
	assert.ErrorIs(t, err, ErrOAuthExchangeFailed)
	assert.NotContains(t, err.Error(), leakedToken, "上游响应体不得被拼入对外错误")
}

func TestListProviders_Sorted(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{
		Google: config.OAuthProviderConfig{Enabled: true, ClientID: "g", ClientSecret: "s", RedirectURL: "https://e/c"},
		WeChat: config.OAuthProviderConfig{Enabled: true, ClientID: "w", ClientSecret: "s", RedirectURL: "https://e/c"},
	})
	metas := svc.ListProviders(context.TODO())
	require.Len(t, metas, 2)
	// 按 ID 升序：google 在 wechat 前
	assert.Equal(t, "google", metas[0].ID)
	assert.Equal(t, "wechat", metas[1].ID)
	assert.Equal(t, "Google", metas[0].Label)
	assert.Equal(t, "微信", metas[1].Label)
}
