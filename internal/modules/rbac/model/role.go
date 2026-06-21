package model

import "time"

type Role struct {
	ID          string
	Name        string
	Code        string
	Description string
	TenantID    string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
