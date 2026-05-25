package dto

// RegisterTenantReq 注册租户请求 (复合结构：租户 + 管理员)
type RegisterTenantReq struct {
	// --- 租户基础信息 ---
	Name         string `json:"name" validate:"required,min=2,max=100" label:"租户名称"`
	Domain       string `json:"domain" validate:"required,alphanum,min=3,max=30" label:"租户域名"`
	Description  string `json:"description,omitempty" validate:"max=255" label:"租户描述"`
	ContactEmail string `json:"contact_email,omitempty" validate:"omitempty,email,max=100" label:"联系邮箱"`
	Region       string `json:"region,omitempty" validate:"max=50" label:"地区"`
	Logo         string `json:"logo,omitempty" validate:"omitempty,url,max=500" label:"Logo地址"`
	Extra        string `json:"extra,omitempty" validate:"max=1000" label:"扩展字段"`

	// --- 初始管理员信息 (必须成对出现) ---
	AdminUser struct {
		Username string `json:"username" validate:"required,alphanum,min=4,max=50" label:"管理员用户名"`
		Password string `json:"password" validate:"required,min=6,max=32" label:"管理员密码"`
		Nickname string `json:"nickname" validate:"required,max=50" label:"管理员昵称"`
		Email    string `json:"email,omitempty" validate:"omitempty,email,max=100" label:"管理员邮箱"`
	} `json:"admin_user" validate:"required"`
}

// UpdateTenantReq 更新租户信息请求
type UpdateTenantReq struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Status      int    `json:"status" validate:"oneof=0 1"` // 0:禁用 1:正常
	Description string `json:"description,omitempty" validate:"max=255"`
}

// TenantResp 租户响应结构
type TenantResp struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Domain       string `json:"domain"`
	Status       int    `json:"status"`
	Description  string `json:"description,omitempty"`
	ContactEmail string `json:"contact_email,omitempty"`
	Region       string `json:"region,omitempty"`
	Logo         string `json:"logo,omitempty"`
	Extra        string `json:"extra,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}
