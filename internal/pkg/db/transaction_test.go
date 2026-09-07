package db

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newMockTxManager(t *testing.T) (*TxManager, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm: %v", err)
	}
	return NewTxManager(gormDB), mock
}

// TestWithTx_CommitsAndExposesTx 验证成功路径提交，且回调中能通过 GetTx/GetDB 取到事务句柄
func TestWithTx_CommitsAndExposesTx(t *testing.T) {
	m, mock := newMockTxManager(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	var gotTx *gorm.DB
	err := m.WithTx(context.Background(), func(ctx context.Context, tx *gorm.DB) error {
		gotTx = tx
		assert.Same(t, tx, GetTx(ctx))
		assert.Same(t, tx, GetDB(ctx, m.DB()))
		return nil
	})

	assert.NoError(t, err)
	assert.NotNil(t, gotTx)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestWithTx_RollsBackOnError 验证回调返回错误时事务回滚并传播错误
func TestWithTx_RollsBackOnError(t *testing.T) {
	m, mock := newMockTxManager(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	sentinel := errors.New("business error")
	err := m.WithTx(context.Background(), func(_ context.Context, _ *gorm.DB) error {
		return sentinel
	})

	assert.ErrorIs(t, err, sentinel)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestWithTx_PanicsRollBack 回调 panic 时事务不残留
func TestWithTx_PanicsRollBack(t *testing.T) {
	m, mock := newMockTxManager(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	assert.Panics(t, func() {
		_ = m.WithTx(context.Background(), func(_ context.Context, _ *gorm.DB) error {
			panic("boom")
		})
	})
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetTx_OutsideTransaction 无事务上下文返回 nil
func TestGetTx_OutsideTransaction(t *testing.T) {
	m, _ := newMockTxManager(t)
	assert.Nil(t, GetTx(context.Background()))
	assert.Same(t, m.DB(), GetDB(context.Background(), m.DB()))
}
