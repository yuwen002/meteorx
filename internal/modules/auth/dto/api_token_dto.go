package dto

import "time"

// CreateAPITokenReq 创建 API Token 请求
type CreateAPITokenReq struct {
	Name      string `json:"name" validate:"required,min=1,max=64" label:"名称"`
	ExpiresIn string `json:"expires_in" validate:"omitempty" label:"有效期"` // Go Duration 格式，如 720h（30天）；空表示使用默认值
}

// CreateAPITokenResp 创建 API Token 响应（含明文令牌，仅创建时返回一次）
type CreateAPITokenResp struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Token     string    `json:"token"` // 明文令牌，仅此一次返回
	ExpiresAt *string   `json:"expires_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// APITokenResp API Token 列表项响应
type APITokenResp struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	LastUsedAt *string   `json:"last_used_at,omitempty"`
	ExpiresAt  *string   `json:"expires_at,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	Revoked    bool      `json:"revoked"`
}

// RevokeAPITokenReq 撤销 API Token 请求
type RevokeAPITokenReq struct {
	ID string `json:"id" validate:"required" label:"令牌ID"`
}