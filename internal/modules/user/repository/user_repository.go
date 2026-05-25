package repository

import (
	"context"
	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/user/model"
	"time"

	"gorm.io/gorm"
)

// UserPO 内部数据库模型
type UserPO struct {
	ID       string `gorm:"primaryKey;size:26;comment:用户ID"`
	TenantID string `gorm:"index;size:26;not null;comment:租户ID"`
	// 注意复合唯一索引：同一个租户下用户名唯一
	Username  string         `gorm:"size:50;not null;uniqueIndex:idx_tenant_username;comment:用户名"`
	Password  string         `gorm:"size:255;not null;comment:密码"`
	Nickname  string         `gorm:"size:50;comment:昵称"`
	Email     string         `gorm:"size:100;comment:邮箱"`
	Role      string         `gorm:"size:20;default:'user';comment:角色"`
	Status    int            `gorm:"default:1;comment:状态"`
	IsMaster  bool           `gorm:"default:false;comment:是否为主管理员"`
	CreatedAt time.Time      `gorm:"comment:创建时间"`
	UpdatedAt time.Time      `gorm:"comment:更新时间"`
	DeletedAt gorm.DeletedAt `gorm:"index;comment:删除时间"`
}

func (UserPO) TableName() string {
	return "users"
}

// 转换逻辑
func (record UserPO) toDomain() *model.User {
	u := &model.User{
		ID:        record.ID,
		TenantID:  record.TenantID,
		Username:  record.Username,
		Password:  record.Password,
		Nickname:  record.Nickname,
		Email:     record.Email,
		Role:      record.Role,
		Status:    record.Status,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
	if record.DeletedAt.Valid {
		u.DeletedAt = &record.DeletedAt.Time
	}
	return u
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&UserPO{})
}

// 接口实现
type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u *model.User) error {
	record := UserPO{
		ID: u.ID, TenantID: u.TenantID, Username: u.Username,
		Password: u.Password, Nickname: u.Nickname, Role: u.Role,
		Status: u.Status, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}
	return r.db.WithContext(ctx).Create(&record).Error
}

// GetByUsername 根据租户ID和用户名查询用户
// 支持两种登录场景：
// 1. 平台超级管理员登录：tenantID为空时，查询系统租户下的主管理员用户
// 2. 普通租户用户登录：tenantID不为空时，查询指定租户下的用户
// 参数：
//   - ctx: 上下文
//   - tenantID: 租户ID，为空时表示平台超级管理员登录
//   - username: 用户名
//
// 返回：
//   - *model.User: 用户信息
//   - error: 查询错误
func (r *userRepository) GetByUsername(ctx context.Context, tenantID, username string) (*model.User, error) {
	var record UserPO

	query := r.db.WithContext(ctx).Model(&UserPO{})
	if tenantID == "" {
		// 🚀 平台超级管理员登录路径：
		query = query.Where("tenant_id = ? AND username = ? AND is_master = ?", contextx.SystemTenantID, username, true)
	} else {
		// 🚀 普通租户用户登录路径：
		query = query.Where("tenant_id = ? AND username = ?", tenantID, username)
	}

	err := query.First(&record).Error
	if err != nil {
		return nil, err
	}
	return record.toDomain(), nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	var record UserPO
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return record.toDomain(), nil
}
