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

// dingTalkProvider 钉钉 OAuth2 适配（新版 OAuth2.0 授权登录）。
type dingTalkProvider struct {
	name         string
	enabled      bool
	clientID     string
	clientSecret string
	redirectURL  string
}

func newDingTalkProvider(cred providerCred) *dingTalkProvider {
	return &dingTalkProvider{
		name:         cred.code,
		enabled:      cred.enabled,
		clientID:     cred.clientID,
		clientSecret: cred.clientSecret,
		redirectURL:  cred.redirectURL,
	}
}

func (p *dingTalkProvider) Name() string    { return model.ProviderDingTalk }
func (p *dingTalkProvider) IsEnabled() bool { return p.enabled && p.clientID != "" }

func (p *dingTalkProvider) RedirectURL(state string) string {
	q := url.Values{}
	q.Set("client_id", p.clientID)
	q.Set("response_type", "code")
	q.Set("scope", "openid")
	q.Set("redirect_uri", p.redirectURL)
	q.Set("state", state)
	q.Set("prompt", "consent")
	return "https://login.dingtalk.com/oauth2/auth?" + q.Encode()
}

func (p *dingTalkProvider) Exchange(ctx context.Context, code string) (*dto.OAuthUserInfo, error) {
	tokenBody, _ := json.Marshal(map[string]string{
		"clientId":     p.clientID,
		"clientSecret": p.clientSecret,
		"code":         code,
		"grantType":    "authorization_code",
	})
	raw, err := postJSON(ctx, "https://api.dingtalk.com/v1.0/oauth2/accessToken", tokenBody, "")
	if err != nil {
		return nil, ErrOAuthExchangeFailed
	}
	var tr struct {
		AccessToken string `json:"accessToken"`
		ErrMsg      string `json:"errMsg"`
		ErrCode     int    `json:"code"`
	}
	if err := json.Unmarshal(raw, &tr); err != nil || tr.AccessToken == "" {
		// 响应体可能含有效令牌，只写服务端日志，对外仅返回语义化错误
		logger.Warnf("[oauth] dingtalk token exchange failed: %s", safeBodyPreview(raw))
		return nil, ErrOAuthExchangeFailed
	}

	raw, err = postJSON(ctx, "https://api.dingtalk.com/v1.0/oauth2/userInfo", nil, tr.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUserInfoFailed, err)
	}
	var ui struct {
		Nickname  string `json:"nickname"`
		OpenID    string `json:"openid"`
		UnionID   string `json:"unionid"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatarUrl"`
	}
	if err := json.Unmarshal(raw, &ui); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUserInfoFailed, err)
	}

	providerID := ui.UnionID
	if providerID == "" {
		providerID = ui.OpenID
	}
	if providerID == "" {
		return nil, fmt.Errorf("%w: missing openid/unionid in userinfo", ErrOAuthUserInfoFailed)
	}

	return &dto.OAuthUserInfo{
		Provider:    model.ProviderDingTalk,
		ProviderID:  providerID,
		Email:       ui.Email,
		Name:        ui.Nickname,
		AvatarURL:   ui.AvatarURL,
		AccessToken: tr.AccessToken,
	}, nil
}
