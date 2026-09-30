package repository

import (
	"context"
	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/user/model"
	"time"

	"gorm.io/gorm"
)

// UserPO 内部数据库模型（角色由 user_roles 关联表管理）
type UserPO struct {
	ID            string         `gorm:"primaryKey;size:26;comment:用户ID"`
	TenantID      string         `gorm:"index;size:26;not null;comment:租户ID"`
	Username      string         `gorm:"size:50;not null;uniqueIndex:idx_tenant_username;comment:用户名"`
	Password      string         `gorm:"size:255;not null;comment:密码"`
	Nickname      string         `gorm:"size:50;comment:昵称"`
	Email         string         `gorm:"size:100;comment:邮箱"`
	EmailVerified bool           `gorm:"default:false;comment:邮箱是否已验证"`
	Phone         string         `gorm:"size:20;comment:手机号"`
	Avatar        string         `gorm:"size:500;comment:头像URL"`
	Status        int            `gorm:"default:1;comment:状态"`
	IsMaster      bool           `gorm:"default:false;comment:是否为主管理员"`
	CreatedAt     time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt     gorm.DeletedAt `gorm:"index;comment:删除时间"`
}

// TableName 返回用户表名 users。
func (UserPO) TableName() string {
	return "users"
}

// 转换逻辑（角色从 user_roles 关联查询，不在此处填充）
func (record UserPO) toDomain() *model.User {
	u := &model.User{
		ID:            record.ID,
		TenantID:      record.TenantID,
		Username:      record.Username,
		Password:      record.Password,
		Nickname:      record.Nickname,
		Email:         record.Email,
		EmailVerified: record.EmailVerified,
		Phone:         record.Phone,
		Avatar:        record.Avatar,
		Status:        record.Status,
		IsMaster:      record.IsMaster,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
	if record.DeletedAt.Valid {
		u.DeletedAt = &record.DeletedAt.Time
	}
	return u
}

// AutoMigrate 自动迁移用户表结构。
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&UserPO{})
}

// 接口实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储实例。
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 新建用户记录。
func (r *userRepository) Create(ctx context.Context, u *model.User) error {
	record := UserPO{
		ID:            u.ID,
		TenantID:      u.TenantID,
		Username:      u.Username,
		Password:      u.Password,
		Nickname:      u.Nickname,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Phone:         u.Phone,
		Avatar:        u.Avatar,
		Status:        u.Status,
		IsMaster:      u.IsMaster,
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

// GetByID 按主键查询用户。
func (r *userRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	var record UserPO
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return record.toDomain(), nil
}

// GetByEmail 按邮箱查询用户。
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var record UserPO
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&record).Error
	if err != nil {
		return nil, err
	}
	return record.toDomain(), nil
}

// UsernameExists 全局检查用户名是否已存在（跨所有租户）
func (r *userRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&UserPO{}).Where("username = ?", username).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListByTenant 根据租户ID查询用户列表（支持分页、关键字搜索和状态筛选）
func (r *userRepository) ListByTenant(ctx context.Context, tenantID string, page, pageSize int, keyword string, status *int) ([]*model.User, int64, error) {
	var records []UserPO
	var total int64

	// 构建查询条件
	query := r.db.WithContext(ctx).Model(&UserPO{}).Where("tenant_id = ?", tenantID)

	// 如果有搜索关键字，按用户名、昵称、邮箱模糊搜索
	if keyword != "" {
		query = query.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 如果指定了状态，按状态筛选
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// 查询总数
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	if pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}
	err = query.Find(&records).Error
	if err != nil {
		return nil, 0, err
	}

	var users []*model.User
	for _, record := range records {
		users = append(users, record.toDomain())
	}
	return users, total, nil
}

// Update 更新用户信息（角色由 user_roles 关联表管理，此处不处理）
func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	updates := map[string]interface{}{
		"nickname":       user.Nickname,
		"email":          user.Email,
		"email_verified": user.EmailVerified,
		"phone":          user.Phone,
		"avatar":         user.Avatar,
		"status":         user.Status,
	}
	if user.Password != "" {
		updates["password"] = user.Password
	}
	return r.db.WithContext(ctx).Model(&UserPO{}).Where("id = ?", user.ID).Updates(updates).Error
}

// Delete 删除用户（软删除）
func (r *userRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&UserPO{}, "id = ?", id).Error
}

// ListMasterAdmins 查询所有系统管理员（支持分页和关键词搜索）
func (r *userRepository) ListMasterAdmins(ctx context.Context, page, pageSize int, keyword string) ([]*model.User, int64, error) {
	var records []UserPO
	var total int64

	// 构建基础查询条件
	baseQuery := r.db.WithContext(ctx).Model(&UserPO{}).Where("is_master = ?", true)
	if keyword != "" {
		like := "%" + keyword + "%"
		baseQuery = baseQuery.Where("username LIKE ? OR nickname LIKE ?", like, like)
	}

	// 先查询总数
	err := baseQuery.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	if pageSize > 0 {
		offset := (page - 1) * pageSize
		baseQuery = baseQuery.Offset(offset).Limit(pageSize)
	}
	err = baseQuery.Find(&records).Error
	if err != nil {
		return nil, 0, err
	}

	var users []*model.User
	for _, record := range records {
		users = append(users, record.toDomain())
	}
	return users, total, nil
}

// UpdateStatus 更新用户状态
func (r *userRepository) UpdateStatus(ctx context.Context, id string, status int) error {
	result := r.db.WithContext(ctx).Model(&UserPO{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// FindDeletedMasterAdmins 查询已删除的系统管理员列表
func (r *userRepository) FindDeletedMasterAdmins(ctx context.Context, page, pageSize int, keyword string) ([]*model.User, int64, error) {
	var records []UserPO
	var total int64

	// 构建查询条件：已删除 + 系统管理员
	query := r.db.WithContext(ctx).Unscoped().Model(&UserPO{}).
		Where("is_master = ? AND deleted_at IS NOT NULL", true)

	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("username LIKE ? OR nickname LIKE ?", like, like)
	}

	// 查询总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
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

	var users []*model.User
	for _, record := range records {
		users = append(users, record.toDomain())
	}
	return users, total, nil
}

// RestoreMasterAdmin 恢复已删除的系统管理员
func (r *userRepository) RestoreMasterAdmin(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Unscoped().Model(&UserPO{}).
		Where("id = ? AND is_master = ? AND deleted_at IS NOT NULL", id, true).
		Update("deleted_at", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// PermanentDeleteMasterAdmin 永久删除系统管理员（从数据库中物理删除）
func (r *userRepository) PermanentDeleteMasterAdmin(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Unscoped().
		Where("id = ? AND is_master = ? AND deleted_at IS NOT NULL", id, true).
		Delete(&UserPO{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// BatchUpdateStatus 批量更新系统管理员状态
func (r *userRepository) BatchUpdateStatus(ctx context.Context, ids []string, status int) (int64, error) {
	result := r.db.WithContext(ctx).Model(&UserPO{}).
		Where("id IN ? AND is_master = ?", ids, true).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// BatchDelete 批量删除系统管理员
func (r *userRepository) BatchDelete(ctx context.Context, ids []string) (int64, error) {
	result := r.db.WithContext(ctx).Model(&UserPO{}).
		Where("id IN ? AND is_master = ?", ids, true).
		Delete(&UserPO{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// ListAllTenantUsers 分页查询所有租户普通用户（排除系统管理员，支持关键词搜索）。
func (r *userRepository) ListAllTenantUsers(ctx context.Context, page, pageSize int, keyword string) ([]*model.User, int64, error) {
	var records []UserPO
	var total int64

	// 构建基础查询条件：排除系统管理员
	baseQuery := r.db.WithContext(ctx).Model(&UserPO{}).Where("is_master = ?", false)
	if keyword != "" {
		like := "%" + keyword + "%"
		baseQuery = baseQuery.Where("username LIKE ? OR nickname LIKE ?", like, like)
	}

	// 先查询总数
	err := baseQuery.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	if pageSize > 0 {
		offset := (page - 1) * pageSize
		baseQuery = baseQuery.Offset(offset).Limit(pageSize)
	}
	err = baseQuery.Find(&records).Error
	if err != nil {
		return nil, 0, err
	}

	var users []*model.User
	for _, record := range records {
		users = append(users, record.toDomain())
	}
	return users, total, nil
}

// FindDeletedTenantUsers 查询指定租户的已删除用户列表（回收站）
func (r *userRepository) FindDeletedTenantUsers(ctx context.Context, tenantID string, page, pageSize int, keyword string) ([]*model.User, int64, error) {
	var records []UserPO
	var total int64

	query := r.db.WithContext(ctx).Unscoped().Model(&UserPO{}).
		Where("tenant_id = ? AND is_master = ? AND deleted_at IS NOT NULL", tenantID, false)

	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("username LIKE ? OR nickname LIKE ?", like, like)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

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

	var users []*model.User
	for _, record := range records {
		users = append(users, record.toDomain())
	}
	return users, total, nil
}

// FindAllDeletedTenantUsers 查询所有租户的已删除用户列表（回收站，排除系统管理员）
func (r *userRepository) FindAllDeletedTenantUsers(ctx context.Context, page, pageSize int, keyword string) ([]*model.User, int64, error) {
	var records []UserPO
	var total int64

	query := r.db.WithContext(ctx).Unscoped().Model(&UserPO{}).
		Where("is_master = ? AND deleted_at IS NOT NULL", false)

	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("username LIKE ? OR nickname LIKE ?", like, like)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

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

	var users []*model.User
	for _, record := range records {
		users = append(users, record.toDomain())
	}
	return users, total, nil
}

// RestoreTenantUser 恢复已删除的租户用户
func (r *userRepository) RestoreTenantUser(ctx context.Context, tenantID, userID string) error {
	result := r.db.WithContext(ctx).Unscoped().Model(&UserPO{}).
		Where("id = ? AND tenant_id = ? AND is_master = ? AND deleted_at IS NOT NULL", userID, tenantID, false).
		Update("deleted_at", nil)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// PermanentDeleteTenantUser 永久删除租户用户（物理删除）
func (r *userRepository) PermanentDeleteTenantUser(ctx context.Context, tenantID, userID string) error {
	result := r.db.WithContext(ctx).Unscoped().
		Where("id = ? AND tenant_id = ? AND is_master = ? AND deleted_at IS NOT NULL", userID, tenantID, false).
		Delete(&UserPO{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// BatchDeleteTenantUsers 批量删除租户用户（指定租户，排除系统管理员）
func (r *userRepository) BatchDeleteTenantUsers(ctx context.Context, tenantID string, ids []string) (int64, error) {
	result := r.db.WithContext(ctx).Model(&UserPO{}).
		Where("id IN ? AND tenant_id = ? AND is_master = ?", ids, tenantID, false).
		Delete(&UserPO{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// BatchUpdateTenantUserStatus 批量更新租户用户状态（指定租户，排除系统管理员）
func (r *userRepository) BatchUpdateTenantUserStatus(ctx context.Context, tenantID string, ids []string, status int) (int64, error) {
	result := r.db.WithContext(ctx).Model(&UserPO{}).
		Where("id IN ? AND tenant_id = ? AND is_master = ?", ids, tenantID, false).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// CountByTenant 统计指定租户下的用户总数
func (r *userRepository) CountByTenant(ctx context.Context, tenantID string) (int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&UserPO{})
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	}
	err := query.Count(&total).Error
	return total, err
}

// CountAllUsers 统计所有用户总数（跨租户）
func (r *userRepository) CountAllUsers(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&UserPO{}).Count(&total).Error
	return total, err
}

// UpdateEmailVerified 更新用户邮箱验证状态
func (r *userRepository) UpdateEmailVerified(ctx context.Context, userID string, verified bool) error {
	return r.db.WithContext(ctx).Model(&UserPO{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"email_verified": verified,
			"updated_at":     time.Now(),
		}).Error
}

// GetByPhone 根据手机号查询用户
func (r *userRepository) GetByPhone(ctx context.Context, phone string) (*model.User, error) {
	var record UserPO
	err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&record).Error
	if err != nil {
		return nil, err
	}
	return record.toDomain(), nil
}
