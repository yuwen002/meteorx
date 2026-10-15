// Package repository 提供 RBAC 模块的数据访问实现与表结构迁移。
package repository

import "gorm.io/gorm"

// Models 返回本模块全部待迁移模型，作为 AutoMigrate 与脏列巡检的单一模型来源。
func Models() []interface{} {
	return []interface{}{&RolePO{}, &PermissionPO{}, &RolePermissionPO{}, &UserRolePO{}}
}

// AutoMigrate 自动迁移 RBAC 相关表结构（角色/用户角色/权限/角色权限）。
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(Models()...)
}
