package model

import "time"

// Tenant 这是一个纯净的领域模型，不带任何第三方框架的 Tag
type Tenant struct {
	ID        string
	Name      string
	Domain    string
	Status    int
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
