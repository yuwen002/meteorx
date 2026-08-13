// Package service 套餐与订阅业务逻辑层
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"meteorx/internal/modules/plan/dto"
	"meteorx/internal/modules/plan/model"
	"meteorx/internal/modules/plan/repository"
	"meteorx/pkg/ulid"
)

// 业务错误定义
var (
	ErrPlanNotFound      = errors.New("plan not found")
	ErrPlanCodeConflict  = errors.New("plan code already exists")
	ErrPlanInUse         = errors.New("plan is in use by tenants")
	ErrSubscriptionEmpty = errors.New("tenant has no active subscription")
	ErrPlanExpired       = errors.New("plan has expired")
	ErrUserLimitExceeded = errors.New("user limit exceeded")
	ErrExpiredAtInPast   = errors.New("expires_at must be in the future")
)

// PlanService 套餐与订阅服务
type PlanService struct {
	planRepo repository.PlanRepository
	subRepo  repository.SubscriptionRepository
	userRepo repository.UserCounter
}

// NewPlanService 创建套餐服务
func NewPlanService(
	planRepo repository.PlanRepository,
	subRepo repository.SubscriptionRepository,
	userRepo repository.UserCounter,
) *PlanService {
	return &PlanService{
		planRepo: planRepo,
		subRepo:  subRepo,
		userRepo: userRepo,
	}
}

// ---- 套餐管理 ----

// CreatePlan 创建套餐
func (s *PlanService) CreatePlan(ctx context.Context, req dto.CreatePlanReq) (*dto.PlanResp, error) {
	// 检查编码唯一性
	existing, err := s.planRepo.GetByCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrPlanCodeConflict
	}
	if req.UserLimit <= 0 {
		// 使用 -1 表示不限制，0 或负数统一转为 -1
		if req.UserLimit != -1 {
			return nil, fmt.Errorf("用户数上限必须大于 0 或为 -1（不限制）")
		}
	}

	plan := &model.Plan{
		ID:          ulid.Generate(),
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		UserLimit:   req.UserLimit,
		Price:       req.Price,
		Status:      req.Status,
	}

	if err := s.planRepo.Create(ctx, plan); err != nil {
		return nil, err
	}
	return s.getPlanResp(ctx, plan.ID)
}

// UpdatePlan 更新套餐
func (s *PlanService) UpdatePlan(ctx context.Context, id string, req dto.UpdatePlanReq) (*dto.PlanResp, error) {
	existing, err := s.planRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrPlanNotFound
	}

	plan := &model.Plan{
		Name:        req.Name,
		Description: req.Description,
		UserLimit:   req.UserLimit,
		Price:       req.Price,
		Status:      req.Status,
	}
	if existing.Name == plan.Name && existing.Description == plan.Description &&
		existing.UserLimit == plan.UserLimit && existing.Price == plan.Price && existing.Status == plan.Status {
		return nil, nil
	}

	if err := s.planRepo.Update(ctx, id, plan); err != nil {
		return nil, err
	}
	return s.getPlanResp(ctx, id)
}

// DeletePlan 删除套餐
func (s *PlanService) DeletePlan(ctx context.Context, id string) error {
	_, err := s.planRepo.GetByID(ctx, id)
	if err != nil {
		return ErrPlanNotFound
	}

	// 检查是否有租户正在使用
	cnt, err := s.subRepo.CountByPlan(ctx, id)
	if err != nil {
		return err
	}
	if cnt > 0 {
		return ErrPlanInUse
	}

	return s.planRepo.Delete(ctx, id)
}

// ListPlans 分页查询套餐
func (s *PlanService) ListPlans(ctx context.Context, page, pageSize int, keyword string, status *int) ([]*dto.PlanResp, int64, error) {
	plans, total, err := s.planRepo.FindPage(ctx, page, pageSize, keyword, status)
	if err != nil {
		return nil, 0, err
	}

	// 批量查询每套餐的租户订阅数
	planIDs := make([]string, len(plans))
	for i, p := range plans {
		planIDs[i] = p.ID
	}
	subMap, _ := s.subRepo.ListActiveByPlans(ctx, planIDs)

	resp := make([]*dto.PlanResp, 0, len(plans))
	for _, p := range plans {
		resp = append(resp, &dto.PlanResp{
			ID:            p.ID,
			Name:          p.Name,
			Code:          p.Code,
			Description:   p.Description,
			UserLimit:     p.UserLimit,
			Price:         p.Price,
			Status:        p.Status,
			SubscriberCnt: subMap[p.ID],
			CreatedAt:     p.CreatedAt,
			UpdatedAt:     p.UpdatedAt,
		})
	}
	return resp, total, nil
}

// ListEnabledPlans 获取所有启用的套餐（下拉选择用，不分页）
func (s *PlanService) ListEnabledPlans(ctx context.Context) ([]*dto.PlanResp, error) {
	plans, err := s.planRepo.ListAllEnabled(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]*dto.PlanResp, 0, len(plans))
	for _, p := range plans {
		resp = append(resp, &dto.PlanResp{
			ID:          p.ID,
			Name:        p.Name,
			Code:        p.Code,
			Description: p.Description,
			UserLimit:   p.UserLimit,
			Price:       p.Price,
			Status:      p.Status,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}
	return resp, nil
}

// GetPlanResp 获取套餐详情响应
func (s *PlanService) getPlanResp(ctx context.Context, id string) (*dto.PlanResp, error) {
	plan, err := s.planRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrPlanNotFound
	}
	cnt, err := s.subRepo.CountByPlan(ctx, id)
	if err != nil {
		cnt = 0
	}
	return &dto.PlanResp{
		ID:            plan.ID,
		Name:          plan.Name,
		Code:          plan.Code,
		Description:   plan.Description,
		UserLimit:     plan.UserLimit,
		Price:         plan.Price,
		Status:        plan.Status,
		SubscriberCnt: cnt,
		CreatedAt:     plan.CreatedAt,
		UpdatedAt:     plan.UpdatedAt,
	}, nil
}

// ---- 订阅管理 ----

// AssignPlan 为租户分配/变更套餐
// 同一租户只保留一条生效订阅，变更时旧订阅标记为取消
func (s *PlanService) AssignPlan(ctx context.Context, tenantID string, req dto.AssignPlanReq) error {
	// 校验套餐存在且启用
	plan, err := s.planRepo.GetByID(ctx, req.PlanID)
	if err != nil {
		return ErrPlanNotFound
	}
	if plan.Status == model.StatusDisabled {
		return fmt.Errorf("套餐已停用，无法分配")
	}

	// 解析到期时间
	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", req.ExpiresAt, time.Local)
		if err != nil {
			return fmt.Errorf("到期时间格式错误")
		}
		if t.Before(time.Now()) {
			return ErrExpiredAtInPast
		}
		expiresAt = &t
	}

	// 取消旧的生效订阅
	oldSub, err := s.subRepo.GetActiveByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	if oldSub != nil {
		if oldSub.PlanID == plan.ID {
			return fmt.Errorf("该租户已使用该套餐，无需重复分配")
		}
		if err := s.subRepo.UpdateStatus(ctx, oldSub.ID, model.SubscriptionCancelled); err != nil {
			return err
		}
	}

	// 创建新订阅
	sub := &model.TenantSubscription{
		ID:        ulid.Generate(),
		TenantID:  tenantID,
		PlanID:    plan.ID,
		Status:    model.SubscriptionActive,
		StartedAt: time.Now(),
		ExpiresAt: expiresAt,
	}
	return s.subRepo.Create(ctx, sub)
}

// ExpireSubscriptions 将已到期订阅标记为过期
// 返回被标记的租户ID列表
func (s *PlanService) ExpireSubscriptions(ctx context.Context) ([]string, error) {
	subs, err := s.subRepo.FindExpiredActive(ctx)
	if err != nil {
		return nil, err
	}
	var expiredTenantIDs []string
	for _, sub := range subs {
		if err := s.subRepo.UpdateStatus(ctx, sub.ID, model.SubscriptionExpired); err != nil {
			continue
		}
		expiredTenantIDs = append(expiredTenantIDs, sub.TenantID)
	}
	return expiredTenantIDs, nil
}

// GetCurrentPlan 获取当前租户套餐与用量
func (s *PlanService) GetCurrentPlan(ctx context.Context, tenantID string) (*dto.CurrentPlanResp, error) {
	sub, err := s.subRepo.GetActiveByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubscriptionEmpty
	}

	plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return nil, ErrPlanNotFound
	}

	currentUsers, err := s.userRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		currentUsers = 0
	}

	resp := &dto.CurrentPlanResp{
		TenantID:     tenantID,
		PlanID:       plan.ID,
		PlanName:     plan.Name,
		PlanCode:     plan.Code,
		UserLimit:    plan.UserLimit,
		CurrentUsers: currentUsers,
		StartedAt:    sub.StartedAt.Format("2006-01-02 15:04:05"),
		Status:       sub.Status,
	}
	if sub.ExpiresAt != nil {
		exp := sub.ExpiresAt.Format("2006-01-02 15:04:05")
		resp.ExpiresAt = &exp
		resp.EffectiveDays = int(sub.ExpiresAt.Sub(time.Now()).Hours() / 24)
		if resp.EffectiveDays < 0 {
			resp.EffectiveDays = 0
		}
	}
	return resp, nil
}

// UpdateSubscriptionExpiry 更新当前生效订阅的到期时间（续期）
func (s *PlanService) UpdateSubscriptionExpiry(ctx context.Context, tenantID string, expiresAtStr string) error {
	if expiresAtStr == "" {
		return fmt.Errorf("到期时间不能为空")
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", expiresAtStr, time.Local)
	if err != nil {
		return fmt.Errorf("到期时间格式错误")
	}
	if t.Before(time.Now()) {
		return ErrExpiredAtInPast
	}

	sub, err := s.subRepo.GetActiveByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrSubscriptionEmpty
	}
	if sub.ExpiresAt != nil {
		now := time.Now()
		if t.Before(*sub.ExpiresAt) && !now.After(*sub.ExpiresAt) {
			return fmt.Errorf("新的到期时间不能早于当前到期时间")
		}
	}
	// 重新创建一条订阅记录，保留历史
	newSub := &model.TenantSubscription{
		ID:        ulid.Generate(),
		TenantID:  tenantID,
		PlanID:    sub.PlanID,
		Status:    model.SubscriptionActive,
		StartedAt: sub.StartedAt,
		ExpiresAt: &t,
	}
	if err := s.subRepo.Create(ctx, newSub); err != nil {
		return err
	}
	return s.subRepo.UpdateStatus(ctx, sub.ID, model.SubscriptionCancelled)
}

// GetTenantPlansBrief 批量查询多个租户的当前套餐摘要（租户列表展示用）
func (s *PlanService) GetTenantPlansBrief(ctx context.Context, tenantIDs []string) (map[string]*dto.TenantPlanBrief, error) {
	result := make(map[string]*dto.TenantPlanBrief)
	if len(tenantIDs) == 0 {
		return result, nil
	}

	subs, err := s.subRepo.ListActiveByTenants(ctx, tenantIDs)
	if err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return result, nil
	}

	// 收集涉及的套餐ID
	planIDs := make([]string, 0, len(subs))
	seen := make(map[string]bool)
	for _, sub := range subs {
		if !seen[sub.PlanID] {
			seen[sub.PlanID] = true
			planIDs = append(planIDs, sub.PlanID)
		}
	}

	// 批量查询套餐名称
	plans := make(map[string]*model.Plan)
	for _, pid := range planIDs {
		p, err := s.planRepo.GetByID(ctx, pid)
		if err == nil && p != nil {
			plans[p.ID] = p
		}
	}

	now := time.Now()
	for _, sub := range subs {
		brief := &dto.TenantPlanBrief{
			TenantID: sub.TenantID,
		}
		if p, ok := plans[sub.PlanID]; ok {
			brief.PlanName = p.Name
		}
		if sub.ExpiresAt != nil && sub.ExpiresAt.Before(now) {
			brief.Expired = true
		}
		result[sub.TenantID] = brief
	}
	return result, nil
}

// CheckUserLimit 实现 QuotaVerifier 接口：校验租户用户数是否达上限
func (s *PlanService) CheckUserLimit(ctx context.Context, tenantID string) (bool, int64, int, error) {
	sub, err := s.subRepo.GetActiveByTenant(ctx, tenantID)
	if err != nil {
		return false, 0, 0, err
	}
	if sub == nil {
		// 无订阅时不限制（避免阻塞现有流程），返回 unlimited
		return false, 0, -1, nil
	}
	// 订阅到期无效
	if sub.ExpiresAt != nil && sub.ExpiresAt.Before(time.Now()) {
		plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
		if err != nil {
			return false, 0, 0, err
		}
		users, err := s.userRepo.CountByTenant(ctx, tenantID)
		if err != nil {
			return false, 0, 0, err
		}
		return true, users, plan.UserLimit, ErrPlanExpired
	}

	plan, err := s.planRepo.GetByID(ctx, sub.PlanID)
	if err != nil {
		return false, 0, 0, err
	}
	// UserLimit 为 -1 或 0 表示不限制
	if plan.UserLimit <= 0 {
		return false, 0, plan.UserLimit, nil
	}

	users, err := s.userRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		return false, 0, 0, err
	}
	if users >= int64(plan.UserLimit) {
		return true, users, plan.UserLimit, ErrUserLimitExceeded
	}
	return false, users, plan.UserLimit, nil
}

var _ repository.QuotaVerifier = (*PlanService)(nil)