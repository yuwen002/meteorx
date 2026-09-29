package repository

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/modules/rbac/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newPermTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func TestPermRepo_Create(t *testing.T) {
	gormDB, mock := newPermTestDB(t)
	repo := NewPermissionRepository(gormDB)

	mock.ExpectExec("INSERT INTO `permissions`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), &model.Permission{
		ID: "p-001", Name: "Create User", Code: "user:create", Resource: "user", Action: "create", Status: 1,
	})
	assert.NoError(t, err)
}

func TestPermRepo_GetByID(t *testing.T) {
	gormDB, mock := newPermTestDB(t)
	repo := NewPermissionRepository(gormDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "name", "code", "description", "resource", "action", "status", "created_at", "updated_at",
	}).AddRow("p-001", "Create User", "user:create", "Permission to create users", "user", "create", 1, now, now)

	mock.ExpectQuery("SELECT .+ FROM `permissions` WHERE id = \\? .+ LIMIT \\?").
		WithArgs("p-001", 1).
		WillReturnRows(rows)

	perm, err := repo.GetByID(context.Background(), "p-001")
	require.NoError(t, err)
	assert.Equal(t, "p-001", perm.ID)
	assert.Equal(t, "user:create", perm.Code)
}

func TestPermRepo_GetByCode(t *testing.T) {
	gormDB, mock := newPermTestDB(t)
	repo := NewPermissionRepository(gormDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "name", "code", "description", "resource", "action", "status", "created_at", "updated_at",
	}).AddRow("p-001", "Create User", "user:create", "Permission to create users", "user", "create", 1, now, now)

	mock.ExpectQuery("SELECT .+ FROM `permissions` WHERE code = \\? .+ LIMIT \\?").
		WithArgs("user:create", 1).
		WillReturnRows(rows)

	perm, err := repo.GetByCode(context.Background(), "user:create")
	require.NoError(t, err)
	assert.Equal(t, "user:create", perm.Code)
}

func TestPermRepo_Delete(t *testing.T) {
	gormDB, mock := newPermTestDB(t)
	repo := NewPermissionRepository(gormDB)

	mock.ExpectExec("DELETE FROM `permissions` WHERE id = \\?").
		WithArgs("p-001").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), "p-001")
	assert.NoError(t, err)
}

func TestPermRepo_Count(t *testing.T) {
	gormDB, mock := newPermTestDB(t)
	repo := NewPermissionRepository(gormDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(42)
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `permissions`").WillReturnRows(rows)

	total, err := repo.Count(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(42), total)
}