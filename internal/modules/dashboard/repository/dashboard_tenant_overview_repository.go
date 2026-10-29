// Package repository 提供数据看板数据访问层
package repository

import (
	"context"
	"log/slog"
	"meteorx/internal/modules/dashboard/model"
	"time"
)

// GetTenantOverview 聚合单个租户的总览：成员/任务/Wiki/存储。
// 复用健康度的批量聚合助手（aggregateUsers/aggregateStorage/aggregateWikiDocs）取
// 租户级慢变事实，另以状态条件聚合补齐任务分布与 Wiki 空间/浏览量。
// 沿用容错策略：单维度查询失败仅记日志、按零值降级，不影响其余维度返回。
// SQL 全部使用 COUNT/SUM(CASE...) 条件聚合，兼容 MySQL 与 SQLite。
func (r *dashboardRepository) GetTenantOverview(ctx context.Context, tenantID string) (*model.TenantOverview, error) {
	ids := []string{tenantID}

	overview := &model.TenantOverview{}

	// 成员：复用租户级用户聚合（total/active）
	if u, ok := r.aggregateUsers(ctx, ids)[tenantID]; ok {
		overview.Member = model.TenantMemberOverview{Total: u.total, Active: u.active}
	}

	// 存储：复用租户级文件体积聚合
	overview.StorageUsed = r.aggregateStorage(ctx, ids)[tenantID]

	// Wiki：文档数复用聚合助手，空间数/浏览量单独统计
	overview.Wiki.Documents = r.aggregateWikiDocs(ctx, ids)[tenantID]
	overview.Wiki.Spaces = r.countWikiSpaces(ctx, tenantID)
	overview.Wiki.Views = r.sumWikiViews(ctx, tenantID)

	// 任务：按状态条件聚合出分布 + 逾期，完成率在 Go 侧计算（保持与任务模块一致的 4 位精度）
	overview.Task = r.aggregateTenantTasks(ctx, tenantID)

	return overview, nil
}

// aggregateTenantTasks 统计指定租户的任务状态分布与逾期量，并给出完成率。
func (r *dashboardRepository) aggregateTenantTasks(ctx context.Context, tenantID string) model.TenantTaskOverview {
	var row struct {
		Total      int64 `gorm:"column:total"`
		Pending    int64 `gorm:"column:pending"`
		InProgress int64 `gorm:"column:in_progress"`
		Completed  int64 `gorm:"column:completed"`
		Overdue    int64 `gorm:"column:overdue"`
	}
	err := r.db.WithContext(ctx).Table("tasks").
		Select(`COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) AS pending,
			COALESCE(SUM(CASE WHEN status = 'in_progress' THEN 1 ELSE 0 END), 0) AS in_progress,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) AS completed,
			COALESCE(SUM(CASE WHEN status <> 'completed' AND due_date IS NOT NULL AND due_date < ? THEN 1 ELSE 0 END), 0) AS overdue`,
			time.Now()).
		Where("tenant_id = ?", tenantID).
		Where("deleted_at IS NULL").
		Scan(&row).Error
	if err != nil {
		slog.Warn("Dashboard: tenant overview task aggregation failed", "error", err)
		return model.TenantTaskOverview{}
	}
	return model.TenantTaskOverview{
		Pending:        row.Pending,
		InProgress:     row.InProgress,
		Completed:      row.Completed,
		Overdue:        row.Overdue,
		Total:          row.Total,
		CompletionRate: overviewCompletionRate(row.Completed, row.Total),
	}
}

// countWikiSpaces 统计租户内未软删的知识空间数。
func (r *dashboardRepository) countWikiSpaces(ctx context.Context, tenantID string) int64 {
	var count int64
	err := r.db.WithContext(ctx).Table("wiki_spaces").
		Where("tenant_id = ?", tenantID).
		Where("deleted_at IS NULL").
		Count(&count).Error
	if err != nil {
		slog.Warn("Dashboard: tenant overview wiki space count failed", "error", err)
		return 0
	}
	return count
}

// sumWikiViews 汇总租户内未软删文档的累计浏览量。
func (r *dashboardRepository) sumWikiViews(ctx context.Context, tenantID string) int64 {
	var views int64
	err := r.db.WithContext(ctx).Table("wiki_documents").
		Select("COALESCE(SUM(view_count), 0)").
		Where("tenant_id = ?", tenantID).
		Where("deleted_at IS NULL").
		Scan(&views).Error
	if err != nil {
		slog.Warn("Dashboard: tenant overview wiki views aggregation failed", "error", err)
		return 0
	}
	return views
}

// overviewCompletionRate 计算完成率（0~1，保留四位小数）；总数为 0 时返回 0。
func overviewCompletionRate(completed, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(int64(float64(completed)/float64(total)*10000+0.5)) / 10000
}
