package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"meteorx/internal/modules/dashboard/model"
	"meteorx/pkg/timeutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newDashTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{SkipDefaultTransaction: true})
	require.NoError(t, err)
	return gormDB, mock
}

var countQuery = regexp.QuoteMeta("SELECT count(*) FROM")

func expectTenantStats(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(countQuery).WithArgs().WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(8))
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
}

func expectUserStats(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(50))
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(20))
}

func expectSubscriptionStats(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(12))
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT COUNT\\(DISTINCT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))
}

func expectAuditStats(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(30))
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(90))
	mock.ExpectQuery(countQuery).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	mock.ExpectQuery("SELECT .* as key, COUNT\\(\\*\\) as count FROM").WillReturnRows(
		sqlmock.NewRows([]string{"key", "count"}).AddRow("login", 50).AddRow("create", 40),
	)
	mock.ExpectQuery("SELECT .* as key, COUNT\\(\\*\\) as count FROM").WillReturnRows(
		sqlmock.NewRows([]string{"key", "count"}).AddRow("auth", 60).AddRow("wiki", 30),
	)
}

func TestDashboardRepo_GetOverview(t *testing.T) {
	gormDB, mock := newDashTestDB(t)

	expectTenantStats(mock)
	expectUserStats(mock)
	expectSubscriptionStats(mock)
	expectAuditStats(mock)

	repo := NewDashboardRepository(gormDB)
	overview, err := repo.GetOverview(context.Background())
	require.NoError(t, err)
	require.NotNil(t, overview)

	assert.Equal(t, int64(10), overview.TenantStats.Total)
	assert.Equal(t, int64(8), overview.TenantStats.Enabled)
	assert.Equal(t, int64(2), overview.TenantStats.Disabled)
	assert.Equal(t, int64(50), overview.UserStats.Total)
	assert.Equal(t, int64(15), overview.SubscriptionStats.Total)
	assert.Equal(t, int64(12), overview.SubscriptionStats.Active)
	assert.Equal(t, int64(100), overview.AuditStats.Total)
	assert.Equal(t, int64(90), overview.AuditStats.Success)
}

func TestDashboardRepo_GetOverview_PartialFailure(t *testing.T) {
	gormDB, mock := newDashTestDB(t)

	mock.ExpectQuery(countQuery).WillReturnError(gorm.ErrInvalidDB)

	expectUserStats(mock)
	expectSubscriptionStats(mock)
	expectAuditStats(mock)

	repo := NewDashboardRepository(gormDB)
	overview, err := repo.GetOverview(context.Background())
	require.NoError(t, err)
	require.NotNil(t, overview)

	assert.Equal(t, int64(0), overview.TenantStats.Total, "tenant stats should be zero on DB error")
	assert.Equal(t, int64(50), overview.UserStats.Total, "user stats should still be returned")
}

func TestDashboardRepo_GetTrend(t *testing.T) {
	gormDB, mock := newDashTestDB(t)

	// 分桶日期由实现按“业务时区当天”往前回推得出，测试必须以同一口径生成相对日期键，
	// 否则硬编码绝对日期只在特定一天成立（跨天/跨时区/CI 上必然全 0）。
	// days=2 → 窗口为 [昨天, 今天]，与 GetTrend 中 start=StartOfDay(now).AddDate(0,0,-(days-1)) 对齐。
	start := timeutil.StartOfDay(time.Now()).AddDate(0, 0, -1)
	keyYesterday := start.Format("2006-01-02")
	keyToday := start.AddDate(0, 0, 1).Format("2006-01-02")

	// 租户/用户查询带 deleted_at IS NULL，审计日志不带，均使用同一个 group-by-day 正则。
	groupPattern := "SELECT DATE\\(created_at\\) as day, COUNT\\(\\*\\) as count FROM .*tenants.* GROUP BY DATE\\(created_at\\)"
	mock.ExpectQuery(groupPattern).WillReturnRows(
		sqlmock.NewRows([]string{"day", "count"}).AddRow(keyToday, 2),
	)
	mock.ExpectQuery("SELECT DATE\\(created_at\\) as day, COUNT\\(\\*\\) as count FROM .*users.* GROUP BY DATE\\(created_at\\)").WillReturnRows(
		sqlmock.NewRows([]string{"day", "count"}).AddRow(keyYesterday, 3).AddRow(keyToday, 1),
	)
	mock.ExpectQuery("SELECT DATE\\(created_at\\) as day, COUNT\\(\\*\\) as count FROM .*audit_logs.* GROUP BY DATE\\(created_at\\)").WillReturnRows(
		sqlmock.NewRows([]string{"day", "count"}).AddRow(keyToday, 40),
	)

	repo := NewDashboardRepository(gormDB)
	trend, err := repo.GetTrend(context.Background(), 2)
	require.NoError(t, err)
	require.NotNil(t, trend)

	assert.Equal(t, 2, trend.Days)
	require.Len(t, trend.List, 2)
	// 缺失日期以零值补齐，命中日期回填计数
	assert.Equal(t, int64(0), trend.List[0].NewTenants)
	assert.Equal(t, int64(3), trend.List[0].NewUsers)
	assert.Equal(t, int64(2), trend.List[1].NewTenants)
	assert.Equal(t, int64(1), trend.List[1].NewUsers)
	assert.Equal(t, int64(40), trend.List[1].AuditLogs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardRepo_GetTrend_DefaultsAndClamps(t *testing.T) {
	gormDB, mock := newDashTestDB(t)

	// days<=0 应回退为默认 30 天，因此需要为 3 张表各预期一次查询
	for i := 0; i < 3; i++ {
		mock.ExpectQuery("SELECT DATE\\(created_at\\) as day").WillReturnRows(sqlmock.NewRows([]string{"day", "count"}))
	}

	repo := NewDashboardRepository(gormDB)
	trend, err := repo.GetTrend(context.Background(), 0)
	require.NoError(t, err)
	assert.Equal(t, 30, trend.Days)
	assert.Len(t, trend.List, 30)
}

func TestNewDashboardRepository(t *testing.T) {
	gormDB, _ := newDashTestDB(t)
	repo := NewDashboardRepository(gormDB)
	assert.NotNil(t, repo)

	var _ = repo
}

func TestDashboardModel_JSON(t *testing.T) {
	overview := &model.DashboardOverview{
		TenantStats: model.TenantStats{Total: 10, Enabled: 8, Disabled: 2},
		UserStats:   model.UserStats{Total: 50},
		SubscriptionStats: model.SubscriptionStats{
			Total:  15,
			Active: 12,
		},
		AuditStats: model.AuditStats{
			Total:   100,
			Success: 90,
			Failure: 10,
		},
	}
	assert.Equal(t, int64(10), overview.TenantStats.Total)
	assert.Equal(t, int64(50), overview.UserStats.Total)
	assert.Equal(t, int64(12), overview.SubscriptionStats.Active)
	assert.Equal(t, int64(10), overview.AuditStats.Failure)
}
