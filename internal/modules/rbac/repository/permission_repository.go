package repository

import (
	"context"
	"errors"
	"meteorx/internal/modules/rbac/model"
	"time"

	"gorm.io/gorm"
)

type PermissionPO struct {
	ID          string    `gorm:"primaryKey;size:26;comment:权限ID"`
	Name        string    `gorm:"size:50;not null;comment:权限名称"`
	Code        string    `gorm:"size:100;not null;uniqueIndex;comment:权限编码"`
	Description string    `gorm:"size:255;comment:权限描述"`
	Resource    string    `gorm:"size:50;not null;comment:资源类型"`
	Action      string    `gorm:"size:50;not null;comment:操作类型"`
	Status      int       `gorm:"default:1;comment:状态:1-启用 0-禁用"`
	CreatedAt   time.Time `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime;comment:更新时间"`
}

func (PermissionPO) TableName() string {
	return "permissions"
}

func (record PermissionPO) toDomain() *model.Permission {
	return &model.Permission{
		ID:          record.ID,
		Name:        record.Name,
		Code:        record.Code,
		Description: record.Description,
		Resource:    record.Resource,
		Action:      record.Action,
		Status:      record.Status,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

type permissionRepository struct {
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &permissionRepository{db: db}
}

func (r *permissionRepository) Create(ctx context.Context, permission *model.Permission) error {
	record := PermissionPO{
		ID:          permission.ID,
		Name:        permission.Name,
		Code:        permission.Code,
		Description: permission.Description,
		Resource:    permission.Resource,
		Action:      permission.Action,
		Status:      permission.Status,
	}
	return r.db.WithContext(ctx).Create(&record).Error
}

func (r *permissionRepository) GetByID(ctx context.Context, id string) (*model.Permission, error) {
	var record PermissionPO
	if err := r.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return record.toDomain(), nil
}

func (r *permissionRepository) GetByCode(ctx context.Context, code string) (*model.Permission, error) {
	var record PermissionPO
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&record).Error; err != nil {
		return nil, err
	}
	return record.toDomain(), nil
}

func (r *permissionRepository) List(ctx context.Context, page, pageSize int, resource, keyword string) ([]*model.Permission, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&PermissionPO{})
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []PermissionPO
	offset := (page - 1) * pageSize
	if page <= 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	permissions := make([]*model.Permission, len(records))
	for i, record := range records {
		permissions[i] = record.toDomain()
	}
	return permissions, total, nil
}

func (r *permissionRepository) Update(ctx context.Context, permission *model.Permission) error {
	return r.db.WithContext(ctx).Model(&PermissionPO{}).Where("id = ?", permission.ID).Updates(map[string]interface{}{
		"name":        permission.Name,
		"code":        permission.Code,
		"description": permission.Description,
		"resource":    permission.Resource,
		"action":      permission.Action,
		"status":      permission.Status,
		"updated_at":  time.Now(),
	}).Error
}

func (r *permissionRepository) UpdateStatus(ctx context.Context, id string, status int) error {
	result := r.db.WithContext(ctx).Model(&PermissionPO{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("权限不存在")
	}
	return nil
}

func (r *permissionRepository) BatchUpdateStatus(ctx context.Context, ids []string, status int) (int64, error) {
	result := r.db.WithContext(ctx).Model(&PermissionPO{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *permissionRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&PermissionPO{}, "id = ?", id).Error
}

func (r *permissionRepository) BatchDelete(ctx context.Context, ids []string) (int64, error) {
	result := r.db.WithContext(ctx).Model(&PermissionPO{}).
		Where("id IN ?", ids).
		Delete(&PermissionPO{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}