package dto

import "time"

// AdminCreateTenantReq 运营后台管理员手工创建租户请求
type AdminCreateTenantReq struct {
	Name         string `json:"name" validate:"required,min=2,max=100" label:"租户名称"`
	Domain       string `json:"domain" validate:"required,alphanum,min=3,max=30" label:"租户域名"`
	Description  string `json:"description,omitempty" validate:"max=255" label:"租户描述"`
	ContactEmail string `json:"contact_email,omitempty" validate:"omitempty,email,max=100" label:"联系邮箱"`
	Region       string `json:"region,omitempty" validate:"max=50" label:"地区"`
	Logo         string `json:"logo,omitempty" validate:"omitempty,url,max=500" label:"Logo地址"`
	Status       int    `json:"status" validate:"oneof=0 1" label:"租户状态"` // 运营后台允许指定状态(0:禁用 1:启用)
	Extra        string `json:"extra,omitempty" validate:"max=1000" label:"扩展字段"`

	// 初始绑定的管理员信息
	AdminUser struct {
		Username string `json:"username" validate:"required,alphanum,min=4,max=50" label:"管理员用户名"`
		Password string `json:"password" validate:"required,min=6,max=32" label:"管理员密码"`
		Nickname string `json:"nickname" validate:"required,max=50" label:"管理员昵称"`
		Email    string `json:"email,omitempty" validate:"omitempty,email,max=100" label:"管理员邮箱"`
	} `json:"admin_user" validate:"required"`
}

// AdminTenantResp 运营后台专用的租户详情详情视图
type AdminTenantResp struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Domain       string    `json:"domain"`
	Status       int       `json:"status"`
	Description  string    `json:"description"`
	ContactEmail string    `json:"contact_email"`
	Region       string    `json:"region"`
	Logo         string    `json:"logo"`
	Extra        string    `json:"extra"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
