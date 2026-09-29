// Package dto 定义邀请模块的请求/响应数据传输对象。
package dto

// CreateInvitationReq 创建邀请请求。管理员邀请外部用户加入租户。
type CreateInvitationReq struct {
	Email   string   `json:"email" validate:"required,email" label:"邮箱"`     // 被邀请人邮箱
	RoleIDs []string `json:"role_ids" validate:"required,min=1" label:"角色"` // 分配的角色 ID 列表
}

// InvitationResp 邀请响应，返回给前端的邀请详情。
type InvitationResp struct {
	ID         string   `json:"id"`
	TenantID   string   `json:"tenant_id"`
	TenantName string   `json:"tenant_name"`   // 租户名称（仅查询时填充）
	Email      string   `json:"email"`         // 被邀请人邮箱
	RoleIDs    []string `json:"role_ids"`      // 分配的角色 ID 列表
	Status     string   `json:"status"`        // 状态：pending / accepted / cancelled / expired
	InvitedBy  string   `json:"invited_by"`    // 邀请人用户 ID
	ExpiresAt  string   `json:"expires_at"`    // 过期时间
	AcceptedAt string   `json:"accepted_at,omitempty"` // 接受时间（仅已接受时有值）
	CreatedAt  string   `json:"created_at"`    // 创建时间
}

// AcceptInvitationReq 接受邀请请求。被邀请人填写注册信息并接受邀请。
type AcceptInvitationReq struct {
	Token    string `json:"token" validate:"required" label:"邀请令牌"`                // 邮件链接中携带的邀请令牌
	Username string `json:"username" validate:"required,alphanum,min=4,max=50" label:"用户名"` // 设置的登录用户名
	Password string `json:"password" validate:"required,min=6,max=32" label:"密码"`      // 设置的密码
	Nickname string `json:"nickname" validate:"required,max=50" label:"昵称"`           // 设置的昵称
}

// AcceptInvitationResp 接受邀请响应。
type AcceptInvitationResp struct {
	Message string `json:"message"` // 提示信息
}

// ListInvitationsReq 邀请列表查询请求。
type ListInvitationsReq struct {
	Page     int    `json:"page"`       // 页码
	PageSize int    `json:"page_size"`  // 每页条数
	Keyword  string `json:"keyword"`    // 按邮箱搜索
	Status   string `json:"status"`     // 按状态筛选
}