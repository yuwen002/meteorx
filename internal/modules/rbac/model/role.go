package model

import "time"

const (
	RoleStatusDisabled = 0 // 禁用
	RoleStatusEnabled  = 1 // 启用
)

// 角色作用域：控制角色可在哪个上下文被分配
const (
	RoleScopeSystem = "system" // 仅系统管理接口可分配
	RoleScopeTenant = "tenant" // 仅租户接口可分配
	RoleScopeAll    = "all"    // 所有上下文均可分配
)

type Role struct {
	ID          string
	Name        string
	Code        string
	Description string
	TenantID    string
	IsSystem    bool
	Scope       string // 作用域: system/tenant/all
	Status      int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}
