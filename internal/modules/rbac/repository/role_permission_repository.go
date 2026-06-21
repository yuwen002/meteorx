package repository

import (
	"context"
	"meteorx/internal/modules/rbac/model"
	"time"

	"gorm.io/gorm"
)

type RolePermissionPO struct {
	RoleID       string    `gorm:"primaryKey;size:26;comment:角色ID"`
	PermissionID string    `gorm:"primaryKey;size:26;comment:权限ID"`
	CreatedAt    time.Time `gorm:"autoCreateTime;comment:创建时间"`
}

func (RolePermissionPO) TableName() string {
	return "role_permissions"
}

type rolePermissionRepository struct {
	db *gorm.DB
}

func NewRolePermissionRepository(db *gorm.DB) RolePermissionRepository {
	return &rolePermissionRepository{db: db}
}

func (r *rolePermissionRepository) BindPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除旧绑定
		if err := tx.Where("role_id = ?", roleID).Delete(&RolePermissionPO{}).Error; err != nil {
			return err
		}
		// 插入新绑定
		for _, pid := range permissionIDs {
			record := RolePermissionPO{
				RoleID:       roleID,
				PermissionID: pid,
				CreatedAt:    time.Now(),
			}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *rolePermissionRepository) GetPermissionsByRoleID(ctx context.Context, roleID string) ([]*model.Permission, error) {
	var records []PermissionPO
	if err := r.db.WithContext(ctx).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&records).Error; err != nil {
		return nil, err
	}

	permissions := make([]*model.Permission, len(records))
	for i, record := range records {
		permissions[i] = record.toDomain()
	}
	return permissions, nil
}

func (r *rolePermissionRepository) GetPermissionCodesByRoleID(ctx context.Context, roleID string) ([]string, error) {
	var codes []string
	if err := r.db.WithContext(ctx).Model(&PermissionPO{}).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Pluck("permissions.code", &codes).Error; err != nil {
		return nil, err
	}
	return codes, nil
}

func (r *rolePermissionRepository) UnbindPermission(ctx context.Context, roleID, permissionID string) error {
	return r.db.WithContext(ctx).
		Where("role_id = ? AND permission_id = ?", roleID, permissionID).
		Delete(&RolePermissionPO{}).Error
}
