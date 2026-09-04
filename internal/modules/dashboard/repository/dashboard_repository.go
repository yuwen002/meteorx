// Package repository 提供数据看板数据访问层
package repository

import (
	"context"
	"log/slog"
	"meteorx/internal/modules/dashboard/model"
	"time"

	"gorm.io/gorm"
)

// DashboardRepository 数据看板仓库接口
type DashboardRepository interface {
	// GetOverview 获取运营数据总览
	GetOverview(ctx context.Context) (*model.DashboardOverview, error)
}

// dashboardRepository 数据看板仓库实现
type dashboardRepository struct {
	db *gorm.DB
}

// NewDashboardRepository 创建数据看板仓库
func NewDashboardRepository(db *gorm.DB) DashboardRepository {
	return &dashboardRepository{db: db}
}

// GetOverview 获取运营数据总览
// 采用容错策略：单个模块查询失败不影响其他模块，失败时返回零值并记录日志
func (r *dashboardRepository) GetOverview(ctx context.Context) (*model.DashboardOverview, error) {
	overview := &model.DashboardOverview{}

	var err error

	overview.TenantStats, err = r.getTenantStats(ctx)
	if err != nil {
		slog.Warn("Dashboard: tenant stats query failed", "error", err)
	}

	overview.UserStats, err = r.getUserStats(ctx)
	if err != nil {
		slog.Warn("Dashboard: user stats query failed", "error", err)
	}

	overview.SubscriptionStats, err = r.getSubscriptionStats(ctx)
	if err != nil {
		slog.Warn("Dashboard: subscription stats query failed", "error", err)
	}

	overview.AuditStats, err = r.getAuditStats(ctx)
	if err != nil {
		slog.Warn("Dashboard: audit stats query failed", "error", err)
	}

	return overview, nil
}

// getTenantStats 租户统计（基于 tenants 表）
func (r *dashboardRepository) getTenantStats(ctx context.Context) (model.TenantStats, error) {
	var total, enabled, disabled, todayNew, weekNew, monthNew int64
	db := r.db.WithContext(ctx)

	startToday := time.Now().Truncate(24 * time.Hour)
	startWeek := time.Now().AddDate(0, 0, -6)
	startWeek = time.Date(startWeek.Year(), startWeek.Month(), startWeek.Day(), 0, 0, 0, 0, startWeek.Location())
	startMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Now().Location())

	if err := db.Table("tenants").Count(&total).Error; err != nil {
		return model.TenantStats{}, err
	}
	if err := db.Table("tenants").Where("status = ?", 1).Count(&enabled).Error; err != nil {
		return model.TenantStats{}, err
	}
	if err := db.Table("tenants").Where("status = ?", 0).Count(&disabled).Error; err != nil {
		return model.TenantStats{}, err
	}
	if err := db.Table("tenants").Where("created_at >= ?", startToday).Count(&todayNew).Error; err != nil {
		return model.TenantStats{}, err
	}
	if err := db.Table("tenants").Where("created_at >= ?", startWeek).Count(&weekNew).Error; err != nil {
		return model.TenantStats{}, err
	}
	if err := db.Table("tenants").Where("created_at >= ?", startMonth).Count(&monthNew).Error; err != nil {
		return model.TenantStats{}, err
	}

	return model.TenantStats{
		Total:    total,
		Enabled:  enabled,
		Disabled: disabled,
		TodayNew: todayNew,
		WeekNew:  weekNew,
		MonthNew: monthNew,
	}, nil
}

// getUserStats 用户统计（基于 users 表）
func (r *dashboardRepository) getUserStats(ctx context.Context) (model.UserStats, error) {
	var total, todayNew, weekNew, monthNew int64
	db := r.db.WithContext(ctx)

	startToday := time.Now().Truncate(24 * time.Hour)
	startWeek := time.Now().AddDate(0, 0, -6)
	startWeek = time.Date(startWeek.Year(), startWeek.Month(), startWeek.Day(), 0, 0, 0, 0, startWeek.Location())
	startMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Now().Location())

	if err := db.Table("users").Count(&total).Error; err != nil {
		return model.UserStats{}, err
	}
	if err := db.Table("users").Where("created_at >= ?", startToday).Count(&todayNew).Error; err != nil {
		return model.UserStats{}, err
	}
	if err := db.Table("users").Where("created_at >= ?", startWeek).Count(&weekNew).Error; err != nil {
		return model.UserStats{}, err
	}
	if err := db.Table("users").Where("created_at >= ?", startMonth).Count(&monthNew).Error; err != nil {
		return model.UserStats{}, err
	}

	return model.UserStats{
		Total:    total,
		TodayNew: todayNew,
		WeekNew:  weekNew,
		MonthNew: monthNew,
	}, nil
}

// getSubscriptionStats 订阅统计（基于 tenant_subscriptions 表）
func (r *dashboardRepository) getSubscriptionStats(ctx context.Context) (model.SubscriptionStats, error) {
	var total, active, expired, cancelled, activeTenant int64
	db := r.db.WithContext(ctx)

	if err := db.Table("tenant_subscriptions").Count(&total).Error; err != nil {
		return model.SubscriptionStats{}, err
	}
	if err := db.Table("tenant_subscriptions").Where("status = ?", 1).Count(&active).Error; err != nil {
		return model.SubscriptionStats{}, err
	}
	if err := db.Table("tenant_subscriptions").Where("status = ?", 2).Count(&expired).Error; err != nil {
		return model.SubscriptionStats{}, err
	}
	if err := db.Table("tenant_subscriptions").Where("status = ?", 3).Count(&cancelled).Error; err != nil {
		return model.SubscriptionStats{}, err
	}
	if err := db.Table("tenant_subscriptions").
		Where("status = ?", 1).
		Distinct("tenant_id").
		Count(&activeTenant).Error; err != nil {
		return model.SubscriptionStats{}, err
	}

	return model.SubscriptionStats{
		Total:        total,
		Active:       active,
		Expired:      expired,
		Cancelled:    cancelled,
		ActiveTenant: activeTenant,
	}, nil
}

// getAuditStats 审计统计（基于 audit_logs 表）
func (r *dashboardRepository) getAuditStats(ctx context.Context) (model.AuditStats, error) {
	var total, today, success, failure int64
	db := r.db.WithContext(ctx)

	startToday := time.Now().Truncate(24 * time.Hour)

	if err := db.Table("audit_logs").Count(&total).Error; err != nil {
		return model.AuditStats{}, err
	}
	if err := db.Table("audit_logs").Where("created_at >= ?", startToday).Count(&today).Error; err != nil {
		return model.AuditStats{}, err
	}
	if err := db.Table("audit_logs").Where("result = ?", "success").Count(&success).Error; err != nil {
		return model.AuditStats{}, err
	}
	if err := db.Table("audit_logs").Where("result = ?", "failure").Count(&failure).Error; err != nil {
		return model.AuditStats{}, err
	}

	actionStats, err := r.getGroupStats(ctx, "action")
	if err != nil {
		return model.AuditStats{}, err
	}
	moduleStats, err := r.getGroupStats(ctx, "module")
	if err != nil {
		return model.AuditStats{}, err
	}

	return model.AuditStats{
		Total:       total,
		Today:       today,
		Success:     success,
		Failure:     failure,
		ActionStats: actionStats,
		ModuleStats: moduleStats,
	}, nil
}

// getGroupStats 按指定字段分组统计
func (r *dashboardRepository) getGroupStats(ctx context.Context, column string) (map[string]int64, error) {
	type row struct {
		Key   string `gorm:"column:key"`
		Count int64  `gorm:"column:count"`
	}
	var rows []row
	if err := r.db.WithContext(ctx).Table("audit_logs").
		Select(column + " as key, COUNT(*) as count").
		Group(column).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	stats := make(map[string]int64, len(rows))
	for _, r := range rows {
		stats[r.Key] = r.Count
	}
	return stats, nil
}
