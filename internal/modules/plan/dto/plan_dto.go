package dto

import "time"

// CreatePlanReq 创建套餐请求
type CreatePlanReq struct {
	Name        string  `json:"name" validate:"required,min=2,max=100" label:"套餐名称"`
	Code        string  `json:"code" validate:"required,min=2,max=50" label:"套餐编码"`
	Description string  `json:"description,omitempty" validate:"max=255" label:"套餐描述"`
	UserLimit   int     `json:"user_limit" validate:"min=-1" label:"用户数上限"`
	Price       float64 `json:"price" validate:"min=0" label:"月费价格"`
	Status      int     `json:"status" validate:"oneof=0 1" label:"套餐状态"`
}

// UpdatePlanReq 更新套餐请求
type UpdatePlanReq struct {
	Name        string  `json:"name,omitempty" validate:"omitempty,min=2,max=100" label:"套餐名称"`
	Description string  `json:"description,omitempty" validate:"max=255" label:"套餐描述"`
	UserLimit   int     `json:"user_limit,omitempty" validate:"omitempty,min=-1" label:"用户数上限"`
	Price       float64 `json:"price,omitempty" validate:"omitempty,min=0" label:"月费价格"`
	Status      int     `json:"status,omitempty" validate:"omitempty,oneof=0 1" label:"套餐状态"`
}

// PlanResp 套餐响应
type PlanResp struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Code          string    `json:"code"`
	Description   string    `json:"description"`
	UserLimit     int       `json:"user_limit"`
	Price         float64   `json:"price"`
	Status        int       `json:"status"`
	SubscriberCnt int64     `json:"subscriber_cnt"` // 正在使用该套餐的租户数
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AssignPlanReq 为租户分配/变更套餐请求
type AssignPlanReq struct {
	PlanID    string `json:"plan_id" validate:"required" label:"套餐ID"`
	ExpiresAt string `json:"expires_at,omitempty" validate:"omitempty,datetime=2006-01-02 15:04:05" label:"到期时间"`
}

// CurrentPlanResp 当前租户套餐与用量响应
type CurrentPlanResp struct {
	TenantID       string     `json:"tenant_id"`
	PlanID         string     `json:"plan_id"`
	PlanName       string     `json:"plan_name"`
	PlanCode       string     `json:"plan_code"`
	UserLimit      int        `json:"user_limit"`
	CurrentUsers   int64      `json:"current_users"`
	StartedAt      string     `json:"started_at"`
	ExpiresAt      *string    `json:"expires_at"`
	Status         int        `json:"status"`
	EffectiveDays  int        `json:"effective_days"` // 剩余有效天数
}

// TenantPlanBrief 租户套餐摘要（用于租户列表展示）
type TenantPlanBrief struct {
	TenantID  string `json:"tenant_id"`
	PlanName  string `json:"plan_name"`
	Expired   bool   `json:"expired"`
}