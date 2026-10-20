package repository

import (
	"context"
	"time"

	"meteorx/internal/modules/plan/model"
)

// MockSubscriptionRepository 订阅仓库的内存实现（用于测试）
type MockSubscriptionRepository struct {
	subs []*model.TenantSubscription
	// updateStatusErr 非 nil 时使 UpdateStatus 失败，用于验证写库出错时的重入语义
	updateStatusErr error
}

// SetUpdateStatusErr 注入 UpdateStatus 失败（仅测试使用）
func (m *MockSubscriptionRepository) SetUpdateStatusErr(err error) {
	m.updateStatusErr = err
}

// StatusOf 读取指定订阅的当前状态（仅测试断言使用，未找到返回 0）
func (m *MockSubscriptionRepository) StatusOf(id string) int {
	for _, s := range m.subs {
		if s.ID == id {
			return s.Status
		}
	}
	return 0
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
	if m.updateStatusErr != nil {
		return m.updateStatusErr
	}
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

// FindExpiringSoon 返回 within 时间内即将到期且仍生效的订阅
func (m *MockSubscriptionRepository) FindExpiringSoon(_ context.Context, within time.Duration) ([]*model.TenantSubscription, error) {
	now := nowFunc()
	limit := now.Add(within)
	var result []*model.TenantSubscription
	for _, s := range m.subs {
		if s.Status == model.SubscriptionActive && s.ExpiresAt != nil &&
			!s.ExpiresAt.Before(now) && s.ExpiresAt.Before(limit) {
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
