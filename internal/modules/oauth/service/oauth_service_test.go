package service

import (
	"context"
	"testing"

	"meteorx/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestOAuthService(cfg config.OAuthConfig) *OAuthService {
	return NewOAuthService(cfg, nil, nil, nil, nil, nil, nil, nil, nil)
}

func TestGetRedirectURL_UnsupportedProvider(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})

	_, _, err := svc.GetRedirectURL(context.TODO(), "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provider")
}

func TestGetRedirectURL_GoogleDisabled(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})

	_, _, err := svc.GetRedirectURL(context.TODO(), "google")
	assert.Error(t, err)
	assert.Equal(t, ErrOAuthProviderDisabled, err)
}

func TestGetRedirectURL_GitHubDisabled(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})

	_, _, err := svc.GetRedirectURL(context.TODO(), "github")
	assert.Error(t, err)
	assert.Equal(t, ErrOAuthProviderDisabled, err)
}

func TestGetRedirectURL_GoogleEnabled(t *testing.T) {
	cfg := config.OAuthConfig{
		Google: config.OAuthProviderConfig{
			Enabled:      true,
			ClientID:     "google-client-id",
			ClientSecret: "secret",
			RedirectURL:  "https://example.com/oauth/google/callback",
		},
	}
	svc := newTestOAuthService(cfg)

	url, state, err := svc.GetRedirectURL(context.TODO(), "google")
	assert.NoError(t, err)
	assert.Contains(t, url, "accounts.google.com")
	assert.Contains(t, url, "google-client-id")
	assert.Contains(t, url, "openid+email+profile")
	assert.NotEmpty(t, state, "state should be generated")
	assert.Contains(t, url, "state=", "state param should be in redirect URL")
}

func TestGetRedirectURL_GitHubEnabled(t *testing.T) {
	cfg := config.OAuthConfig{
		GitHub: config.OAuthProviderConfig{
			Enabled:      true,
			ClientID:     "github-client-id",
			ClientSecret: "secret",
			RedirectURL:  "https://example.com/oauth/github/callback",
		},
	}
	svc := newTestOAuthService(cfg)

	url, state, err := svc.GetRedirectURL(context.TODO(), "github")
	assert.NoError(t, err)
	assert.Contains(t, url, "github.com")
	assert.Contains(t, url, "github-client-id")
	assert.NotEmpty(t, state, "state should be generated")
	assert.Contains(t, url, "state=", "state param should be in redirect URL")
}

func TestGoogleRedirectURL_ContainsState(t *testing.T) {
	cfg := config.OAuthConfig{
		Google: config.OAuthProviderConfig{
			ClientID:    "test-client-id",
			RedirectURL: "https://example.com/callback",
		},
	}
	svc := newTestOAuthService(cfg)

	p, ok := svc.registry.Get("google")
	require.True(t, ok)
	url := p.RedirectURL("test-state-value")
	assert.Contains(t, url, "https://accounts.google.com/o/oauth2/v2/auth")
	assert.Contains(t, url, "client_id=test-client-id")
	assert.Contains(t, url, "response_type=code")
	assert.Contains(t, url, "scope=openid+email+profile")
	assert.Contains(t, url, "state=test-state-value")
}

func TestGitHubRedirectURL_ContainsState(t *testing.T) {
	cfg := config.OAuthConfig{
		GitHub: config.OAuthProviderConfig{
			ClientID:    "gh-client-id",
			RedirectURL: "https://example.com/github/callback",
		},
	}
	svc := newTestOAuthService(cfg)

	p, ok := svc.registry.Get("github")
	require.True(t, ok)
	url := p.RedirectURL("test-state-value")
	assert.Contains(t, url, "https://github.com/login/oauth/authorize")
	assert.Contains(t, url, "client_id=gh-client-id")
	assert.Contains(t, url, "read%3Auser+user%3Aemail")
	assert.Contains(t, url, "state=test-state-value")
}

func TestListProviders_OnlyEnabled(t *testing.T) {
	cfg := config.OAuthConfig{
		Google: config.OAuthProviderConfig{Enabled: true, ClientID: "g", ClientSecret: "s", RedirectURL: "https://e/c"},
		GitHub: config.OAuthProviderConfig{Enabled: false, ClientID: "gh", ClientSecret: "s", RedirectURL: "https://e/c"},
		WeChat: config.OAuthProviderConfig{Enabled: true, ClientID: "wx", ClientSecret: "s", RedirectURL: "https://e/c"},
	}
	svc := newTestOAuthService(cfg)
	metas := svc.ListProviders(context.TODO())
	ids := make([]string, 0, len(metas))
	for _, m := range metas {
		ids = append(ids, m.ID)
	}
	assert.Contains(t, ids, "google")
	assert.Contains(t, ids, "wechat")
	assert.NotContains(t, ids, "github", "disabled provider should not be listed")
}

func TestNewProviderRegistry_OIDC(t *testing.T) {
	cfg := config.OAuthConfig{
		OIDC: map[string]config.OIDCProviderConfig{
			"acme": {
				Enabled:      true,
				ClientID:     "acme-id",
				ClientSecret: "acme-secret",
				RedirectURL:  "https://e/c",
				AuthURL:      "https://idp/acme/authorize",
				TokenURL:     "https://idp/acme/token",
				UserInfoURL:  "https://idp/acme/userinfo",
			},
		},
	}
	svc := newTestOAuthService(cfg)
	p, ok := svc.registry.Get("oidc_acme")
	require.True(t, ok)
	assert.Equal(t, "oidc_acme", p.Name())
	assert.True(t, p.IsEnabled())
	assert.Contains(t, p.RedirectURL("st"), "idp/acme/authorize")
}

func TestLogin_UnsupportedProvider(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})

	_, _, _, _, _, _, err := svc.Login(context.TODO(), "unknown", "code123", "state123", "tenant-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provider")
}

// TestLogin_DisabledProviderRejected 未启用的 provider 不得发起出站交换，直接拒绝。
func TestLogin_DisabledProviderRejected(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})

	_, _, _, _, _, _, err := svc.Login(context.TODO(), "google", "code123", "state123", "tenant-id")
	assert.Equal(t, ErrOAuthProviderDisabled, err)
}

// TestBindOAuth_DisabledProviderRejected 未启用的 provider 不得绑定。
func TestBindOAuth_DisabledProviderRejected(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})

	err := svc.BindOAuth(context.TODO(), "user-1", "github", "code123", "state123")
	assert.Equal(t, ErrOAuthProviderDisabled, err)
}

func TestNewOAuthService(t *testing.T) {
	cfg := config.OAuthConfig{
		Google: config.OAuthProviderConfig{Enabled: true},
		GitHub: config.OAuthProviderConfig{Enabled: false},
	}
	svc := NewOAuthService(cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.cfg)
}

func TestGenerateState(t *testing.T) {
	state1, err := generateState()
	assert.NoError(t, err)
	assert.NotEmpty(t, state1)

	state2, err := generateState()
	assert.NoError(t, err)
	assert.NotEmpty(t, state2)

	assert.NotEqual(t, state1, state2, "each state should be unique")
}

func TestValidateAndConsumeState_NoRedis(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})
	err := svc.validateAndConsumeState(context.TODO(), "any-state")
	assert.NoError(t, err, "without Redis, state validation should be skipped")
}

func TestSaveState_NoRedis(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})
	err := svc.saveState(context.TODO(), "any-state")
	assert.NoError(t, err, "without Redis, state saving should be skipped")
}
