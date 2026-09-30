// Package repository 提供 RBAC 模块的数据访问实现与表结构迁移。
package repository

import "gorm.io/gorm"

// AutoMigrate 自动迁移 RBAC 相关表结构（角色/用户角色/权限/角色权限）。
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&RolePO{}, &PermissionPO{}, &RolePermissionPO{}, &UserRolePO{})
}
