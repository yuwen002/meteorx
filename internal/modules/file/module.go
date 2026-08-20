package file

import (
	"meteorx/internal/modules/file/model"

	"gorm.io/gorm"
)

// AutoMigrate 自动迁移文件表
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&model.File{})
}
