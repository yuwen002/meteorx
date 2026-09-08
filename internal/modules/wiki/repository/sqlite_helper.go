package repository

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"meteorx/internal/modules/wiki/model"
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
		&model.WikiTag{},
		&model.DocumentTag{},
		&model.WikiAttachment{},
		&model.WikiComment{},
		&model.WikiShare{},
	); err != nil {
		return nil, err
	}

	return db, nil
}