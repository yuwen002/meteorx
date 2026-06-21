package model

import "time"

type Permission struct {
	ID          string
	Name        string
	Code        string
	Description string
	Resource    string
	Action      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
