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
	Username string `json:"username" validate:"required,username,min=4,max=20" label:"用户名"`
	Password string `json:"password" validate:"required,min=6,max=32" label:"密码"`
	Nickname string `json:"nickname" validate:"required" label:"昵称"`
	Email    string `json:"email" validate:"required,email" label:"邮箱"`
}

// LoginResp 登录成功响应
type LoginResp struct {
	Token       string            `json:"token"`       // JWT 令牌
	User        *userdto.UserResp `json:"user"`  // 用户信息
	Permissions []string          `json:"permissions"`    // 用户所有权限码（前端用于按钮/菜单权限控制）
}

// LoginErrorResp 登录失败响应（包含安全提示）
type LoginErrorResp struct {
	Message          string `json:"message"`           // 错误信息
	RemainingAttempts int   `json:"remaining_attempts"` // 剩余尝试次数
	Locked           bool   `json:"locked"`            // 是否已锁定
	LockoutDuration  int64  `json:"lockout_duration"`  // 锁定剩余时间（秒）
}

type ForgotPasswordReq struct {
	Email string `json:"email" validate:"required,email" label:"邮箱"`
}

type ForgotPasswordResp struct {
	Message string `json:"message"`
}

type ResetPasswordReq struct {
	Token       string `json:"token" validate:"required" label:"重置令牌"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=32" label:"新密码"`
}

type ResetPasswordResp struct {
	Message string `json:"message"`
}