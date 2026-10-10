package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"meteorx/internal/modules/invitation/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newInvTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func sampleInv() *model.Invitation {
	return &model.Invitation{
		ID:        "inv001",
		TenantID:  "t1",
		Email:     "alice@example.com",
		Token:     "tok-abc",
		RoleIDs:   `["role-1"]`,
		Status:    model.InvitationStatusPending,
		InvitedBy: "u1",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestInvRepo_Create(t *testing.T) {
	gormDB, mock := newInvTestDB(t)
	repo := NewInvitationRepository(gormDB)
	inv := sampleInv()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `invitations`")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), inv)
	assert.NoError(t, err)
}

func TestInvRepo_GetByToken(t *testing.T) {
	gormDB, mock := newInvTestDB(t)
	repo := NewInvitationRepository(gormDB)
	inv := sampleInv()

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "email", "token", "role_ids", "status", "invited_by", "expires_at", "accepted_at", "created_at", "updated_at"}).
		AddRow(inv.ID, inv.TenantID, inv.Email, inv.Token, inv.RoleIDs, inv.Status, inv.InvitedBy, inv.ExpiresAt, nil, inv.CreatedAt, inv.UpdatedAt)
	mock.ExpectQuery("SELECT .* FROM `invitations` WHERE token").
		WillReturnRows(rows)

	result, err := repo.GetByToken(context.Background(), "tok-abc")
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, inv.ID, result.ID)
	assert.Equal(t, inv.Token, result.Token)
}

func TestInvRepo_GetByID(t *testing.T) {
	gormDB, mock := newInvTestDB(t)
	repo := NewInvitationRepository(gormDB)
	inv := sampleInv()

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "email", "token", "role_ids", "status", "invited_by", "expires_at", "accepted_at", "created_at", "updated_at"}).
		AddRow(inv.ID, inv.TenantID, inv.Email, inv.Token, inv.RoleIDs, inv.Status, inv.InvitedBy, inv.ExpiresAt, nil, inv.CreatedAt, inv.UpdatedAt)
	mock.ExpectQuery("SELECT .* FROM `invitations` WHERE id").
		WillReturnRows(rows)

	result, err := repo.GetByID(context.Background(), "inv001")
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "inv001", result.ID)
}

func TestInvRepo_GetByID_NotFound(t *testing.T) {
	gormDB, mock := newInvTestDB(t)
	repo := NewInvitationRepository(gormDB)

	mock.ExpectQuery("SELECT .* FROM `invitations` WHERE id").
		WillReturnError(gorm.ErrRecordNotFound)

	result, err := repo.GetByID(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestInvRepo_UpdateStatus(t *testing.T) {
	gormDB, mock := newInvTestDB(t)
	repo := NewInvitationRepository(gormDB)

	mock.ExpectExec("UPDATE `invitations` SET").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateStatus(context.Background(), "inv001", model.InvitationStatusAccepted)
	assert.NoError(t, err)
}

func TestInvRepo_Delete(t *testing.T) {
	gormDB, mock := newInvTestDB(t)
	repo := NewInvitationRepository(gormDB)

	mock.ExpectExec("DELETE FROM `invitations`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Delete(context.Background(), "inv001")
	assert.NoError(t, err)
}

func TestInvRepo_CountPendingByTenant(t *testing.T) {
	gormDB, mock := newInvTestDB(t)
	repo := NewInvitationRepository(gormDB)

	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `invitations`").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.CountPendingByTenant(context.Background(), "t1")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestInvRepo_FindByEmailAndTenant(t *testing.T) {
	gormDB, mock := newInvTestDB(t)
	repo := NewInvitationRepository(gormDB)
	inv := sampleInv()

	rows := sqlmock.NewRows([]string{"id", "tenant_id", "email", "token", "role_ids", "status", "invited_by", "expires_at", "accepted_at", "created_at", "updated_at"}).
		AddRow(inv.ID, inv.TenantID, inv.Email, inv.Token, inv.RoleIDs, inv.Status, inv.InvitedBy, inv.ExpiresAt, nil, inv.CreatedAt, inv.UpdatedAt)
	mock.ExpectQuery("SELECT .* FROM `invitations` WHERE email").
		WillReturnRows(rows)

	result, err := repo.FindByEmailAndTenant(context.Background(), "alice@example.com", "t1")
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "alice@example.com", result.Email)
}

func TestInvRepo_ExpireOutdated(t *testing.T) {
	gormDB, mock := newInvTestDB(t)
	repo := NewInvitationRepository(gormDB)

	mock.ExpectExec("UPDATE `invitations` SET").
		WillReturnResult(sqlmock.NewResult(1, 2))

	affected, err := repo.ExpireOutdated(context.Background(), time.Now())
	assert.NoError(t, err)
	assert.Equal(t, int64(2), affected)
}