package repository

import (
	"context"
	"meteorx/internal/modules/rbac/model"
	"time"

	"gorm.io/gorm"
)

type RolePO struct {
	ID          string    `gorm:"primaryKey;size:26;comment:角色ID"`
	Name        string    `gorm:"size:50;not null;comment:角色名称"`
	Code        string    `gorm:"size:50;not null;index;comment:角色编码"`
	Description string    `gorm:"size:255;comment:角色描述"`
	TenantID    string    `gorm:"index;size:26;comment:租户ID"`
	IsSystem    bool      `gorm:"default:false;comment:是否系统内置"`
	CreatedAt   time.Time `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime;comment:更新时间"`
}

func (RolePO) TableName() string {
	return "roles"
}

func (record RolePO) toDomain() *model.Role {
	return &model.Role{
		ID:          record.ID,
		Name:        record.Name,
		Code:        record.Code,
		Description: record.Description,
		TenantID:    record.TenantID,
		IsSystem:    record.IsSystem,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
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
		"updated_at":  time.Now(),
	}).Error
}

func (r *roleRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&RolePO{}, "id = ?", id).Error
}
