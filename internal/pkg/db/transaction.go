// Package db 提供数据库事务管理工具，封装 GORM 事务操作和上下文传递。
package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// TxManager 事务管理器，封装 GORM 事务并提供上下文传递。
type TxManager struct {
	db *gorm.DB
}

// NewTxManager 创建事务管理器。
func NewTxManager(db *gorm.DB) *TxManager {
	return &TxManager{db: db}
}

// DB 返回底层 GORM 连接。
func (m *TxManager) DB() *gorm.DB {
	return m.db
}

// WithTx 在一个事务内执行 fn，并将 tx 存入上下文供下游仓库复用。
func (m *TxManager) WithTx(ctx context.Context, fn func(ctx context.Context, tx *gorm.DB) error) error {
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx, tx)
	})
}

type txKey struct{}

// GetTx 从上下文读取事务 DB，不存在时返回 nil。
func GetTx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return nil
}

// GetDB 优先返回上下文中的事务 DB，否则回退到默认 DB。
func GetDB(ctx context.Context, defaultDB *gorm.DB) *gorm.DB {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return defaultDB
}

// Repository 支持注入事务连接的仓库约束。
type Repository interface {
	SetTx(tx *gorm.DB)
}

// WithReadCommitted 返回使用读已提交隔离级别的 DB（预留扩展，目前直接返回）。
func WithReadCommitted(db *gorm.DB) *gorm.DB {
	return db
}

// ErrTransactionFailed 事务执行失败的哨兵错误。
var ErrTransactionFailed = fmt.Errorf("transaction failed")
