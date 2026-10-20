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
	// ExpiryRemindedAt 到期提醒最近发送时间，nil 表示从未提醒过；
	// 去重靠这一列而不是任务内存，因此重启与多副本都不会重发
	ExpiryRemindedAt *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
