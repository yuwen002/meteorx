package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"meteorx/internal/modules/dashboard/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// newHealthTestDB 建内存 SQLite 并创建健康度聚合涉及的全部表
func newHealthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)

	stmts := []string{
		`CREATE TABLE tenants (id TEXT PRIMARY KEY, name TEXT, status INTEGER, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE users (id TEXT PRIMARY KEY, tenant_id TEXT, status INTEGER, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE tasks (id TEXT PRIMARY KEY, tenant_id TEXT, status TEXT, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE wiki_documents (id TEXT PRIMARY KEY, tenant_id TEXT, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE files (id TEXT PRIMARY KEY, tenant_id TEXT, file_size INTEGER, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE audit_logs (id INTEGER PRIMARY KEY AUTOINCREMENT, tenant_id TEXT, created_at DATETIME)`,
		`CREATE TABLE tenant_plans (id TEXT PRIMARY KEY, name TEXT, code TEXT, deleted_at DATETIME)`,
		`CREATE TABLE tenant_subscriptions (id TEXT PRIMARY KEY, tenant_id TEXT, plan_id TEXT, status INTEGER, started_at DATETIME, expires_at DATETIME, created_at DATETIME, deleted_at DATETIME)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

func seedHealthData(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Now()

	// 两个租户
	require.NoError(t, db.Exec(`INSERT INTO tenants (id, name, status, created_at) VALUES (?, ?, ?, ?)`, "tA", "Alpha Corp", 1, now.Add(-time.Hour)).Error)
	require.NoError(t, db.Exec(`INSERT INTO tenants (id, name, status, created_at) VALUES (?, ?, ?, ?)`, "tB", "Beta Inc", 1, now).Error)

	// tA 用户：1 有效 + 1 停用 + 1 软删（应被排除）
	require.NoError(t, db.Exec(`INSERT INTO users (id, tenant_id, status, created_at) VALUES ('u1','tA',1,?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO users (id, tenant_id, status, created_at) VALUES ('u2','tA',0,?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO users (id, tenant_id, status, created_at, deleted_at) VALUES ('u3','tA',1,?,?)`, now, now).Error)
	// tB 用户：仅 1 停用
	require.NoError(t, db.Exec(`INSERT INTO users (id, tenant_id, status, created_at) VALUES ('u4','tB',0,?)`, now).Error)

	// tA 任务：2 completed + 1 pending
	require.NoError(t, db.Exec(`INSERT INTO tasks (id, tenant_id, status, created_at) VALUES ('tk1','tA','completed',?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO tasks (id, tenant_id, status, created_at) VALUES ('tk2','tA','completed',?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO tasks (id, tenant_id, status, created_at) VALUES ('tk3','tA','pending',?)`, now).Error)

	// tA wiki：4 篇 + 1 软删
	for i := 0; i < 4; i++ {
		require.NoError(t, db.Exec(`INSERT INTO wiki_documents (id, tenant_id, created_at) VALUES (?, 'tA', ?)`, "w"+string(rune('a'+i)), now).Error)
	}
	require.NoError(t, db.Exec(`INSERT INTO wiki_documents (id, tenant_id, created_at, deleted_at) VALUES ('wx','tA',?,?)`, now, now).Error)

	// tA 文件：1000 + 2000 有效，5000 软删（不计入存储）
	require.NoError(t, db.Exec(`INSERT INTO files (id, tenant_id, file_size, created_at) VALUES ('f1','tA',1000,?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO files (id, tenant_id, file_size, created_at) VALUES ('f2','tA',2000,?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO files (id, tenant_id, file_size, created_at, deleted_at) VALUES ('f3','tA',5000,?,?)`, now, now).Error)

	// tA 审计：近 7 天 25 条 + 1 条久远（不计入 7d，但影响最近活动时间）
	for i := 0; i < 25; i++ {
		require.NoError(t, db.Exec(`INSERT INTO audit_logs (tenant_id, created_at) VALUES ('tA', ?)`, now).Error)
	}
	require.NoError(t, db.Exec(`INSERT INTO audit_logs (tenant_id, created_at) VALUES ('tA', ?)`, now.Add(-30*24*time.Hour)).Error)

	// tA 订阅：生效，剩余约 30 天，套餐"专业版"
	require.NoError(t, db.Exec(`INSERT INTO tenant_plans (id, name, code) VALUES ('p1','专业版','pro')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO tenant_subscriptions (id, tenant_id, plan_id, status, started_at, expires_at, created_at) VALUES ('s1','tA','p1',1,?,?,?)`,
		now.Add(-time.Hour), now.Add(30*24*time.Hour), now).Error)
	// tB 无订阅
}

func TestGetTenantHealth_Aggregation(t *testing.T) {
	db := newHealthTestDB(t)
	seedHealthData(t, db)
	repo := NewDashboardRepository(db)

	list, err := repo.GetTenantHealth(context.Background(), model.HealthQuery{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(2), list.Total)
	require.Len(t, list.Items, 2)

	// 按 created_at DESC：tB(now) 在前，tA(-1h) 在后
	byID := map[string]model.TenantHealthItem{}
	for _, it := range list.Items {
		byID[it.TenantID] = it
	}

	a := byID["tA"]
	require.Equal(t, "Alpha Corp", a.TenantName)
	require.Equal(t, int64(2), a.UserCount)       // 排除软删 u3
	require.Equal(t, int64(1), a.ActiveUserCount) // 仅 u1
	require.Equal(t, int64(3), a.TaskCount)
	require.Equal(t, int64(2), a.TaskCompletedCount)
	require.Equal(t, int64(4), a.WikiDocCount)   // 排除软删 wx
	require.Equal(t, int64(3000), a.StorageUsed) // 排除软删 f3
	require.Equal(t, int64(25), a.AuditCount7d)  // 排除 30 天前
	require.NotNil(t, a.LastActivityAt)
	require.Equal(t, "专业版", a.PlanName)
	require.Equal(t, 1, a.SubscriptionStatus)
	require.NotNil(t, a.SubscriptionExpiresAt)
	require.True(t, a.DaysRemaining >= 29 && a.DaysRemaining <= 30, "剩余天数应约 30，实际 %d", a.DaysRemaining)
	require.Equal(t, "healthy", a.HealthLevel)
	require.True(t, a.HealthScore >= 70, "健康租户评分应 >=70，实际 %d", a.HealthScore)

	b := byID["tB"]
	require.Equal(t, "Beta Inc", b.TenantName)
	require.Equal(t, int64(1), b.UserCount)
	require.Equal(t, int64(0), b.TaskCount)
	require.Equal(t, int64(0), b.WikiDocCount)
	require.Equal(t, int64(0), b.StorageUsed)
	require.Equal(t, int64(0), b.AuditCount7d)
	require.Nil(t, b.LastActivityAt)
	require.Equal(t, 0, b.SubscriptionStatus)
	require.Equal(t, "", b.PlanName)
	require.Equal(t, "at_risk", b.HealthLevel)

	// 分布统计
	require.Equal(t, int64(1), list.Healthy)
	require.Equal(t, int64(0), list.Moderate)
	require.Equal(t, int64(1), list.AtRisk)
}

func TestGetTenantHealth_KeywordAndPaging(t *testing.T) {
	db := newHealthTestDB(t)
	seedHealthData(t, db)
	repo := NewDashboardRepository(db)

	// 关键字过滤
	list, err := repo.GetTenantHealth(context.Background(), model.HealthQuery{Page: 1, PageSize: 20, Keyword: "Alpha"})
	require.NoError(t, err)
	require.Equal(t, int64(1), list.Total)
	require.Len(t, list.Items, 1)
	require.Equal(t, "tA", list.Items[0].TenantID)

	// 分页：每页 1 条
	page2, err := repo.GetTenantHealth(context.Background(), model.HealthQuery{Page: 2, PageSize: 1})
	require.NoError(t, err)
	require.Equal(t, int64(2), page2.Total)
	require.Len(t, page2.Items, 1)
}

func TestGetTenantHealthDetail(t *testing.T) {
	db := newHealthTestDB(t)
	seedHealthData(t, db)
	repo := NewDashboardRepository(db)
	ctx := context.Background()

	item, err := repo.GetTenantHealthDetail(ctx, "tA")
	require.NoError(t, err)
	require.Equal(t, "tA", item.TenantID)
	require.Equal(t, "Alpha Corp", item.TenantName)
	require.Equal(t, int64(2), item.UserCount)
	require.Equal(t, int64(1), item.ActiveUserCount)
	require.Equal(t, int64(3), item.TaskCount)
	require.Equal(t, int64(2), item.TaskCompletedCount)
	require.Equal(t, int64(4), item.WikiDocCount)
	require.Equal(t, int64(3000), item.StorageUsed)
	require.Equal(t, int64(25), item.AuditCount7d)
	require.NotNil(t, item.LastActivityAt)
	require.Equal(t, "专业版", item.PlanName)
	require.Equal(t, 1, item.SubscriptionStatus)
	require.Equal(t, "healthy", item.HealthLevel)

	// 不存在的租户
	_, err = repo.GetTenantHealthDetail(ctx, "missing")
	require.True(t, errors.Is(err, ErrTenantNotFound))
}
