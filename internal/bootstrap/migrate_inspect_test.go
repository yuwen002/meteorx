package bootstrap

import (
	"regexp"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// inspectWidget 巡检测试用模型：仅 id/name 两列，表名固定为 widgets。
type inspectWidget struct {
	ID   string
	Name string
}

// TableName 指定表名，便于 mock 断言与 schema 解析。
func (inspectWidget) TableName() string { return "widgets" }

// newInspectDB 构造基于 sqlmock 的 gorm 连接（复用项目 mysql+sqlmock 测试范式）。
func newInspectDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{SkipDefaultTransaction: true})
	require.NoError(t, err)
	return gormDB, mock
}

func TestOrphanColumns_Pure(t *testing.T) {
	expected := map[string]struct{}{"id": {}, "name": {}}
	actual := []string{"id", "legacy_col", "name", "dropped_col"}

	orphans := orphanColumns(actual, expected)

	// 只报出模型已无的列，保持数据库返回顺序；命中模型列的被过滤。
	assert.Equal(t, []string{"legacy_col", "dropped_col"}, orphans)
}

func TestAuditOrphanColumns_ReportsOrphan(t *testing.T) {
	gormDB, mock := newInspectDB(t)

	rows := sqlmock.NewRows([]string{"COLUMN_NAME"}).
		AddRow("id").AddRow("name").AddRow("legacy_col")
	mock.ExpectQuery(regexp.QuoteMeta("information_schema")).
		WithArgs("widgets").
		WillReturnRows(rows)

	reports, err := auditOrphanColumns(gormDB, []interface{}{&inspectWidget{}})
	require.NoError(t, err)
	require.Len(t, reports, 1)
	assert.Equal(t, "widgets", reports[0].Table)
	assert.Equal(t, []string{"legacy_col"}, reports[0].OrphanColumns)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAuditOrphanColumns_NoOrphan(t *testing.T) {
	gormDB, mock := newInspectDB(t)

	// 库里列与模型完全一致 → 无孤儿列。
	rows := sqlmock.NewRows([]string{"COLUMN_NAME"}).AddRow("id").AddRow("name")
	mock.ExpectQuery(regexp.QuoteMeta("information_schema")).
		WithArgs("widgets").
		WillReturnRows(rows)

	reports, err := auditOrphanColumns(gormDB, []interface{}{&inspectWidget{}})
	require.NoError(t, err)
	assert.Empty(t, reports)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAuditOrphanColumns_SkipsMissingTable(t *testing.T) {
	gormDB, mock := newInspectDB(t)

	// 全新库该表尚未创建 → 查询返回空 → 跳过，不计入报告。
	rows := sqlmock.NewRows([]string{"COLUMN_NAME"})
	mock.ExpectQuery(regexp.QuoteMeta("information_schema")).
		WithArgs("widgets").
		WillReturnRows(rows)

	reports, err := auditOrphanColumns(gormDB, []interface{}{&inspectWidget{}})
	require.NoError(t, err)
	assert.Empty(t, reports)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestMigrationModels_Parseable 断言所有真实模块模型均可被 schema 解析且表名非空，
// 防止某模块 Models() 遗漏或被改名后巡检静默漏表。
func TestMigrationModels_Parseable(t *testing.T) {
	models := migrationModels()
	assert.NotEmpty(t, models)

	cache := &sync.Map{}
	for _, m := range models {
		s, err := schema.Parse(m, cache, schema.NamingStrategy{})
		require.NoError(t, err, "模型 %T 应可解析", m)
		assert.NotEmpty(t, s.Table, "模型 %T 表名不应为空", m)
		assert.NotEmpty(t, s.FieldsByDBName, "模型 %T 应至少有一列", m)
	}
}
