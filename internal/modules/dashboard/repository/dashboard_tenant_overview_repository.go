// Package repository 提供数据看板数据访问层
package repository

import (
	"context"
	"log/slog"
	"meteorx/internal/modules/dashboard/model"
	"time"

	"golang.org/x/sync/errgroup"
)

// GetTenantOverview 聚合单个租户的总览：成员/任务/Wiki/存储。
// 复用健康度的批量聚合助手（aggregateUsers/aggregateStorage）取租户级慢变事实，
// 任务分布、Wiki 空间/文档/浏览量以专用条件聚合补齐。
// 沿用容错策略：单维度查询失败仅记日志、按零值降级，不影响其余维度返回。
// 四个维度互不依赖，经 errgroup 并行下发；结果按租户短 TTL 缓存，命中直接返回。
// SQL 全部使用 COUNT/SUM 条件聚合，兼容 MySQL 与 SQLite。
func (r *dashboardRepository) GetTenantOverview(ctx context.Context, tenantID string) (*model.TenantOverview, error) {
	key := tenantOverviewCacheKey(tenantID)
	if cached, ok := cacheGet[model.TenantOverview](ctx, r.rdb, key); ok {
		return &cached, nil
	}

	ids := []string{tenantID}
	overview := &model.TenantOverview{}
	var g errgroup.Group

	// 成员：复用租户级用户聚合（total/active）
	g.Go(func() error {
		if u, ok := r.aggregateUsers(ctx, ids)[tenantID]; ok {
			overview.Member = model.TenantMemberOverview{Total: u.total, Active: u.active}
		}
		return nil
	})

	// 存储：复用租户级文件体积聚合
	g.Go(func() error {
		overview.StorageUsed = r.aggregateStorage(ctx, ids)[tenantID]
		return nil
	})

	// Wiki：文档数与浏览量合并为一次扫描，空间数在另一张表单独统计
	g.Go(func() error {
		overview.Wiki = r.aggregateTenantWiki(ctx, tenantID)
		return nil
	})

	// 任务：按状态条件聚合出分布 + 逾期，完成率在 Go 侧计算（保持与任务模块一致的 4 位精度）
	g.Go(func() error {
		overview.Task = r.aggregateTenantTasks(ctx, tenantID)
		return nil
	})
	_ = g.Wait()

	cacheSet(ctx, r.rdb, key, overview, r.cacheTTL)
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

// aggregateTenantWiki 一次扫描 wiki_documents 得到文档数与累计浏览量（COALESCE 保证空租户返回 0），
// 空间数在 wiki_spaces 单独统计；均排除软删除行。替代此前对 wiki_documents 的两次单遍扫描。
func (r *dashboardRepository) aggregateTenantWiki(ctx context.Context, tenantID string) model.TenantWikiOverview {
	// 空间数与文档/浏览量分属两张表，先取空间数，wiki_documents 查询失败不影响已得部分
	wiki := model.TenantWikiOverview{Spaces: r.countWikiSpaces(ctx, tenantID)}
	var row struct {
		Documents int64 `gorm:"column:documents"`
		Views     int64 `gorm:"column:views"`
	}
	err := r.db.WithContext(ctx).Table("wiki_documents").
		Select(`COUNT(*) AS documents, COALESCE(SUM(view_count), 0) AS views`).
		Where("tenant_id = ?", tenantID).
		Where("deleted_at IS NULL").
		Scan(&row).Error
	if err != nil {
		slog.Warn("Dashboard: tenant overview wiki aggregation failed", "error", err)
		return wiki
	}
	wiki.Documents = row.Documents
	wiki.Views = row.Views
	return wiki
}

// overviewCompletionRate 计算完成率（0~1，保留四位小数）；总数为 0 时返回 0。
func overviewCompletionRate(completed, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(int64(float64(completed)/float64(total)*10000+0.5)) / 10000
}
