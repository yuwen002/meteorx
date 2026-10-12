// Package repository 提供数据看板数据访问层
package repository

import (
	"context"
	"log/slog"
	"meteorx/internal/modules/dashboard/model"
	"meteorx/pkg/timeutil"
	"time"

	"gorm.io/gorm"
)

// healthAuditWindow 健康度活跃度统计窗口（近 7 天）
const healthAuditWindow = 7 * 24 * time.Hour

// GetTenantHealth 按租户聚合健康度（分页）。
// 先按租户分页取基集，再以 IN + GROUP BY tenant_id 批量聚合各资源维度，
// 避免 N+1 查询；单张聚合表失败时记录日志并按零值降级，不影响其余维度。
func (r *dashboardRepository) GetTenantHealth(ctx context.Context, query model.HealthQuery) (*model.TenantHealthList, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	size := query.PageSize
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	db := r.db.WithContext(ctx)

	// 1) 租户基集：软删除过滤 + 关键字匹配 + 分页
	newBase := func() *gorm.DB {
		q := db.Table("tenants").Where("deleted_at IS NULL")
		if query.Keyword != "" {
			q = q.Where("name LIKE ?", "%"+query.Keyword+"%")
		}
		return q
	}

	var total int64
	if err := newBase().Count(&total).Error; err != nil {
		return nil, err
	}

	type tenantRow struct {
		ID     string  `gorm:"column:id"`
		Name   *string `gorm:"column:name"`
		Status int     `gorm:"column:status"`
	}
	var tenants []tenantRow
	if err := newBase().Select("id, name, status").
		Order("created_at DESC").
		Offset((page - 1) * size).
		Limit(size).
		Scan(&tenants).Error; err != nil {
		return nil, err
	}

	result := &model.TenantHealthList{
		Items:    make([]model.TenantHealthItem, 0, len(tenants)),
		Total:    total,
		Page:     page,
		PageSize: size,
	}
	if len(tenants) == 0 {
		return result, nil
	}

	ids := make([]string, 0, len(tenants))
	for _, t := range tenants {
		ids = append(ids, t.ID)
	}

	// 2) 各维度批量聚合
	now := time.Now()
	auditSince := now.Add(-healthAuditWindow)
	userAgg := r.aggregateUsers(ctx, ids)
	taskAgg := r.aggregateTasks(ctx, ids)
	wikiAgg := r.aggregateWikiDocs(ctx, ids)
	storageAgg := r.aggregateStorage(ctx, ids)
	auditAgg := r.aggregateAudit(ctx, ids, auditSince)
	subAgg := r.aggregateSubscription(ctx, ids)

	// 3) 组装 + 评分
	for _, t := range tenants {
		item := model.TenantHealthItem{
			TenantID:   t.ID,
			TenantName: strVal(t.Name),
			Status:     t.Status,
		}
		if u, ok := userAgg[t.ID]; ok {
			item.UserCount = u.total
			item.ActiveUserCount = u.active
		}
		if tk, ok := taskAgg[t.ID]; ok {
			item.TaskCount = tk.total
			item.TaskCompletedCount = tk.completed
		}
		item.WikiDocCount = wikiAgg[t.ID]
		item.StorageUsed = storageAgg[t.ID]
		if a, ok := auditAgg[t.ID]; ok {
			item.AuditCount7d = a.count7d
			if !a.lastAt.IsZero() {
				la := a.lastAt
				item.LastActivityAt = &la
			}
		}
		if s, ok := subAgg[t.ID]; ok {
			item.PlanName = s.planName
			item.SubscriptionStatus = s.status
			item.SubscriptionExpiresAt = s.expiresAt
			item.DaysRemaining = daysRemaining(s.expiresAt, now)
		}

		item.HealthScore, item.HealthLevel = scoreTenantHealth(item)
		switch item.HealthLevel {
		case "healthy":
			result.Healthy++
		case "moderate":
			result.Moderate++
		default:
			result.AtRisk++
		}
		result.Items = append(result.Items, item)
	}

	return result, nil
}

// ---------- 各维度聚合 ----------

type userAggRow struct {
	TenantID string `gorm:"column:tenant_id"`
	Total    int64  `gorm:"column:total"`
	Active   int64  `gorm:"column:active"`
}

type userAgg struct{ total, active int64 }

func (r *dashboardRepository) aggregateUsers(ctx context.Context, ids []string) map[string]userAgg {
	out := make(map[string]userAgg, len(ids))
	var rows []userAggRow
	err := r.db.WithContext(ctx).Table("users").
		Select("tenant_id, COUNT(*) as total, SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) as active").
		Where("tenant_id IN ?", ids).
		Where("deleted_at IS NULL").
		Group("tenant_id").
		Scan(&rows).Error
	if err != nil {
		slog.Warn("Dashboard: tenant health user aggregation failed", "error", err)
		return out
	}
	for _, row := range rows {
		out[row.TenantID] = userAgg{total: row.Total, active: row.Active}
	}
	return out
}

type taskAggRow struct {
	TenantID  string `gorm:"column:tenant_id"`
	Total     int64  `gorm:"column:total"`
	Completed int64  `gorm:"column:completed"`
}

type taskAgg struct{ total, completed int64 }

func (r *dashboardRepository) aggregateTasks(ctx context.Context, ids []string) map[string]taskAgg {
	out := make(map[string]taskAgg, len(ids))
	var rows []taskAggRow
	err := r.db.WithContext(ctx).Table("tasks").
		Select("tenant_id, COUNT(*) as total, SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed").
		Where("tenant_id IN ?", ids).
		Where("deleted_at IS NULL").
		Group("tenant_id").
		Scan(&rows).Error
	if err != nil {
		slog.Warn("Dashboard: tenant health task aggregation failed", "error", err)
		return out
	}
	for _, row := range rows {
		out[row.TenantID] = taskAgg{total: row.Total, completed: row.Completed}
	}
	return out
}

func (r *dashboardRepository) aggregateWikiDocs(ctx context.Context, ids []string) map[string]int64 {
	out := make(map[string]int64, len(ids))
	type row struct {
		TenantID string `gorm:"column:tenant_id"`
		Total    int64  `gorm:"column:total"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("wiki_documents").
		Select("tenant_id, COUNT(*) as total").
		Where("tenant_id IN ?", ids).
		Where("deleted_at IS NULL").
		Group("tenant_id").
		Scan(&rows).Error
	if err != nil {
		slog.Warn("Dashboard: tenant health wiki aggregation failed", "error", err)
		return out
	}
	for _, r := range rows {
		out[r.TenantID] = r.Total
	}
	return out
}

func (r *dashboardRepository) aggregateStorage(ctx context.Context, ids []string) map[string]int64 {
	out := make(map[string]int64, len(ids))
	type row struct {
		TenantID string `gorm:"column:tenant_id"`
		Used     int64  `gorm:"column:used"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Table("files").
		Select("tenant_id, COALESCE(SUM(file_size), 0) as used").
		Where("tenant_id IN ?", ids).
		Where("deleted_at IS NULL").
		Group("tenant_id").
		Scan(&rows).Error
	if err != nil {
		slog.Warn("Dashboard: tenant health storage aggregation failed", "error", err)
		return out
	}
	for _, r := range rows {
		out[r.TenantID] = r.Used
	}
	return out
}

type auditAgg struct {
	count7d int64
	lastAt  time.Time
}

// aggregateAudit 统计各租户近 7 天审计量与最近活动时间。
// 因 MAX(created_at) 在不同驱动下返回类型不一致（MySQL/Postgres 为 time.Time，
// SQLite 为无类型文本），改用 Rows 手动扫描到 any 再归一化，保证跨库兼容。
func (r *dashboardRepository) aggregateAudit(ctx context.Context, ids []string, since time.Time) map[string]auditAgg {
	out := make(map[string]auditAgg, len(ids))
	rows, err := r.db.WithContext(ctx).Table("audit_logs").
		Select("tenant_id, SUM(CASE WHEN created_at >= ? THEN 1 ELSE 0 END) as count7d, MAX(created_at) as last_at", since).
		Where("tenant_id IN ?", ids).
		Group("tenant_id").
		Rows()
	if err != nil {
		slog.Warn("Dashboard: tenant health audit aggregation failed", "error", err)
		return out
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			tenantID string
			count7d  int64
			lastRaw  any
		)
		if err := rows.Scan(&tenantID, &count7d, &lastRaw); err != nil {
			slog.Warn("Dashboard: tenant health audit scan failed", "error", err)
			return out
		}
		a := auditAgg{count7d: count7d}
		if ts, ok := scanTime(lastRaw); ok {
			a.lastAt = ts
		}
		out[tenantID] = a
	}
	return out
}

// scanTime 将驱动返回的聚合时间列（time.Time / string / []byte / nil）归一化为 time.Time。
func scanTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			return time.Time{}, false
		}
		return t, true
	case *time.Time:
		if t == nil || t.IsZero() {
			return time.Time{}, false
		}
		return *t, true
	case string:
		return parseFlexTime(t)
	case []byte:
		return parseFlexTime(string(t))
	default:
		return time.Time{}, false
	}
}

// parseFlexTime 尝试以多种常见布局解析时间字符串（兼容 SQLite 无类型文本）。
func parseFlexTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}
	for _, l := range layouts {
		if ts, err := time.Parse(l, s); err == nil {
			return ts, true
		}
	}
	return time.Time{}, false
}

type subAggRow struct {
	TenantID  string     `gorm:"column:tenant_id"`
	Status    int        `gorm:"column:status"`
	ExpiresAt *time.Time `gorm:"column:expires_at"`
	PlanName  *string    `gorm:"column:plan_name"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

type subAgg struct {
	status    int
	expiresAt *time.Time
	planName  string
}

// aggregateSubscription 为每个租户挑选当前订阅：优先生效(status=1)，其次按创建时间最新。
func (r *dashboardRepository) aggregateSubscription(ctx context.Context, ids []string) map[string]subAgg {
	out := make(map[string]subAgg, len(ids))
	var rows []subAggRow
	err := r.db.WithContext(ctx).Table("tenant_subscriptions").
		Select("tenant_subscriptions.tenant_id as tenant_id, tenant_subscriptions.status as status, tenant_subscriptions.expires_at as expires_at, tenant_subscriptions.created_at as created_at, tenant_plans.name as plan_name").
		Joins("LEFT JOIN tenant_plans ON tenant_plans.id = tenant_subscriptions.plan_id").
		Where("tenant_subscriptions.tenant_id IN ?", ids).
		Where("tenant_subscriptions.deleted_at IS NULL").
		Order("tenant_subscriptions.status ASC, tenant_subscriptions.created_at DESC").
		Scan(&rows).Error
	if err != nil {
		slog.Warn("Dashboard: tenant health subscription aggregation failed", "error", err)
		return out
	}
	// 结果已按 status 升序、created_at 降序排列，每个租户首个即当前订阅
	for _, row := range rows {
		if _, ok := out[row.TenantID]; ok {
			continue
		}
		s := subAgg{status: row.Status, expiresAt: row.ExpiresAt}
		if row.PlanName != nil {
			s.planName = *row.PlanName
		}
		out[row.TenantID] = s
	}
	return out
}

// ---------- 评分与工具 ----------

// daysRemaining 计算订阅剩余天数；expiresAt 为 nil 表示长期有效返回 -1；已过期返回 0。
func daysRemaining(expiresAt *time.Time, now time.Time) int {
	if expiresAt == nil {
		return -1
	}
	d := int(timeutil.StartOfDay(*expiresAt).Sub(timeutil.StartOfDay(now)).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}

// scoreTenantHealth 依据订阅、近 7 天活跃度与资源足迹计算 0-100 健康度评分与分级。
// 订阅（满分 40）：生效 40 / 到期 15 / 取消或无订阅 0；
// 活跃度（满分 30）：30 * min(1, 近7天审计数 / 20)；
// 资源足迹（满分 30）：30 * min(1, (用户+任务+文档) / 100)。
func scoreTenantHealth(item model.TenantHealthItem) (int, string) {
	score := 0

	switch item.SubscriptionStatus {
	case 1:
		score += 40
	case 2:
		score += 15
	default:
		// 取消(3)或无订阅(0) 记 0 分
	}

	score += int(30 * ratio(float64(item.AuditCount7d), 20))

	footprint := float64(item.UserCount + item.TaskCount + item.WikiDocCount)
	score += int(30 * ratio(footprint, 100))

	if score > 100 {
		score = 100
	} else if score < 0 {
		score = 0
	}

	var level string
	switch {
	case score >= 70:
		level = "healthy"
	case score >= 40:
		level = "moderate"
	default:
		level = "at_risk"
	}
	return score, level
}

// ratio 计算 value/limit 并钳制到 [0,1]。
func ratio(value, limit float64) float64 {
	if limit <= 0 {
		return 0
	}
	r := value / limit
	if r > 1 {
		return 1
	}
	if r < 0 {
		return 0
	}
	return r
}

// strVal 安全取出字符串指针值。
func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
