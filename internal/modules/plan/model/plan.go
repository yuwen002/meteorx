// Package model 定义套餐模块的领域模型，包括套餐和订阅实体。
package model

import "time"

// Plan 套餐领域模型（纯净领域模型，不含第三方框架 Tag）
type Plan struct {
	ID          string
	Name        string
	Code        string
	Description string
	UserLimit   int
	Price       float64
	Status      int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

const (
	StatusDisabled = 0
	StatusEnabled  = 1
)
