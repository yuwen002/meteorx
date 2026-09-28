package model

import "time"

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

func (APIToken) TableName() string {
	return "api_tokens"
}