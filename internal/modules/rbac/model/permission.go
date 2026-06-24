package model

import "time"

const (
	PermissionStatusDisabled = 0
	PermissionStatusEnabled  = 1
)

type Permission struct {
	ID          string
	Name        string
	Code        string
	Description string
	Resource    string
	Action      string
	Status      int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}