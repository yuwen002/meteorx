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
// 用于映射数据库表结构，可选字段使用 *string 指针以区分 NULL 与空字符串
type TenantPO struct {
	ID           string         `gorm:"primaryKey;size:26;comment:'租户唯一标识'"`        // 租户唯一标识符
	Name         *string        `gorm:"size:100;comment:'租户名称'"`                      // 租户显示名称
	Domain       string         `gorm:"size:255;uniqueIndex;comment:'租户域名，全局唯一'"` // 租户访问域名，全局唯一
	Status       int            `gorm:"comment:'状态：1-启用 0-禁用'"`                     // 租户状态：1-启用，0-禁用
	Description  *string        `gorm:"size:255;comment:'租户简述'"`                     // 租户简述
	ContactEmail *string        `gorm:"size:100;comment:'联系邮箱'"`                     // 联系邮箱
	Region       *string        `gorm:"size:50;comment:'地区/数据中心'"`                  // 地区/数据中心
	Logo         *string        `gorm:"size:500;comment:'租户Logo URL'"`               // 租户 Logo URL
	Extra        *string        `gorm:"type:text;comment:'扩展字段(JSON格式)'"`           // 扩展字段，存储 JSON 格式
	CreatedAt    time.Time      `gorm:"comment:'创建时间'"`                               // 记录创建时间
	UpdatedAt    time.Time      `gorm:"comment:'更新时间'"`                               // 记录最后更新时间
	DeletedAt    gorm.DeletedAt `gorm:"index;comment:'软删除时间'"`                       // 软删除时间戳，支持 GORM 软删除
}

// TableName 指定 GORM 使用的表名
// 返回数据库中对应的表名
func (TenantPO) TableName() string {
	return "tenants"
}

// strPtr 将字符串转换为指针，空字符串返回 nil
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// strVal 安全地取出字符串指针的值，nil 返回空字符串
func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// toDomain 将 GORM 模型转换为纯净的业务模型
// 隔离数据库层与业务层，避免业务模型被数据库框架污染
// 返回值：*model.Tenant - 纯业务模型，不包含数据库相关字段
func (record TenantPO) toDomain() *model.Tenant {
	tenant := &model.Tenant{
		ID:           record.ID,
		Name:         strVal(record.Name),
		Domain:       record.Domain,
		Status:       record.Status,
		Description:  strVal(record.Description),
		ContactEmail: strVal(record.ContactEmail),
		Region:       strVal(record.Region),
		Logo:         strVal(record.Logo),
		Extra:        strVal(record.Extra),
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
		Name:         strPtr(t.Name),
		Domain:       t.Domain,
		Status:       t.Status,
		Description:  strPtr(t.Description),
		ContactEmail: strPtr(t.ContactEmail),
		Region:       strPtr(t.Region),
		Logo:         strPtr(t.Logo),
		Extra:        strPtr(t.Extra),
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
	if t.DeletedAt != nil {
		record.DeletedAt = gorm.DeletedAt{Time: *t.DeletedAt, Valid: true}
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return err
	}

	// 将 GORM 自动生成的 CreatedAt/UpdatedAt 回写到领域模型
	t.CreatedAt = record.CreatedAt
	t.UpdatedAt = record.UpdatedAt

	return nil
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
// CreateTenantWithAdmin 在创建租户的同时创建管理员用户
// 这是一个跨模块的数据库操作，使用事务确保数据一致性
// 参数:
//   - ctx: 上下文信息，用于传递请求范围的数据、取消信号和截止日期
//   - t: 租户模型对象，包含租户的基本信息
//   - u: 用户模型对象，将作为该租户的管理员
//
// 返回值:
//   - error: 操作过程中发生的错误，如果成功则为nil
func (r *tenantRepository) CreateTenantWithAdmin(ctx context.Context, t *model.Tenant, u *userModel.User) error {
	// 1. 组装本模块的 PO (Persistent Object)
	// 将领域模型转换为持久化对象，以便与数据库交互
	tenantPO := &TenantPO{
		ID:           t.ID,                    // 租户ID
		Name:         strPtr(t.Name),          // 租户名称
		Domain:       t.Domain,                // 租户域名
		Status:       t.Status,                // 租户状态
		Description:  strPtr(t.Description),   // 租户描述
		ContactEmail: strPtr(t.ContactEmail),  // 联系邮箱
		Region:       strPtr(t.Region),        // 租户所在区域
		Logo:         strPtr(t.Logo),          // 租户Logo
		Extra:        strPtr(t.Extra),         // 额外信息，以JSON格式存储
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
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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

	if err != nil {
		return err
	}

	// 4. 将 GORM 自动生成的 CreatedAt/UpdatedAt 回写到领域模型
	// GORM 在 Create 后会自动将数据库生成的时间戳填充到 PO 对象中
	t.CreatedAt = tenantPO.CreatedAt
	t.UpdatedAt = tenantPO.UpdatedAt

	return nil
}

// UpdateStatus 修改租户状态
// UpdateStatus 更新租户状态的方法
// 参数:
//   - ctx: 上下文信息，用于控制请求的超时、取消等
//   - id: 要更新的租户ID
//   - status: 新的状态值
//
// 返回值:
//   - error: 操作过程中可能出现的错误
func (r *tenantRepository) UpdateStatus(ctx context.Context, id string, status int) error {
	// 使用GORM的WithContext方法设置上下文
	// 使用Model方法指定操作的数据模型为TenantPO
	// 使用Where方法添加条件，筛选ID匹配的记录
	// 使用Update方法更新status字段
	// 返回操作过程中可能出现的错误
	return r.db.WithContext(ctx).Model(&TenantPO{}).Where("id = ?", id).Update("status", status).Error
}

// Update 更新租户信息
func (r *tenantRepository) Update(ctx context.Context, id string, t *model.Tenant) error {
	updates := map[string]interface{}{}
	if t.Name != "" {
		updates["name"] = t.Name
	}
	if t.Description != "" {
		updates["description"] = t.Description
	}
	if t.ContactEmail != "" {
		updates["contact_email"] = t.ContactEmail
	}
	if t.Region != "" {
		updates["region"] = t.Region
	}
	if t.Logo != "" {
		updates["logo"] = t.Logo
	}
	if t.Extra != "" {
		updates["extra"] = t.Extra
	}

	if len(updates) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Model(&TenantPO{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 软删除租户
func (r *tenantRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&TenantPO{}, "id = ?", id).Error
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
		tenants = append(tenants, po.toDomain())
	}

	return tenants, total, nil
}

// findMissingIDs 在给定 ID 列表中找出数据库中不存在的 ID
func (r *tenantRepository) findMissingIDs(ctx context.Context, ids []string) ([]string, error) {
	var existingIDs []string
	if err := r.db.WithContext(ctx).Model(&TenantPO{}).Where("id IN ?", ids).Pluck("id", &existingIDs).Error; err != nil {
		return nil, err
	}
	// 用 map 快速查找，差集即为不存在的 ID
	existingMap := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existingMap[id] = struct{}{}
	}
	var missing []string
	for _, id := range ids {
		if _, found := existingMap[id]; !found {
			missing = append(missing, id)
		}
	}
	return missing, nil
}

// BatchUpdateStatus 批量更新租户状态
// 先检查哪些 ID 不存在，将其作为 failedIDs 返回，只对存在的记录执行更新
func (r *tenantRepository) BatchUpdateStatus(ctx context.Context, ids []string, status int) (int64, []string, error) {
	missingIDs, err := r.findMissingIDs(ctx, ids)
	if err != nil {
		return 0, nil, err
	}
	// 计算实际可操作的 ID（排除不存在的）
	validIDs := make([]string, 0, len(ids)-len(missingIDs))
	missingSet := make(map[string]struct{}, len(missingIDs))
	for _, id := range missingIDs {
		missingSet[id] = struct{}{}
	}
	for _, id := range ids {
		if _, skip := missingSet[id]; !skip {
			validIDs = append(validIDs, id)
		}
	}
	if len(validIDs) == 0 {
		return 0, missingIDs, nil
	}
	result := r.db.WithContext(ctx).Model(&TenantPO{}).Where("id IN ?", validIDs).Update("status", status)
	return result.RowsAffected, missingIDs, result.Error
}

// BatchDelete 批量软删除租户
// 先检查哪些 ID 不存在，将其作为 failedIDs 返回，只对存在的记录执行软删除
func (r *tenantRepository) BatchDelete(ctx context.Context, ids []string) (int64, []string, error) {
	missingIDs, err := r.findMissingIDs(ctx, ids)
	if err != nil {
		return 0, nil, err
	}
	// 计算实际可操作的 ID（排除不存在的）
	validIDs := make([]string, 0, len(ids)-len(missingIDs))
	missingSet := make(map[string]struct{}, len(missingIDs))
	for _, id := range missingIDs {
		missingSet[id] = struct{}{}
	}
	for _, id := range ids {
		if _, skip := missingSet[id]; !skip {
			validIDs = append(validIDs, id)
		}
	}
	if len(validIDs) == 0 {
		return 0, missingIDs, nil
	}
	result := r.db.WithContext(ctx).Where("id IN ?", validIDs).Delete(&TenantPO{})
	return result.RowsAffected, missingIDs, result.Error
}

// FindDeleted 分页查询已软删除的租户列表
// 使用 GORM 的 Unscoped 绕过软删除过滤，手动筛选 deleted_at IS NOT NULL 的记录
func (r *tenantRepository) FindDeleted(ctx context.Context, page, pageSize int, name string) ([]*model.Tenant, int64, error) {
	var pos []*TenantPO
	var total int64

	query := r.db.WithContext(ctx).Unscoped().Model(&TenantPO{}).Where("deleted_at IS NOT NULL")
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("deleted_at DESC").Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	// 将 PO 数组转换为 Model 数组
	var tenants []*model.Tenant
	for _, po := range pos {
		t := po.toDomain()
		tenants = append(tenants, t)
	}

	return tenants, total, nil
}

// GetByName 根据租户名称查询租户信息
// 用于更新租户时校验名称唯一性
func (r *tenantRepository) GetByName(ctx context.Context, name string) (*model.Tenant, error) {
	var record TenantPO
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return record.toDomain(), nil
}

// Restore 恢复已软删除的租户
// 使用 Unscoped 找到已软删除的记录，并将 deleted_at 置为 NULL
func (r *tenantRepository) Restore(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Unscoped().Model(&TenantPO{}).
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Update("deleted_at", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("tenant not found or not deleted")
	}
	return nil
}
