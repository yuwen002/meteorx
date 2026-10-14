// Package model 提供数据看板相关的领域模型
package model

import "time"

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

// TrendDay 单日增长趋势数据点（按自然日分桶的新增量）
type TrendDay struct {
	Date       string `json:"date"`        // 业务时区自然日，格式 YYYY-MM-DD
	NewTenants int64  `json:"new_tenants"` // 当日新增租户数
	NewUsers   int64  `json:"new_users"`   // 当日新增用户数
	AuditLogs  int64  `json:"audit_logs"`  // 当日审计日志数（活跃度指标）
}

// GrowthTrend 一段时间的增长趋势时间序列
type GrowthTrend struct {
	Days int        `json:"days"` // 统计窗口天数（含今日）
	List []TrendDay `json:"list"` // 按日期升序排列，缺失日期以零值补齐
}

// HealthQuery 租户健康度列表查询参数
type HealthQuery struct {
	Page     int    // 页码，从 1 开始
	PageSize int    // 每页条数
	Keyword  string // 按租户名称模糊匹配
}

// TenantHealthItem 单个租户的健康度聚合结果
type TenantHealthItem struct {
	TenantID              string     `json:"tenant_id"`
	TenantName            string     `json:"tenant_name"`
	Status                int        `json:"status"`                  // 租户状态：1-启用 0-禁用
	PlanName              string     `json:"plan_name"`               // 当前套餐名称，无订阅时为空
	SubscriptionStatus    int        `json:"subscription_status"`     // 0-无订阅 1-生效 2-到期 3-取消
	SubscriptionExpiresAt *time.Time `json:"subscription_expires_at"` // 订阅到期时间，nil 表示长期有效
	DaysRemaining         int        `json:"days_remaining"`          // 订阅剩余天数，-1 表示长期有效
	UserCount             int64      `json:"user_count"`              // 用户总数
	ActiveUserCount       int64      `json:"active_user_count"`       // 有效用户数
	TaskCount             int64      `json:"task_count"`              // 任务总数
	TaskCompletedCount    int64      `json:"task_completed_count"`    // 已完成任务数
	WikiDocCount          int64      `json:"wiki_doc_count"`          // Wiki 文档总数
	StorageUsed           int64      `json:"storage_used"`            // 存储占用（字节）
	AuditCount7d          int64      `json:"audit_count_7d"`          // 近 7 天审计日志数
	LastActivityAt        *time.Time `json:"last_activity_at"`        // 最近活动时间
	HealthScore           int        `json:"health_score"`            // 健康度评分 0-100
	HealthLevel           string     `json:"health_level"`            // healthy/moderate/at_risk
}

// TenantHealthList 租户健康度列表（含分页与健康度分布统计）
type TenantHealthList struct {
	Items    []TenantHealthItem `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	Healthy  int64              `json:"healthy"`  // 健康租户数
	Moderate int64              `json:"moderate"` // 中等租户数
	AtRisk   int64              `json:"at_risk"`  // 风险租户数
}
