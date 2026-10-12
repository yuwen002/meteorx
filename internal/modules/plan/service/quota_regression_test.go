package service

import (
	"context"
	"errors"
	"testing"

	"meteorx/internal/modules/plan/dto"
	"meteorx/internal/modules/plan/repository"
)

// stubStorageUsage 存储用量桩
type stubStorageUsage struct {
	used int64
	err  error
}

func (s *stubStorageUsage) SumSizeByTenant(_ context.Context, _ string) (int64, error) {
	return s.used, s.err
}

// stubWikiUsage Wiki 空间用量桩
type stubWikiUsage struct {
	count int64
	err   error
}

func (s *stubWikiUsage) CountSpacesByTenant(_ context.Context, _ string) (int64, error) {
	return s.count, s.err
}

// newQuotaService 构造带配额套餐的服务：创建套餐并分配给 tenant-1
func newQuotaService(t *testing.T, req dto.CreatePlanReq) (*PlanService, *dto.PlanResp) {
	t.Helper()
	planRepo := repository.NewMockPlanRepository()
	subRepo := repository.NewMockSubscriptionRepository()
	svc := newTestService(planRepo, subRepo, map[string]int64{"tenant-1": 3})

	plan, err := svc.CreatePlan(context.Background(), req)
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}
	if err := svc.AssignPlan(context.Background(), "tenant-1", dto.AssignPlanReq{PlanID: plan.ID}); err != nil {
		t.Fatalf("AssignPlan failed: %v", err)
	}
	return svc, plan
}

// TestCreatePlanQuotaNormalization 未显式设置（0）的配额统一归一化为 -1（不限）
func TestCreatePlanQuotaNormalization(t *testing.T) {
	svc, plan := newQuotaService(t, dto.CreatePlanReq{
		Name: "免费版", Code: "free", UserLimit: 10, Status: 1,
	})
	if plan.StorageLimit != -1 || plan.ApiRateLimit != -1 || plan.WikiSpaceLimit != -1 {
		t.Errorf("expected quotas normalized to -1, got storage=%d api=%d wiki=%d",
			plan.StorageLimit, plan.ApiRateLimit, plan.WikiSpaceLimit)
	}

	// 显式设置的配额应原样保留
	ctx := context.Background()
	if _, err := svc.CreatePlan(ctx, dto.CreatePlanReq{
		Name: "企业版", Code: "enterprise", UserLimit: 500, Status: 1,
		StorageLimit: 1024, ApiRateLimit: 600, WikiSpaceLimit: 20,
	}); err != nil {
		t.Fatalf("CreatePlan enterprise failed: %v", err)
	}
	ent, err := svc.planRepo.GetByCode(ctx, "enterprise")
	if err != nil || ent == nil {
		t.Fatalf("GetByCode enterprise failed: %v", err)
	}
	if ent.StorageLimit != 1024 || ent.ApiRateLimit != 600 || ent.WikiSpaceLimit != 20 {
		t.Errorf("expected explicit quotas kept, got storage=%d api=%d wiki=%d",
			ent.StorageLimit, ent.ApiRateLimit, ent.WikiSpaceLimit)
	}
}

// TestGetTenantQuotaLimits 三类限额查询：有订阅返回套餐值，无订阅返回 -1
func TestGetTenantQuotaLimits(t *testing.T) {
	ctx := context.Background()
	svc, _ := newQuotaService(t, dto.CreatePlanReq{
		Name: "标准版", Code: "standard", UserLimit: 50, Status: 1,
		StorageLimit: 1073741824, ApiRateLimit: 300, WikiSpaceLimit: 5,
	})

	rate, err := svc.GetTenantApiRate(ctx, "tenant-1")
	if err != nil || rate != 300 {
		t.Errorf("expected api rate 300, got %d (err=%v)", rate, err)
	}
	storage, err := svc.GetTenantStorageLimit(ctx, "tenant-1")
	if err != nil || storage != 1073741824 {
		t.Errorf("expected storage 1073741824, got %d (err=%v)", storage, err)
	}
	wiki, err := svc.GetTenantWikiSpaceLimit(ctx, "tenant-1")
	if err != nil || wiki != 5 {
		t.Errorf("expected wiki space limit 5, got %d (err=%v)", wiki, err)
	}

	// 无生效订阅的租户统一返回 -1（不限制）
	for name, got := range map[string]int64{
		"apiRate":   int64(mustInt(svc.GetTenantApiRate(ctx, "tenant-none"))),
		"storage":   mustInt64(svc.GetTenantStorageLimit(ctx, "tenant-none")),
		"wikiSpace": int64(mustInt(svc.GetTenantWikiSpaceLimit(ctx, "tenant-none"))),
	} {
		if got != -1 {
			t.Errorf("%s: expected -1 for tenant without subscription, got %d", name, got)
		}
	}
}

func mustInt(v int, err error) int {
	if err != nil {
		panic(err)
	}
	return v
}

func mustInt64(v int64, err error) int64 {
	if err != nil {
		panic(err)
	}
	return v
}

// TestGetCurrentPlanUsageBackfill GetCurrentPlan 回填存储/Wiki 用量，失败时保持零值
func TestGetCurrentPlanUsageBackfill(t *testing.T) {
	ctx := context.Background()
	svc, _ := newQuotaService(t, dto.CreatePlanReq{
		Name: "专业版", Code: "pro", UserLimit: 20, Status: 1,
		StorageLimit: 1073741824, ApiRateLimit: 60, WikiSpaceLimit: 3,
	})

	// 正常注入：用量回填
	svc.SetUsageProviders(&stubStorageUsage{used: 524288000}, &stubWikiUsage{count: 2})
	cur, err := svc.GetCurrentPlan(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("GetCurrentPlan failed: %v", err)
	}
	if cur.CurrentStorage != 524288000 {
		t.Errorf("expected current_storage 524288000, got %d", cur.CurrentStorage)
	}
	if cur.CurrentWikiSpaces != 2 {
		t.Errorf("expected current_wiki_spaces 2, got %d", cur.CurrentWikiSpaces)
	}
	if cur.StorageLimit != 1073741824 || cur.ApiRateLimit != 60 || cur.WikiSpaceLimit != 3 {
		t.Errorf("quota fields mismatch: storage=%d api=%d wiki=%d",
			cur.StorageLimit, cur.ApiRateLimit, cur.WikiSpaceLimit)
	}

	// 提供者报错：best-effort，保持零值且不影响主流程
	svc.SetUsageProviders(&stubStorageUsage{err: errors.New("db down")}, &stubWikiUsage{err: errors.New("db down")})
	cur, err = svc.GetCurrentPlan(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("GetCurrentPlan with failing providers failed: %v", err)
	}
	if cur.CurrentStorage != 0 || cur.CurrentWikiSpaces != 0 {
		t.Errorf("expected zero usage on provider error, got storage=%d wiki=%d", cur.CurrentStorage, cur.CurrentWikiSpaces)
	}

	// 未注入提供者：保持零值
	fresh := newTestService(repository.NewMockPlanRepository(), repository.NewMockSubscriptionRepository(), nil)
	if _, err := fresh.CreatePlan(ctx, dto.CreatePlanReq{Name: "基础版", Code: "basic", UserLimit: 5, Status: 1}); err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}
	plan, _ := fresh.planRepo.GetByCode(ctx, "basic")
	if err := fresh.AssignPlan(ctx, "tenant-2", dto.AssignPlanReq{PlanID: plan.ID}); err != nil {
		t.Fatalf("AssignPlan failed: %v", err)
	}
	cur, err = fresh.GetCurrentPlan(ctx, "tenant-2")
	if err != nil {
		t.Fatalf("GetCurrentPlan without providers failed: %v", err)
	}
	if cur.CurrentStorage != 0 || cur.CurrentWikiSpaces != 0 {
		t.Errorf("expected zero usage without providers, got storage=%d wiki=%d", cur.CurrentStorage, cur.CurrentWikiSpaces)
	}
}
