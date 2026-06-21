package model

import "time"

type RolePermission struct {
	RoleID       string
	PermissionID string
	CreatedAt    time.Time
}
