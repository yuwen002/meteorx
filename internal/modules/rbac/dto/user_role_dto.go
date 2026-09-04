package dto

import (
	"meteorx/internal/modules/rbac/model"
)

// AssignUserRolesReq 为用户分配角色请求
type AssignUserRolesReq struct {
	RoleIDs []string `json:"role_ids" validate:"required,min=1"` // 角色ID列表
}

// UserRoleResp 用户-角色关系响应（联表查询后含用户和角色详情）
type UserRoleResp struct {
	UserID    string              `json:"user_id"`
	RoleID    string              `json:"role_id"`
	CreatedAt string              `json:"created_at"`
	User      *SimpleUserInfoResp `json:"user"`
	Role      *SimpleRoleResp     `json:"role"`
}

// SimpleUserInfoResp 简化的用户信息（仅在 user-role 列表中展示）
type SimpleUserInfoResp struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Status   int    `json:"status"`
	IsMaster bool   `json:"is_master"`
}

// ToUserRoleResp 将 model.UserRole 转为 UserRoleResp
func ToUserRoleResp(ur *model.UserRole) *UserRoleResp {
	resp := &UserRoleResp{
		UserID:    ur.UserID,
		RoleID:    ur.RoleID,
		CreatedAt: ur.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if ur.User != nil {
		resp.User = &SimpleUserInfoResp{
			ID:       ur.User.ID,
			TenantID: ur.User.TenantID,
			Username: ur.User.Username,
			Nickname: ur.User.Nickname,
			Email:    ur.User.Email,
			Status:   ur.User.Status,
			IsMaster: ur.User.IsMaster,
		}
	}
	if ur.Role != nil {
		resp.Role = &SimpleRoleResp{
			ID:    ur.Role.ID,
			Name:  ur.Role.Name,
			Code:  ur.Role.Code,
			Scope: ur.Role.Scope,
		}
	}
	return resp
}

// BatchAssignUserRolesReq 批量为用户分配角色请求
type BatchAssignUserRolesReq struct {
	UserRoleAssignments []UserRoleAssignment `json:"assignments" validate:"required,min=1"` // 用户角色分配列表
}

// UserRoleAssignment 单个用户角色分配
type UserRoleAssignment struct {
	UserID  string   `json:"user_id" validate:"required"`        // 用户ID
	RoleIDs []string `json:"role_ids" validate:"required,min=1"` // 角色ID列表
}

// RemoveUserRoleReq 删除用户单个角色请求
type RemoveUserRoleReq struct {
	RoleID string `json:"role_id" validate:"required"` // 角色ID
}
