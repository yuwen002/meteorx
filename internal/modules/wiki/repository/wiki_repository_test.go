package repository

import (
	"context"
	"testing"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// newTestDB 基于 sqlmock 构建一个 GORM DB，并断言测试结束时所有预期的 SQL 都已被消费。
func newTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
		_ = sqlDB.Close()
	})
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	return gormDB, mock
}

// tenantCtx 构造一个指定租户的上下文。
func tenantCtx(tenantID string) context.Context {
	return contextx.SetVars(context.Background(), tenantID, "user-1", []string{"admin"})
}

// masterCtx 构造平台超级管理员上下文。
func masterCtx() context.Context {
	return contextx.SetVars(context.Background(), contextx.SystemTenantID, "admin", []string{"superadmin"})
}

func TestGetNodeByID_ScopedByTenant(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewWikiRepository(gormDB).(*wikiRepository)

	// 普通租户读取节点：必须携带 tenant_id = tenant-1
	mock.ExpectQuery("SELECT \\* FROM `wiki_nodes` WHERE tenant_id = \\? AND id = \\? ORDER BY `wiki_nodes`\\.`id` LIMIT \\?").
		WithArgs("tenant-1", "node-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "space_id", "title"}).
			AddRow("node-1", "tenant-1", "space-1", "Hello"))

	node, err := repo.GetNodeByID(tenantCtx("tenant-1"), "node-1")
	assert.NoError(t, err)
	assert.NotNil(t, node)
	assert.Equal(t, "node-1", node.ID)
}

func TestGetNodeByID_NotInOtherTenant(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewWikiRepository(gormDB).(*wikiRepository)

	// 租户 A 尝试读取租户 B 的节点：查询带了 tenant_id = A，sqlmock 返回空 → 应返回 ErrWikiNodeNotFound
	mock.ExpectQuery("SELECT \\* FROM `wiki_nodes` WHERE tenant_id = \\? AND id = \\? ORDER BY `wiki_nodes`\\.`id` LIMIT \\?").
		WithArgs("tenant-A", "node-of-B", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "space_id", "title"}))

	_, err := repo.GetNodeByID(tenantCtx("tenant-A"), "node-of-B")
	assert.ErrorIs(t, err, ErrWikiNodeNotFound)
}

func TestGetNodeByID_MasterBypassesFilter(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewWikiRepository(gormDB).(*wikiRepository)

	// 平台超级管理员：不应带 tenant 过滤，可直接读取任意租户数据
	mock.ExpectQuery("SELECT \\* FROM `wiki_nodes` WHERE id = \\? ORDER BY `wiki_nodes`\\.`id` LIMIT \\?").
		WithArgs("node-x", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "space_id", "title"}).
			AddRow("node-x", "tenant-B", "space-9", "Cross"))

	node, err := repo.GetNodeByID(masterCtx(), "node-x")
	assert.NoError(t, err)
	assert.Equal(t, "node-x", node.ID)
}

func TestUpdateNode_ScopedByTenant(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewWikiRepository(gormDB).(*wikiRepository)

	// 更新节点必须携带 tenant_id 条件，防止跨租户覆盖。
	// Updates(map) 的列与值均被参数化、顺序不定，因此 SET 段全部用 AnyArg，
	// 只精确定位 WHERE 中必须出现的租户条件。
	mock.ExpectExec("UPDATE `wiki_nodes` SET .* WHERE tenant_id = \\? AND id = \\?").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "tenant-1", "node-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	node := &model.WikiNode{ID: "node-1", TenantID: "tenant-1", Title: "新标题"}
	err := repo.UpdateNode(tenantCtx("tenant-1"), node)
	assert.NoError(t, err)
}

func TestDeleteNode_ScopedByTenant(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewWikiRepository(gormDB).(*wikiRepository)

	mock.ExpectExec("DELETE FROM `wiki_nodes` WHERE tenant_id = \\? AND id = \\?").
		WithArgs("tenant-1", "node-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteNode(tenantCtx("tenant-1"), "node-1")
	assert.NoError(t, err)
}

func TestUpdateDocument_OptimisticLock(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewWikiRepository(gormDB).(*wikiRepository)

	// 乐观锁：更新必须携带 current_ver = 期望版本
	mock.ExpectExec("UPDATE `wiki_documents` SET .* WHERE tenant_id = \\? AND \\(id = \\? AND current_ver = \\?\\)").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "tenant-1", "doc-1", 3).
		WillReturnResult(sqlmock.NewResult(0, 1))

	doc := &model.Document{ID: "doc-1", TenantID: "tenant-1", CurrentVer: 4}
	err := repo.UpdateDocument(tenantCtx("tenant-1"), doc, 3)
	assert.NoError(t, err)
}

func TestUpdateDocument_VersionConflict(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewWikiRepository(gormDB).(*wikiRepository)

	// 0 行受影响 + 文档仍存在 → 版本冲突
	mock.ExpectExec("UPDATE `wiki_documents` SET .* WHERE tenant_id = \\? AND \\(id = \\? AND current_ver = \\?\\)").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "tenant-1", "doc-1", 3).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `wiki_documents` WHERE tenant_id = \\? AND id = \\?").
		WithArgs("tenant-1", "doc-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	doc := &model.Document{ID: "doc-1", TenantID: "tenant-1", CurrentVer: 4}
	err := repo.UpdateDocument(tenantCtx("tenant-1"), doc, 3)
	assert.ErrorIs(t, err, ErrDocumentVersionConflict)
}

func TestUpdateDocument_NotFound(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewWikiRepository(gormDB).(*wikiRepository)

	// 0 行受影响 + 文档不存在 → ErrDocumentNotFound
	mock.ExpectExec("UPDATE `wiki_documents` SET .* WHERE tenant_id = \\? AND \\(id = \\? AND current_ver = \\?\\)").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "tenant-1", "doc-1", 3).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `wiki_documents` WHERE tenant_id = \\? AND id = \\?").
		WithArgs("tenant-1", "doc-1").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	doc := &model.Document{ID: "doc-1", TenantID: "tenant-1", CurrentVer: 4}
	err := repo.UpdateDocument(tenantCtx("tenant-1"), doc, 3)
	assert.ErrorIs(t, err, ErrDocumentNotFound)
}

func TestScope_NoTenantReturnsNoRows(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewWikiRepository(gormDB).(*wikiRepository)

	// 缺少租户上下文时，Scope 返回 1=0，杜绝越权访问
	mock.ExpectQuery("SELECT \\* FROM `wiki_nodes` WHERE 1 = 0 AND id = \\? ORDER BY `wiki_nodes`\\.`id` LIMIT \\?").
		WithArgs("node-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "space_id", "title"}))

	_, err := repo.GetNodeByID(context.Background(), "node-1")
	assert.ErrorIs(t, err, ErrWikiNodeNotFound)
}
