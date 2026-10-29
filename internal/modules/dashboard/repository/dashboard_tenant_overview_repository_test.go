package repository

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// newOverviewTestDB 建内存 SQLite 并创建租户总览聚合涉及的全部表
// （含 aggregateUsers/aggregateStorage/aggregateWikiDocs 及任务分布/Wiki 空间浏览量）。
func newOverviewTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	// GetTenantOverview 各维度经 errgroup 并行下发；内存 SQLite 每条新连接是独立空库，
	// 限为单连接让并发查询共享同一份数据，避免新连接上 no such table。
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	stmts := []string{
		`CREATE TABLE users (id TEXT PRIMARY KEY, tenant_id TEXT, status INTEGER, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE files (id TEXT PRIMARY KEY, tenant_id TEXT, file_size INTEGER, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE tasks (id TEXT PRIMARY KEY, tenant_id TEXT, status TEXT, due_date DATETIME, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE wiki_spaces (id TEXT PRIMARY KEY, tenant_id TEXT, created_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE wiki_documents (id TEXT PRIMARY KEY, tenant_id TEXT, view_count INTEGER, created_at DATETIME, deleted_at DATETIME)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

func seedOverviewData(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Now()
	past := now.Add(-48 * time.Hour)
	future := now.Add(48 * time.Hour)

	// tA 用户：2 有效 + 1 停用 + 1 软删（软删不计入）→ total 3 / active 2
	require.NoError(t, db.Exec(`INSERT INTO users (id, tenant_id, status, created_at) VALUES ('u1','tA',1,?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO users (id, tenant_id, status, created_at) VALUES ('u2','tA',1,?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO users (id, tenant_id, status, created_at) VALUES ('u3','tA',0,?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO users (id, tenant_id, status, created_at, deleted_at) VALUES ('u4','tA',1,?,?)`, now, now).Error)

	// tA 文件：1000 + 2000 有效，5000 软删（不计入存储）→ 3000
	require.NoError(t, db.Exec(`INSERT INTO files (id, tenant_id, file_size, created_at) VALUES ('f1','tA',1000,?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO files (id, tenant_id, file_size, created_at) VALUES ('f2','tA',2000,?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO files (id, tenant_id, file_size, created_at, deleted_at) VALUES ('f3','tA',5000,?,?)`, now, now).Error)

	// tA wiki 空间：2 有效 + 1 软删 → 2
	require.NoError(t, db.Exec(`INSERT INTO wiki_spaces (id, tenant_id, created_at) VALUES ('s1','tA',?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO wiki_spaces (id, tenant_id, created_at) VALUES ('s2','tA',?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO wiki_spaces (id, tenant_id, created_at, deleted_at) VALUES ('s3','tA',?,?)`, now, now).Error)

	// tA wiki 文档：浏览量 10+20+30=60 + 1 软删（浏览 999 不计入）
	require.NoError(t, db.Exec(`INSERT INTO wiki_documents (id, tenant_id, view_count, created_at) VALUES ('d1','tA',10,?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO wiki_documents (id, tenant_id, view_count, created_at) VALUES ('d2','tA',20,?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO wiki_documents (id, tenant_id, view_count, created_at) VALUES ('d3','tA',30,?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO wiki_documents (id, tenant_id, view_count, created_at, deleted_at) VALUES ('d4','tA',999,?,?)`, now, now).Error)

	// tA 任务：pending(tk1) / in_progress(tk2 未来) / completed(tk3 过期但不算逾期) / pending 逾期(tk4) / 软删(tk5)
	// → total 4, pending 2, in_progress 1, completed 1, overdue 1(tk4), completion_rate 0.25
	require.NoError(t, db.Exec(`INSERT INTO tasks (id, tenant_id, status, due_date, created_at) VALUES ('tk1','tA','pending',NULL,?)`, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO tasks (id, tenant_id, status, due_date, created_at) VALUES ('tk2','tA','in_progress',?,?)`, future, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO tasks (id, tenant_id, status, due_date, created_at) VALUES ('tk3','tA','completed',?,?)`, past, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO tasks (id, tenant_id, status, due_date, created_at) VALUES ('tk4','tA','pending',?,?)`, past, now).Error)
	require.NoError(t, db.Exec(`INSERT INTO tasks (id, tenant_id, status, due_date, created_at, deleted_at) VALUES ('tk5','tA','pending',?,?,?)`, past, now, now).Error)
}

func TestGetTenantOverview_Aggregation(t *testing.T) {
	db := newOverviewTestDB(t)
	seedOverviewData(t, db)
	repo := NewDashboardRepository(db)

	ov, err := repo.GetTenantOverview(context.Background(), "tA")
	require.NoError(t, err)

	require.Equal(t, int64(3), ov.Member.Total)
	require.Equal(t, int64(2), ov.Member.Active)

	require.Equal(t, int64(2), ov.Wiki.Spaces)
	require.Equal(t, int64(3), ov.Wiki.Documents)
	require.Equal(t, int64(60), ov.Wiki.Views)

	require.Equal(t, int64(3000), ov.StorageUsed)

	require.Equal(t, int64(4), ov.Task.Total)
	require.Equal(t, int64(2), ov.Task.Pending)
	require.Equal(t, int64(1), ov.Task.InProgress)
	require.Equal(t, int64(1), ov.Task.Completed)
	require.Equal(t, int64(1), ov.Task.Overdue)
	require.InDelta(t, 0.25, ov.Task.CompletionRate, 0.0001)
}

func TestGetTenantOverview_UnknownTenantIsZero(t *testing.T) {
	db := newOverviewTestDB(t)
	seedOverviewData(t, db)
	repo := NewDashboardRepository(db)

	ov, err := repo.GetTenantOverview(context.Background(), "missing")
	require.NoError(t, err)
	require.Equal(t, int64(0), ov.Member.Total)
	require.Equal(t, int64(0), ov.Task.Total)
	require.Equal(t, float64(0), ov.Task.CompletionRate)
	require.Equal(t, int64(0), ov.Wiki.Documents)
	require.Equal(t, int64(0), ov.StorageUsed)
}
