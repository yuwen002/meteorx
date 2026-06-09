package dto

// UserListResp 如果你以后需要用户列表的分页显示，也可以在这里预留
type UserListResp struct {
	Items []*UserResp `json:"items"`
	Total int64       `json:"total"`
}

type CreateUserReq struct {
	Username string `json:"username" validate:"required,alphanum,min=4,max=50"`
	Password string `json:"password" validate:"required,min=6,max=32"`
	Nickname string `json:"nickname" validate:"required,max=50"`
	Email    string `json:"email,omitempty" validate:"omitempty,email"`
	Role     string `json:"role" validate:"required,oneof=admin user"` // 租户用户角色
}

// CreateMasterAdminReq 创建系统管理员请求（不需要指定角色，自动设为 superadmin）
type CreateMasterAdminReq struct {
	Username string `json:"username" validate:"required,alphanum,min=4,max=50"`
	Password string `json:"password" validate:"required,min=6,max=32"`
	Nickname string `json:"nickname" validate:"required,max=50"`
	Email    string `json:"email,omitempty" validate:"omitempty,email"`
}

// AdminCreateTenantUserReq 系统管理员为指定租户创建用户请求
type AdminCreateTenantUserReq struct {
	TenantID string `json:"tenant_id" validate:"required"` // 需要指定租户ID
	Username string `json:"username" validate:"required,alphanum,min=4,max=50"`
	Password string `json:"password" validate:"required,min=6,max=32"`
	Nickname string `json:"nickname" validate:"required,max=50"`
	Email    string `json:"email,omitempty" validate:"omitempty,email"`
	Role     string `json:"role" validate:"required,oneof=admin user superadmin"` // 租户用户角色
}

type UpdateUserReq struct {
	Nickname string `json:"nickname,omitempty" validate:"max=50"`
	Email    string `json:"email,omitempty" validate:"omitempty,email"`
	Role     string `json:"role,omitempty" validate:"oneof=admin user superadmin"`
	Status   int    `json:"status,omitempty" validate:"oneof=0 1"`
}

type UserResp struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Status    int    `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
