package repository

import (
	"meteorx/internal/modules/wiki/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupSQLiteMemory 创建内存 SQLite 数据库并自动迁移表结构
func setupSQLiteMemory() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=busy_timeout(5000)"), &gorm.Config{
		Logger: logger.Discard, // 测试时关闭日志
	})
	if err != nil {
		return nil, err
	}

	// 自动迁移所有 Wiki 相关表（与生产 AutoMigrate 保持对齐，确保 purge 级联等涉及全表的操作可真实执行）
	if err := db.AutoMigrate(
		&model.WikiSpace{},
		&model.WikiNode{},
		&model.Document{},
		&model.DocumentRevision{},
		&model.WikiSpaceMember{},
		&model.WikiNodePermission{},
		&model.TrashItem{},
		&model.Attachment{},
		&model.Tag{},
		&model.DocumentTag{},
		&model.Comment{},
		&model.ShareLink{},
		&model.DocumentTemplate{},
		&model.DocumentAccessLog{},
		&model.DocumentSubscription{},
		&model.Notification{},
		&model.EditLock{},
		&model.ReviewComment{},
	); err != nil {
		return nil, err
	}

	return db, nil
}
