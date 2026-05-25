package dto

type RegisterUserReq struct {
	TenantID string `json:"tenant_id" validate:"required" label:"租户ID"`
	Username string `json:"username" validate:"required,alphanum,min=4,max=20" label:"用户名"`
	Password string `json:"password" validate:"required,min=6,max=32" label:"密码"`
	Nickname string `json:"nickname" validate:"required" label:"昵称"`
}
