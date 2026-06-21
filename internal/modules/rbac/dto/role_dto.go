package dto

// CreateRoleReq 创建角色请求
type CreateRoleReq struct {
	Name        string `json:"name" validate:"required,max=50"`
	Code        string `json:"code" validate:"required,max=50"`
	Description string `json:"description" validate:"max=255"`
}

// UpdateRoleReq 更新角色请求
type UpdateRoleReq struct {
	Name        string `json:"name" validate:"max=50"`
	Code        string `json:"code" validate:"max=50"`
	Description string `json:"description" validate:"max=255"`
}

// RoleResp 角色响应
type RoleResp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	TenantID    string `json:"tenant_id"`
	IsSystem    bool   `json:"is_system"`
	CreatedAt   string `json:"created_at"`
}

// ListRolesQuery 角色列表查询参数
type ListRolesQuery struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Keyword  string `json:"keyword" form:"keyword"`
}

// BindRolePermissionsReq 绑定角色权限请求
type BindRolePermissionsReq struct {
	PermissionIDs []string `json:"permission_ids" validate:"required"`
}
