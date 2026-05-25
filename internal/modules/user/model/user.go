package model

import "time"

type User struct {
	ID        string
	TenantID  string // 所属租户
	Username  string // 登录名
	Password  string // 加密后的哈希值
	Nickname  string
	Email     string
	Role      string // admin, user
	Status    int    // 1: 正常, 0: 禁用
	IsMaster  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
