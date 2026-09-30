// Package model 定义 OAuth 模块的领域模型，存储第三方账号关联信息。
package model

import "time"

// OAuthAccount 第三方 OAuth 账号与本系统用户的绑定关系。
type OAuthAccount struct {
	ID          string
	UserID      string
	Provider    string
	ProviderID  string
	Email       string
	AccessToken string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const (
	ProviderGoogle = "google"
	ProviderGitHub = "github"
)
