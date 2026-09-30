package model

import "time"

// RolePermission 角色与权限的多对多关联，可附带角色/权限详情用于联表查询。
type RolePermission struct {
	RoleID       string
	PermissionID string
	CreatedAt    time.Time
	Role         *Role       // 角色详情（列表联表查询时加载）
	Permission   *Permission // 权限详情（列表联表查询时加载）
}
