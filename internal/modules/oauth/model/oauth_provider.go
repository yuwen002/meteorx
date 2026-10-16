// Package model 定义 OAuth 模块的领域模型，存储第三方账号关联信息。
package model

import "time"

// OAuthProvider 第三方登录提供商的运行时配置（平台级）。
// 与 config.yaml 的静态配置解耦：后台可增删改并即时生效，无需重启。
// ClientSecret 在领域层为明文，仓储层负责加密存储（见 repository 的 SecretEnc 字段）。
type OAuthProvider struct {
	ID           string
	Kind         string   // provider 类型：google/github/wechat/feishu/dingtalk/qq/microsoft/oidc
	Code         string   // provider 标识（唯一），固定类型时等于 Kind；oidc 时由管理员自定义
	DisplayName  string   // 展示名
	Enabled      bool     // 是否启用
	ClientID     string   // 客户端 ID
	ClientSecret string   // 客户端密钥（明文，落库前加密）
	RedirectURL  string   // 授权回调地址
	Tenant       string   // Microsoft 目录租户 ID 或 common/organizations
	AuthURL      string   // OIDC 授权端点
	TokenURL     string   // OIDC 令牌端点
	UserInfoURL  string   // OIDC 用户信息端点
	Scopes       []string // OIDC 授权范围
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// OAuth Provider 类型（Kind / 固定 provider 的 Code）
const (
	ProviderKindGoogle    = "google"
	ProviderKindGitHub    = "github"
	ProviderKindWeChat    = "wechat"
	ProviderKindFeishu    = "feishu"
	ProviderKindDingTalk  = "dingtalk"
	ProviderKindQQ        = "qq"
	ProviderKindMicrosoft = "microsoft"
	ProviderKindOIDC      = "oidc"
)

// ValidProviderKind 判断 provider 类型是否受支持
func ValidProviderKind(kind string) bool {
	switch kind {
	case ProviderKindGoogle, ProviderKindGitHub, ProviderKindWeChat,
		ProviderKindFeishu, ProviderKindDingTalk, ProviderKindQQ,
		ProviderKindMicrosoft, ProviderKindOIDC:
		return true
	}
	return false
}

// FixedProviderKinds 返回所有内置（端点与逻辑固化）的 provider 类型
func FixedProviderKinds() []string {
	return []string{
		ProviderKindGoogle, ProviderKindGitHub, ProviderKindWeChat,
		ProviderKindFeishu, ProviderKindDingTalk, ProviderKindQQ,
		ProviderKindMicrosoft,
	}
}

// NeedsEndpoints 该类型是否需要管理员提供授权/令牌/用户信息端点（目前仅通用 OIDC）
func NeedsEndpoints(kind string) bool {
	return kind == ProviderKindOIDC
}
