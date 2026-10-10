package repository

import (
	"context"
	"regexp"
	"testing"

	"meteorx/internal/modules/dashboard/model"

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

func TestNewDashboardRepository(t *testing.T) {
	gormDB, _ := newDashTestDB(t)
	repo := NewDashboardRepository(gormDB)
	assert.NotNil(t, repo)

	var _ DashboardRepository = repo
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