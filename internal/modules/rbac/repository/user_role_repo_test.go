package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newUserRoleTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func TestUserRoleRepo_GetRoleIDsByUserID(t *testing.T) {
	gormDB, mock := newUserRoleTestDB(t)
	repo := NewUserRoleRepository(gormDB)

	rows := sqlmock.NewRows([]string{"user_id", "role_id", "created_at"}).
		AddRow("u-001", "r-001", time.Now()).
		AddRow("u-001", "r-002", time.Now())

	mock.ExpectQuery("SELECT .+ FROM `user_roles` WHERE user_id = \\?").
		WithArgs("u-001").
		WillReturnRows(rows)

	roleIDs, err := repo.GetRoleIDsByUserID(context.Background(), "u-001")
	require.NoError(t, err)
	assert.Equal(t, []string{"r-001", "r-002"}, roleIDs)
}

func TestUserRoleRepo_DeleteByUserID(t *testing.T) {
	gormDB, mock := newUserRoleTestDB(t)
	repo := NewUserRoleRepository(gormDB)

	mock.ExpectExec("DELETE FROM `user_roles` WHERE user_id = \\?").
		WithArgs("u-001").
		WillReturnResult(sqlmock.NewResult(1, 2))

	err := repo.DeleteByUserID(context.Background(), "u-001")
	assert.NoError(t, err)
}

func TestUserRoleRepo_CountByRoleID(t *testing.T) {
	gormDB, mock := newUserRoleTestDB(t)
	repo := NewUserRoleRepository(gormDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(5)
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `user_roles` WHERE role_id = \\?").
		WithArgs("r-001").
		WillReturnRows(rows)

	count, err := repo.CountByRoleID(context.Background(), "r-001")
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestUserRoleRepo_CountByUserID(t *testing.T) {
	gormDB, mock := newUserRoleTestDB(t)
	repo := NewUserRoleRepository(gormDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(3)
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `user_roles` WHERE user_id = \\?").
		WithArgs("u-001").
		WillReturnRows(rows)

	count, err := repo.CountByUserID(context.Background(), "u-001")
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestUserRoleRepo_DeleteByUserIDAndRoleID(t *testing.T) {
	gormDB, mock := newUserRoleTestDB(t)
	repo := NewUserRoleRepository(gormDB)

	mock.ExpectExec("DELETE FROM `user_roles` WHERE user_id = \\? AND role_id = \\?").
		WithArgs("u-001", "r-001").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.DeleteByUserIDAndRoleID(context.Background(), "u-001", "r-001")
	assert.NoError(t, err)
}

func TestUserRoleRepo_GetUserIDsByRoleID(t *testing.T) {
	gormDB, mock := newUserRoleTestDB(t)
	repo := NewUserRoleRepository(gormDB)

	rows := sqlmock.NewRows([]string{"user_id", "role_id", "created_at"}).
		AddRow("u-001", "r-001", time.Now()).
		AddRow("u-002", "r-001", time.Now())

	mock.ExpectQuery("SELECT .+ FROM `user_roles` WHERE role_id = \\?").
		WithArgs("r-001").
		WillReturnRows(rows)

	userIDs, err := repo.GetUserIDsByRoleID(context.Background(), "r-001")
	require.NoError(t, err)
	assert.Equal(t, []string{"u-001", "u-002"}, userIDs)
}