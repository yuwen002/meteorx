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
	"sync"
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

// defaultDisplayName 按内置规则推导展示名（Google/微信/…或 OIDC 去掉前缀）
func defaultDisplayName(id string) string {
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

// ProviderSpec 描述一个 provider 的完整参数，是「配置来源」与「Provider 实现」之间的中介。
// 配置可来自 config.yaml（静态）或 oauth_providers 表（运行时），二者最终都转换为 Spec 再构建。
type ProviderSpec struct {
	Kind         string   // provider 类型（model.ProviderKindXxx）
	Code         string   // provider 标识（唯一），固定类型时等于 Kind
	DisplayName  string   // 展示名，为空时按内置规则推导
	Enabled      bool     // 是否启用
	ClientID     string   // 客户端 ID
	ClientSecret string   // 客户端密钥（明文）
	RedirectURL  string   // 授权回调地址
	Tenant       string   // Microsoft 目录租户
	AuthURL      string   // OIDC 授权端点
	TokenURL     string   // OIDC 令牌端点
	UserInfoURL  string   // OIDC 用户信息端点
	Scopes       []string // OIDC 授权范围
}

// providerCred 各 provider 共用的凭据字段
type providerCred struct {
	code         string
	enabled      bool
	clientID     string
	clientSecret string
	redirectURL  string
}

// BuildProvider 按类型把 Spec 构建为具体 Provider 实现；未知类型返回 nil。
func BuildProvider(spec ProviderSpec) Provider {
	cred := providerCred{
		code:         spec.Code,
		enabled:      spec.Enabled,
		clientID:     spec.ClientID,
		clientSecret: spec.ClientSecret,
		redirectURL:  spec.RedirectURL,
	}
	scopes := spec.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}

	switch spec.Kind {
	case model.ProviderKindGoogle:
		return &standardOAuthProvider{
			name:         spec.Code,
			enabled:      cred.enabled,
			clientID:     cred.clientID,
			clientSecret: cred.clientSecret,
			redirectURL:  cred.redirectURL,
			authURL:      "https://accounts.google.com/o/oauth2/v2/auth",
			tokenURL:     "https://oauth2.googleapis.com/token",
			userInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
			scopes:       scopes,
		}
	case model.ProviderKindGitHub:
		return newGitHubProvider(cred)
	case model.ProviderKindWeChat:
		return newWeChatProvider(cred)
	case model.ProviderKindFeishu:
		return newFeishuProvider(cred)
	case model.ProviderKindDingTalk:
		return newDingTalkProvider(cred)
	case model.ProviderKindQQ:
		return newQQProvider(cred)
	case model.ProviderKindMicrosoft:
		tenant := spec.Tenant
		if tenant == "" {
			tenant = "common"
		}
		base := "https://login.microsoftonline.com/" + tenant + "/oauth2/v2.0"
		return &standardOAuthProvider{
			name:         spec.Code,
			enabled:      cred.enabled,
			clientID:     cred.clientID,
			clientSecret: cred.clientSecret,
			redirectURL:  cred.redirectURL,
			authURL:      base + "/authorize",
			tokenURL:     base + "/token",
			userInfoURL:  "https://graph.microsoft.com/oidc/userinfo",
			scopes:       scopes,
		}
	case model.ProviderKindOIDC:
		return &standardOAuthProvider{
			name:         spec.Code,
			enabled:      cred.enabled,
			clientID:     cred.clientID,
			clientSecret: cred.clientSecret,
			redirectURL:  cred.redirectURL,
			authURL:      spec.AuthURL,
			tokenURL:     spec.TokenURL,
			userInfoURL:  spec.UserInfoURL,
			scopes:       scopes,
		}
	default:
		return nil
	}
}

// ProviderRegistry 所有已配置 provider 的注册表，业务层通过 Get 按名取用。
// 支持热重载：Replace 原子替换内部快照，多副本下配合定时刷新即可免重启生效。
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	labels    map[string]string // code → 展示名（DB 配置优先于内置推导）
}

// NewProviderRegistryFromSpecs 依据 Spec 列表构建注册表；未知类型或标识为空的 Spec 被跳过。
func NewProviderRegistryFromSpecs(specs []ProviderSpec) *ProviderRegistry {
	m := make(map[string]Provider, len(specs))
	labels := make(map[string]string, len(specs))
	for _, spec := range specs {
		if spec.Code == "" {
			continue
		}
		p := BuildProvider(spec)
		if p == nil {
			logger.Warnf("[oauth] skip provider %q: unknown kind %q", spec.Code, spec.Kind)
			continue
		}
		m[spec.Code] = p
		if spec.DisplayName != "" {
			labels[spec.Code] = spec.DisplayName
		}
	}
	return &ProviderRegistry{providers: m, labels: labels}
}

// NewProviderRegistry 依据 config.yaml 构建注册表（作为 DB 运行时配置的种子与兜底）。
// 已知固定 provider（google/github/wechat/feishu/dingtalk/qq/microsoft）始终登记，
// 其 IsEnabled 同时要求 enabled=true 且已填写 client_id；未配置的 provider 虽存在但处于禁用态。
// 通用 OIDC 仅在配置（填写 client_id）时登记，其 provider 标识为 "oidc_<key>"。
func NewProviderRegistry(cfg config.OAuthConfig) *ProviderRegistry {
	specs := ProviderSpecsFromConfig(cfg)
	return NewProviderRegistryFromSpecs(specs)
}

// ProviderSpecsFromConfig 把 config.yaml 的静态配置转换为 ProviderSpec 列表。
// 提取为公开函数以便「首次启动种子化」复用同一套转换逻辑。
func ProviderSpecsFromConfig(cfg config.OAuthConfig) []ProviderSpec {
	specs := make([]ProviderSpec, 0, len(cfg.OIDC)+7)

	specs = append(specs, ProviderSpec{
		Kind: model.ProviderKindGoogle, Code: model.ProviderGoogle,
		Enabled: cfg.Google.Enabled, ClientID: cfg.Google.ClientID,
		ClientSecret: cfg.Google.ClientSecret, RedirectURL: cfg.Google.RedirectURL,
	})
	for _, item := range []struct {
		kind string
		cfg  config.OAuthProviderConfig
	}{
		{model.ProviderKindGitHub, cfg.GitHub},
		{model.ProviderKindWeChat, cfg.WeChat},
		{model.ProviderKindFeishu, cfg.Feishu},
		{model.ProviderKindDingTalk, cfg.DingTalk},
		{model.ProviderKindQQ, cfg.QQ},
	} {
		specs = append(specs, ProviderSpec{
			Kind: item.kind, Code: item.kind,
			Enabled: item.cfg.Enabled, ClientID: item.cfg.ClientID,
			ClientSecret: item.cfg.ClientSecret, RedirectURL: item.cfg.RedirectURL,
		})
	}
	specs = append(specs, ProviderSpec{
		Kind: model.ProviderKindMicrosoft, Code: model.ProviderMicrosoft,
		Enabled: cfg.Microsoft.Enabled, ClientID: cfg.Microsoft.ClientID,
		ClientSecret: cfg.Microsoft.ClientSecret, RedirectURL: cfg.Microsoft.RedirectURL,
		Tenant: cfg.Microsoft.Tenant,
	})

	for key, oc := range cfg.OIDC {
		if oc.ClientID == "" {
			continue
		}
		specs = append(specs, ProviderSpec{
			Kind: model.ProviderKindOIDC, Code: "oidc_" + key,
			DisplayName: oc.DisplayName,
			Enabled:     oc.Enabled, ClientID: oc.ClientID,
			ClientSecret: oc.ClientSecret, RedirectURL: oc.RedirectURL,
			AuthURL: oc.AuthURL, TokenURL: oc.TokenURL, UserInfoURL: oc.UserInfoURL,
			Scopes: oc.Scopes,
		})
	}
	return specs
}

// Get 按名获取 provider，未登记返回 (nil, false)。
func (r *ProviderRegistry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// Enabled 返回所有已启用的 provider 展示元信息（按 ID 排序）。
func (r *ProviderRegistry) Enabled() []ProviderMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	metas := make([]ProviderMeta, 0, len(r.providers))
	for _, p := range r.providers {
		if !p.IsEnabled() {
			continue
		}
		label := r.labels[p.Name()]
		if label == "" {
			label = defaultDisplayName(p.Name())
		}
		metas = append(metas, ProviderMeta{ID: p.Name(), Label: label})
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].ID < metas[j].ID })
	return metas
}

// Replace 用新的 provider 集合原子替换当前快照（热重载入口）。
func (r *ProviderRegistry) Replace(providers map[string]Provider, labels map[string]string) {
	if labels == nil {
		labels = map[string]string{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = providers
	r.labels = labels
}
