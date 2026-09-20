package service

import (
	"testing"

	"meteorx/internal/config"

	"github.com/stretchr/testify/assert"
)

func newTestOAuthService(cfg config.OAuthConfig) *OAuthService {
	return NewOAuthService(cfg, nil, nil, nil, nil, nil, nil)
}

func TestGetRedirectURL_UnsupportedProvider(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})

	_, err := svc.GetRedirectURL("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provider")
}

func TestGetRedirectURL_GoogleDisabled(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})

	_, err := svc.GetRedirectURL("google")
	assert.Error(t, err)
	assert.Equal(t, ErrOAuthProviderDisabled, err)
}

func TestGetRedirectURL_GitHubDisabled(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})

	_, err := svc.GetRedirectURL("github")
	assert.Error(t, err)
	assert.Equal(t, ErrOAuthProviderDisabled, err)
}

func TestGetRedirectURL_GoogleEnabled(t *testing.T) {
	cfg := config.OAuthConfig{
		Google: config.OAuthProviderConfig{
			Enabled:     true,
			ClientID:    "google-client-id",
			ClientSecret: "secret",
			RedirectURL: "https://example.com/oauth/google/callback",
		},
	}
	svc := newTestOAuthService(cfg)

	url, err := svc.GetRedirectURL("google")
	assert.NoError(t, err)
	assert.Contains(t, url, "accounts.google.com")
	assert.Contains(t, url, "google-client-id")
	assert.Contains(t, url, "openid+email+profile")
}

func TestGetRedirectURL_GitHubEnabled(t *testing.T) {
	cfg := config.OAuthConfig{
		GitHub: config.OAuthProviderConfig{
			Enabled:     true,
			ClientID:    "github-client-id",
			ClientSecret: "secret",
			RedirectURL: "https://example.com/oauth/github/callback",
		},
	}
	svc := newTestOAuthService(cfg)

	url, err := svc.GetRedirectURL("github")
	assert.NoError(t, err)
	assert.Contains(t, url, "github.com")
	assert.Contains(t, url, "github-client-id")
}

func TestBuildGoogleRedirectURL(t *testing.T) {
	cfg := config.OAuthConfig{
		Google: config.OAuthProviderConfig{
			ClientID:    "test-client-id",
			RedirectURL: "https://example.com/callback",
		},
	}
	svc := newTestOAuthService(cfg)

	url := svc.buildGoogleRedirectURL()
	assert.Contains(t, url, "https://accounts.google.com/o/oauth2/v2/auth")
	assert.Contains(t, url, "client_id=test-client-id")
	assert.Contains(t, url, "response_type=code")
	assert.Contains(t, url, "scope=openid+email+profile")
	assert.Contains(t, url, "access_type=online")
	assert.Contains(t, url, "prompt=select_account")
}

func TestBuildGitHubRedirectURL(t *testing.T) {
	cfg := config.OAuthConfig{
		GitHub: config.OAuthProviderConfig{
			ClientID:    "gh-client-id",
			RedirectURL: "https://example.com/github/callback",
		},
	}
	svc := newTestOAuthService(cfg)

	url := svc.buildGitHubRedirectURL()
	assert.Contains(t, url, "https://github.com/login/oauth/authorize")
	assert.Contains(t, url, "client_id=gh-client-id")
	assert.Contains(t, url, "read%3Auser+user%3Aemail")
}

func TestLogin_UnsupportedProvider(t *testing.T) {
	svc := newTestOAuthService(config.OAuthConfig{})

	_, _, _, _, _, err := svc.Login(nil, "unknown", "code123", "tenant-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provider")
}

func TestNewOAuthService(t *testing.T) {
	cfg := config.OAuthConfig{
		Google: config.OAuthProviderConfig{Enabled: true},
		GitHub: config.OAuthProviderConfig{Enabled: false},
	}
	svc := NewOAuthService(cfg, nil, nil, nil, nil, nil, nil)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.cfg)
}