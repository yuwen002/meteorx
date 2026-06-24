package model

import "time"

type UserRole struct {
	UserID    string
	RoleID    string
	CreatedAt time.Time
}