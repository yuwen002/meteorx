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

func newRoleTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func TestRoleRepo_Create(t *testing.T) {
	gormDB, mock := newRoleTestDB(t)
	repo := NewRoleRepository(gormDB)

	mock.ExpectExec("INSERT INTO `roles`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), &model.Role{
		ID: "r-001", Name: "Admin", Code: "admin", TenantID: "t-001", Scope: "tenant", Status: 1,
	})
	assert.NoError(t, err)
}

func TestRoleRepo_GetByID(t *testing.T) {
	gormDB, mock := newRoleTestDB(t)
	repo := NewRoleRepository(gormDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "name", "code", "description", "tenant_id", "is_system", "scope", "status", "created_at", "updated_at", "deleted_at",
	}).AddRow("r-001", "Admin", "admin", "Administrator role", "t-001", true, "system", 1, now, now, nil)

	mock.ExpectQuery("SELECT .+ FROM `roles` WHERE id = \\? .+ LIMIT \\?").
		WithArgs("r-001", 1).
		WillReturnRows(rows)

	role, err := repo.GetByID(context.Background(), "r-001")
	require.NoError(t, err)
	assert.Equal(t, "r-001", role.ID)
	assert.Equal(t, "admin", role.Code)
}

func TestRoleRepo_GetByCode(t *testing.T) {
	gormDB, mock := newRoleTestDB(t)
	repo := NewRoleRepository(gormDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "name", "code", "description", "tenant_id", "is_system", "scope", "status", "created_at", "updated_at", "deleted_at",
	}).AddRow("r-001", "Admin", "admin", "Administrator role", "t-001", true, "system", 1, now, now, nil)

	mock.ExpectQuery("SELECT .+ FROM `roles` WHERE code = \\? AND tenant_id = \\? .+ LIMIT \\?").
		WithArgs("admin", "t-001", 1).
		WillReturnRows(rows)

	role, err := repo.GetByCode(context.Background(), "t-001", "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Code)
}

func TestRoleRepo_Update(t *testing.T) {
	gormDB, mock := newRoleTestDB(t)
	repo := NewRoleRepository(gormDB)

	mock.ExpectExec("UPDATE `roles` SET .+ WHERE id = \\? AND `roles`\\.\\`deleted_at\\` IS NULL").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Update(context.Background(), &model.Role{
		ID: "r-001", Name: "Admin Updated", Code: "admin", Description: "updated", Scope: "tenant", Status: 1,
	})
	assert.NoError(t, err)
}

func TestRoleRepo_UpdateStatus(t *testing.T) {
	gormDB, mock := newRoleTestDB(t)
	repo := NewRoleRepository(gormDB)

	mock.ExpectExec("UPDATE `roles` SET .+ WHERE id = \\? AND `roles`\\.\\`deleted_at\\` IS NULL").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateStatus(context.Background(), "r-001", 0)
	assert.NoError(t, err)
}