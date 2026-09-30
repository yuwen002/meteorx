// Package model 定义认证模块的领域模型，包括 API Token 等实体。
package model

import "time"

// APIToken API 访问令牌领域模型，仅持久化令牌哈希而非明文。
type APIToken struct {
	ID           string
	Name         string
	TokenHash    string
	UserID       string
	TenantID     string
	AllowedPaths string // 可访问的 URL 路径列表，JSON 数组格式；空表示不限制（默认权限）
	ExpiresAt    *time.Time
	LastUsedAt   *time.Time
	CreatedAt    time.Time
	RevokedAt    *time.Time
}

// TableName 返回 API 令牌表名 api_tokens。
func (APIToken) TableName() string {
	return "api_tokens"
}
