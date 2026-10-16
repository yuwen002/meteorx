package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"meteorx/internal/modules/oauth/dto"
	"meteorx/internal/modules/oauth/model"
	"meteorx/pkg/logger"
)

// weChatProvider 微信网站应用扫码登录适配。
// 说明：微信不返回邮箱，ProviderID 优先取 unionid（同一开放平台下跨应用一致），否则取 openid。
type weChatProvider struct {
	name         string
	enabled      bool
	clientID     string
	clientSecret string
	redirectURL  string
}

func newWeChatProvider(cred providerCred) *weChatProvider {
	return &weChatProvider{
		name:         cred.code,
		enabled:      cred.enabled,
		clientID:     cred.clientID,
		clientSecret: cred.clientSecret,
		redirectURL:  cred.redirectURL,
	}
}

func (p *weChatProvider) Name() string    { return model.ProviderWeChat }
func (p *weChatProvider) IsEnabled() bool { return p.enabled && p.clientID != "" }

func (p *weChatProvider) RedirectURL(state string) string {
	q := url.Values{}
	q.Set("appid", p.clientID)
	q.Set("redirect_uri", p.redirectURL)
	q.Set("response_type", "code")
	q.Set("scope", "snsapi_login")
	q.Set("state", state)
	return "https://open.weixin.qq.com/connect/qrconnect?" + q.Encode()
}

func (p *weChatProvider) Exchange(ctx context.Context, code string) (*dto.OAuthUserInfo, error) {
	tokenURL := "https://api.weixin.qq.com/sns/oauth2/access_token?" + url.Values{
		"appid":      {p.clientID},
		"secret":     {p.clientSecret},
		"code":       {code},
		"grant_type": {"authorization_code"},
	}.Encode()

	tb, err := getPlain(ctx, tokenURL)
	if err != nil {
		return nil, ErrOAuthExchangeFailed
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		OpenID      string `json:"openid"`
		UnionID     string `json:"unionid"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(tb, &tr); err != nil || tr.AccessToken == "" {
		// 响应体可能含有效令牌，只写服务端日志，对外仅返回语义化错误
		logger.Warnf("[oauth] wechat token exchange failed: %s", safeBodyPreview(tb))
		return nil, ErrOAuthExchangeFailed
	}
	if tr.ErrCode != 0 {
		logger.Warnf("[oauth] wechat token errcode=%d errmsg=%s", tr.ErrCode, tr.ErrMsg)
		return nil, ErrOAuthExchangeFailed
	}

	infoURL := "https://api.weixin.qq.com/sns/userinfo?" + url.Values{
		"access_token": {tr.AccessToken},
		"openid":       {tr.OpenID},
	}.Encode()
	ub, err := getPlain(ctx, infoURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUserInfoFailed, err)
	}
	var wu struct {
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		Nickname   string `json:"nickname"`
		HeadImgURL string `json:"headimgurl"`
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}
	if err := json.Unmarshal(ub, &wu); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUserInfoFailed, err)
	}
	// errcode != 0 表示上游拒绝，errmsg 内容不可控，只记日志不对外回显
	if wu.ErrCode != 0 {
		logger.Warnf("[oauth] wechat userinfo errcode=%d errmsg=%s", wu.ErrCode, wu.ErrMsg)
		return nil, ErrOAuthUserInfoFailed
	}

	providerID := wu.UnionID
	if providerID == "" {
		providerID = wu.OpenID
	}
	if providerID == "" {
		return nil, fmt.Errorf("%w: missing openid/unionid in userinfo", ErrOAuthUserInfoFailed)
	}

	// 微信不提供邮箱，留空由上层引导补全
	return &dto.OAuthUserInfo{
		Provider:    model.ProviderWeChat,
		ProviderID:  providerID,
		Email:       "",
		Name:        wu.Nickname,
		AvatarURL:   wu.HeadImgURL,
		AccessToken: tr.AccessToken,
	}, nil
}

// getPlain 发起 GET 请求并读取原始响应体（微信接口不要求 Bearer）。
func getPlain(ctx context.Context, urlStr string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
}
