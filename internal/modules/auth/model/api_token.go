package model

import "time"

type APIToken struct {
	ID          string
	Name        string
	TokenHash   string
	UserID      string
	TenantID    string
	ExpiresAt   *time.Time
	LastUsedAt  *time.Time
	CreatedAt   time.Time
	RevokedAt   *time.Time
}

func (APIToken) TableName() string {
	return "api_tokens"
}