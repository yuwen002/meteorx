package model

import "time"

// 订阅状态
const (
	SubscriptionActive    = 1 // 生效中
	SubscriptionExpired   = 2 // 已到期
	SubscriptionCancelled = 3 // 已取消
)

// TenantSubscription 租户套餐订阅领域模型
// 采用独立订阅表记录租户与套餐的关系，可保留变更历史
type TenantSubscription struct {
	ID        string
	TenantID  string
	PlanID    string
	Status    int
	StartedAt time.Time
	ExpiresAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
