// Package dto 提供数据看板模块的数据传输对象
package dto

import "meteorx/internal/modules/dashboard/model"

// DashboardOverviewResp 数据看板总览响应
type DashboardOverviewResp struct {
	TenantStats       TenantStatsResp       `json:"tenant_stats"`
	UserStats         UserStatsResp         `json:"user_stats"`
	SubscriptionStats SubscriptionStatsResp `json:"subscription_stats"`
	AuditStats        AuditStatsResp        `json:"audit_stats"`
}

// TenantStatsResp 租户统计响应
type TenantStatsResp struct {
	Total    int64 `json:"total"`
	Enabled  int64 `json:"enabled"`
	Disabled int64 `json:"disabled"`
	TodayNew int64 `json:"today_new"`
	WeekNew  int64 `json:"week_new"`
	MonthNew int64 `json:"month_new"`
}

// UserStatsResp 用户统计响应
type UserStatsResp struct {
	Total    int64 `json:"total"`
	TodayNew int64 `json:"today_new"`
	WeekNew  int64 `json:"week_new"`
	MonthNew int64 `json:"month_new"`
}

// SubscriptionStatsResp 订阅统计响应
type SubscriptionStatsResp struct {
	Total        int64 `json:"total"`
	Active       int64 `json:"active"`
	Expired      int64 `json:"expired"`
	Cancelled    int64 `json:"cancelled"`
	ActiveTenant int64 `json:"active_tenant"`
}

// AuditStatsResp 审计统计响应
type AuditStatsResp struct {
	Total       int64            `json:"total"`
	Today       int64            `json:"today"`
	Success     int64            `json:"success"`
	Failure     int64            `json:"failure"`
	ActionStats map[string]int64 `json:"action_stats"`
	ModuleStats map[string]int64 `json:"module_stats"`
}

// ToDashboardOverviewResp 将领域模型转换为响应 DTO
func ToDashboardOverviewResp(m *model.DashboardOverview) *DashboardOverviewResp {
	return &DashboardOverviewResp{
		TenantStats: TenantStatsResp{
			Total:    m.TenantStats.Total,
			Enabled:  m.TenantStats.Enabled,
			Disabled: m.TenantStats.Disabled,
			TodayNew: m.TenantStats.TodayNew,
			WeekNew:  m.TenantStats.WeekNew,
			MonthNew: m.TenantStats.MonthNew,
		},
		UserStats: UserStatsResp{
			Total:    m.UserStats.Total,
			TodayNew: m.UserStats.TodayNew,
			WeekNew:  m.UserStats.WeekNew,
			MonthNew: m.UserStats.MonthNew,
		},
		SubscriptionStats: SubscriptionStatsResp{
			Total:        m.SubscriptionStats.Total,
			Active:       m.SubscriptionStats.Active,
			Expired:      m.SubscriptionStats.Expired,
			Cancelled:    m.SubscriptionStats.Cancelled,
			ActiveTenant: m.SubscriptionStats.ActiveTenant,
		},
		AuditStats: AuditStatsResp{
			Total:       m.AuditStats.Total,
			Today:       m.AuditStats.Today,
			Success:     m.AuditStats.Success,
			Failure:     m.AuditStats.Failure,
			ActionStats: m.AuditStats.ActionStats,
			ModuleStats: m.AuditStats.ModuleStats,
		},
	}
}
