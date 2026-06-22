package repository

import (
	"context"
	"errors"
	"meteorx/internal/modules/rbac/model"
	"time"

	"gorm.io/gorm"
)

type RolePO struct {
	ID          string         `gorm:"primaryKey;size:26;comment:角色ID"`
	Name        string         `gorm:"size:50;not null;comment:角色名称"`
	Code        string         `gorm:"size:50;not null;index;comment:角色编码"`
	Description string         `gorm:"size:255;comment:角色描述"`
	TenantID    string         `gorm:"index;size:26;comment:租户ID"`
	IsSystem    bool           `gorm:"default:false;comment:是否系统内置"`
	Status      int            `gorm:"default:1;comment:状态:1-启用 0-禁用"`
	CreatedAt   time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt   gorm.DeletedAt `gorm:"index;comment:软删除时间"`
}

func (RolePO) TableName() string {
	return "roles"
}

func (record RolePO) toDomain() *model.Role {
	role := &model.Role{
		ID:          record.ID,
		Name:        record.Name,
		Code:        record.Code,
		Description: record.Description,
		TenantID:    record.TenantID,
		IsSystem:    record.IsSystem,
		Status:      record.Status,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
	if record.DeletedAt.Valid {
		role.DeletedAt = &record.DeletedAt.Time
	}
	return role
}

type roleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) Create(ctx context.Context, role *model.Role) error {
	record := RolePO{
		ID:          role.ID,
		Name:        role.Name,
		Code:        role.Code,
		Description: role.Description,
		TenantID:    role.TenantID,
		IsSystem:    role.IsSystem,
		Status:      role.Status,
	}
	return r.db.WithContext(ctx).Create(&record).Error
}

func (r *roleRepository) GetByID(ctx context.Context, id string) (*model.Role, error) {
	var record RolePO
	if err := r.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return record.toDomain(), nil
}

func (r *roleRepository) GetByCode(ctx context.Context, tenantID, code string) (*model.Role, error) {
	var record RolePO
	query := r.db.WithContext(ctx).Where("code = ?", code)
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if err := query.First(&record).Error; err != nil {
		return nil, err
	}
	return record.toDomain(), nil
}

func (r *roleRepository) List(ctx context.Context, tenantID string, page, pageSize int, keyword string) ([]*model.Role, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&RolePO{})
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []RolePO
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

	roles := make([]*model.Role, len(records))
	for i, record := range records {
		roles[i] = record.toDomain()
	}
	return roles, total, nil
}

func (r *roleRepository) Update(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Model(&RolePO{}).Where("id = ?", role.ID).Updates(map[string]interface{}{
		"name":        role.Name,
		"code":        role.Code,
		"description": role.Description,
		"tenant_id":   role.TenantID,
		"status":      role.Status,
		"updated_at":  time.Now(),
	}).Error
}

func (r *roleRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&RolePO{}, "id = ?", id).Error
}

// FindDeleted 分页查询已软删除的角色列表
func (r *roleRepository) FindDeleted(ctx context.Context, page, pageSize int, keyword string) ([]*model.Role, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Unscoped().Model(&RolePO{}).Where("deleted_at IS NOT NULL")
	if keyword != "" {
		query = query.Where("name LIKE ? OR code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []RolePO
	offset := (page - 1) * pageSize
	if page <= 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	if err := query.Order("deleted_at DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	roles := make([]*model.Role, len(records))
	for i, record := range records {
		roles[i] = record.toDomain()
	}
	return roles, total, nil
}

// Restore 恢复已软删除的角色
func (r *roleRepository) Restore(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Unscoped().Model(&RolePO{}).
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Update("deleted_at", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("角色不存在或未删除")
	}
	return nil
}
