package repository

import (
	"context"
	"time"

	"meteorx/internal/modules/user/model"

	"gorm.io/gorm"
)

// ImportRepository 汇总批量导入所需的原子写入能力。
// 之所以与 UserRepository 分开，是因为导入的语义是"整批成功或整批失败"：
// 用户记录与 user_roles 关联必须落在同一个事务里，与逐条 CRUD 的调用方式不同，
// 塞进 UserRepository 会把事务细节泄漏给所有调用方。
type ImportRepository interface {
	// CheckUsernames 批量检查用户名占用情况。
	//   - active：库中现存（未软删除）的同名用户，语义与 UsernameExists 的"跨租户全局唯一"一致；
	//   - trash：指定租户回收站里的同名用户。软删除行仍然占用 (tenant_id, username)
	//     唯一索引，不提前提示的话整批插入会直接撞库报 500。
	CheckUsernames(ctx context.Context, tenantID string, usernames []string) (active map[string]struct{}, trash map[string]struct{}, err error)

	// CreateUsersWithRoles 在单个事务内批量创建用户并写入角色关联，
	// 任一步失败整体回滚，不会留下"有用户无角色"的半成品。
	CreateUsersWithRoles(ctx context.Context, users []*model.User, roleIDsByUserID map[string][]string) error
}

// importRepository ImportRepository 的 GORM 实现。
type importRepository struct {
	db *gorm.DB
}

// NewImportRepository 创建批量导入仓储实例。
func NewImportRepository(db *gorm.DB) ImportRepository {
	return &importRepository{db: db}
}

// CheckUsernames 批量查询用户名占用：一条 SQL 查全局现存，一条 SQL 查本租户回收站。
func (r *importRepository) CheckUsernames(ctx context.Context, tenantID string, usernames []string) (map[string]struct{}, map[string]struct{}, error) {
	active := make(map[string]struct{}, len(usernames))
	trash := make(map[string]struct{})
	if len(usernames) == 0 {
		return active, trash, nil
	}

	var activeNames []string
	if err := r.db.WithContext(ctx).Model(&UserPO{}).
		Where("username IN ?", usernames).
		Pluck("username", &activeNames).Error; err != nil {
		return nil, nil, err
	}
	for _, name := range activeNames {
		active[name] = struct{}{}
	}

	var trashNames []string
	if err := r.db.WithContext(ctx).Unscoped().Model(&UserPO{}).
		Where("tenant_id = ? AND username IN ? AND deleted_at IS NOT NULL", tenantID, usernames).
		Pluck("username", &trashNames).Error; err != nil {
		return nil, nil, err
	}
	for _, name := range trashNames {
		trash[name] = struct{}{}
	}
	return active, trash, nil
}

// CreateUsersWithRoles 单事务批量写入用户与角色关联。
func (r *importRepository) CreateUsersWithRoles(ctx context.Context, users []*model.User, roleIDsByUserID map[string][]string) error {
	if len(users) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		records := make([]UserPO, len(users))
		for i, u := range users {
			records[i] = userPOFromModel(u)
		}
		if err := tx.CreateInBatches(&records, 100).Error; err != nil {
			return err
		}

		links := make([]userRoleLink, 0, len(users))
		for _, u := range users {
			for _, roleID := range roleIDsByUserID[u.ID] {
				links = append(links, userRoleLink{UserID: u.ID, RoleID: roleID})
			}
		}
		if len(links) == 0 {
			return nil
		}
		return tx.CreateInBatches(&links, 200).Error
	})
}

// userPOFromModel 把领域模型转为持久化对象（与 userRepository.Create 保持同一映射口径）。
func userPOFromModel(u *model.User) UserPO {
	return UserPO{
		ID:            u.ID,
		TenantID:      u.TenantID,
		Username:      u.Username,
		Password:      u.Password,
		Nickname:      u.Nickname,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Phone:         u.Phone,
		Avatar:        u.Avatar,
		Department:    u.Department,
		Position:      u.Position,
		Remark:        u.Remark,
		Status:        u.Status,
		IsMaster:      u.IsMaster,
		TOTPSecret:    u.TOTPSecret,
		TOTPEnabled:   u.TOTPEnabled,
		LastLoginAt:   u.LastLoginAt,
	}
}

// userRoleLink 批量写 user_roles 的轻量映射：只声明导入需要写入的列，
// 避免 user 模块的仓储层直接依赖 rbac 的持久化类型。
type userRoleLink struct {
	UserID    string    `gorm:"primaryKey;size:26"`
	RoleID    string    `gorm:"primaryKey;size:26"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// TableName 返回用户-角色关联表名。
func (userRoleLink) TableName() string {
	return "user_roles"
}
