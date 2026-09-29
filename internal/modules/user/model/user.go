package model

import "time"

type User struct {
	ID            string
	TenantID      string // 所属租户
	Username      string // 登录名
	Password      string // 加密后的哈希值
	Nickname      string
	Email         string
	EmailVerified bool   // 邮箱是否已验证
	Phone         string // 手机号
	Avatar        string // 头像URL
	Roles         []string // 角色编码列表（从 user_roles 表关联查询得到）
	Status        int      // 1: 正常, 0: 禁用
	IsMaster      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}