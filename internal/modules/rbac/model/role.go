// Package model 定义 RBAC 模块的领域模型，包括角色、权限及其关联实体。
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

// Role 角色领域模型，区分系统级/租户级作用域，支持软删除。
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
