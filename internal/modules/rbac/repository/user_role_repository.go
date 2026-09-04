package repository

import (
	"context"
	"errors"
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

// DeleteByUserIDAndRoleID 删除用户的单个角色
func (r *userRoleRepository) DeleteByUserIDAndRoleID(ctx context.Context, userID, roleID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Delete(&UserRolePO{}).Error
}

// GetUserIDsByRoleID 查询拥有某角色的所有用户ID
func (r *userRoleRepository) GetUserIDsByRoleID(ctx context.Context, roleID string) ([]string, error) {
	var records []UserRolePO
	if err := r.db.WithContext(ctx).Where("role_id = ?", roleID).Find(&records).Error; err != nil {
		return nil, err
	}
	userIDs := make([]string, len(records))
	for i, record := range records {
		userIDs[i] = record.UserID
	}
	return userIDs, nil
}

// CheckUserExists 校验用户是否存在（用于存在性校验）
func (r *userRoleRepository) CheckUserExists(ctx context.Context, userID string) error {
	var count int64
	err := r.db.WithContext(ctx).Table("users").Where("id = ?", userID).Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("用户不存在")
	}
	return nil
}

// CountByUserID 查询某用户已绑定的角色数量
func (r *userRoleRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&UserRolePO{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// ListUserRoles 分页查询用户-角色关系，联表返回用户和角色信息
func (r *userRoleRepository) ListUserRoles(ctx context.Context, page, pageSize int, userID, roleID string) ([]*model.UserRole, int64, error) {
	type row struct {
		UserID     string    `gorm:"column:user_id"`
		RoleID     string    `gorm:"column:role_id"`
		CreatedAt  time.Time `gorm:"column:created_at"`
		TenantID   string    `gorm:"column:tenant_id"`
		Username   string    `gorm:"column:username"`
		Nickname   string    `gorm:"column:nickname"`
		Email      string    `gorm:"column:email"`
		UserStatus int       `gorm:"column:user_status"`
		IsMaster   bool      `gorm:"column:is_master"`
		RoleName   string    `gorm:"column:role_name"`
		RoleCode   string    `gorm:"column:role_code"`
		RoleScope  string    `gorm:"column:role_scope"`
	}

	var total int64
	var rows []row

	baseQuery := r.db.WithContext(ctx).Table("user_roles ur").
		Joins("JOIN users u ON u.id = ur.user_id").
		Joins("JOIN roles r ON r.id = ur.role_id")

	if userID != "" {
		baseQuery = baseQuery.Where("ur.user_id = ?", userID)
	}
	if roleID != "" {
		baseQuery = baseQuery.Where("ur.role_id = ?", roleID)
	}

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := baseQuery.Select(
		"ur.user_id", "ur.role_id", "ur.created_at",
		"u.tenant_id", "u.username", "u.nickname", "u.email",
		"u.status as user_status", "u.is_master",
		"r.name as role_name", "r.code as role_code", "r.scope as role_scope",
	)

	if page > 0 && pageSize > 0 {
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	if err := query.Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*model.UserRole, len(rows))
	for i, item := range rows {
		result[i] = &model.UserRole{
			UserID:    item.UserID,
			RoleID:    item.RoleID,
			CreatedAt: item.CreatedAt,
			User: &model.RoleUserInfo{
				ID:       item.UserID,
				TenantID: item.TenantID,
				Username: item.Username,
				Nickname: item.Nickname,
				Email:    item.Email,
				Status:   item.UserStatus,
				IsMaster: item.IsMaster,
			},
			Role: &model.Role{
				ID:    item.RoleID,
				Name:  item.RoleName,
				Code:  item.RoleCode,
				Scope: item.RoleScope,
			},
		}
	}
	return result, total, nil
}
