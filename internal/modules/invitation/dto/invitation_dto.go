package dto

type CreateInvitationReq struct {
	Email   string   `json:"email" validate:"required,email" label:"邮箱"`
	RoleIDs []string `json:"role_ids" validate:"required,min=1" label:"角色"`
}

type InvitationResp struct {
	ID         string   `json:"id"`
	TenantID   string   `json:"tenant_id"`
	TenantName string   `json:"tenant_name"`
	Email      string   `json:"email"`
	RoleIDs    []string `json:"role_ids"`
	Status     string   `json:"status"`
	InvitedBy  string   `json:"invited_by"`
	ExpiresAt  string   `json:"expires_at"`
	AcceptedAt string   `json:"accepted_at,omitempty"`
	CreatedAt  string   `json:"created_at"`
}

type AcceptInvitationReq struct {
	Token    string `json:"token" validate:"required" label:"邀请令牌"`
	Username string `json:"username" validate:"required,alphanum,min=4,max=50" label:"用户名"`
	Password string `json:"password" validate:"required,min=6,max=32" label:"密码"`
	Nickname string `json:"nickname" validate:"required,max=50" label:"昵称"`
}

type AcceptInvitationResp struct {
	Message string `json:"message"`
}

type ListInvitationsReq struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Keyword  string `json:"keyword"`
	Status   string `json:"status"`
}