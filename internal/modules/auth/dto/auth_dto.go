package dto

import userdto "meteorx/internal/modules/user/dto"

// LoginReq 用户登录请求
type LoginReq struct {
	TenantID string `json:"tenant_id" validate:"omitempty" label:"租户ID"` // 👈 改为 omitempty，上帝账号不需要租户ID
	Username string `json:"username" validate:"required" label:"用户名"`
	Password string `json:"password" validate:"required" label:"密码"`
}
type RegisterUserReq struct {
	TenantID string `json:"tenant_id" validate:"required" label:"租户ID"`
	Username string `json:"username" validate:"required,alphanum,min=4,max=20" label:"用户名"`
	Password string `json:"password" validate:"required,min=6,max=32" label:"密码"`
	Nickname string `json:"nickname" validate:"required" label:"昵称"`
	Email    string `json:"email" validate:"required,email" label:"邮箱"`
}

// LoginResp 登录成功响应
type LoginResp struct {
	Token string            `json:"token"` // JWT 令牌
	User  *userdto.UserResp `json:"user"`  // 返回用户信息给前端展示
}
