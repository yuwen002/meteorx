package model

import "time"

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
