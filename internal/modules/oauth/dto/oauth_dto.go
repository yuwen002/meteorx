package dto

// OAuthLoginRequest 第三方登录请求
type OAuthLoginRequest struct {
	Provider string `json:"provider" validate:"required,oneof=google github"` // 提供商
	Code     string `json:"code" validate:"required"`                         // 授权码
	State    string `json:"state,omitempty"`                                  // CSRF 状态码
}

// OAuthRedirectResponse OAuth2 跳转链接响应
type OAuthRedirectResponse struct {
	URL string `json:"url"` // 跳转链接
}

// OAuthLoginResponse 登录成功响应
type OAuthLoginResponse struct {
	Token       string      `json:"token"`
	User        interface{} `json:"user"`
	Permissions []string    `json:"permissions"`
	IsNewUser   bool        `json:"is_new_user"` // 是否首次登录（新用户）
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
