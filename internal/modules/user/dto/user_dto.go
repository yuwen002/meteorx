package dto

// UserListResp 如果你以后需要用户列表的分页显示，也可以在这里预留
type UserListResp struct {
	Items []*UserResp `json:"items"`
	Total int64       `json:"total"`
}

type CreateUserReq struct {
	Username string   `json:"username" validate:"required,alphanum,min=4,max=50"`
	Password string   `json:"password" validate:"required,min=6,max=32"`
	Nickname string   `json:"nickname" validate:"required,max=50"`
	Email    string   `json:"email,omitempty" validate:"omitempty,email"`
	RoleIDs  []string `json:"role_ids" validate:"required,min=1"` // 角色ID列表，需存在于 roles 表中
}

// CreateMasterAdminReq 创建系统管理员请求（可指定角色，不指定则默认为 superadmin）
type CreateMasterAdminReq struct {
	Username string `json:"username" validate:"required,alphanum,min=4,max=50"`
	Password string `json:"password" validate:"required,min=6,max=32"`
	Nickname string `json:"nickname" validate:"required,max=50"`
	Email    string `json:"email,omitempty" validate:"omitempty,email"`
	RoleID   string `json:"role_id,omitempty" validate:"omitempty"` // 角色ID，只能指定一个，不指定则默认为 superadmin
}

// AdminCreateTenantUserReq 系统管理员为指定租户创建用户请求
type AdminCreateTenantUserReq struct {
	TenantID string   `json:"tenant_id" validate:"required" label:"租户ID"` // 需要指定租户ID
	Username string   `json:"username" validate:"required,username,min=4,max=50" label:"用户名"`
	Password string   `json:"password" validate:"required,min=6,max=32" label:"密码"`
	Nickname string   `json:"nickname" validate:"required,max=50" label:"昵称"`
	Email    string   `json:"email,omitempty" validate:"omitempty,email" label:"邮箱"`
	RoleIDs  []string `json:"role_ids" validate:"required,min=1" label:"角色"` // 角色ID列表
}

type UpdateUserReq struct {
	Nickname string   `json:"nickname,omitempty" validate:"max=50"`
	Email    string   `json:"email,omitempty" validate:"omitempty,email"`
	RoleIDs  []string `json:"role_ids,omitempty" validate:"omitempty,min=1"` // 可选，用于单独更新角色
	Status   *int     `json:"status,omitempty" validate:"omitempty,oneof=0 1"`
}

// UpdateMasterAdminReq 更新系统管理员请求
 type UpdateMasterAdminReq struct {
	Nickname string `json:"nickname,omitempty" validate:"max=50"`
	Email    string `json:"email,omitempty" validate:"omitempty,email"`
	RoleID   string `json:"role_id,omitempty" validate:"omitempty"` // 角色ID，只能指定一个
	Status   *int   `json:"status,omitempty" validate:"omitempty,oneof=0 1"`
}

type UserResp struct {
	ID         string   `json:"id"`
	TenantID   string   `json:"tenant_id"`
	TenantName string   `json:"tenant_name"`
	Username   string   `json:"username"`
	Nickname   string   `json:"nickname"`
	Email      string   `json:"email"`
	Roles      []string `json:"roles"`     // 角色编码列表
	RoleIDs    []string `json:"role_ids"`
	Status     int      `json:"status"`
	IsMaster   bool     `json:"is_master"` // 是否系统管理员
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
	DeletedAt  string   `json:"deleted_at,omitempty"` // 删除时间，仅已删除记录返回
}

// AssignUserRolesReq 为用户分配角色
type AssignUserRolesReq struct {
	RoleIDs []string `json:"role_ids" validate:"required,min=1"` // 角色ID列表，全覆盖式更新
}

// ChangePasswordReq 修改密码请求
type ChangePasswordReq struct {
	OldPassword     string `json:"old_password" validate:"required,min=6,max=32"`
	NewPassword     string `json:"new_password" validate:"required,min=6,max=32"`
	ConfirmPassword string `json:"confirm_password" validate:"required,min=6,max=32,eqfield=NewPassword"`
}

// UpdateUserStatusReq 更新用户状态请求
type UpdateUserStatusReq struct {
	Status int `json:"status" validate:"oneof=0 1"` // 状态: 1-启用 0-禁用
}

// BatchUpdateUserStatusReq 批量更新用户状态请求
type BatchUpdateUserStatusReq struct {
	IDs    []string `json:"ids" validate:"required,min=1"` // 用户ID列表
	Status int      `json:"status" validate:"oneof=0 1"`    // 状态: 1-启用 0-禁用
}

// BatchDeleteUsersReq 批量删除用户请求
type BatchDeleteUsersReq struct {
	IDs []string `json:"ids" validate:"required,min=1"` // 用户ID列表
}