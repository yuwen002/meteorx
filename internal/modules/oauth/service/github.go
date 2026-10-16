package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"meteorx/internal/modules/oauth/dto"
	"meteorx/internal/modules/oauth/model"
	"meteorx/pkg/logger"
)

// githubProvider GitHub OAuth2 适配（令牌端点需 Accept: application/json，邮箱需额外查询）。
type githubProvider struct {
	name         string
	enabled      bool
	clientID     string
	clientSecret string
	redirectURL  string
}

func newGitHubProvider(cred providerCred) *githubProvider {
	return &githubProvider{
		name:         cred.code,
		enabled:      cred.enabled,
		clientID:     cred.clientID,
		clientSecret: cred.clientSecret,
		redirectURL:  cred.redirectURL,
	}
}

func (p *githubProvider) Name() string    { return model.ProviderGitHub }
func (p *githubProvider) IsEnabled() bool { return p.enabled && p.clientID != "" }

func (p *githubProvider) RedirectURL(state string) string {
	q := url.Values{}
	q.Set("client_id", p.clientID)
	q.Set("redirect_uri", p.redirectURL)
	q.Set("scope", "read:user user:email")
	q.Set("state", state)
	return "https://github.com/login/oauth/authorize?" + q.Encode()
}

func (p *githubProvider) Exchange(ctx context.Context, code string) (*dto.OAuthUserInfo, error) {
	v := url.Values{}
	v.Set("code", code)
	v.Set("client_id", p.clientID)
	v.Set("client_secret", p.clientSecret)
	v.Set("redirect_uri", p.redirectURL)

	body, err := httpPostForm(ctx, "https://github.com/login/oauth/access_token", v, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return nil, ErrOAuthExchangeFailed
	}
	var tr struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tr); err != nil || tr.AccessToken == "" {
		// 响应体可能含有效令牌，只写服务端日志，对外仅返回语义化错误
		logger.Warnf("[oauth] github token exchange failed: %s", safeBodyPreview(body))
		return nil, ErrOAuthExchangeFailed
	}

	ub, err := httpGetJSON(ctx, "https://api.github.com/user", tr.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUserInfoFailed, err)
	}
	var gu struct {
		ID     int    `json:"id"`
		Email  string `json:"email"`
		Name   string `json:"name"`
		Login  string `json:"login"`
		Avatar string `json:"avatar_url"`
	}
	if err := json.Unmarshal(ub, &gu); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUserInfoFailed, err)
	}
	// id 缺失时 %d 会退化为 "0"，所有异常账号会撞同一 ProviderID
	if gu.ID == 0 {
		return nil, fmt.Errorf("%w: missing user id in userinfo", ErrOAuthUserInfoFailed)
	}

	email := gu.Email
	if email == "" {
		email = fetchGitHubPrimaryEmail(ctx, tr.AccessToken)
	}
	name := gu.Name
	if name == "" {
		name = gu.Login
	}

	return &dto.OAuthUserInfo{
		Provider:    model.ProviderGitHub,
		ProviderID:  fmt.Sprintf("%d", gu.ID),
		Email:       email,
		Name:        name,
		AvatarURL:   gu.Avatar,
		AccessToken: tr.AccessToken,
	}, nil
}

// fetchGitHubPrimaryEmail 查询 GitHub 主邮箱（部分账号 user 接口不返回 email）。
func fetchGitHubPrimaryEmail(ctx context.Context, accessToken string) string {
	body, err := httpGetJSON(ctx, "https://api.github.com/user/emails", accessToken)
	if err != nil {
		return ""
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return ""
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	if len(emails) > 0 {
		return emails[0].Email
	}
	return ""
}
