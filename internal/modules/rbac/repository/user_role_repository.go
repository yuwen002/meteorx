package repository

import (
	"context"
	"meteorx/internal/modules/rbac/model"
	"time"

	"gorm.io/gorm"
)

// UserRolePO 用户-角色关联表（多对多）
type UserRolePO struct {
	UserID    string    `gorm:"primaryKey;size:26;comment:用户ID"`
	RoleID    string    `gorm:"primaryKey;size:26;comment:角色ID"`
	CreatedAt time.Time `gorm:"autoCreateTime;comment:创建时间"`
}

func (UserRolePO) TableName() string {
	return "user_roles"
}

func (record UserRolePO) toDomain() *model.UserRole {
	return &model.UserRole{
		UserID:    record.UserID,
		RoleID:    record.RoleID,
		CreatedAt: record.CreatedAt,
	}
}

type userRoleRepository struct {
	db *gorm.DB
}

func NewUserRoleRepository(db *gorm.DB) UserRoleRepository {
	return &userRoleRepository{db: db}
}

// AssignRoles 为用户分配角色（先清空旧的角色后批量插入）
func (r *userRoleRepository) AssignRoles(ctx context.Context, userID string, roleIDs []string) error {
	if len(roleIDs) == 0 {
		// 没有角色时直接清空
		return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&UserRolePO{}).Error
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 先清空当前用户的所有角色
		if err := tx.Where("user_id = ?", userID).Delete(&UserRolePO{}).Error; err != nil {
			return err
		}
		// 2. 批量插入新角色
		records := make([]UserRolePO, len(roleIDs))
		for i, roleID := range roleIDs {
			records[i] = UserRolePO{
				UserID: userID,
				RoleID: roleID,
			}
		}
		return tx.Create(&records).Error
	})
}

// GetRoleIDsByUserID 查询用户关联的所有角色ID
func (r *userRoleRepository) GetRoleIDsByUserID(ctx context.Context, userID string) ([]string, error) {
	var records []UserRolePO
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&records).Error; err != nil {
		return nil, err
	}
	roleIDs := make([]string, len(records))
	for i, record := range records {
		roleIDs[i] = record.RoleID
	}
	return roleIDs, nil
}

// GetRoleCodesByUserID 查询用户关联的所有角色编码（联查 roles 表）
func (r *userRoleRepository) GetRoleCodesByUserID(ctx context.Context, userID string) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND roles.status = ?", userID, 1).
		Pluck("roles.code", &codes).Error
	return codes, err
}

// BatchGetRoleIDsByUserIDs 批量查询多个用户的角色ID
func (r *userRoleRepository) BatchGetRoleIDsByUserIDs(ctx context.Context, userIDs []string) (map[string][]string, error) {
	var records []UserRolePO
	if err := r.db.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&records).Error; err != nil {
		return nil, err
	}
	result := make(map[string][]string)
	for _, record := range records {
		result[record.UserID] = append(result[record.UserID], record.RoleID)
	}
	return result, nil
}

// DeleteByUserID 删除指定用户的所有角色
func (r *userRoleRepository) DeleteByUserID(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&UserRolePO{}).Error
}

// CountByRoleID 查询某角色被多少用户使用
func (r *userRoleRepository) CountByRoleID(ctx context.Context, roleID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&UserRolePO{}).Where("role_id = ?", roleID).Count(&count).Error
	return count, err
}