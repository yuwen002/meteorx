// Package repository 提供租户数据访问层的实现
// 基于 GORM 框架操作数据库，实现 TenantRepository 接口
// 包含数据库模型定义、迁移方法和基础的 CRUD 操作
package repository

import (
	"context"
	"errors"
	userModel "meteorx/internal/modules/user/model"
	userRepo "meteorx/internal/modules/user/repository"
	"time"

	"meteorx/internal/modules/tenant/model"

	"gorm.io/gorm"
)

// 确保 tenantRepository 完整实现了 TenantRepository 接口
// 如果接口方法未实现，编译时会报错
var _ TenantRepository = (*tenantRepository)(nil)

// TenantPO 仅限仓库内部使用的 GORM 模型
// 用于映射数据库表结构，包含所有持久化字段
type TenantPO struct {
	ID           string         `gorm:"primaryKey;size:26;comment:'租户唯一标识'"`      // 租户唯一标识符
	Name         string         `gorm:"size:100;comment:'租户名称'"`                  // 租户显示名称
	Domain       string         `gorm:"size:255;uniqueIndex;comment:'租户域名，全局唯一'"` // 租户访问域名，全局唯一
	Status       int            `gorm:"comment:'状态：1-启用 0-禁用'"`                   // 租户状态：1-启用，0-禁用
	Description  string         `gorm:"size:255;comment:'租户简述'"`                  // 租户简述
	ContactEmail string         `gorm:"size:100;comment:'联系邮箱'"`                  // 联系邮箱
	Region       string         `gorm:"size:50;comment:'地区/数据中心'"`                // 地区/数据中心
	Logo         string         `gorm:"size:500;comment:'租户Logo URL'"`            // 租户 Logo URL
	Extra        string         `gorm:"type:text;comment:'扩展字段(JSON格式)'"`         // 扩展字段，存储 JSON 格式
	CreatedAt    time.Time      `gorm:"comment:'创建时间'"`                           // 记录创建时间
	UpdatedAt    time.Time      `gorm:"comment:'更新时间'"`                           // 记录最后更新时间
	DeletedAt    gorm.DeletedAt `gorm:"index;comment:'软删除时间'"`                    // 软删除时间戳，支持 GORM 软删除
}

// TableName 指定 GORM 使用的表名
// 返回数据库中对应的表名
func (TenantPO) TableName() string {
	return "tenants"
}

// toDomain 将 GORM 模型转换为纯净的业务模型
// 隔离数据库层与业务层，避免业务模型被数据库框架污染
// 返回值：*model.Tenant - 纯业务模型，不包含数据库相关字段
func (record TenantPO) toDomain() *model.Tenant {
	tenant := &model.Tenant{
		ID:           record.ID,
		Name:         record.Name,
		Domain:       record.Domain,
		Status:       record.Status,
		Description:  record.Description,
		ContactEmail: record.ContactEmail,
		Region:       record.Region,
		Logo:         record.Logo,
		Extra:        record.Extra,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
	if record.DeletedAt.Valid {
		tenant.DeletedAt = &record.DeletedAt.Time
	}
	return tenant
}

// AutoMigrate 在 repository 包内部执行 GORM 模型迁移
// 自动创建或更新数据库表结构，确保与模型定义一致
// 参数：db - GORM 数据库连接实例
// 返回值：error - 迁移过程中的错误
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&TenantPO{})
}

// tenantRepository 租户仓储实现
// 封装数据库操作，提供租户数据的持久化功能
type tenantRepository struct {
	db *gorm.DB // GORM 数据库连接实例
}

// NewTenantRepository 创建基于 GORM 的 TenantRepository 实现
// 工厂函数，初始化租户仓储实例
// 参数：db - GORM 数据库连接实例
// 返回值：TenantRepository - 租户仓储接口实例
func NewTenantRepository(db *gorm.DB) TenantRepository {
	return &tenantRepository{db: db}
}

// GetByID 根据租户 ID 查询租户信息
// 参数：
//
//	ctx - 上下文，用于传递请求截止时间和取消信号
//	id - 租户唯一标识符
//
// 返回值：
//
//	*model.Tenant - 找到的租户信息
//	error - 查询过程中的错误，如记录不存在或数据库错误
func (r *tenantRepository) GetByID(ctx context.Context, id string) (*model.Tenant, error) {
	var record TenantPO
	if err := r.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return record.toDomain(), nil
}

// Create 创建新的租户记录
// 将业务模型转换为数据库模型并持久化到数据库
// 参数：
//
//	ctx - 上下文，用于传递请求截止时间和取消信号
//	t - 待创建的租户业务模型
//
// 返回值：
//
//	error - 创建过程中的错误，如数据验证失败或数据库错误
func (r *tenantRepository) Create(ctx context.Context, t *model.Tenant) error {
	record := TenantPO{
		ID:           t.ID,
		Name:         t.Name,
		Domain:       t.Domain,
		Status:       t.Status,
		Description:  t.Description,
		ContactEmail: t.ContactEmail,
		Region:       t.Region,
		Logo:         t.Logo,
		Extra:        t.Extra,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
	if t.DeletedAt != nil {
		record.DeletedAt = gorm.DeletedAt{Time: *t.DeletedAt, Valid: true}
	}
	return r.db.WithContext(ctx).Create(&record).Error
}

// GetByDomain 根据域名查询租户信息
// 用于多租户系统中根据访问域名识别租户
// 参数：
//
//	ctx - 上下文，用于传递请求截止时间和取消信号
//	domain - 租户域名
//
// 返回值：
//
//	*model.Tenant - 找到的租户信息
//	error - 查询过程中的错误，如记录不存在或数据库错误
func (r *tenantRepository) GetByDomain(ctx context.Context, domain string) (*model.Tenant, error) {
	var record TenantPO
	err := r.db.WithContext(ctx).Where("domain = ?", domain).First(&record).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 没找到不是错误，返回空对象
		}
		return nil, err // 真正的数据库错误（如断网）
	}
	return record.toDomain(), nil
}

// CreateTenantWithAdmin 在物理库层开启 GORM 事务，两张表同时落库
func (r *tenantRepository) CreateTenantWithAdmin(ctx context.Context, t *model.Tenant, u *userModel.User) error {
	// 1. 组装本模块的 PO
	tenantPO := &TenantPO{
		ID:           t.ID,
		Name:         t.Name,
		Domain:       t.Domain,
		Status:       t.Status,
		Description:  t.Description,
		ContactEmail: t.ContactEmail,
		Region:       t.Region,
		Logo:         t.Logo,
		Extra:        t.Extra,
	}

	// 2. 跨包直接组装邻居的 PO (名字带 PO 后缀，优雅易懂)
	userPO := &userRepo.UserPO{
		ID:       u.ID,
		TenantID: t.ID, // 强绑定租户ID
		Username: u.Username,
		Password: u.Password,
		Nickname: u.Nickname,
		Email:    u.Email,
		Role:     u.Role,
		Status:   u.Status,
		IsMaster: u.IsMaster,
	}

	// 3. 执行本地事务闭包
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 写入租户表
		if err := tx.Create(tenantPO).Error; err != nil {
			return err
		}

		// 写入用户表
		if err := tx.Create(userPO).Error; err != nil {
			return err
		}

		return nil // 自动提交
	})

}

// UpdateStatus 修改租户状态
func (r *tenantRepository) UpdateStatus(ctx context.Context, id string, status int) error {
	return r.db.WithContext(ctx).Model(&TenantPO{}).Where("id = ?", id).Update("status", status).Error
}

// FindPage 分页查询租户列表
func (r *tenantRepository) FindPage(ctx context.Context, page, pageSize int, name string) ([]*model.Tenant, int64, error) {
	var pos []*TenantPO
	var total int64

	query := r.db.WithContext(ctx).Model(&TenantPO{})
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	// 将 PO 数组转换为对外干净的 Model 数组
	var tenants []*model.Tenant
	for _, po := range pos {
		tenants = append(tenants, &model.Tenant{
			ID:           po.ID,
			Name:         po.Name,
			Domain:       po.Domain,
			Status:       po.Status,
			ContactEmail: po.ContactEmail,
			CreatedAt:    po.CreatedAt,
		})
	}

	return tenants, total, nil
}
