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

// UpdateCurrentTenantReq 租户更新当前租户信息请求
type UpdateCurrentTenantReq struct {
	Name         string `json:"name,omitempty" validate:"omitempty,min=2,max=100" label:"租户名称"`
	Logo         string `json:"logo,omitempty" validate:"omitempty,url,max=500" label:"Logo地址"`
	Description  string `json:"description,omitempty" validate:"max=255" label:"租户描述"`
	ContactEmail string `json:"contact_email,omitempty" validate:"omitempty,email,max=100" label:"联系邮箱"`
	Region       string `json:"region,omitempty" validate:"max=50" label:"地区"`
	Extra        string `json:"extra,omitempty" validate:"max=1000" label:"扩展字段"`
}

// GetInitStatusResp 租户初始化状态响应
type GetInitStatusResp struct {
	Status      string `json:"status"`      // pending, initializing, completed, failed
	Message     string `json:"message"`     // 状态描述信息
	Progress    int    `json:"progress"`    // 进度百分比 0-100
	Initialized bool   `json:"initialized"` // 是否已完成初始化
}

// ApplyCancellationReq 租户申请注销请求
type ApplyCancellationReq struct {
	Reason string `json:"reason" validate:"required,max=500" label:"注销原因"`
}

// ApplyCancellationResp 租户申请注销响应
type ApplyCancellationResp struct {
	AppliedAt    string `json:"applied_at"`    // 申请时间
	Status       string `json:"status"`       // pending, approved, rejected
	EstimatedDay int    `json:"estimated_day"` // 预计注销天数
}
