package service

import (
	"bytes"
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

// feishuProvider 飞书 OAuth2 适配（授权/令牌/用户信息均走开放平台 v1 接口）。
type feishuProvider struct {
	name         string
	enabled      bool
	clientID     string // 飞书称 app_id
	clientSecret string // 飞书称 app_secret
	redirectURL  string
}

func newFeishuProvider(cred providerCred) *feishuProvider {
	return &feishuProvider{
		name:         cred.code,
		enabled:      cred.enabled,
		clientID:     cred.clientID,
		clientSecret: cred.clientSecret,
		redirectURL:  cred.redirectURL,
	}
}

func (p *feishuProvider) Name() string    { return model.ProviderFeishu }
func (p *feishuProvider) IsEnabled() bool { return p.enabled && p.clientID != "" }

func (p *feishuProvider) RedirectURL(state string) string {
	q := url.Values{}
	q.Set("app_id", p.clientID)
	q.Set("redirect_uri", p.redirectURL)
	q.Set("state", state)
	return "https://www.feishu.cn/open-apis/authen/v1/authorize?" + q.Encode()
}

func (p *feishuProvider) Exchange(ctx context.Context, code string) (*dto.OAuthUserInfo, error) {
	tokenBody, _ := json.Marshal(map[string]string{
		"app_id":     p.clientID,
		"app_secret": p.clientSecret,
		"code":       code,
		"grant_type": "authorization_code",
	})
	raw, err := postJSON(ctx, "https://open.feishu.cn/open-apis/authen/v1/access_token", tokenBody, "")
	if err != nil {
		return nil, ErrOAuthExchangeFailed
	}
	var tr struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			AccessToken string `json:"access_token"`
			OpenID      string `json:"open_id"`
			UnionID     string `json:"union_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &tr); err != nil || tr.Data.AccessToken == "" {
		// 响应体可能含有效令牌，只写服务端日志，对外仅返回语义化错误
		logger.Warnf("[oauth] feishu token exchange failed: %s", safeBodyPreview(raw))
		return nil, ErrOAuthExchangeFailed
	}
	if tr.Code != 0 {
		logger.Warnf("[oauth] feishu token code=%d msg=%s", tr.Code, tr.Msg)
		return nil, ErrOAuthExchangeFailed
	}

	raw, err = postJSON(ctx, "https://open.feishu.cn/open-apis/authen/v1/user_info", nil, tr.Data.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUserInfoFailed, err)
	}
	var ui struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			OpenID    string `json:"open_id"`
			UnionID   string `json:"union_id"`
			Name      string `json:"name"`
			Email     string `json:"email"`
			AvatarURL string `json:"avatar_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &ui); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUserInfoFailed, err)
	}
	// code != 0 表示上游拒绝，msg 内容不可控，只记日志不对外回显
	if ui.Code != 0 {
		logger.Warnf("[oauth] feishu userinfo code=%d msg=%s", ui.Code, ui.Msg)
		return nil, ErrOAuthUserInfoFailed
	}

	providerID := ui.Data.UnionID
	if providerID == "" {
		providerID = ui.Data.OpenID
	}
	if providerID == "" {
		return nil, fmt.Errorf("%w: missing open_id/union_id in userinfo", ErrOAuthUserInfoFailed)
	}

	return &dto.OAuthUserInfo{
		Provider:    model.ProviderFeishu,
		ProviderID:  providerID,
		Email:       ui.Data.Email,
		Name:        ui.Data.Name,
		AvatarURL:   ui.Data.AvatarURL,
		AccessToken: tr.Data.AccessToken,
	}, nil
}

// postJSON 提交 JSON（可携带 Bearer 令牌），读取响应体。
func postJSON(ctx context.Context, urlStr string, body []byte, authToken string) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
}
