// Package model 提供数据看板相关的领域模型
package model

// DashboardOverview 数据看板总览统计
type DashboardOverview struct {
	// 租户统计
	TenantStats TenantStats `json:"tenant_stats"`
	// 用户统计
	UserStats UserStats `json:"user_stats"`
	// 订阅/套餐统计
	SubscriptionStats SubscriptionStats `json:"subscription_stats"`
	// 审计/操作统计
	AuditStats AuditStats `json:"audit_stats"`
}

// TenantStats 租户统计
type TenantStats struct {
	Total    int64 `json:"total"`     // 租户总数
	Enabled  int64 `json:"enabled"`   // 启用租户数
	Disabled int64 `json:"disabled"`  // 禁用租户数
	TodayNew int64 `json:"today_new"` // 今日新增租户
	WeekNew  int64 `json:"week_new"`  // 本周新增租户
	MonthNew int64 `json:"month_new"` // 本月新增租户
}

// UserStats 用户统计
type UserStats struct {
	Total    int64 `json:"total"`     // 用户总数
	TodayNew int64 `json:"today_new"` // 今日新增用户
	WeekNew  int64 `json:"week_new"`  // 本周新增用户
	MonthNew int64 `json:"month_new"` // 本月新增用户
}

// SubscriptionStats 订阅统计
type SubscriptionStats struct {
	Total        int64 `json:"total"`         // 订阅总数
	Active       int64 `json:"active"`        // 生效中订阅数
	Expired      int64 `json:"expired"`       // 已到期订阅数
	Cancelled    int64 `json:"cancelled"`     // 已取消订阅数
	ActiveTenant int64 `json:"active_tenant"` // 当前有生效订阅的租户数
}

// AuditStats 审计/操作统计
type AuditStats struct {
	Total       int64            `json:"total"`        // 日志总数
	Today       int64            `json:"today"`        // 今日日志数
	Success     int64            `json:"success"`      // 成功操作数
	Failure     int64            `json:"failure"`      // 失败操作数
	ActionStats map[string]int64 `json:"action_stats"` // 按操作类型统计
	ModuleStats map[string]int64 `json:"module_stats"` // 按模块统计
}
