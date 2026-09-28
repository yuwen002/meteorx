package service

import (
	"testing"

	"meteorx/internal/config"

	"github.com/stretchr/testify/assert"
)

func newTestOAuthService(cfg config.OAuthConfig) *OAuthService {
	return NewOAuthService(cfg, nil, nil, nil, nil, nil, nil, nil, nil)
}

func TestGetRedirectURL_UnsupportedProvider(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})

	_, _, err := svc.GetRedirectURL(nil, "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provider")
}

func TestGetRedirectURL_GoogleDisabled(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})

	_, _, err := svc.GetRedirectURL(nil, "google")
	assert.Error(t, err)
	assert.Equal(t, ErrOAuthProviderDisabled, err)
}

func TestGetRedirectURL_GitHubDisabled(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})

	_, _, err := svc.GetRedirectURL(nil, "github")
	assert.Error(t, err)
	assert.Equal(t, ErrOAuthProviderDisabled, err)
}

func TestGetRedirectURL_GoogleEnabled(t *testing.T) {
	cfg := config.OAuthConfig{
		Google: config.OAuthProviderConfig{
			Enabled:      true,
			ClientID:    "google-client-id",
			ClientSecret: "secret",
			RedirectURL: "https://example.com/oauth/google/callback",
		},
	}
	svc := newTestOAuthService(cfg)

	url, state, err := svc.GetRedirectURL(nil, "google")
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
			ClientID:    "github-client-id",
			ClientSecret: "secret",
			RedirectURL: "https://example.com/oauth/github/callback",
		},
	}
	svc := newTestOAuthService(cfg)

	url, state, err := svc.GetRedirectURL(nil, "github")
	assert.NoError(t, err)
	assert.Contains(t, url, "github.com")
	assert.Contains(t, url, "github-client-id")
	assert.NotEmpty(t, state, "state should be generated")
	assert.Contains(t, url, "state=", "state param should be in redirect URL")
}

func TestBuildGoogleRedirectURL_ContainsState(t *testing.T) {
	cfg := config.OAuthConfig{
		Google: config.OAuthProviderConfig{
			ClientID:    "test-client-id",
			RedirectURL: "https://example.com/callback",
		},
	}
	svc := newTestOAuthService(cfg)

	state := "test-state-value"
	url := svc.buildGoogleRedirectURL(state)
	assert.Contains(t, url, "https://accounts.google.com/o/oauth2/v2/auth")
	assert.Contains(t, url, "client_id=test-client-id")
	assert.Contains(t, url, "response_type=code")
	assert.Contains(t, url, "scope=openid+email+profile")
	assert.Contains(t, url, "access_type=online")
	assert.Contains(t, url, "prompt=select_account")
	assert.Contains(t, url, "state=test-state-value")
}

func TestBuildGitHubRedirectURL_ContainsState(t *testing.T) {
	cfg := config.OAuthConfig{
		GitHub: config.OAuthProviderConfig{
			ClientID:    "gh-client-id",
			RedirectURL: "https://example.com/github/callback",
		},
	}
	svc := newTestOAuthService(cfg)

	state := "test-state-value"
	url := svc.buildGitHubRedirectURL(state)
	assert.Contains(t, url, "https://github.com/login/oauth/authorize")
	assert.Contains(t, url, "client_id=gh-client-id")
	assert.Contains(t, url, "read%3Auser+user%3Aemail")
	assert.Contains(t, url, "state=test-state-value")
}

func TestLogin_UnsupportedProvider(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})

	_, _, _, _, _, _, err := svc.Login(nil, "unknown", "code123", "state123", "tenant-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provider")
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
	err := svc.validateAndConsumeState(nil, "any-state")
	assert.NoError(t, err, "without Redis, state validation should be skipped")
}

func TestSaveState_NoRedis(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})
	err := svc.saveState(nil, "any-state")
	assert.NoError(t, err, "without Redis, state saving should be skipped")
}