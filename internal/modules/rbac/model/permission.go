package model

import "time"

const (
	PermissionStatusDisabled = 0
	PermissionStatusEnabled  = 1
)

// Permission 权限点领域模型，以资源+动作描述一项可授权能力。
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
