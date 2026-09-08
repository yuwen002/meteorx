package repository

import (
	"meteorx/internal/modules/wiki/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupSQLiteMemory 创建内存 SQLite 数据库并自动迁移表结构
func setupSQLiteMemory() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Discard, // 测试时关闭日志
	})
	if err != nil {
		return nil, err
	}

	// 自动迁移所有 Wiki 相关表
	if err := db.AutoMigrate(
		&model.WikiSpace{},
		&model.WikiSpaceMember{},
		&model.WikiNode{},
		&model.Document{},
		&model.DocumentRevision{},
		&model.TrashItem{},
		&model.DocumentTag{},
	); err != nil {
		return nil, err
	}

	return db, nil
}
