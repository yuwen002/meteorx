package service

import (
	"context"
	"errors"
	"fmt"
	"meteorx/internal/modules/tenant/dto"
	tenantModel "meteorx/internal/modules/tenant/model"
	"meteorx/internal/modules/tenant/repository"
	userModel "meteorx/internal/modules/user/model"
	userRepo "meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"
	"meteorx/pkg/ulid"
	"time"
)

// 定义业务层特有的错误，方便 Handler 层判断
var (
	ErrTenantNotFound    = errors.New("tenant not found")
	ErrDomainConflict    = errors.New("domain already exists")
	ErrNameConflict      = errors.New("tenant name already exists")
	ErrTenantNotDeleted  = errors.New("tenant not found or not in deleted state")
	ErrUsernameConflict  = errors.New("username already exists in tenant")
)

type TenantService struct {
	repo     repository.TenantRepository
	userRepo userRepo.UserRepository
}

func NewTenantService(repo repository.TenantRepository, userRepo userRepo.UserRepository) *TenantService {
	return &TenantService{repo: repo, userRepo: userRepo}
}

// Register 注册新租户及其管理员用户
// 该方法会生成租户ID和用户ID，加密管理员密码，创建租户和管理员用户模型，并通过事务持久化到数据库
func (s *TenantService) Register(ctx context.Context, req dto.RegisterTenantReq) (*tenantModel.Tenant, error) {
	// 1. 检查租户域名是否已被占用
	existingTenant, err := s.repo.GetByDomain(ctx, req.Domain)
	if err == nil && existingTenant != nil {
		return nil, ErrDomainConflict
	}

	// 2. 检查用户名是否已被占用（全局唯一）
	usernameExists, err := s.userRepo.UsernameExists(ctx, req.AdminUser.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username existence: %w", err)
	}
	if usernameExists {
		return nil, ErrUsernameConflict
	}

	// 3. 生成符合 size:26 限制的唯一 ID (使用 ULID，高并发安全、支持字典序排序)
	tenantID := ulid.Generate()
	userID := ulid.Generate()

	// 4. 密码加密
	hashedPassword, err := crypto.HashPassword(req.AdminUser.Password)
	if err != nil {
		return nil, fmt.Errorf("password encryption failed: %w", err)
	}

	// 5. 构造数据库租户模型 PO (全量对齐你精致的 DTO)
	tenantPO := &tenantModel.Tenant{
		ID:           tenantID,
		Name:         req.Name, // 修正为 req.Name
		Domain:       req.Domain,
		Description:  req.Description,
		ContactEmail: req.ContactEmail,
		Region:       req.Region,
		Logo:         req.Logo,
		Extra:        req.Extra,
		Status:       1, // 默认正常激活
	}

	// 6. 构造数据库用户模型 PO
	userPO := &userModel.User{
		ID:       userID,
		TenantID: tenantID,
		Username: req.AdminUser.Username,
		Password: string(hashedPassword),
		Nickname: req.AdminUser.Nickname,
		Email:    req.AdminUser.Email,
		Role:     "admin", // 租户创始人，锁定 admin 角色
		IsMaster: false,   // 绝不是 MaaS 平台上帝
		Status:   1,       // 默认激活
	}

	// 7. 抛给持久层执行事务
	if err := s.repo.CreateTenantWithAdmin(ctx, tenantPO, userPO); err != nil {
		return nil, err
	}

	return tenantPO, nil
}

// AdminCreate 后台管理员手动创建租户
func (s *TenantService) AdminCreate(ctx context.Context, req dto.AdminCreateTenantReq) (*tenantModel.Tenant, error) {
	// 1. 唯一 ID 生成
	tenantID := ulid.Generate()
	userID := ulid.Generate()

	// 2. 初始管理员密码加密
	hashedPassword, err := crypto.HashPassword(req.AdminUser.Password)
	if err != nil {
		return nil, fmt.Errorf("admin user password encryption failed: %w", err)
	}

	// 3. 组装纯业务租户模型
	tenant := &tenantModel.Tenant{
		ID:           tenantID,
		Name:         req.Name,
		Domain:       req.Domain,
		Description:  req.Description,
		ContactEmail: req.ContactEmail,
		Region:       req.Region,
		Logo:         req.Logo,
		Extra:        req.Extra,
		Status:       req.Status, // 使用后台指定的的状态
	}

	// 4. 组装纯业务用户模型
	user := &userModel.User{
		ID:       userID,
		TenantID: tenantID,
		Username: req.AdminUser.Username,
		Password: string(hashedPassword),
		Nickname: req.AdminUser.Nickname,
		Email:    req.AdminUser.Email,
		Role:     "admin", // 锁定为该租户的主管理员
		IsMaster: false,   // 后台创建的也只是普通租户管理员，绝非 MaaS 平台超级管理员
		Status:   1,       // 默认激活用户状态
	}

	// 5. 交付底层 Repository 开启强一致性事务落库
	// 完美复用你之前写好的 CreateTenantWithAdmin 事务方法
	if err := s.repo.CreateTenantWithAdmin(ctx, tenant, user); err != nil {
		return nil, err
	}

	return tenant, nil
}

// UpdateTenantStatus 更新租户状态
func (s *TenantService) UpdateTenantStatus(ctx context.Context, id string, status int) error {
	// 1. 先查询租户是否存在
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrTenantNotFound
	}

	// 2. 更新状态
	return s.repo.UpdateStatus(ctx, id, status)
}

// QueryTenantList 分页查询租户列表
func (s *TenantService) QueryTenantList(ctx context.Context, page, pageSize int, name string) ([]*tenantModel.Tenant, int64, error) {
	return s.repo.FindPage(ctx, page, pageSize, name)
}

// AdminUpdate 后台更新租户信息
func (s *TenantService) AdminUpdate(ctx context.Context, id string, req dto.AdminUpdateTenantReq) error {
	// 1. 先查询租户是否存在
	existingTenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrTenantNotFound
	}

	// 2. 如果请求中包含了域名，且与当前域名不同，则需要验证唯一性
	if req.Domain != "" && req.Domain != existingTenant.Domain {
		// 根据域名查询，看是否已被其他租户占用
		conflictTenant, err := s.repo.GetByDomain(ctx, req.Domain)
		if err == nil && conflictTenant != nil {
			// 域名已被占用，且占用的租户不是当前租户
			if conflictTenant.ID != id {
				return ErrDomainConflict
			}
		}
		// 将新的域名赋值给模型
		existingTenant.Domain = req.Domain
	}

	// 3. 如果请求中包含了名称，且与当前名称不同，则需要验证唯一性
	if req.Name != "" && req.Name != existingTenant.Name {
		conflictTenant, err := s.repo.GetByName(ctx, req.Name)
		if err == nil && conflictTenant != nil {
			if conflictTenant.ID != id {
				return ErrNameConflict
			}
		}
	}

	// 4. 构造更新模型，只更新非空字段（或根据你的 ORM 策略赋值）
	// 注意：这里采用全量覆盖赋值，如果你的 ORM 使用零值忽略策略，请确保逻辑正确
	tenant := &tenantModel.Tenant{
		Name:         req.Name,
		Domain:       existingTenant.Domain, // 使用经过唯一性校验的域名
		Description:  req.Description,
		ContactEmail: req.ContactEmail,
		Region:       req.Region,
		Logo:         req.Logo,
		Extra:        req.Extra,
	}

	// 4. 更新租户信息
	return s.repo.Update(ctx, id, tenant)
}

// AdminDetail 后台查询租户详情
func (s *TenantService) AdminDetail(ctx context.Context, id string) (*tenantModel.Tenant, error) {
	tenant, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrTenantNotFound
	}
	return tenant, nil
}

// AdminDelete 后台软删除租户
func (s *TenantService) AdminDelete(ctx context.Context, id string) error {
	// 1. 先查询租户是否存在
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrTenantNotFound
	}

	// 2. 软删除租户
	return s.repo.Delete(ctx, id)
}

// BatchUpdateStatus 批量更新租户状态，返回受影响行数、失败的ID列表和错误
func (s *TenantService) BatchUpdateStatus(ctx context.Context, ids []string, status int) (int64, []string, error) {
	return s.repo.BatchUpdateStatus(ctx, ids, status)
}

// BatchDelete 批量软删除租户，返回受影响行数、失败的ID列表和错误
func (s *TenantService) BatchDelete(ctx context.Context, ids []string) (int64, []string, error) {
	return s.repo.BatchDelete(ctx, ids)
}

// FindDeleted 分页查询已软删除的租户列表
func (s *TenantService) FindDeleted(ctx context.Context, page, pageSize int, name string) ([]*tenantModel.Tenant, int64, error) {
	return s.repo.FindDeleted(ctx, page, pageSize, name)
}

// Restore 恢复已软删除的租户
func (s *TenantService) Restore(ctx context.Context, id string) error {
	err := s.repo.Restore(ctx, id)
	if err != nil {
		return ErrTenantNotDeleted
	}
	return nil
}

// GetCurrentTenant 获取当前租户信息
func (s *TenantService) GetCurrentTenant(ctx context.Context, tenantID string) (*tenantModel.Tenant, error) {
	tenant, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, ErrTenantNotFound
	}
	return tenant, nil
}

// UpdateCurrentTenant 更新当前租户信息
func (s *TenantService) UpdateCurrentTenant(ctx context.Context, tenantID string, req dto.UpdateCurrentTenantReq) error {
	// 1. 先查询租户是否存在
	existingTenant, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return ErrTenantNotFound
	}

	// 2. 构造更新模型，只更新非空字段
	tenant := &tenantModel.Tenant{
		Name:         existingTenant.Name,
		Domain:       existingTenant.Domain,
		Description:  existingTenant.Description,
		ContactEmail: existingTenant.ContactEmail,
		Region:       existingTenant.Region,
		Logo:         existingTenant.Logo,
		Extra:        existingTenant.Extra,
	}

	// 3. 应用请求中的非空字段
	if req.Name != "" {
		// 如果名称发生变化，需要校验唯一性
		if req.Name != existingTenant.Name {
			conflictTenant, err := s.repo.GetByName(ctx, req.Name)
			if err == nil && conflictTenant != nil {
				return ErrNameConflict
			}
		}
		tenant.Name = req.Name
	}
	if req.Logo != "" {
		tenant.Logo = req.Logo
	}
	if req.Description != "" {
		tenant.Description = req.Description
	}
	if req.ContactEmail != "" {
		tenant.ContactEmail = req.ContactEmail
	}
	if req.Region != "" {
		tenant.Region = req.Region
	}
	if req.Extra != "" {
		tenant.Extra = req.Extra
	}

	// 4. 更新租户信息
	return s.repo.Update(ctx, tenantID, tenant)
}

// GetInitStatus 获取租户初始化状态
func (s *TenantService) GetInitStatus(ctx context.Context, tenantID string) (*dto.GetInitStatusResp, error) {
	// 1. 先查询租户是否存在
	tenant, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, ErrTenantNotFound
	}

	// 2. 构造初始化状态响应
	// 这里假设租户创建后即为已完成初始化状态
	// 实际业务中可能需要根据租户的初始化流程字段来判断
	resp := &dto.GetInitStatusResp{
		Status:      "completed",
		Message:     "租户已成功初始化",
		Progress:    100,
		Initialized: tenant.Status == tenantModel.StatusEnabled,
	}

	return resp, nil
}

// ApplyCancellation 申请注销租户
func (s *TenantService) ApplyCancellation(ctx context.Context, tenantID string, req dto.ApplyCancellationReq) (*dto.ApplyCancellationResp, error) {
	// 1. 先查询租户是否存在
	_, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, ErrTenantNotFound
	}

	// 2. 这里可以记录注销申请原因到数据库或发送通知
	// 目前简化处理，直接返回申请成功响应
	// 实际业务中可能需要创建注销申请记录，等待管理员审批

	resp := &dto.ApplyCancellationResp{
		AppliedAt:    time.Now().Format("2006-01-02 15:04:05"),
		Status:       "pending",
		EstimatedDay: 7, // 默认7天后注销
	}

	return resp, nil
}
