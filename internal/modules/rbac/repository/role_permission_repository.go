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

func (r *rolePermissionRepository) GetPermissionsByRoleIDWithResource(ctx context.Context, roleID string, resource string) ([]*model.Permission, error) {
	var records []PermissionPO
	query := r.db.WithContext(ctx).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID)
	
	if resource != "" {
		query = query.Where("permissions.resource = ?", resource)
	}
	
	if err := query.Find(&records).Error; err != nil {
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

func (r *rolePermissionRepository) BatchBindPermissions(ctx context.Context, roleIDs []string, permissionIDs []string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, roleID := range roleIDs {
			for _, permissionID := range permissionIDs {
				// 检查是否已存在
				var existing int64
				if err := tx.Model(&RolePermissionPO{}).
					Where("role_id = ? AND permission_id = ?", roleID, permissionID).
					Count(&existing).Error; err != nil {
					return err
				}
				if existing > 0 {
					continue
				}
				
				record := RolePermissionPO{
					RoleID:       roleID,
					PermissionID: permissionID,
					CreatedAt:    time.Now(),
				}
				if err := tx.Create(&record).Error; err != nil {
					return err
				}
				count++
			}
		}
		return nil
	})
	return count, err
}

func (r *rolePermissionRepository) BatchUnbindPermissions(ctx context.Context, roleIDs []string, permissionIDs []string) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("role_id IN ?", roleIDs).
		Where("permission_id IN ?", permissionIDs).
		Delete(&RolePermissionPO{})
	return result.RowsAffected, result.Error
}

func (r *rolePermissionRepository) List(ctx context.Context, page, pageSize int, roleID, permissionID string) ([]*model.RolePermission, int64, error) {
	type row struct {
		RoleID        string    `gorm:"column:role_id"`
		PermissionID  string    `gorm:"column:permission_id"`
		CreatedAt     time.Time `gorm:"column:created_at"`
		RoleName      string    `gorm:"column:role_name"`
		RoleCode      string    `gorm:"column:role_code"`
		PermissionName string   `gorm:"column:permission_name"`
		PermissionCode string   `gorm:"column:permission_code"`
		PermissionResource string `gorm:"column:permission_resource"`
		PermissionAction string   `gorm:"column:permission_action"`
	}

	var total int64
	var rows []row

	baseQuery := r.db.WithContext(ctx).Table("role_permissions rp").
		Joins("JOIN roles r ON r.id = rp.role_id").
		Joins("JOIN permissions p ON p.id = rp.permission_id")

	if roleID != "" {
		baseQuery = baseQuery.Where("rp.role_id = ?", roleID)
	}
	if permissionID != "" {
		baseQuery = baseQuery.Where("rp.permission_id = ?", permissionID)
	}

	// count
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// select with columns
	query := baseQuery.Select(
		"rp.role_id", "rp.permission_id", "rp.created_at",
		"r.name as role_name", "r.code as role_code",
		"p.name as permission_name", "p.code as permission_code",
		"p.resource as permission_resource", "p.action as permission_action",
	)

	if page > 0 && pageSize > 0 {
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	if err := query.Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	list := make([]*model.RolePermission, len(rows))
	for i, item := range rows {
		list[i] = &model.RolePermission{
			RoleID:       item.RoleID,
			PermissionID: item.PermissionID,
			CreatedAt:    item.CreatedAt,
			Role: &model.Role{
				ID:   item.RoleID,
				Name: item.RoleName,
				Code: item.RoleCode,
			},
			Permission: &model.Permission{
				ID:       item.PermissionID,
				Name:     item.PermissionName,
				Code:     item.PermissionCode,
				Resource: item.PermissionResource,
				Action:   item.PermissionAction,
			},
		}
	}
	return list, total, nil
}

// CountByRoleID 查询某角色已绑定的权限数量
func (r *rolePermissionRepository) CountByRoleID(ctx context.Context, roleID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&RolePermissionPO{}).Where("role_id = ?", roleID).Count(&count).Error
	return count, err
}

// CountByPermissionID 查询某权限被多少角色使用
func (r *rolePermissionRepository) CountByPermissionID(ctx context.Context, permissionID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&RolePermissionPO{}).Where("permission_id = ?", permissionID).Count(&count).Error
	return count, err
}