package model

import "time"

type RolePermission struct {
	RoleID       string
	PermissionID string
	CreatedAt    time.Time
	Role         *Role       // 角色详情（列表联表查询时加载）
	Permission   *Permission // 权限详情（列表联表查询时加载）
}
