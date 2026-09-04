package dto

import "meteorx/internal/modules/rbac/model"

// CreateRoleReq 创建角色请求
type CreateRoleReq struct {
	Name        string `json:"name" validate:"required,max=50"`
	Code        string `json:"code" validate:"required,max=50"`
	Description string `json:"description" validate:"max=255"`
	TenantID    string `json:"tenant_id" validate:"max=26"`                        // 可选，为空则默认为系统级角色(SYSTEM_ROOT)
	IsSystem    bool   `json:"is_system"`                                          // 是否系统内置角色，默认 false
	Scope       string `json:"scope" validate:"omitempty,oneof=system tenant all"` // 作用域: system/tenant/all，默认 tenant
	Status      int    `json:"status"`                                             // 状态: 1-启用 0-禁用，默认 1
}

// UpdateRoleReq 更新角色请求
type UpdateRoleReq struct {
	Name        string `json:"name" validate:"max=50"`
	Code        string `json:"code" validate:"max=50"`
	Description string `json:"description" validate:"max=255"`
	TenantID    string `json:"tenant_id" validate:"max=26"`                        // 可重新分配租户
	Scope       string `json:"scope" validate:"omitempty,oneof=system tenant all"` // 作用域: system/tenant/all
	Status      int    `json:"status"`                                             // 状态: 1-启用 0-禁用
}

// RoleResp 角色响应
type RoleResp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	TenantID    string `json:"tenant_id"`
	IsSystem    bool   `json:"is_system"`
	Scope       string `json:"scope"`
	Status      int    `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ToRoleResp 将 model.Role 转为 RoleResp
func ToRoleResp(r *model.Role) *RoleResp {
	return &RoleResp{
		ID:          r.ID,
		Name:        r.Name,
		Code:        r.Code,
		Description: r.Description,
		TenantID:    r.TenantID,
		IsSystem:    r.IsSystem,
		Scope:       r.Scope,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   r.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
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

// UpdateRoleStatusReq 更改角色状态请求
type UpdateRoleStatusReq struct {
	Status int `json:"status" validate:"oneof=0 1"` // 状态: 1-启用 0-禁用
}

// BatchUpdateRoleStatusReq 批量更改角色状态请求
type BatchUpdateRoleStatusReq struct {
	IDs    []string `json:"ids" validate:"required,min=1"` // 角色ID列表
	Status int      `json:"status" validate:"oneof=0 1"`   // 状态: 1-启用 0-禁用
}

// BatchDeleteRolesReq 批量删除角色请求
type BatchDeleteRolesReq struct {
	IDs []string `json:"ids" validate:"required,min=1"` // 角色ID列表
}

// UnbindRolePermissionReq 解绑角色单个权限请求
type UnbindRolePermissionReq struct {
	PermissionID string `json:"permission_id" validate:"required"` // 权限ID
}

// UnbindRolePermissionsReq 解绑角色多个权限请求
type UnbindRolePermissionsReq struct {
	PermissionIDs []string `json:"permission_ids" validate:"required,min=1"` // 权限ID列表
}

// RolePermissionResp 角色权限关系响应
type RolePermissionResp struct {
	RoleID       string                `json:"role_id"`       // 角色ID
	PermissionID string                `json:"permission_id"` // 权限ID
	CreatedAt    string                `json:"created_at"`    // 绑定时间
	Role         *SimpleRoleResp       `json:"role"`          // 角色信息
	Permission   *SimplePermissionResp `json:"permission"`    // 权限信息
}

// SimpleRoleResp 简化的角色信息
type SimpleRoleResp struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Code  string `json:"code"`
	Scope string `json:"scope"`
}

// SimplePermissionResp 简化的权限信息
type SimplePermissionResp struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// ToRolePermissionResp 将 model.RolePermission 转为 RolePermissionResp
func ToRolePermissionResp(rp *model.RolePermission) *RolePermissionResp {
	resp := &RolePermissionResp{
		RoleID:       rp.RoleID,
		PermissionID: rp.PermissionID,
		CreatedAt:    rp.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if rp.Role != nil {
		resp.Role = &SimpleRoleResp{
			ID:    rp.Role.ID,
			Name:  rp.Role.Name,
			Code:  rp.Role.Code,
			Scope: rp.Role.Scope,
		}
	}
	if rp.Permission != nil {
		resp.Permission = &SimplePermissionResp{
			ID:       rp.Permission.ID,
			Name:     rp.Permission.Name,
			Code:     rp.Permission.Code,
			Resource: rp.Permission.Resource,
			Action:   rp.Permission.Action,
		}
	}
	return resp
}

// BatchBindRolesPermissionsReq 批量为多个角色绑定权限请求
type BatchBindRolesPermissionsReq struct {
	RoleIDs       []string `json:"role_ids" validate:"required,min=1"`       // 角色ID列表
	PermissionIDs []string `json:"permission_ids" validate:"required,min=1"` // 权限ID列表
}

// BatchUnbindRolesPermissionsReq 批量解绑多个角色的权限请求
type BatchUnbindRolesPermissionsReq struct {
	RoleIDs       []string `json:"role_ids" validate:"required,min=1"`       // 角色ID列表
	PermissionIDs []string `json:"permission_ids" validate:"required,min=1"` // 权限ID列表
}
