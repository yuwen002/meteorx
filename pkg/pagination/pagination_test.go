package pagination

import (
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func openGorm(t *testing.T) *gorm.DB {
	t.Helper()
	sqlDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm: %v", err)
	}
	return db
}

// buildSQL 以 DryRun 模式把 Apply* 链式条件构造成最终 SELECT，便于断言 SQL 片段
func buildSQL(t *testing.T, db *gorm.DB, mutate func(tx *gorm.DB) *gorm.DB) string {
	t.Helper()
	var out []map[string]interface{}
	q := mutate(db.Session(&gorm.Session{DryRun: true}).Table("wiki_spaces"))
	_ = q.Find(&out).Error
	sql := q.Statement.SQL.String()
	if sql == "" {
		t.Fatal("built SQL is empty")
	}
	return sql
}

func TestNewPageRequest_ClampsValues(t *testing.T) {
	cases := []struct {
		name     string
		page     int
		pageSize int
		wantPage int
		wantSize int
	}{
		{"默认值", 0, 0, 1, 20},
		{"负数修正", -3, -5, 1, 20},
		{"上限截断", 9, 10000, 9, 100},
		{"正常取值", 3, 50, 3, 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := NewPageRequest(tc.page, tc.pageSize)
			assert.Equal(t, tc.wantPage, req.Page)
			assert.Equal(t, tc.wantSize, req.PageSize)
		})
	}
}

func TestPageRequest_OffsetLimit(t *testing.T) {
	req := NewPageRequest(3, 25)
	assert.Equal(t, 25, req.Limit())
	assert.Equal(t, 50, req.Offset())
}

func TestNewPageResult(t *testing.T) {
	items := []string{"a", "b"}
	res := NewPageResult(items, int64(len(items)), 2, 10)
	assert.Equal(t, items, res.Items)
	assert.Equal(t, int64(2), res.Total)
	assert.Equal(t, 2, res.Page)
	assert.Equal(t, 10, res.PageSize)
}

func TestApplySort_WhitelistedAndOrder(t *testing.T) {
	db := openGorm(t)
	allowed := map[string]string{"created_at": "created_at", "name": "name"}

	desc := buildSQL(t, db, func(tx *gorm.DB) *gorm.DB {
		return ApplySort(tx, "created_at", "DESC", allowed)
	})
	assert.Contains(t, strings.ToUpper(desc), "ORDER BY CREATED_AT DESC")

	asc := buildSQL(t, db, func(tx *gorm.DB) *gorm.DB {
		return ApplySort(tx, "created_at", "", allowed)
	})
	assert.Contains(t, strings.ToUpper(asc), "ORDER BY CREATED_AT ASC")
}

func TestApplySort_UnknownFieldIgnored(t *testing.T) {
	db := openGorm(t)
	allowed := map[string]string{"created_at": "created_at"}
	sql := buildSQL(t, db, func(tx *gorm.DB) *gorm.DB {
		return ApplySort(tx, "created_at; DROP TABLE users", "DESC", allowed)
	})
	upper := strings.ToUpper(sql)
	// 白名单外的“排序字段”直接忽略，不拼接进 SQL
	assert.NotContains(t, upper, "DROP")
	assert.NotContains(t, upper, "ORDER BY")
}

func TestApplyKeyword_SearchableFields(t *testing.T) {
	db := openGorm(t)
	sql := buildSQL(t, db, func(tx *gorm.DB) *gorm.DB {
		return ApplyKeyword(tx, "meeting", "title", "content")
	})
	upper := strings.ToUpper(sql)
	assert.Contains(t, upper, "LIKE ?")
	assert.Contains(t, upper, "TITLE")
	assert.Contains(t, upper, "CONTENT")
}

func TestApplyKeyword_EmptySkipsFilter(t *testing.T) {
	db := openGorm(t)
	sql := buildSQL(t, db, func(tx *gorm.DB) *gorm.DB {
		return ApplyKeyword(tx, "", "title")
	})
	assert.NotContains(t, strings.ToUpper(sql), "LIKE")
}

func TestApplyPagination_SetsLimitOffset(t *testing.T) {
	db := openGorm(t)
	sql := buildSQL(t, db, func(tx *gorm.DB) *gorm.DB {
		return ApplyPagination(tx, 3, 25)
	})
	upper := strings.ToUpper(sql)
	// 参数以占位符绑定（数值由 TestPageRequest_OffsetLimit 覆盖）
	assert.Contains(t, upper, "LIMIT ? OFFSET ?")
}

func TestApplyPagination_ClampsIllegalInput(t *testing.T) {
	db := openGorm(t)
	sql := buildSQL(t, db, func(tx *gorm.DB) *gorm.DB {
		return ApplyPagination(tx, 0, 0)
	})
	upper := strings.ToUpper(sql)
	// 默认 page=1 offset=0：gorm 省略 OFFSET，只保留 LIMIT 20
	assert.Contains(t, upper, "LIMIT ?")
	assert.NotContains(t, upper, "OFFSET")
}

func TestPagination_Helpers(t *testing.T) {
	pg := NewPagination(2, 30)
	pg.SetTotal(99)
	assert.Equal(t, 2, pg.Page)
	assert.Equal(t, 30, pg.PageSize)
	assert.Equal(t, 99, pg.Total)
	assert.Equal(t, 30, pg.Limit())
	assert.Equal(t, 30, pg.Offset())

	res := NewPaginatedResult([]int{1}, 1, 20, 5)
	assert.Equal(t, []int{1}, res.Data)
	assert.Equal(t, 1, res.Pagination.Page)
	assert.Equal(t, 5, res.Pagination.Total)
}
