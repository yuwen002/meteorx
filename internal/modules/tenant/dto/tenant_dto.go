package dto

// RegisterTenantReq 注册租户请求
type RegisterTenantReq struct {
	Name   string `json:"name" validate:"required,min=2,max=100"`
	Domain string `json:"domain" validate:"required,alphanum,min=3,max=30"`
}

// UpdateTenantReq 更新租户信息请求
type UpdateTenantReq struct {
	Name   string `json:"name" validate:"required"`
	Status int    `json:"status" validate:"oneof=0 1"`
}
