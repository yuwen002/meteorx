// Package file 提供文件管理模块，支持上传、下载、存储抽象和本地/云存储后端切换。
package file

import (
	"meteorx/internal/modules/file/model"

	"gorm.io/gorm"
)

// AutoMigrate 自动迁移文件表
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&model.File{})
}
