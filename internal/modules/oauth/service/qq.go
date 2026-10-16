package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"meteorx/internal/modules/oauth/dto"
	"meteorx/internal/modules/oauth/model"
	"meteorx/pkg/logger"
)

// qqProvider QQ 互联 OAuth2 适配。
// 注意：QQ 令牌端点返回 urlencoded（非 JSON），openid 端点返回 JSONP 包裹，
// 且 userinfo 不返回邮箱——均按对应格式解析，并触发邮箱补全。
type qqProvider struct {
	name         string
	enabled      bool
	clientID     string
	clientSecret string
	redirectURL  string
}

func newQQProvider(cred providerCred) *qqProvider {
	return &qqProvider{
		name:         cred.code,
		enabled:      cred.enabled,
		clientID:     cred.clientID,
		clientSecret: cred.clientSecret,
		redirectURL:  cred.redirectURL,
	}
}

func (p *qqProvider) Name() string    { return model.ProviderQQ }
func (p *qqProvider) IsEnabled() bool { return p.enabled && p.clientID != "" }

func (p *qqProvider) RedirectURL(state string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", p.clientID)
	q.Set("redirect_uri", p.redirectURL)
	q.Set("state", state)
	q.Set("scope", "get_user_info")
	return "https://graph.qq.com/oauth2.0/authorize?" + q.Encode()
}

func (p *qqProvider) Exchange(ctx context.Context, code string) (*dto.OAuthUserInfo, error) {
	tokenURL := "https://graph.qq.com/oauth2.0/token?" + url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {p.clientID},
		"client_secret": {p.clientSecret},
		"code":          {code},
		"redirect_uri":  {p.redirectURL},
	}.Encode()
	tb, err := getPlain(ctx, tokenURL)
	if err != nil {
		return nil, ErrOAuthExchangeFailed
	}
	// 返回格式：access_token=xxx&expires_in=xxx&refresh_token=xxx
	vals, err := url.ParseQuery(string(tb))
	if err != nil || vals.Get("access_token") == "" {
		// 响应体含有效令牌，只写服务端日志，对外仅返回语义化错误
		logger.Warnf("[oauth] qq token exchange failed: %s", safeBodyPreview(tb))
		return nil, ErrOAuthExchangeFailed
	}
	accessToken := vals.Get("access_token")

	// openid 端点返回 JSONP：callback( {"client_id":"...","openid":"..."} );
	mb, err := getPlain(ctx, "https://graph.qq.com/oauth2.0/me?access_token="+accessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUserInfoFailed, err)
	}
	openID, err := extractQQOpenID(string(mb))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUserInfoFailed, err)
	}

	infoURL := "https://graph.qq.com/user/get_user_info?" + url.Values{
		"access_token":       {accessToken},
		"oauth_consumer_key": {p.clientID},
		"openid":             {openID},
	}.Encode()
	ub, err := getPlain(ctx, infoURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUserInfoFailed, err)
	}
	var qu struct {
		Ret       int    `json:"ret"`
		Msg       string `json:"msg"`
		Nickname  string `json:"nickname"`
		FigureURL string `json:"figureurl_qq_2"`
	}
	if err := json.Unmarshal(ub, &qu); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUserInfoFailed, err)
	}
	// ret != 0 表示上游拒绝，msg 内容不可控，只记日志不对外回显
	if qu.Ret != 0 {
		logger.Warnf("[oauth] qq get_user_info rejected: ret=%d msg=%s", qu.Ret, qu.Msg)
		return nil, ErrOAuthUserInfoFailed
	}

	// QQ 不提供邮箱，留空由上层引导补全
	return &dto.OAuthUserInfo{
		Provider:    model.ProviderQQ,
		ProviderID:  openID,
		Email:       "",
		Name:        qu.Nickname,
		AvatarURL:   qu.FigureURL,
		AccessToken: accessToken,
	}, nil
}

var qqJSONPCleaner = regexp.MustCompile(`\A\s*callback\(\s*|\s*\)\s*;\s*\z`)

// extractQQOpenID 从 JSONP 响应中提取 openid。
func extractQQOpenID(raw string) (string, error) {
	cleaned := qqJSONPCleaner.ReplaceAllString(raw, "")
	var m struct {
		OpenID string `json:"openid"`
	}
	if err := json.Unmarshal([]byte(cleaned), &m); err != nil || m.OpenID == "" {
		return "", fmt.Errorf("parse openid from %q", strings.TrimSpace(raw))
	}
	return m.OpenID, nil
}
