package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"meteorx/internal/config"
	"meteorx/internal/modules/oauth/dto"
	"meteorx/internal/modules/oauth/model"
	"meteorx/pkg/logger"
)

// Provider 表示一个 OAuth2 / OIDC 身份提供商的适配实现。
// 新增一个 IdP 只需实现该接口并在 NewProviderRegistry 中登记，无需改动业务开关逻辑。
type Provider interface {
	// Name 返回 provider 标识（对应 model.ProviderXxx 或 "oidc_<key>"）
	Name() string
	// IsEnabled 是否已启用（配置 enabled=true 且填写了 client_id）
	IsEnabled() bool
	// RedirectURL 构造授权跳转地址
	RedirectURL(state string) string
	// Exchange 用授权码换取用户信息
	Exchange(ctx context.Context, code string) (*dto.OAuthUserInfo, error)
}

// ProviderMeta 已启用 provider 的展示元信息，供前端动态渲染登录按钮。
type ProviderMeta struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

var providerDisplayNames = map[string]string{
	model.ProviderGoogle:    "Google",
	model.ProviderGitHub:    "GitHub",
	model.ProviderWeChat:    "微信",
	model.ProviderFeishu:    "飞书",
	model.ProviderDingTalk:  "钉钉",
	model.ProviderQQ:        "QQ",
	model.ProviderMicrosoft: "Microsoft",
}

func displayName(id string) string {
	if n, ok := providerDisplayNames[id]; ok {
		return n
	}
	if strings.HasPrefix(id, "oidc_") {
		return strings.TrimPrefix(id, "oidc_")
	}
	return id
}

// ---- 共享 HTTP 辅助 ----

// oauthHTTPClient 共享出站 HTTP 客户端：统一超时，避免 IdP 挂起长时间占用 goroutine 与连接。
var oauthHTTPClient = &http.Client{Timeout: 10 * time.Second}

// maxResponseBodyBytes 上游响应体读取上限，防止异常大响应打爆内存。
const maxResponseBodyBytes = 1 << 20 // 1MB

// safeBodyPreview 截取响应体前 200 字节用于服务端脱敏日志，绝不拼入对外错误。
func safeBodyPreview(body []byte) string {
	s := strings.TrimSpace(string(body))
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

// httpPostForm 以 application/x-www-form-urlencoded 提交并读取响应体。
func httpPostForm(ctx context.Context, tokenURL string, values url.Values, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
}

// httpGetJSON 携带 Bearer 令牌请求 JSON 接口并读取响应体。
func httpGetJSON(ctx context.Context, urlStr, authToken string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Accept", "application/json")
	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		// 状态码可安全对外；响应体可能含令牌，只写服务端日志
		logger.Warnf("[oauth] userinfo request failed (%d): %s", resp.StatusCode, safeBodyPreview(body))
		return nil, fmt.Errorf("userinfo request failed (%d)", resp.StatusCode)
	}
	return body, nil
}

// standardUserInfo 从标准 OAuth2/OIDC 的 userinfo JSON 中解析通用字段。
// 兼容 id / sub 作为 ProviderID，email / name / picture / avatar_url 作为对应字段。
func standardUserInfo(body []byte) (providerID, email, name, avatar string, err error) {
	var m struct {
		ID        string `json:"id"`
		Sub       string `json:"sub"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		Picture   string `json:"picture"`
		AvatarURL string `json:"avatar_url"`
	}
	if err = json.Unmarshal(body, &m); err != nil {
		return "", "", "", "", fmt.Errorf("parse userinfo: %w", err)
	}
	providerID = m.ID
	if providerID == "" {
		providerID = m.Sub
	}
	avatar = m.Picture
	if avatar == "" {
		avatar = m.AvatarURL
	}
	return providerID, m.Email, m.Name, avatar, nil
}

// standardOAuthProvider 通用 OAuth2 授权码流程（POST 表单换 token + Bearer 取 userinfo）。
// 适用于 Google、Microsoft、通用 OIDC 等标准实现。
type standardOAuthProvider struct {
	name         string
	enabled      bool
	clientID     string
	clientSecret string
	redirectURL  string
	authURL      string
	tokenURL     string
	userInfoURL  string
	scopes       []string
	tokenHeaders map[string]string // 部分 IdP 要求 Accept: application/json 等
}

func (p *standardOAuthProvider) Name() string    { return p.name }
func (p *standardOAuthProvider) IsEnabled() bool { return p.enabled && p.clientID != "" }

func (p *standardOAuthProvider) RedirectURL(state string) string {
	q := url.Values{}
	q.Set("client_id", p.clientID)
	q.Set("redirect_uri", p.redirectURL)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(p.scopes, " "))
	q.Set("state", state)
	return p.authURL + "?" + q.Encode()
}

func (p *standardOAuthProvider) Exchange(ctx context.Context, code string) (*dto.OAuthUserInfo, error) {
	v := url.Values{}
	v.Set("code", code)
	v.Set("client_id", p.clientID)
	v.Set("client_secret", p.clientSecret)
	v.Set("redirect_uri", p.redirectURL)
	v.Set("grant_type", "authorization_code")

	body, err := httpPostForm(ctx, p.tokenURL, v, p.tokenHeaders)
	if err != nil {
		return nil, ErrOAuthExchangeFailed
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &tr); err != nil || tr.AccessToken == "" {
		// 响应体可能含有效令牌，只写服务端日志，对外仅返回语义化错误
		logger.Warnf("[oauth] %s token exchange failed: %s", p.name, safeBodyPreview(body))
		return nil, ErrOAuthExchangeFailed
	}

	ub, err := httpGetJSON(ctx, p.userInfoURL, tr.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUserInfoFailed, err)
	}
	providerID, email, name, avatar, err := standardUserInfo(ub)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthUserInfoFailed, err)
	}
	if providerID == "" {
		return nil, fmt.Errorf("%w: missing subject in userinfo", ErrOAuthUserInfoFailed)
	}
	return &dto.OAuthUserInfo{
		Provider:    p.name,
		ProviderID:  providerID,
		Email:       email,
		Name:        name,
		AvatarURL:   avatar,
		AccessToken: tr.AccessToken,
	}, nil
}

// ProviderRegistry 所有已配置 provider 的注册表，业务层通过 Get 按名取用。
type ProviderRegistry struct {
	providers map[string]Provider
}

// NewProviderRegistry 依据配置构建 provider 注册表。
// 已知固定 provider（google/github/wechat/feishu/dingtalk/qq/microsoft）始终登记，
// 其 IsEnabled 同时要求 enabled=true 且已填写 client_id；未配置的 provider 虽存在但处于禁用态。
// 通用 OIDC 仅在配置（填写 client_id）时登记，其 provider 标识为 "oidc_<key>"。
func NewProviderRegistry(cfg config.OAuthConfig) *ProviderRegistry {
	m := map[string]Provider{}

	m[model.ProviderGoogle] = &standardOAuthProvider{
		name:         model.ProviderGoogle,
		enabled:      cfg.Google.Enabled,
		clientID:     cfg.Google.ClientID,
		clientSecret: cfg.Google.ClientSecret,
		redirectURL:  cfg.Google.RedirectURL,
		authURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		tokenURL:     "https://oauth2.googleapis.com/token",
		userInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
		scopes:       []string{"openid", "email", "profile"},
	}

	m[model.ProviderGitHub] = newGitHubProvider(cfg.GitHub)
	m[model.ProviderWeChat] = newWeChatProvider(cfg.WeChat)
	m[model.ProviderFeishu] = newFeishuProvider(cfg.Feishu)
	m[model.ProviderDingTalk] = newDingTalkProvider(cfg.DingTalk)
	m[model.ProviderQQ] = newQQProvider(cfg.QQ)
	m[model.ProviderMicrosoft] = newMicrosoftProvider(cfg.Microsoft)

	for key, oc := range cfg.OIDC {
		if oc.ClientID == "" {
			continue
		}
		name := "oidc_" + key
		scopes := oc.Scopes
		if len(scopes) == 0 {
			scopes = []string{"openid", "email", "profile"}
		}
		m[name] = &standardOAuthProvider{
			name:         name,
			enabled:      oc.Enabled,
			clientID:     oc.ClientID,
			clientSecret: oc.ClientSecret,
			redirectURL:  oc.RedirectURL,
			authURL:      oc.AuthURL,
			tokenURL:     oc.TokenURL,
			userInfoURL:  oc.UserInfoURL,
			scopes:       scopes,
		}
	}

	return &ProviderRegistry{providers: m}
}

// Get 按名获取 provider，未登记返回 (nil, false)。
func (r *ProviderRegistry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// Enabled 返回所有已启用的 provider 展示元信息（按 ID 排序）。
func (r *ProviderRegistry) Enabled() []ProviderMeta {
	metas := make([]ProviderMeta, 0, len(r.providers))
	for _, p := range r.providers {
		if p.IsEnabled() {
			metas = append(metas, ProviderMeta{ID: p.Name(), Label: displayName(p.Name())})
		}
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].ID < metas[j].ID })
	return metas
}
