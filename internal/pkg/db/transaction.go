package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

func (m *TxManager) DB() *gorm.DB {
	return m.db
}

func (m *TxManager) WithTx(ctx context.Context, fn func(ctx context.Context, tx *gorm.DB) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx, tx)
	})
}

type txKey struct{}

func GetTx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return nil
}

func GetDB(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return defaultDB
}

type Repository interface {
	SetTx(tx *gorm.DB)
}

func WithReadCommitted(db *gorm.DB) *gorm.DB {
	return db
}

var ErrTransactionFailed = fmt.Errorf("transaction failed")