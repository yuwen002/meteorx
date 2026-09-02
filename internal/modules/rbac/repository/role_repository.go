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
	Scope       string         `gorm:"size:20;default:'tenant';comment:作用域:system/tenant/all"`
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
		Scope:       record.Scope,
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
		Scope:       role.Scope,
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

// ListByScope 查询指定作用域下可用角色（scope 匹配或为 all，且状态为启用）
// 当 scope = "all" 时，返回所有启用的角色（不限定作用域）
func (r *roleRepository) ListByScope(ctx context.Context, scope string) ([]*model.Role, error) {
	var records []RolePO
	db := r.db.WithContext(ctx)

	if scope == model.RoleScopeAll {
		// scope=all 时返回所有启用的角色
		db = db.Where("status = ?", model.RoleStatusEnabled)
	} else {
		// 指定 scope 时，返回该 scope 或 scope=all 的角色
		db = db.Where("(scope = ? OR scope = ?) AND status = ?", scope, model.RoleScopeAll, model.RoleStatusEnabled)
	}

	if err := db.Order("created_at ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	roles := make([]*model.Role, len(records))
	for i, record := range records {
		roles[i] = record.toDomain()
	}
	return roles, nil
}

// ListSystemAdminRoles 查询系统管理员角色（scope 为 system 或 all，且状态为启用）
func (r *roleRepository) ListSystemAdminRoles(ctx context.Context) ([]*model.Role, error) {
	var records []RolePO
	db := r.db.WithContext(ctx)

	// 查询 scope 为 system 或 all 且状态为启用的角色（不限制 is_system）
	db = db.Where("(scope = ? OR scope = ?) AND status = ?",
		model.RoleScopeSystem, model.RoleScopeAll, model.RoleStatusEnabled)

	if err := db.Order("created_at ASC").Find(&records).Error; err != nil {
		return nil, err
	}
	roles := make([]*model.Role, len(records))
	for i, record := range records {
		roles[i] = record.toDomain()
	}
	return roles, nil
}

func (r *roleRepository) Update(ctx context.Context, role *model.Role) error {
	return r.db.WithContext(ctx).Model(&RolePO{}).Where("id = ?", role.ID).Updates(map[string]interface{}{
		"name":        role.Name,
		"code":        role.Code,
		"description": role.Description,
		"tenant_id":   role.TenantID,
		"scope":       role.Scope,
		"status":      role.Status,
		"updated_at":  time.Now(),
	}).Error
}

// UpdateStatus 仅更新角色状态（启用/禁用）
func (r *roleRepository) UpdateStatus(ctx context.Context, id string, status int) error {
	result := r.db.WithContext(ctx).Model(&RolePO{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("角色不存在")
	}
	return nil
}

// BatchUpdateStatus 批量更新角色状态，返回实际更新的行数
func (r *roleRepository) BatchUpdateStatus(ctx context.Context, ids []string, status int) (int64, error) {
	result := r.db.WithContext(ctx).Model(&RolePO{}).
		Where("id IN ? AND is_system = ?", ids, false).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (r *roleRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&RolePO{}, "id = ?", id).Error
}

// BatchDelete 批量软删除角色，返回实际删除的行数
func (r *roleRepository) BatchDelete(ctx context.Context, ids []string) (int64, error) {
	result := r.db.WithContext(ctx).Model(&RolePO{}).
		Where("id IN ? AND is_system = ?", ids, false).
		Delete(&RolePO{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
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

// PermanentDelete 永久删除角色（物理删除，不可恢复）
func (r *roleRepository) PermanentDelete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Unscoped().Delete(&RolePO{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("角色不存在")
	}
	return nil
}

// BatchPermanentDelete 批量永久删除角色，返回实际删除的数量
func (r *roleRepository) BatchPermanentDelete(ctx context.Context, ids []string) (int64, error) {
	result := r.db.WithContext(ctx).Unscoped().Where("id IN ?", ids).Delete(&RolePO{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// Count 统计角色总数（按租户过滤，不传 tenantID 则统计全部）
func (r *roleRepository) Count(ctx context.Context, tenantID string) (int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&RolePO{})
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	err := query.Count(&total).Error
	return total, err
}