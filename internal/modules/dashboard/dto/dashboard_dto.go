// Package dto 提供数据看板模块的数据传输对象
package dto

import (
	"time"

	"meteorx/internal/modules/dashboard/model"
)

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

// TenantHealthItemResp 单个租户健康度响应
type TenantHealthItemResp struct {
	TenantID              string     `json:"tenant_id"`
	TenantName            string     `json:"tenant_name"`
	Status                int        `json:"status"`
	PlanName              string     `json:"plan_name"`
	SubscriptionStatus    int        `json:"subscription_status"`
	SubscriptionExpiresAt *time.Time `json:"subscription_expires_at"`
	DaysRemaining         int        `json:"days_remaining"`
	UserCount             int64      `json:"user_count"`
	ActiveUserCount       int64      `json:"active_user_count"`
	TaskCount             int64      `json:"task_count"`
	TaskCompletedCount    int64      `json:"task_completed_count"`
	WikiDocCount          int64      `json:"wiki_doc_count"`
	StorageUsed           int64      `json:"storage_used"`
	AuditCount7d          int64      `json:"audit_count_7d"`
	LastActivityAt        *time.Time `json:"last_activity_at"`
	HealthScore           int        `json:"health_score"`
	HealthLevel           string     `json:"health_level"`
}

// TenantHealthListResp 租户健康度列表响应
type TenantHealthListResp struct {
	Items    []TenantHealthItemResp `json:"items"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
	Healthy  int64                  `json:"healthy"`
	Moderate int64                  `json:"moderate"`
	AtRisk   int64                  `json:"at_risk"`
}

// ToTenantHealthListResp 将领域模型转换为健康度列表响应 DTO
func ToTenantHealthListResp(m *model.TenantHealthList) *TenantHealthListResp {
	if m == nil {
		return nil
	}
	items := make([]TenantHealthItemResp, 0, len(m.Items))
	for _, it := range m.Items {
		items = append(items, ToTenantHealthItemResp(it))
	}
	return &TenantHealthListResp{
		Items:    items,
		Total:    m.Total,
		Page:     m.Page,
		PageSize: m.PageSize,
		Healthy:  m.Healthy,
		Moderate: m.Moderate,
		AtRisk:   m.AtRisk,
	}
}

// ToTenantHealthItemResp 将单个租户健康度领域模型转换为响应 DTO
func ToTenantHealthItemResp(it model.TenantHealthItem) TenantHealthItemResp {
	return TenantHealthItemResp{
		TenantID:              it.TenantID,
		TenantName:            it.TenantName,
		Status:                it.Status,
		PlanName:              it.PlanName,
		SubscriptionStatus:    it.SubscriptionStatus,
		SubscriptionExpiresAt: it.SubscriptionExpiresAt,
		DaysRemaining:         it.DaysRemaining,
		UserCount:             it.UserCount,
		ActiveUserCount:       it.ActiveUserCount,
		TaskCount:             it.TaskCount,
		TaskCompletedCount:    it.TaskCompletedCount,
		WikiDocCount:          it.WikiDocCount,
		StorageUsed:           it.StorageUsed,
		AuditCount7d:          it.AuditCount7d,
		LastActivityAt:        it.LastActivityAt,
		HealthScore:           it.HealthScore,
		HealthLevel:           it.HealthLevel,
	}
}
