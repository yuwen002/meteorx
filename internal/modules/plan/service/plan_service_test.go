package service

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/modules/plan/dto"
	"meteorx/internal/modules/plan/model"
	"meteorx/internal/modules/plan/repository"
)

// mockUserRepo 仅供 plan service 测试的最小用户仓库实现
type mockUserRepo struct {
	userCounts map[string]int64
}

func (m *mockUserRepo) CountByTenant(_ context.Context, tenantID string) (int64, error) {
	return m.userCounts[tenantID], nil
}

func (m *mockUserRepo) CountAllUsers(_ context.Context) (int64, error) {
	var total int64
	for _, c := range m.userCounts {
		total += c
	}
	return total, nil
}

func newTestService(planRepo repository.PlanRepository, subRepo repository.SubscriptionRepository, userCounts map[string]int64) *PlanService {
	uRepo := &mockUserRepo{userCounts: userCounts}
	return NewPlanService(planRepo, subRepo, uRepo)
}

func TestCreatePlan(t *testing.T) {
	planRepo := repository.NewMockPlanRepository()
	subRepo := repository.NewMockSubscriptionRepository()
	svc := newTestService(planRepo, subRepo, nil)

	req := dto.CreatePlanReq{
		Name:      "标准版",
		Code:      "standard",
		UserLimit: 100,
		Price:     299,
		Status:    1,
	}
	plan, err := svc.CreatePlan(context.Background(), req)
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}
	if plan.Name != "标准版" {
		t.Errorf("expected name 标准版, got %s", plan.Name)
	}
	if plan.UserLimit != 100 {
		t.Errorf("expected user_limit 100, got %d", plan.UserLimit)
	}

	// 重复创建同 code 应冲突
	if _, err := svc.CreatePlan(context.Background(), req); err != ErrPlanCodeConflict {
		t.Errorf("expected ErrPlanCodeConflict, got %v", err)
	}
}

func TestAssignPlanAndCheckUserLimit(t *testing.T) {
	planRepo := repository.NewMockPlanRepository()
	subRepo := repository.NewMockSubscriptionRepository()
	svc := newTestService(planRepo, subRepo, map[string]int64{"tenant-1": 99})

	// 创建套餐，上限 100
	if _, err := svc.CreatePlan(context.Background(), dto.CreatePlanReq{
		Name: "基础版", Code: "basic", UserLimit: 100, Status: 1,
	}); err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}
	plan, _ := planRepo.GetByCode(context.Background(), "basic")

	// 分配套餐（长期有效）
	if err := svc.AssignPlan(context.Background(), "tenant-1", dto.AssignPlanReq{PlanID: plan.ID}); err != nil {
		t.Fatalf("AssignPlan failed: %v", err)
	}

	// 用户数 99 < 100，未超限
	over, users, limit, err := svc.CheckUserLimit(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("CheckUserLimit failed: %v", err)
	}
	if over {
		t.Errorf("expected not over limit, but over: users=%d limit=%d", users, limit)
	}

	// 用户数来到 100，应超限
	mock := svc.userRepo.(*mockUserRepo)
	mock.userCounts["tenant-1"] = 100
	over, users, limit, err = svc.CheckUserLimit(context.Background(), "tenant-1")
	if err != ErrUserLimitExceeded {
		t.Fatalf("expected ErrUserLimitExceeded, got %v", err)
	}
	if users != 100 || limit != 100 {
		t.Errorf("expected users=100 limit=100, got users=%d limit=%d", users, limit)
	}
}

func TestAssignPlanWithExpiry(t *testing.T) {
	planRepo := repository.NewMockPlanRepository()
	subRepo := repository.NewMockSubscriptionRepository()
	svc := newTestService(planRepo, subRepo, nil)

	if _, err := svc.CreatePlan(context.Background(), dto.CreatePlanReq{
		Name: "专业版", Code: "pro", UserLimit: -1, Status: 1,
	}); err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}
	plan, _ := planRepo.GetByCode(context.Background(), "pro")

	// 过期时间在过去，应报错
	if err := svc.AssignPlan(context.Background(), "tenant-1", dto.AssignPlanReq{
		PlanID: plan.ID, ExpiresAt: "2020-01-01 00:00:00",
	}); err != ErrExpiredAtInPast {
		t.Errorf("expected ErrExpiredAtInPast, got %v", err)
	}

	// 分配带生效到期时间的套餐
	future := time.Now().Add(30 * 24 * time.Hour).Format("2006-01-02 15:04:05")
	if err := svc.AssignPlan(context.Background(), "tenant-1", dto.AssignPlanReq{
		PlanID: plan.ID, ExpiresAt: future,
	}); err != nil {
		t.Fatalf("AssignPlan with future expiry failed: %v", err)
	}

	// 查询当前套餐，剩余天数应>0
	cur, err := svc.GetCurrentPlan(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("GetCurrentPlan failed: %v", err)
	}
	if cur.EffectiveDays < 29 {
		t.Errorf("expected effective_days >= 29, got %d", cur.EffectiveDays)
	}
}

func TestExpireSubscriptions(t *testing.T) {
	planRepo := repository.NewMockPlanRepository()
	subRepo := repository.NewMockSubscriptionRepository()
	svc := newTestService(planRepo, subRepo, nil)

	if _, err := svc.CreatePlan(context.Background(), dto.CreatePlanReq{
		Name: "基础版", Code: "basic", UserLimit: 10, Status: 1,
	}); err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}
	plan, _ := planRepo.GetByCode(context.Background(), "basic")

	// 分配 30 天前到期的套餐（手动插入过期订阅）
	expired := time.Now().Add(-30 * 24 * time.Hour)
	subRepo.Create(context.Background(), &model.TenantSubscription{
		ID: "sub-1", TenantID: "tenant-expired", PlanID: plan.ID,
		Status: model.SubscriptionActive, StartedAt: time.Now().Add(-60 * 24 * time.Hour),
		ExpiresAt: &expired,
	})

	// 分配长期有效的套餐
	if err := svc.AssignPlan(context.Background(), "tenant-active", dto.AssignPlanReq{PlanID: plan.ID}); err != nil {
		t.Fatalf("AssignPlan failed: %v", err)
	}

	expiredIDs, err := svc.ExpireSubscriptions(context.Background())
	if err != nil {
		t.Fatalf("ExpireSubscriptions failed: %v", err)
	}
	if len(expiredIDs) != 1 || expiredIDs[0] != "tenant-expired" {
		t.Errorf("expected expired tenant [tenant-expired], got %v", expiredIDs)
	}
}

func TestDeletePlanInUse(t *testing.T) {
	planRepo := repository.NewMockPlanRepository()
	subRepo := repository.NewMockSubscriptionRepository()
	svc := newTestService(planRepo, subRepo, nil)

	if _, err := svc.CreatePlan(context.Background(), dto.CreatePlanReq{
		Name: "基础版", Code: "basic", UserLimit: 10, Status: 1,
	}); err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}
	plan, _ := planRepo.GetByCode(context.Background(), "basic")

	// 分配套餐使引用
	if err := svc.AssignPlan(context.Background(), "tenant-1", dto.AssignPlanReq{PlanID: plan.ID}); err != nil {
		t.Fatalf("AssignPlan failed: %v", err)
	}

	// 被使用中的套餐不可删除
	if err := svc.DeletePlan(context.Background(), plan.ID); err != ErrPlanInUse {
		t.Errorf("expected ErrPlanInUse, got %v", err)
	}
}
