package dto

// CreatePermissionReq 创建权限请求
type CreatePermissionReq struct {
	Name        string `json:"name" validate:"required,max=50"`
	Code        string `json:"code" validate:"required,max=100"`
	Description string `json:"description" validate:"max=255"`
	Resource    string `json:"resource" validate:"required,max=50"`
	Action      string `json:"action" validate:"required,max=50"`
}

// UpdatePermissionReq 更新权限请求
type UpdatePermissionReq struct {
	Name        string `json:"name" validate:"max=50"`
	Code        string `json:"code" validate:"max=100"`
	Description string `json:"description" validate:"max=255"`
	Resource    string `json:"resource" validate:"max=50"`
	Action      string `json:"action" validate:"max=50"`
}

// PermissionResp 权限响应
type PermissionResp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	CreatedAt   string `json:"created_at"`
}

// ListPermissionsQuery 权限列表查询参数
type ListPermissionsQuery struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Resource string `json:"resource" form:"resource"`
	Keyword  string `json:"keyword" form:"keyword"`
}
