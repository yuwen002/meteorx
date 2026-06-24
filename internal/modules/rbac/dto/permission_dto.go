package dto

import "meteorx/internal/modules/rbac/model"

// CreatePermissionReq 创建权限请求
type CreatePermissionReq struct {
	Name        string `json:"name" validate:"required,max=50"`
	Code        string `json:"code" validate:"required,max=100"`
	Description string `json:"description" validate:"max=255"`
	Resource    string `json:"resource" validate:"required,max=50"`
	Action      string `json:"action" validate:"required,max=50"`
	Status      int    `json:"status" validate:"omitempty,oneof=0 1"` // 状态: 1-启用 0-禁用，默认 1
}

// UpdatePermissionReq 更新权限请求
type UpdatePermissionReq struct {
	Name        string `json:"name" validate:"max=50"`
	Code        string `json:"code" validate:"max=100"`
	Description string `json:"description" validate:"max=255"`
	Resource    string `json:"resource" validate:"max=50"`
	Action      string `json:"action" validate:"max=50"`
	Status      int    `json:"status" validate:"omitempty,oneof=0 1"`
}

// PermissionResp 权限响应
type PermissionResp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Status      int    `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ToPermissionResp 将 model.Permission 转为 PermissionResp
func ToPermissionResp(p *model.Permission) *PermissionResp {
	return &PermissionResp{
		ID:          p.ID,
		Name:        p.Name,
		Code:        p.Code,
		Description: p.Description,
		Resource:    p.Resource,
		Action:      p.Action,
		Status:      p.Status,
		CreatedAt:   p.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ListPermissionsQuery 权限列表查询参数
type ListPermissionsQuery struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Resource string `json:"resource" form:"resource"`
	Keyword  string `json:"keyword" form:"keyword"`
}

// UpdatePermissionStatusReq 更改权限状态请求
type UpdatePermissionStatusReq struct {
	Status int `json:"status" validate:"oneof=0 1"`
}

// BatchUpdatePermissionStatusReq 批量更改权限状态请求
type BatchUpdatePermissionStatusReq struct {
	IDs    []string `json:"ids" validate:"required,min=1"`
	Status int      `json:"status" validate:"oneof=0 1"`
}

// BatchDeletePermissionsReq 批量删除权限请求
type BatchDeletePermissionsReq struct {
	IDs []string `json:"ids" validate:"required,min=1"`
}