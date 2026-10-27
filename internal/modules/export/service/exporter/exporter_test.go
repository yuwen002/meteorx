package exporter

import (
	"bytes"
	"context"
	"encoding/csv"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// 测试用资源表结构，列与生产库对齐（含 deleted_at 以支持导出器的软删除过滤）

type tUser struct {
	ID          string     `gorm:"primaryKey;column:id"`
	TenantID    string     `gorm:"column:tenant_id"`
	Username    string     `gorm:"column:username"`
	Nickname    *string    `gorm:"column:nickname"`
	Email       *string    `gorm:"column:email"`
	Phone       *string    `gorm:"column:phone"`
	Department  *string    `gorm:"column:department"`
	Position    *string    `gorm:"column:position"`
	Status      int        `gorm:"column:status"`
	IsMaster    bool       `gorm:"column:is_master"`
	LastLoginAt *time.Time `gorm:"column:last_login_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
}

func (tUser) TableName() string { return "users" }

type tTask struct {
	ID          string     `gorm:"primaryKey;column:id"`
	TenantID    string     `gorm:"column:tenant_id"`
	CreatorID   string     `gorm:"column:creator_id"`
	AssigneeID  string     `gorm:"column:assignee_id"`
	Title       string     `gorm:"column:title"`
	Status      string     `gorm:"column:status"`
	Priority    string     `gorm:"column:priority"`
	DueDate     *time.Time `gorm:"column:due_date"`
	Visibility  string     `gorm:"column:visibility"`
	CompletedAt *time.Time `gorm:"column:completed_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
}

func (tTask) TableName() string { return "tasks" }

type tAudit struct {
	ID         string    `gorm:"primaryKey;column:id"`
	TenantID   string    `gorm:"column:tenant_id"`
	UserID     *string   `gorm:"column:user_id"`
	Username   *string   `gorm:"column:username"`
	Module     *string   `gorm:"column:module"`
	Action     *string   `gorm:"column:action"`
	Resource   *string   `gorm:"column:resource"`
	Result     *string   `gorm:"column:result"`
	RiskLevel  *string   `gorm:"column:risk_level"`
	ClientIP   *string   `gorm:"column:client_ip"`
	Method     *string   `gorm:"column:method"`
	Path       *string   `gorm:"column:path"`
	StatusCode *int      `gorm:"column:status_code"`
	Duration   *int64    `gorm:"column:duration"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (tAudit) TableName() string { return "audit_logs" }

type tWikiDoc struct {
	ID            string     `gorm:"primaryKey;column:id"`
	TenantID      string     `gorm:"column:tenant_id"`
	NodeID        string     `gorm:"column:node_id"`
	Format        *string    `gorm:"column:format"`
	CurrentVer    int        `gorm:"column:current_ver"`
	ViewCount     int64      `gorm:"column:view_count"`
	PublishStatus *string    `gorm:"column:publish_status"`
	LastEditedBy  *string    `gorm:"column:last_edited_by"`
	PublishedAt   *time.Time `gorm:"column:published_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
	DeletedAt     *time.Time `gorm:"column:deleted_at"`
}

func (tWikiDoc) TableName() string { return "wiki_documents" }

func newExporterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&tUser{}, &tTask{}, &tAudit{}, &tWikiDoc{}))
	return db
}

func ptr[T any](v T) *T { return &v }

// readCSV 解析 buffer 内容为记录切片（含表头行）
func readCSV(t *testing.T, buf *bytes.Buffer) [][]string {
	t.Helper()
	r := csv.NewReader(buf)
	records, err := r.ReadAll()
	require.NoError(t, err)
	return records
}

func TestUserExporter_Stream(t *testing.T) {
	db := newExporterTestDB(t)
	ctx := context.Background()
	now := time.Now()
	require.NoError(t, db.Create(&tUser{ID: "u1", TenantID: "t1", Username: "alice", Email: ptr("a@x.com"), Status: 1, CreatedAt: now}).Error)
	require.NoError(t, db.Create(&tUser{ID: "u2", TenantID: "t1", Username: "bob", Status: 0, CreatedAt: now}).Error)
	require.NoError(t, db.Create(&tUser{ID: "u3", TenantID: "t2", Username: "carol", Status: 1, CreatedAt: now}).Error)
	// 软删除用户不应被导出
	require.NoError(t, db.Create(&tUser{ID: "u4", TenantID: "t1", Username: "ghost", Status: 1, CreatedAt: now, DeletedAt: ptr(now)}).Error)

	var buf bytes.Buffer
	sink := NewCSVSink(&buf)
	exp := NewUserExporter(db)
	rows, err := exp.Stream(ctx, "t1", nil, sink)
	require.NoError(t, sink.Close())
	require.NoError(t, err)
	require.Equal(t, int64(2), rows, "仅导出本租户未删除用户")

	records := readCSV(t, &buf)
	require.Len(t, records, 3, "表头 + 2 数据行")
	require.Equal(t, "ID", records[0][0])
	require.Equal(t, "alice", records[1][1])
}

func TestUserExporter_FilterStatus(t *testing.T) {
	db := newExporterTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&tUser{ID: "u1", TenantID: "t1", Username: "alice", Status: 1, CreatedAt: now}).Error)
	require.NoError(t, db.Create(&tUser{ID: "u2", TenantID: "t1", Username: "bob", Status: 0, CreatedAt: now}).Error)

	var buf bytes.Buffer
	sink := NewCSVSink(&buf)
	rows, err := NewUserExporter(db).Stream(context.Background(), "t1", map[string]any{"status": "1"}, sink)
	require.NoError(t, sink.Close())
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
}

func TestUserExporter_FilterIDs(t *testing.T) {
	db := newExporterTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&tUser{ID: "u1", TenantID: "t1", Username: "alice", CreatedAt: now}).Error)
	require.NoError(t, db.Create(&tUser{ID: "u2", TenantID: "t1", Username: "bob", CreatedAt: now}).Error)
	require.NoError(t, db.Create(&tUser{ID: "u3", TenantID: "t2", Username: "carol", CreatedAt: now}).Error)
	require.NoError(t, db.Create(&tUser{ID: "u4", TenantID: "t1", Username: "ghost", CreatedAt: now, DeletedAt: ptr(now)}).Error)

	// JSON 反序列化后是 []any：选中的他租户用户与已软删用户不得被导出
	var buf bytes.Buffer
	sink := NewCSVSink(&buf)
	rows, err := NewUserExporter(db).Stream(context.Background(), "t1", map[string]any{"ids": []any{"u1", "u3", "u4"}}, sink)
	require.NoError(t, sink.Close())
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
	records := readCSV(t, &buf)
	require.Len(t, records, 2, "表头 + 1 数据行")
	require.Equal(t, "alice", records[1][1])

	// 代码内直接构造的 []string 同样支持
	var buf2 bytes.Buffer
	sink2 := NewCSVSink(&buf2)
	rows2, err := NewUserExporter(db).Stream(context.Background(), "t1", map[string]any{"ids": []string{"u1", "u2"}}, sink2)
	require.NoError(t, sink2.Close())
	require.NoError(t, err)
	require.Equal(t, int64(2), rows2)
}

func TestTaskExporter_Stream(t *testing.T) {
	db := newExporterTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&tTask{ID: "k1", TenantID: "t1", Title: "写周报", Status: "pending", Priority: "normal", Visibility: "personal", CreatedAt: now}).Error)
	require.NoError(t, db.Create(&tTask{ID: "k2", TenantID: "t1", Title: "评审", Status: "done", Priority: "high", Visibility: "team", CreatedAt: now}).Error)
	require.NoError(t, db.Create(&tTask{ID: "k3", TenantID: "t2", Title: "他租户", Status: "pending", Priority: "normal", CreatedAt: now}).Error)

	var buf bytes.Buffer
	sink := NewCSVSink(&buf)
	rows, err := NewTaskExporter(db).Stream(context.Background(), "t1", map[string]any{"status": "done"}, sink)
	require.NoError(t, sink.Close())
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
	records := readCSV(t, &buf)
	require.Equal(t, "评审", records[1][1])
}

func TestAuditExporter_Stream(t *testing.T) {
	db := newExporterTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&tAudit{ID: "a1", TenantID: "t1", Username: ptr("alice"), Module: ptr("auth"), Action: ptr("login"), Result: ptr("success"), RiskLevel: ptr("low"), StatusCode: ptr(200), Duration: ptr(int64(15)), CreatedAt: now}).Error)
	require.NoError(t, db.Create(&tAudit{ID: "a2", TenantID: "t1", Username: ptr("bob"), Module: ptr("user"), Action: ptr("delete"), Result: ptr("fail"), RiskLevel: ptr("high"), StatusCode: ptr(500), Duration: ptr(int64(99)), CreatedAt: now}).Error)
	require.NoError(t, db.Create(&tAudit{ID: "a3", TenantID: "t2", Username: ptr("x"), Module: ptr("auth"), CreatedAt: now}).Error)

	var buf bytes.Buffer
	sink := NewCSVSink(&buf)
	rows, err := NewAuditExporter(db).Stream(context.Background(), "t1", map[string]any{"module": "auth"}, sink)
	require.NoError(t, sink.Close())
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
	records := readCSV(t, &buf)
	require.Equal(t, "alice", records[1][1])
	require.Equal(t, "200", records[1][11])
	require.Equal(t, "15", records[1][12])
}

func TestWikiDocumentExporter_Stream(t *testing.T) {
	db := newExporterTestDB(t)
	now := time.Now()
	require.NoError(t, db.Create(&tWikiDoc{ID: "d1", TenantID: "t1", NodeID: "n1", Format: ptr("markdown"), CurrentVer: 3, ViewCount: 100, PublishStatus: ptr("published"), CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&tWikiDoc{ID: "d2", TenantID: "t1", NodeID: "n2", Format: ptr("markdown"), CurrentVer: 1, ViewCount: 5, PublishStatus: ptr("draft"), CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&tWikiDoc{ID: "d3", TenantID: "t2", NodeID: "n3", PublishStatus: ptr("published"), CreatedAt: now, UpdatedAt: now}).Error)

	var buf bytes.Buffer
	sink := NewCSVSink(&buf)
	rows, err := NewWikiDocumentExporter(db).Stream(context.Background(), "t1", map[string]any{"publishStatus": "published"}, sink)
	require.NoError(t, sink.Close())
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
	records := readCSV(t, &buf)
	require.Equal(t, "d1", records[1][0])
	require.Equal(t, "3", records[1][3])
	require.Equal(t, "100", records[1][4])
}
