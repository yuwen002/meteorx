package dto

// AssignUserRolesReq 为用户分配角色请求
type AssignUserRolesReq struct {
	RoleIDs []string `json:"role_ids" validate:"required,min=1"` // 角色ID列表
}

// UserRoleResp 用户角色响应
type UserRoleResp struct {
	UserID    string `json:"user_id"`
	RoleID    string `json:"role_id"`
	RoleName  string `json:"role_name"`
	RoleCode  string `json:"role_code"`
	CreatedAt string `json:"created_at"`
}

// BatchAssignUserRolesReq 批量为用户分配角色请求
type BatchAssignUserRolesReq struct {
	UserRoleAssignments []UserRoleAssignment `json:"assignments" validate:"required,min=1"` // 用户角色分配列表
}

// UserRoleAssignment 单个用户角色分配
type UserRoleAssignment struct {
	UserID  string   `json:"user_id" validate:"required"`  // 用户ID
	RoleIDs []string `json:"role_ids" validate:"required,min=1"` // 角色ID列表
}

// RemoveUserRoleReq 删除用户单个角色请求
type RemoveUserRoleReq struct {
	RoleID string `json:"role_id" validate:"required"` // 角色ID
}
