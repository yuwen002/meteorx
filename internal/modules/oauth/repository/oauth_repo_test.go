package repository

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/modules/oauth/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newOAuthTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func TestOAuthRepo_Create(t *testing.T) {
	gormDB, mock := newOAuthTestDB(t)
	repo := NewOAuthAccountRepository(gormDB)

	mock.ExpectExec("INSERT INTO `o_auth_accounts`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), &model.OAuthAccount{
		ID: "oa-001", UserID: "u-001", Provider: "google", ProviderID: "google-12345", Email: "alice@gmail.com", AccessToken: "token",
	})
	assert.NoError(t, err)
}

func TestOAuthRepo_GetByProviderAndProviderID(t *testing.T) {
	gormDB, mock := newOAuthTestDB(t)
	repo := NewOAuthAccountRepository(gormDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "provider", "provider_id", "email", "access_token", "created_at", "updated_at",
	}).AddRow("oa-001", "u-001", "google", "google-12345", "alice@gmail.com", "token", now, now)

	mock.ExpectQuery("SELECT .+ FROM `o_auth_accounts` WHERE provider = \\? AND provider_id = \\? .+ LIMIT \\?").
		WithArgs("google", "google-12345", 1).
		WillReturnRows(rows)

	account, err := repo.GetByProviderAndProviderID(context.Background(), "google", "google-12345")
	require.NoError(t, err)
	assert.Equal(t, "oa-001", account.ID)
	assert.Equal(t, "google", account.Provider)
}

func TestOAuthRepo_ListByUserID(t *testing.T) {
	gormDB, mock := newOAuthTestDB(t)
	repo := NewOAuthAccountRepository(gormDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "provider", "provider_id", "email", "access_token", "created_at", "updated_at",
	}).
		AddRow("oa-001", "u-001", "google", "google-12345", "alice@gmail.com", "token1", now, now).
		AddRow("oa-002", "u-001", "github", "github-67890", "alice@github.com", "token2", now, now)

	mock.ExpectQuery("SELECT .+ FROM `o_auth_accounts` WHERE user_id = \\?").
		WithArgs("u-001").
		WillReturnRows(rows)

	accounts, err := repo.ListByUserID(context.Background(), "u-001")
	require.NoError(t, err)
	assert.Len(t, accounts, 2)
	assert.Equal(t, "google", accounts[0].Provider)
	assert.Equal(t, "github", accounts[1].Provider)
}

func TestOAuthRepo_DeleteByID(t *testing.T) {
	gormDB, mock := newOAuthTestDB(t)
	repo := NewOAuthAccountRepository(gormDB)

	mock.ExpectExec("DELETE FROM `o_auth_accounts` WHERE id = \\?").
		WithArgs("oa-001").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.DeleteByID(context.Background(), "oa-001")
	assert.NoError(t, err)
}

func TestOAuthRepo_GetByUserIDAndProvider(t *testing.T) {
	gormDB, mock := newOAuthTestDB(t)
	repo := NewOAuthAccountRepository(gormDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "provider", "provider_id", "email", "access_token", "created_at", "updated_at",
	}).AddRow("oa-001", "u-001", "google", "google-12345", "alice@gmail.com", "token", now, now)

	mock.ExpectQuery("SELECT .+ FROM `o_auth_accounts` WHERE user_id = \\? AND provider = \\? .+ LIMIT \\?").
		WithArgs("u-001", "google", 1).
		WillReturnRows(rows)

	account, err := repo.GetByUserIDAndProvider(context.Background(), "u-001", "google")
	require.NoError(t, err)
	assert.Equal(t, "oa-001", account.ID)
	assert.Equal(t, "google", account.Provider)
}