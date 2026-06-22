package model

import "time"

const (
	RoleStatusDisabled = 0 // 禁用
	RoleStatusEnabled  = 1 // 启用
)

type Role struct {
	ID          string
	Name        string
	Code        string
	Description string
	TenantID    string
	IsSystem    bool
	Status      int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}
