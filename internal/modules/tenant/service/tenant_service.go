package service

import (
	"context"
	"errors"
	"fmt"
	planDto "meteorx/internal/modules/plan/dto"
	planModel "meteorx/internal/modules/plan/model"
	planRepo "meteorx/internal/modules/plan/repository"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	"meteorx/internal/modules/tenant/dto"
	tenantModel "meteorx/internal/modules/tenant/model"
	"meteorx/internal/modules/tenant/repository"
	userModel "meteorx/internal/modules/user/model"
	userRepo "meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"
	"meteorx/pkg/idgen"
	"meteorx/pkg/logger"
	"time"
)

// 定义业务层特有的错误，方便 Handler 层判断
var (
	ErrTenantNotFound   = errors.New("tenant not found")
	ErrDomainConflict   = errors.New("domain already exists")
	ErrNameConflict     = errors.New("tenant name already exists")
	ErrTenantNotDeleted = errors.New("tenant not found or not in deleted state")
	ErrUsernameConflict = errors.New("username already exists in tenant")
)

// TenantPlanProvider 租户套餐摘要查询接口（由 plan 模块实现，避免循环依赖）
type TenantPlanProvider interface {
	// GetTenantPlansBrief 批量查询多个租户的当前套餐摘要
	GetTenantPlansBrief(ctx context.Context, tenantIDs []string) (map[string]*planDto.TenantPlanBrief, error)
}

type TenantService struct {
	repo               repository.TenantRepository
	userRepo           userRepo.UserRepository
	roleRepo           rbacRepo.RoleRepository
	planProvider       TenantPlanProvider
	planAssignProvider TenantPlanAssignProvider
	subRepo            planRepo.SubscriptionRepository
}

func NewTenantService(
	repo repository.TenantRepository,
	userRepo userRepo.UserRepository,
	roleRepo rbacRepo.RoleRepository,
) *TenantService {
	return &TenantService{
		repo:     repo,
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

// SetPlanProvider 注入套餐摘要查询器（由 bootstrap 组装，避免循环依赖）
func (s *TenantService) SetPlanProvider(p TenantPlanProvider) {
	s.planProvider = p
}

// SetSubscriptionRepository 注入订阅仓库（用于注销时取消生效订阅）
func (s *TenantService) SetSubscriptionRepository(sub planRepo.SubscriptionRepository) {
	s.subRepo = sub
}

// TenantPlanAssignProvider 套餐分配接口（由 plan 模块实现，避免循环依赖）
type TenantPlanAssignProvider interface {
	AssignPlan(ctx context.Context, tenantID string, req planDto.AssignPlanReq) error
}

// planAssignProvider 套餐分配器（由 bootstrap 组装注入）
func (s *TenantService) SetPlanAssignProvider(p TenantPlanAssignProvider) {
	s.planAssignProvider = p
}

// GetTenantPlanBriefs 批量查询租户套餐摘要（暴露给 handler 做列表增强）
func (s *TenantService) GetTenantPlanBriefs(ctx context.Context, tenantIDs []string) (map[string]*planDto.TenantPlanBrief, error) {
	if s.planProvider == nil {
		return nil, nil
	}
	return s.planProvider.GetTenantPlansBrief(ctx, tenantIDs)
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

	// 3. 校验默认租户管理员角色是否存在且启用
	defaultRole := "tenant_admin"
	role, err := s.roleRepo.GetByCode(ctx, "", defaultRole)
	if err != nil {
		return nil, fmt.Errorf("默认租户管理员角色未配置，请联系管理员初始化角色: %w", err)
	}
	if role.Status == 0 {
		return nil, fmt.Errorf("默认租户管理员角色已禁用，请联系管理员")
	}

	// 4. 生成符合 size:26 限制的唯一 ID (使用 ULID，高并发安全、支持字典序排序)
	tenantID := idgen.New()
	userID := idgen.New()

	// 5. 密码加密
	hashedPassword, err := crypto.HashPassword(req.AdminUser.Password)
	if err != nil {
		return nil, fmt.Errorf("password encryption failed: %w", err)
	}

	// 6. 构造数据库租户模型 PO (全量对齐你精致的 DTO)
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

	// 7. 构造数据库用户模型 PO (角色信息存放在 user_roles 关联表)
	userPO := &userModel.User{
		ID:       userID,
		TenantID: tenantID,
		Username: req.AdminUser.Username,
		Password: hashedPassword,
		Nickname: req.AdminUser.Nickname,
		Email:    req.AdminUser.Email,
		Roles:    []string{defaultRole},
		IsMaster: false, // 绝不是 MaaS 平台上帝
		Status:   1,     // 默认激活
	}

	// 8. 抛给持久层执行事务：租户、用户与默认角色关联三张表原子落库，
	// 任一环节失败整体回滚，避免遗留"有租户无角色"的脏数据
	if err := s.repo.CreateTenantWithAdmin(ctx, tenantPO, userPO, []string{role.ID}); err != nil {
		return nil, err
	}

	return tenantPO, nil
}

// AdminCreate 后台管理员手动创建租户
func (s *TenantService) AdminCreate(ctx context.Context, req dto.AdminCreateTenantReq) (*tenantModel.Tenant, error) {
	// 1. 唯一 ID 生成
	tenantID := idgen.New()
	userID := idgen.New()

	// 2. 初始管理员密码加密
	hashedPassword, err := crypto.HashPassword(req.AdminUser.Password)
	if err != nil {
		return nil, fmt.Errorf("admin user password encryption failed: %w", err)
	}

	// 3. 校验默认租户管理员角色是否存在且启用
	defaultRole := "tenant_admin"
	role, err := s.roleRepo.GetByCode(ctx, "", defaultRole)
	if err != nil {
		return nil, fmt.Errorf("默认租户管理员角色未配置，请联系管理员初始化角色: %w", err)
	}
	if role.Status == 0 {
		return nil, fmt.Errorf("默认租户管理员角色已禁用，请联系管理员")
	}

	// 4. 组装纯业务租户模型
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

	// 5. 组装纯业务用户模型 (角色信息存放在 user_roles 关联表)
	user := &userModel.User{
		ID:       userID,
		TenantID: tenantID,
		Username: req.AdminUser.Username,
		Password: hashedPassword,
		Nickname: req.AdminUser.Nickname,
		Email:    req.AdminUser.Email,
		Roles:    []string{defaultRole},
		IsMaster: false, // 后台创建的也只是普通租户管理员，绝非 MaaS 平台超级管理员
		Status:   1,     // 默认激活用户状态
	}

	// 6. 交付底层 Repository 开启强一致性事务落库（租户、用户与默认角色关联原子写入）
	if err := s.repo.CreateTenantWithAdmin(ctx, tenant, user, []string{role.ID}); err != nil {
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
func (s *TenantService) QueryTenantList(ctx context.Context, page, pageSize int, name string, status *int) ([]*tenantModel.Tenant, int64, error) {
	return s.repo.FindPage(ctx, page, pageSize, name, status)
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

// AdminHardDelete 后台物理删除租户（彻底销毁，不可恢复）
// 仅应对测试数据或严重违规场景，操作需二次确认
func (s *TenantService) AdminHardDelete(ctx context.Context, id string) error {
	// 1. 先查询租户是否存在
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrTenantNotFound
	}

	// 2. 物理删除租户
	if err := s.repo.HardDelete(ctx, id); err != nil {
		return err
	}

	// 3. 若存在生效订阅，同时取消（失败不再静默吞掉，明确告知调用方需人工处理）
	if s.subRepo != nil {
		activeSub, err := s.subRepo.GetActiveByTenant(ctx, id)
		if err != nil {
			return fmt.Errorf("租户已物理删除，但查询其生效订阅失败: %w", err)
		}
		if activeSub != nil {
			if err := s.subRepo.UpdateStatus(ctx, activeSub.ID, planModel.SubscriptionCancelled); err != nil {
				return fmt.Errorf("租户已物理删除，但取消其生效订阅失败: %w", err)
			}
		}
	}

	return nil
}

// AdminUpdatePlan 后台为租户分配/变更套餐
func (s *TenantService) AdminUpdatePlan(ctx context.Context, id string, req planDto.AssignPlanReq) error {
	// 1. 先查询租户是否存在
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrTenantNotFound
	}

	// 2. 调用套餐分配器
	if s.planAssignProvider == nil {
		return fmt.Errorf("套餐分配器未初始化")
	}
	return s.planAssignProvider.AssignPlan(ctx, id, req)
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
// 创建一条待审批的注销申请记录，等待平台管理员审批
func (s *TenantService) ApplyCancellation(ctx context.Context, tenantID string, req dto.ApplyCancellationReq) (*dto.ApplyCancellationResp, error) {
	// 1. 先查询租户是否存在
	tenant, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, ErrTenantNotFound
	}

	// 2. 检查是否已有未处理的注销申请（待审批或已通过）
	existing, err := s.repo.GetPendingCancelRequestByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("tenant already has a pending cancellation request")
	}

	// 3. 创建注销申请记录
	now := time.Now()
	cancelReq := &tenantModel.CancelRequest{
		ID:         idgen.New(),
		TenantID:   tenant.ID,
		TenantName: tenant.Name,
		Reason:     req.Reason,
		Status:     tenantModel.CancelRequestStatusPending,
		AppliedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo.CreateCancelRequest(ctx, cancelReq); err != nil {
		return nil, err
	}

	resp := &dto.ApplyCancellationResp{
		AppliedAt:    now.Format("2006-01-02 15:04:05"),
		Status:       "pending",
		EstimatedDay: 7, // 默认7天后注销
	}

	return resp, nil
}

// ListCancelRequests 分页查询注销申请（平台管理员）
func (s *TenantService) ListCancelRequests(ctx context.Context, page, pageSize int, status int, keyword string) (*dto.CancelRequestListResp, error) {
	items, total, err := s.repo.FindCancelRequests(ctx, page, pageSize, status, keyword)
	if err != nil {
		return nil, err
	}

	respItems := make([]*dto.CancelRequestResp, len(items))
	for i, c := range items {
		respItems[i] = toCancelRequestResp(c)
	}
	return &dto.CancelRequestListResp{Items: respItems, Total: total}, nil
}

// ApproveCancellation 审批通过注销申请（平台管理员）
// effectiveDays 为生效天数：0 表示立即执行，>0 表示 N 天后执行
func (s *TenantService) ApproveCancellation(ctx context.Context, requestID, approverID string, req dto.AdminApproveCancelReq) (*dto.CancelRequestResp, error) {
	// 1. 查询申请
	cancelReq, err := s.repo.GetCancelRequestByID(ctx, requestID)
	if err != nil {
		return nil, ErrTenantNotFound
	}
	if cancelReq.Status != tenantModel.CancelRequestStatusPending {
		return nil, errors.New("cancellation request is not pending")
	}

	// 2. 更新申请状态
	now := time.Now()
	cancelReq.Status = tenantModel.CancelRequestStatusApproved
	cancelReq.ApproverID = approverID
	cancelReq.ReviewRemark = req.ReviewRemark
	cancelReq.ApprovedAt = &now
	cancelReq.UpdatedAt = now

	// 3. 计算生效时间
	if req.EffectiveDays > 0 {
		effectiveAt := now.AddDate(0, 0, req.EffectiveDays)
		cancelReq.EffectiveAt = &effectiveAt
	} else {
		cancelReq.EffectiveAt = &now
	}

	if err := s.repo.UpdateCancelRequest(ctx, cancelReq); err != nil {
		return nil, err
	}

	// 4. 若立即生效，直接执行注销
	if req.EffectiveDays == 0 {
		if err := s.ExecuteCancellation(ctx, cancelReq); err != nil {
			return nil, err
		}
	}

	return toCancelRequestResp(cancelReq), nil
}

// RejectCancellation 驳回注销申请（平台管理员）
func (s *TenantService) RejectCancellation(ctx context.Context, requestID, approverID string, req dto.AdminRejectCancelReq) (*dto.CancelRequestResp, error) {
	// 1. 查询申请
	cancelReq, err := s.repo.GetCancelRequestByID(ctx, requestID)
	if err != nil {
		return nil, ErrTenantNotFound
	}
	if cancelReq.Status != tenantModel.CancelRequestStatusPending {
		return nil, errors.New("cancellation request is not pending")
	}

	// 2. 更新申请状态为已驳回
	now := time.Now()
	cancelReq.Status = tenantModel.CancelRequestStatusRejected
	cancelReq.ApproverID = approverID
	cancelReq.ReviewRemark = req.ReviewRemark
	cancelReq.ApprovedAt = &now
	cancelReq.UpdatedAt = now

	if err := s.repo.UpdateCancelRequest(ctx, cancelReq); err != nil {
		return nil, err
	}

	return toCancelRequestResp(cancelReq), nil
}

// ExecuteCancellation 执行租户注销
// 1) 软删除租户 2) 取消其生效订阅 3) 标记申请完成
// 订阅查询/取消失败时返回错误且不标记完成：软删除幂等，失败可在下次任务重试，避免订阅状态静默不一致
func (s *TenantService) ExecuteCancellation(ctx context.Context, cancelReq *tenantModel.CancelRequest) error {
	// 1. 软删除租户
	if err := s.repo.Delete(ctx, cancelReq.TenantID); err != nil {
		return err
	}

	// 2. 取消生效订阅（若存在）
	if s.subRepo != nil {
		activeSub, err := s.subRepo.GetActiveByTenant(ctx, cancelReq.TenantID)
		if err != nil {
			return fmt.Errorf("查询租户生效订阅失败: %w", err)
		}
		if activeSub != nil {
			if err := s.subRepo.UpdateStatus(ctx, activeSub.ID, planModel.SubscriptionCancelled); err != nil {
				return fmt.Errorf("取消租户生效订阅失败: %w", err)
			}
		}
	}

	// 3. 标记申请完成
	now := time.Now()
	cancelReq.Status = tenantModel.CancelRequestStatusCompleted
	cancelReq.CompletedAt = &now
	cancelReq.UpdatedAt = now
	return s.repo.UpdateCancelRequest(ctx, cancelReq)
}

// ExecuteDueCancellations 执行所有已到期的注销申请（定时任务调用）
func (s *TenantService) ExecuteDueCancellations(ctx context.Context) (int, error) {
	now := time.Now()
	dueRequests, err := s.repo.FindApprovedDueCancelRequests(ctx, now)
	if err != nil {
		return 0, err
	}

	executed := 0
	for _, r := range dueRequests {
		if err := s.ExecuteCancellation(ctx, r); err != nil {
			// 单个失败不阻断后续，但记录日志便于人工介入（下轮任务自动重试）
			logger.Errorf("执行租户注销失败 tenant=%s request=%s: %v", r.TenantID, r.ID, err)
			continue
		}
		executed++
	}
	return executed, nil
}

// toCancelRequestResp 转换为响应 DTO
func toCancelRequestResp(c *tenantModel.CancelRequest) *dto.CancelRequestResp {
	resp := &dto.CancelRequestResp{
		ID:           c.ID,
		TenantID:     c.TenantID,
		TenantName:   c.TenantName,
		Reason:       c.Reason,
		Status:       c.Status,
		StatusText:   cancelRequestStatusText(c.Status),
		ApproverID:   c.ApproverID,
		ReviewRemark: c.ReviewRemark,
		AppliedAt:    c.AppliedAt.Format("2006-01-02 15:04:05"),
		CreatedAt:    c.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    c.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if c.EffectiveAt != nil {
		resp.EffectiveAt = c.EffectiveAt.Format("2006-01-02 15:04:05")
	}
	if c.ApprovedAt != nil {
		resp.ApprovedAt = c.ApprovedAt.Format("2006-01-02 15:04:05")
	}
	if c.CompletedAt != nil {
		resp.CompletedAt = c.CompletedAt.Format("2006-01-02 15:04:05")
	}
	return resp
}

func cancelRequestStatusText(status int) string {
	switch status {
	case tenantModel.CancelRequestStatusPending:
		return "待审批"
	case tenantModel.CancelRequestStatusApproved:
		return "已通过"
	case tenantModel.CancelRequestStatusRejected:
		return "已驳回"
	case tenantModel.CancelRequestStatusCompleted:
		return "已完成"
	default:
		return "未知"
	}
}
