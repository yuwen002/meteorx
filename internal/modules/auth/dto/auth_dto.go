// Package dto 定义认证模块的请求与响应数据结构。
package dto

import userdto "meteorx/internal/modules/user/dto"

// LoginReq 用户登录请求
type LoginReq struct {
	TenantID string `json:"tenant_id" validate:"omitempty" label:"租户ID"` // 👈 改为 omitempty，上帝账号不需要租户ID
	Username string `json:"username" validate:"required" label:"用户名"`
	Password string `json:"password" validate:"required" label:"密码"`
}

// RegisterUserReq 新用户注册请求
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
	User        *userdto.UserResp `json:"user"`        // 用户信息
	Permissions []string          `json:"permissions"` // 用户所有权限码（前端用于按钮/菜单权限控制）
}

// LoginErrorResp 登录失败响应（包含安全提示）
type LoginErrorResp struct {
	Message           string `json:"message"`            // 错误信息
	RemainingAttempts int    `json:"remaining_attempts"` // 剩余尝试次数
	Locked            bool   `json:"locked"`             // 是否已锁定
	LockoutDuration   int64  `json:"lockout_duration"`   // 锁定剩余时间（秒）
}

// ForgotPasswordReq 忘记密码请求（发送重置链接到邮箱）
type ForgotPasswordReq struct {
	Email string `json:"email" validate:"required,email" label:"邮箱"`
}

// ForgotPasswordResp 忘记密码处理响应
type ForgotPasswordResp struct {
	Message string `json:"message"`
}

// ResetPasswordReq 重置密码请求（携重置令牌与新密码）
type ResetPasswordReq struct {
	Token       string `json:"token" validate:"required" label:"重置令牌"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=32" label:"新密码"`
}

// ResetPasswordResp 重置密码响应
type ResetPasswordResp struct {
	Message string `json:"message"`
}

// SendEmailVerificationReq 发送邮箱验证链接请求
type SendEmailVerificationReq struct {
	Email string `json:"email" validate:"required,email" label:"邮箱"` // 需要验证的邮箱地址
}

// SendEmailVerificationResp 发送邮箱验证链接响应
type SendEmailVerificationResp struct {
	Message string `json:"message"` // 提示信息
}

// VerifyEmailReq 验证邮箱请求（通过邮件中的令牌）
type VerifyEmailReq struct {
	Token string `json:"token" validate:"required" label:"验证令牌"` // 邮件链接中携带的验证令牌
}

// VerifyEmailResp 验证邮箱响应
type VerifyEmailResp struct {
	Message string `json:"message"` // 提示信息
}
