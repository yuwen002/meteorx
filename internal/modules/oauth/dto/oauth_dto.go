// Package dto 定义 OAuth 模块的请求与响应数据结构。
package dto

import "time"

// OAuthLoginRequest 第三方登录请求
type OAuthLoginRequest struct {
	Provider string `json:"provider" validate:"required,oneof=google github"` // 提供商
	Code     string `json:"code" validate:"required"`                         // 授权码
	State    string `json:"state" validate:"required"`                        // CSRF 状态码（必填）
	TenantID string `json:"tenant_id" validate:"required"`                    // 租户ID（OAuth登录时选择租户）
}

// OAuthRedirectResponse OAuth2 跳转链接响应
type OAuthRedirectResponse struct {
	URL   string `json:"url"`
	State string `json:"state"` // CSRF 状态码，前端需保存并在回调时回传
}

// OAuthLoginResponse 登录成功响应
type OAuthLoginResponse struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refresh_token,omitempty"`
	User         interface{} `json:"user"`
	Permissions  []string    `json:"permissions"`
	IsNewUser    bool        `json:"is_new_user"` // 是否首次登录（新用户）
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
	Provider string `json:"provider" validate:"required,oneof=google github"` // 提供商
}

// BindOAuthRequest 绑定已有第三方账号请求（已登录用户绑定新 provider）
type BindOAuthRequest struct {
	Provider string `json:"provider" validate:"required,oneof=google github"` // 提供商
	Code     string `json:"code" validate:"required"`                         // 授权码
	State    string `json:"state" validate:"required"`                        // CSRF 状态码
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
