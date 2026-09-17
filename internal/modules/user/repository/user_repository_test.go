package repository

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/modules/user/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

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

func sampleUser() *model.User {
	now := time.Now()
	return &model.User{
		ID:        "u-001",
		TenantID:  "t-001",
		Username:  "alice",
		Password:  "$2a$hashed",
		Nickname:  "Alice",
		Email:     "alice@example.com",
		Status:    1,
		IsMaster:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestCreate(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	mock.ExpectExec("INSERT INTO `users`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), sampleUser())
	assert.NoError(t, err)
}

func TestGetByID(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "tenant_id", "username", "password", "nickname", "email", "status", "is_master", "created_at", "updated_at", "deleted_at"}).
		AddRow("u-001", "t-001", "alice", "$2a$hashed", "Alice", "alice@example.com", 1, false, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `users` WHERE id = \\? .+ LIMIT \\?").
		WithArgs("u-001", 1).
		WillReturnRows(rows)

	user, err := repo.GetByID(context.Background(), "u-001")
	assert.NoError(t, err)
	assert.Equal(t, "u-001", user.ID)
	assert.Equal(t, "alice", user.Username)
}

func TestGetByUsername(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "tenant_id", "username", "password", "nickname", "email", "status", "is_master", "created_at", "updated_at", "deleted_at"}).
		AddRow("u-001", "t-001", "alice", "$2a$hashed", "Alice", "alice@example.com", 1, false, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `users` WHERE .+ LIMIT \\?").
		WithArgs("t-001", "alice", 1).
		WillReturnRows(rows)

	user, err := repo.GetByUsername(context.Background(), "t-001", "alice")
	assert.NoError(t, err)
	assert.Equal(t, "alice", user.Username)
}

func TestGetByUsername_Superadmin(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "tenant_id", "username", "password", "nickname", "email", "status", "is_master", "created_at", "updated_at", "deleted_at"}).
		AddRow("root-1", "SYSTEM_ROOT", "root", "$2a$hashed", "Root", "root@meteorx.com", 1, true, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `users` WHERE .+ LIMIT \\?").
		WithArgs("SYSTEM_ROOT", "root", true, 1).
		WillReturnRows(rows)

	user, err := repo.GetByUsername(context.Background(), "", "root")
	assert.NoError(t, err)
	assert.True(t, user.IsMaster)
	assert.Equal(t, "SYSTEM_ROOT", user.TenantID)
}

func TestGetByEmail(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "tenant_id", "username", "password", "nickname", "email", "status", "is_master", "created_at", "updated_at", "deleted_at"}).
		AddRow("u-001", "t-001", "alice", "$2a$hashed", "Alice", "alice@example.com", 1, false, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `users` WHERE email = \\? .+ LIMIT \\?").
		WithArgs("alice@example.com", 1).
		WillReturnRows(rows)

	user, err := repo.GetByEmail(context.Background(), "alice@example.com")
	assert.NoError(t, err)
	assert.Equal(t, "alice@example.com", user.Email)
}

func TestUsernameExists_True(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT count.+FROM `users` WHERE username = \\?").
		WithArgs("alice").
		WillReturnRows(rows)

	exists, err := repo.UsernameExists(context.Background(), "alice")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestUsernameExists_False(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery("SELECT count.+FROM `users` WHERE username = \\?").
		WithArgs("nobody").
		WillReturnRows(rows)

	exists, err := repo.UsernameExists(context.Background(), "nobody")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestListByTenant(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	now := time.Now()
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mock.ExpectQuery("SELECT count.+FROM `users` WHERE tenant_id = \\?").
		WithArgs("t-001").
		WillReturnRows(countRows)

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "username", "password", "nickname", "email", "status", "is_master", "created_at", "updated_at", "deleted_at"}).
		AddRow("u-001", "t-001", "alice", "$2a$", "Alice", "a@t.com", 1, false, now, now, nil).
		AddRow("u-002", "t-001", "bob", "$2a$", "Bob", "b@t.com", 1, false, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `users` WHERE .+ LIMIT \\?").
		WithArgs("t-001", 10).
		WillReturnRows(rows)

	users, total, err := repo.ListByTenant(context.Background(), "t-001", 1, 10, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, users, 2)
}

func TestListByTenant_WithKeyword(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	now := time.Now()
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT count.+FROM `users` WHERE .+").
		WillReturnRows(countRows)

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "username", "password", "nickname", "email", "status", "is_master", "created_at", "updated_at", "deleted_at"}).
		AddRow("u-001", "t-001", "alice", "$2a$", "Alice", "a@t.com", 1, false, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `users` WHERE .+ LIMIT \\?").
		WillReturnRows(rows)

	users, total, err := repo.ListByTenant(context.Background(), "t-001", 1, 10, "alice", nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, users, 1)
}

func TestListByTenant_WithStatus(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	now := time.Now()
	status := 0
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT count.+FROM `users` WHERE .+").
		WillReturnRows(countRows)

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "username", "password", "nickname", "email", "status", "is_master", "created_at", "updated_at", "deleted_at"}).
		AddRow("u-001", "t-001", "bob", "$2a$", "Bob", "b@t.com", 0, false, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `users` WHERE .+ LIMIT \\?").
		WillReturnRows(rows)

	users, total, err := repo.ListByTenant(context.Background(), "t-001", 1, 10, "", &status)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, users, 1)
	assert.Equal(t, 0, users[0].Status)
}

func TestListMasterAdmins(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	now := time.Now()
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT count.+FROM `users` WHERE is_master = \\?").
		WithArgs(true).
		WillReturnRows(countRows)

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "username", "password", "nickname", "email", "status", "is_master", "created_at", "updated_at", "deleted_at"}).
		AddRow("root-1", "SYSTEM_ROOT", "root", "$2a$", "Root", "root@m.com", 1, true, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `users` WHERE .+ LIMIT \\?").
		WithArgs(true, 10).
		WillReturnRows(rows)

	users, total, err := repo.ListMasterAdmins(context.Background(), 1, 10, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, users, 1)
	assert.True(t, users[0].IsMaster)
}

func TestListAllTenantUsers(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	now := time.Now()
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mock.ExpectQuery("SELECT count.+FROM `users` WHERE is_master = \\?").
		WithArgs(false).
		WillReturnRows(countRows)

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "username", "password", "nickname", "email", "status", "is_master", "created_at", "updated_at", "deleted_at"}).
		AddRow("u-001", "t-001", "alice", "$2a$", "Alice", "a@t.com", 1, false, now, now, nil).
		AddRow("u-002", "t-002", "bob", "$2a$", "Bob", "b@t.com", 1, false, now, now, nil)
	mock.ExpectQuery("SELECT .+ FROM `users` WHERE .+ LIMIT \\?").
		WithArgs(false, 10).
		WillReturnRows(rows)

	users, total, err := repo.ListAllTenantUsers(context.Background(), 1, 10, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, users, 2)
}

func TestUpdate(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	mock.ExpectExec("UPDATE `users` SET .+ WHERE id = \\?").
		WillReturnResult(sqlmock.NewResult(0, 1))

	u := sampleUser()
	u.Nickname = "Alice Updated"
	err := repo.Update(context.Background(), u)
	assert.NoError(t, err)
}

func TestUpdate_WithPassword(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	mock.ExpectExec("UPDATE `users` SET .+ WHERE id = \\?").
		WillReturnResult(sqlmock.NewResult(0, 1))

	u := sampleUser()
	u.Password = "newpass"
	err := repo.Update(context.Background(), u)
	assert.NoError(t, err)
}

func TestDelete(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	mock.ExpectExec("UPDATE `users` SET .+ WHERE .+").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(context.Background(), "u-001")
	assert.NoError(t, err)
}

func TestUpdateStatus(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	mock.ExpectExec("UPDATE `users` SET .+ WHERE id = \\?").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateStatus(context.Background(), "u-001", 0)
	assert.NoError(t, err)
}

func TestFindDeletedMasterAdmins(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	now := time.Now()
	deletedAt := gorm.DeletedAt{Time: now, Valid: true}
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT count.+FROM `users` WHERE is_master = \\? AND deleted_at IS NOT NULL").
		WithArgs(true).
		WillReturnRows(countRows)

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "username", "password", "nickname", "email", "status", "is_master", "created_at", "updated_at", "deleted_at"}).
		AddRow("root-old", "SYSTEM_ROOT", "oldroot", "$2a$", "Old", "old@m.com", 1, true, now, now, deletedAt)
	mock.ExpectQuery("SELECT .+ FROM `users` WHERE is_master = \\? AND deleted_at IS NOT NULL.*ORDER BY deleted_at DESC LIMIT \\?").
		WillReturnRows(rows)

	users, total, err := repo.FindDeletedMasterAdmins(context.Background(), 1, 10, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, users, 1)
	assert.NotNil(t, users[0].DeletedAt)
}

func TestRestoreMasterAdmin(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	mock.ExpectExec("UPDATE `users` SET .+ WHERE .+").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.RestoreMasterAdmin(context.Background(), "root-1")
	assert.NoError(t, err)
}

func TestPermanentDeleteMasterAdmin(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	mock.ExpectExec("DELETE FROM `users` WHERE .+").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.PermanentDeleteMasterAdmin(context.Background(), "root-1")
	assert.NoError(t, err)
}

func TestBatchUpdateStatus(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	mock.ExpectExec("UPDATE `users` SET .+ WHERE .+").
		WillReturnResult(sqlmock.NewResult(0, 2))

	count, err := repo.BatchUpdateStatus(context.Background(), []string{"u-001", "u-002"}, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestBatchDelete(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	mock.ExpectExec("UPDATE `users` SET .+ WHERE .+").
		WillReturnResult(sqlmock.NewResult(0, 2))

	count, err := repo.BatchDelete(context.Background(), []string{"u-001", "u-002"})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestFindDeletedTenantUsers(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	now := time.Now()
	deletedAt := gorm.DeletedAt{Time: now, Valid: true}
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT count.+FROM `users` WHERE tenant_id = \\? AND is_master = \\? AND deleted_at IS NOT NULL").
		WithArgs("t-001", false).
		WillReturnRows(countRows)

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "username", "password", "nickname", "email", "status", "is_master", "created_at", "updated_at", "deleted_at"}).
		AddRow("u-old", "t-001", "olduser", "$2a$", "Old", "old@t.com", 1, false, now, now, deletedAt)
	mock.ExpectQuery("SELECT .+ FROM `users` WHERE .+ ORDER BY deleted_at DESC LIMIT \\?").
		WillReturnRows(rows)

	users, total, err := repo.FindDeletedTenantUsers(context.Background(), "t-001", 1, 10, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, users, 1)
	assert.NotNil(t, users[0].DeletedAt)
}

func TestFindAllDeletedTenantUsers(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	now := time.Now()
	deletedAt := gorm.DeletedAt{Time: now, Valid: true}
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT count.+FROM `users` WHERE is_master = \\? AND deleted_at IS NOT NULL").
		WithArgs(false).
		WillReturnRows(countRows)

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "username", "password", "nickname", "email", "status", "is_master", "created_at", "updated_at", "deleted_at"}).
		AddRow("u-old", "t-001", "olduser", "$2a$", "Old", "old@t.com", 1, false, now, now, deletedAt)
	mock.ExpectQuery("SELECT .+ FROM `users` WHERE .+ ORDER BY deleted_at DESC LIMIT \\?").
		WillReturnRows(rows)

	users, total, err := repo.FindAllDeletedTenantUsers(context.Background(), 1, 10, "")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, users, 1)
}

func TestRestoreTenantUser(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	mock.ExpectExec("UPDATE `users` SET .+ WHERE .+").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.RestoreTenantUser(context.Background(), "t-001", "u-001")
	assert.NoError(t, err)
}

func TestPermanentDeleteTenantUser(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	mock.ExpectExec("DELETE FROM `users` WHERE .+").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.PermanentDeleteTenantUser(context.Background(), "t-001", "u-001")
	assert.NoError(t, err)
}

func TestBatchUpdateTenantUserStatus(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	mock.ExpectExec("UPDATE `users` SET .+ WHERE .+").
		WillReturnResult(sqlmock.NewResult(0, 2))

	count, err := repo.BatchUpdateTenantUserStatus(context.Background(), "t-001", []string{"u-001", "u-002"}, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestBatchDeleteTenantUsers(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	mock.ExpectExec("UPDATE `users` SET .+ WHERE .+").
		WillReturnResult(sqlmock.NewResult(0, 2))

	count, err := repo.BatchDeleteTenantUsers(context.Background(), "t-001", []string{"u-001", "u-002"})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestCountByTenant(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(5)
	mock.ExpectQuery("SELECT count.+FROM `users` WHERE tenant_id = \\?").
		WithArgs("t-001").
		WillReturnRows(countRows)

	count, err := repo.CountByTenant(context.Background(), "t-001")
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestCountAllUsers(t *testing.T) {
	gormDB, mock := newTestDB(t)
	repo := NewUserRepository(gormDB)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(42)
	mock.ExpectQuery("SELECT count.+FROM `users`").
		WillReturnRows(countRows)

	count, err := repo.CountAllUsers(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(42), count)
}