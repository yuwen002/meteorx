package dto

// UserResp 用户通用响应 DTO
// 仅包含允许展示给前端的字段，严格隐藏 Password 等敏感信息
type UserResp struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	Email     string `json:"email"`
	Role      string `json:"role"`   // 如: admin, user
	Status    int    `json:"status"` // 1: 正常, 0: 禁用
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// UserListResp 如果你以后需要用户列表的分页显示，也可以在这里预留
type UserListResp struct {
	Items []*UserResp `json:"items"`
	Total int64       `json:"total"`
}
