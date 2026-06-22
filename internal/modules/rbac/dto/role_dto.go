package dto

// CreateRoleReq 创建角色请求
type CreateRoleReq struct {
	Name        string `json:"name" validate:"required,max=50"`
	Code        string `json:"code" validate:"required,max=50"`
	Description string `json:"description" validate:"max=255"`
	TenantID    string `json:"tenant_id" validate:"max=26"` // 可选，为空则默认为系统级角色(SYSTEM_ROOT)
	IsSystem    bool   `json:"is_system"`                   // 是否系统内置角色，默认 false
	Status      int    `json:"status"`                      // 状态: 1-启用 0-禁用，默认 1
}

// UpdateRoleReq 更新角色请求
type UpdateRoleReq struct {
	Name        string `json:"name" validate:"max=50"`
	Code        string `json:"code" validate:"max=50"`
	Description string `json:"description" validate:"max=255"`
	TenantID    string `json:"tenant_id" validate:"max=26"` // 可重新分配租户
	Status      int    `json:"status"`                      // 状态: 1-启用 0-禁用
}

// RoleResp 角色响应
type RoleResp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	TenantID    string `json:"tenant_id"`
	IsSystem    bool   `json:"is_system"`
	Status      int    `json:"status"`
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
