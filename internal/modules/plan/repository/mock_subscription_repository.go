package repository

import (
	"context"
	"time"

	"meteorx/internal/modules/plan/model"
)

// MockSubscriptionRepository 订阅仓库的内存实现（用于测试）
type MockSubscriptionRepository struct {
	subs []*model.TenantSubscription
}

func NewMockSubscriptionRepository() *MockSubscriptionRepository {
	return &MockSubscriptionRepository{subs: make([]*model.TenantSubscription, 0)}
}

func (m *MockSubscriptionRepository) GetActiveByTenant(_ context.Context, tenantID string) (*model.TenantSubscription, error) {
	for i := len(m.subs) - 1; i >= 0; i-- {
		if m.subs[i].TenantID == tenantID && m.subs[i].Status == model.SubscriptionActive {
			return m.subs[i], nil
		}
	}
	return nil, nil
}

func (m *MockSubscriptionRepository) Create(_ context.Context, sub *model.TenantSubscription) error {
	m.subs = append(m.subs, sub)
	return nil
}

func (m *MockSubscriptionRepository) UpdateStatus(_ context.Context, id string, status int) error {
	for _, s := range m.subs {
		if s.ID == id {
			s.Status = status
			return nil
		}
	}
	return nil
}

func (m *MockSubscriptionRepository) FindExpiredActive(_ context.Context) ([]*model.TenantSubscription, error) {
	var result []*model.TenantSubscription
	for _, s := range m.subs {
		if s.Status == model.SubscriptionActive && s.ExpiresAt != nil && s.ExpiresAt.Before(nowFunc()) {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *MockSubscriptionRepository) CountByPlan(_ context.Context, planID string) (int64, error) {
	var count int64
	for _, s := range m.subs {
		if s.PlanID == planID && s.Status == model.SubscriptionActive {
			count++
		}
	}
	return count, nil
}

func (m *MockSubscriptionRepository) ListActiveByPlans(_ context.Context, planIDs []string) (map[string]int64, error) {
	result := make(map[string]int64)
	for _, s := range m.subs {
		if s.Status != model.SubscriptionActive {
			continue
		}
		for _, pid := range planIDs {
			if s.PlanID == pid {
				result[pid]++
			}
		}
	}
	return result, nil
}

func (m *MockSubscriptionRepository) ListActiveByTenants(_ context.Context, tenantIDs []string) ([]*model.TenantSubscription, error) {
	var result []*model.TenantSubscription
	idSet := make(map[string]bool, len(tenantIDs))
	for _, id := range tenantIDs {
		idSet[id] = true
	}
	for _, s := range m.subs {
		if s.Status == model.SubscriptionActive && idSet[s.TenantID] {
			result = append(result, s)
		}
	}
	return result, nil
}

// nowFunc 返回当前时间（供测试可替换）
var nowFunc = func() time.Time { return time.Now() }