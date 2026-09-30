package model

import "time"

// UserRole 用户与角色的多对多关联，可附带用户/角色详情用于联表查询。
type UserRole struct {
	UserID    string
	RoleID    string
	CreatedAt time.Time
	User      *RoleUserInfo
	Role      *Role
}

// RoleUserInfo 用户角色关系中的用户信息子集（避免依赖 user 模块的 User）
type RoleUserInfo struct {
	ID       string
	TenantID string
	Username string
	Nickname string
	Email    string
	Status   int
	IsMaster bool
}
