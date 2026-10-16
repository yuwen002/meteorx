// Package dto 定义 OAuth 模块的请求与响应数据结构。
package dto

import "time"

// OAuthLoginRequest 第三方登录请求
type OAuthLoginRequest struct {
	Provider string `json:"provider" validate:"required"` // 提供商（白名单由服务层根据已配置 provider 校验）
	Code     string `json:"code" validate:"required"`     // 授权码
	State    string `json:"state" validate:"required"`    // CSRF 状态码（必填）
	TenantID string `json:"tenant_id" validate:"required"` // 租户ID（OAuth登录时选择租户）
}

// OAuthRedirectResponse OAuth2 跳转链接响应
type OAuthRedirectResponse struct {
	URL   string `json:"url"`
	State string `json:"state"` // CSRF 状态码，前端需保存并在回调时回传
}

// OAuthLoginResponse 登录成功响应
type OAuthLoginResponse struct {
	Token           string      `json:"token"`
	RefreshToken    string      `json:"refresh_token,omitempty"`
	User            interface{} `json:"user"`
	Permissions     []string    `json:"permissions"`
	IsNewUser       bool        `json:"is_new_user"`       // 是否首次登录（新用户）
	NeedCompleteEmail bool      `json:"need_complete_email"` // 第三方未返回邮箱，需前端引导补全
}

// OAuthUserInfo OAuth2 用户信息（第三方返回）
type OAuthUserInfo struct {
	Provider    string `json:"provider"`
	ProviderID  string `json:"provider_id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	AvatarURL   string `json:"avatar_url"`
	AccessToken string `json:"access_token"`
}

// TenantOption 租户选项（用于 OAuth 登录时选择租户）
type TenantOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TenantListResponse 租户列表响应
type TenantListResponse struct {
	Tenants []TenantOption `json:"tenants"`
}

// OAuthAccountResp 已绑定的第三方账号信息
type OAuthAccountResp struct {
	ID        string    `json:"id"`
	Provider  string    `json:"provider"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// OAuthAccountListResponse 已绑定账号列表响应
type OAuthAccountListResponse struct {
	Accounts []OAuthAccountResp `json:"accounts"`
}

// UnbindOAuthRequest 解绑第三方账号请求
type UnbindOAuthRequest struct {
	Provider string `json:"provider" validate:"required"` // 提供商
}

// BindOAuthRequest 绑定已有第三方账号请求（已登录用户绑定新 provider）
type BindOAuthRequest struct {
	Provider string `json:"provider" validate:"required"` // 提供商
	Code     string `json:"code" validate:"required"`     // 授权码
	State    string `json:"state" validate:"required"`    // CSRF 状态码
}

// CompleteEmailRequest 第三方登录后补全邮箱请求
type CompleteEmailRequest struct {
	Email string `json:"email" validate:"required,email"` // 补全的邮箱
}

// RefreshTokenRequest 刷新 Token 请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"` // 刷新令牌
}

// RefreshTokenResponse 刷新 Token 响应
type RefreshTokenResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

// ProviderMeta 已启用第三方登录提供商的展示元信息
type ProviderMeta struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// SaveOAuthProviderReq 新建/更新第三方登录提供商配置请求
type SaveOAuthProviderReq struct {
	Kind        string   `json:"kind" validate:"required"`             // provider 类型：google/github/wechat/feishu/dingtalk/qq/microsoft/oidc
	Code        string   `json:"code" validate:"required"`             // provider 标识；固定类型必须等于 kind，oidc 形如 oidc_<key>
	DisplayName string   `json:"display_name,omitempty"`              // 展示名，留空按内置规则推导
	Enabled     bool     `json:"enabled"`                             // 是否启用
	ClientID    string   `json:"client_id" validate:"required"`       // 客户端 ID
	ClientSecret string   `json:"client_secret,omitempty"`            // 客户端密钥；更新时留空表示保持原值
	RedirectURL string   `json:"redirect_url" validate:"required"`    // 授权回调地址
	Tenant      string   `json:"tenant,omitempty"`                    // Microsoft 目录租户（common/organizations/租户ID）
	AuthURL     string   `json:"auth_url,omitempty"`                  // OIDC 授权端点
	TokenURL    string   `json:"token_url,omitempty"`                 // OIDC 令牌端点
	UserInfoURL string   `json:"userinfo_url,omitempty"`              // OIDC 用户信息端点
	Scopes      []string `json:"scopes,omitempty"`                    // OIDC 授权范围
}

// SetOAuthProviderStatusReq 切换提供商启用状态请求。
// Enabled 用指针区分「未传该字段」（校验失败）与「显式传 false」（禁用）。
type SetOAuthProviderStatusReq struct {
	Enabled *bool `json:"enabled" validate:"required"` // 是否启用
}

// OAuthProviderResp 第三方登录提供商配置响应（client_secret 仅返回脱敏值）
type OAuthProviderResp struct {
	Code              string    `json:"code"`
	Kind              string    `json:"kind"`
	DisplayName       string    `json:"display_name"`
	Enabled           bool      `json:"enabled"`
	ClientID          string    `json:"client_id"`
	ClientSecretMask  string    `json:"client_secret_mask"`
	HasSecret         bool      `json:"has_secret"`
	RedirectURL       string    `json:"redirect_url"`
	Tenant            string    `json:"tenant,omitempty"`
	AuthURL           string    `json:"auth_url,omitempty"`
	TokenURL          string    `json:"token_url,omitempty"`
	UserInfoURL       string    `json:"userinfo_url,omitempty"`
	Scopes            []string  `json:"scopes,omitempty"`
	EditableEndpoint  bool      `json:"editable_endpoint"` // 是否可自行配置端点（通用 OIDC）
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// OAuthProviderListResponse 提供商配置列表响应
type OAuthProviderListResponse struct {
	Providers []OAuthProviderResp `json:"providers"`
}
