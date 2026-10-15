// Package file 提供文件管理模块，支持上传、下载、存储抽象和本地/云存储后端切换。
package file

import (
	"meteorx/internal/modules/file/model"

	"gorm.io/gorm"
)

// Models 返回本模块全部待迁移模型，作为 AutoMigrate 与脏列巡检的单一模型来源。
func Models() []interface{} {
	return []interface{}{&model.File{}}
}

// AutoMigrate 自动迁移文件表
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(Models()...)
}
