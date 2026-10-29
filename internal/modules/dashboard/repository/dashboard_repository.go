// Package repository 提供数据看板数据访问层
package repository

import (
	"context"
	"log/slog"
	"meteorx/internal/cache"
	"meteorx/internal/modules/dashboard/model"
	"meteorx/pkg/timeutil"
	"time"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// DashboardRepository 数据看板仓库接口
type DashboardRepository interface {
	// GetOverview 获取运营数据总览
	GetOverview(ctx context.Context) (*model.DashboardOverview, error)
	// GetTenantHealth 按租户聚合健康度（分页）
	GetTenantHealth(ctx context.Context, query model.HealthQuery) (*model.TenantHealthList, error)
	// GetTenantHealthDetail 获取单个租户健康度详情
	GetTenantHealthDetail(ctx context.Context, tenantID string) (*model.TenantHealthItem, error)
	// GetTrend 获取最近 days 天的增长趋势时间序列（按自然日分桶）
	GetTrend(ctx context.Context, days int) (*model.GrowthTrend, error)
	// GetTenantOverview 聚合单个租户的总览（成员/任务/Wiki/存储），供租户工作台首屏
	GetTenantOverview(ctx context.Context, tenantID string) (*model.TenantOverview, error)
}

// trendMaxDays 增长趋势查询的最大天数上限，避免一次性拉取过大区间压库。
const trendMaxDays = 365

// dashboardRepository 数据看板仓库实现
type dashboardRepository struct {
	db *gorm.DB
	// rdb 为可选的 Redis 缓存客户端，nil 或不可用时缓存路径整体降级为直接查库；
	// cacheTTL 为聚合结果短缓存时长，仅经 WithCache 注入后置为正值。
	rdb      *cache.Redis
	cacheTTL time.Duration
}

// NewDashboardRepository 创建数据看板仓库。
// 可变选项（如 WithCache）用于注入可选依赖，不传时行为与旧签名一致，保持既有调用点/测试向后兼容。
func NewDashboardRepository(db *gorm.DB, opts ...RepoOption) DashboardRepository {
	r := &dashboardRepository{db: db}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// GetOverview 获取运营数据总览
// 采用容错策略：单个模块查询失败不影响其他模块，失败时返回零值并记录日志。
// 四个维度是彼此独立的聚合，经 errgroup 并行下发；结果按短 TTL 缓存，命中直接返回。
func (r *dashboardRepository) GetOverview(ctx context.Context) (*model.DashboardOverview, error) {
	if cached, ok := cacheGet[model.DashboardOverview](ctx, r.rdb, cacheKeyOverview); ok {
		return &cached, nil
	}

	overview := &model.DashboardOverview{}
	var g errgroup.Group
	g.Go(func() error {
		stats, err := r.getTenantStats(ctx)
		if err != nil {
			slog.Warn("Dashboard: tenant stats query failed", "error", err)
			return nil
		}
		overview.TenantStats = stats
		return nil
	})
	g.Go(func() error {
		stats, err := r.getUserStats(ctx)
		if err != nil {
			slog.Warn("Dashboard: user stats query failed", "error", err)
			return nil
		}
		overview.UserStats = stats
		return nil
	})
	g.Go(func() error {
		stats, err := r.getSubscriptionStats(ctx)
		if err != nil {
			slog.Warn("Dashboard: subscription stats query failed", "error", err)
			return nil
		}
		overview.SubscriptionStats = stats
		return nil
	})
	g.Go(func() error {
		stats, err := r.getAuditStats(ctx)
		if err != nil {
			slog.Warn("Dashboard: audit stats query failed", "error", err)
			return nil
		}
		overview.AuditStats = stats
		return nil
	})
	_ = g.Wait()

	cacheSet(ctx, r.rdb, cacheKeyOverview, overview, r.cacheTTL)
	return overview, nil
}

// getTenantStats 租户统计（基于 tenants 表）
func (r *dashboardRepository) getTenantStats(ctx context.Context) (model.TenantStats, error) {
	var total, enabled, disabled, todayNew, weekNew, monthNew int64
	db := r.db.WithContext(ctx)

	now := time.Now()
	// 统一使用业务时区的自然日/月边界，避免 Truncate(24h) 在 UTC 截断导致的跨时区偏差
	startToday := timeutil.StartOfDay(now)
	startWeek := timeutil.StartOfDay(now.AddDate(0, 0, -6))
	startMonth := timeutil.StartOfMonth(now)

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

	now := time.Now()
	// 统一使用业务时区的自然日/月边界，避免 Truncate(24h) 在 UTC 截断导致的跨时区偏差
	startToday := timeutil.StartOfDay(now)
	startWeek := timeutil.StartOfDay(now.AddDate(0, 0, -6))
	startMonth := timeutil.StartOfMonth(now)

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

	startToday := timeutil.StartOfDay(time.Now())

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

// GetTrend 获取最近 days 天的增长趋势时间序列
// 以今日为终点向前回推 days-1 天构成闭区间，逐日按 created_at 分桶统计新增量；
// 沿用 GetOverview 的容错策略：单表查询失败仅记日志、该指标置零，不阻断整体返回。
func (r *dashboardRepository) GetTrend(ctx context.Context, days int) (*model.GrowthTrend, error) {
	if days <= 0 {
		days = 30
	}
	if days > trendMaxDays {
		days = trendMaxDays
	}

	now := time.Now()
	start := timeutil.StartOfDay(now).AddDate(0, 0, -(days - 1))

	tenantMap, err := r.countByDay(ctx, "tenants", start, true)
	if err != nil {
		slog.Warn("Dashboard: tenant trend query failed", "error", err)
	}
	userMap, err := r.countByDay(ctx, "users", start, true)
	if err != nil {
		slog.Warn("Dashboard: user trend query failed", "error", err)
	}
	auditMap, err := r.countByDay(ctx, "audit_logs", start, false)
	if err != nil {
		slog.Warn("Dashboard: audit trend query failed", "error", err)
	}

	list := make([]model.TrendDay, days)
	for i := 0; i < days; i++ {
		day := start.AddDate(0, 0, i)
		key := day.Format("2006-01-02")
		list[i] = model.TrendDay{
			Date:       key,
			NewTenants: tenantMap[key],
			NewUsers:   userMap[key],
			AuditLogs:  auditMap[key],
		}
	}
	return &model.GrowthTrend{Days: days, List: list}, nil
}

// countByDay 按自然日分组统计指定表在 since 之后的记录数，返回 map["2006-01-02"]count。
// withDeleted 为 true 时附加 deleted_at IS NULL，软删除记录不计入增长。
func (r *dashboardRepository) countByDay(ctx context.Context, table string, since time.Time, withDeleted bool) (map[string]int64, error) {
	type row struct {
		Day   string `gorm:"column:day"`
		Count int64  `gorm:"column:count"`
	}
	query := r.db.WithContext(ctx).Table(table).
		Select("DATE(created_at) as day, COUNT(*) as count").
		Where("created_at >= ?", since)
	if withDeleted {
		query = query.Where("deleted_at IS NULL")
	}

	var rows []row
	if err := query.Group("DATE(created_at)").Scan(&rows).Error; err != nil {
		return nil, err
	}

	stats := make(map[string]int64, len(rows))
	for _, rr := range rows {
		stats[rr.Day] = rr.Count
	}
	return stats, nil
}
