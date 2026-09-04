package repository

import (
	"context"
	"errors"

	"meteorx/internal/modules/plan/model"
)

// ErrNotFound 模拟仓储中记录不存在
var ErrNotFound = errors.New("record not found")

// MockPlanRepository 套餐仓库的内存实现（用于测试）
type MockPlanRepository struct {
	plans []*model.Plan
}

func NewMockPlanRepository() *MockPlanRepository {
	return &MockPlanRepository{plans: make([]*model.Plan, 0)}
}

func (m *MockPlanRepository) Create(_ context.Context, plan *model.Plan) error {
	m.plans = append(m.plans, plan)
	return nil
}

func (m *MockPlanRepository) GetByID(_ context.Context, id string) (*model.Plan, error) {
	for _, p := range m.plans {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MockPlanRepository) GetByCode(_ context.Context, code string) (*model.Plan, error) {
	for _, p := range m.plans {
		if p.Code == code {
			return p, nil
		}
	}
	return nil, nil
}

func (m *MockPlanRepository) Update(_ context.Context, id string, plan *model.Plan) error {
	for i, p := range m.plans {
		if p.ID == id {
			if plan.Name != "" {
				m.plans[i].Name = plan.Name
			}
			if plan.Description != "" {
				m.plans[i].Description = plan.Description
			}
			if plan.UserLimit >= -1 {
				m.plans[i].UserLimit = plan.UserLimit
			}
			if plan.Price >= 0 {
				m.plans[i].Price = plan.Price
			}
			if plan.Status >= 0 {
				m.plans[i].Status = plan.Status
			}
			return nil
		}
	}
	return ErrNotFound
}

func (m *MockPlanRepository) Delete(_ context.Context, id string) error {
	for i, p := range m.plans {
		if p.ID == id {
			m.plans = append(m.plans[:i], m.plans[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (m *MockPlanRepository) FindPage(_ context.Context, _page, _pageSize int, keyword string, status *int) ([]*model.Plan, int64, error) {
	var result []*model.Plan
	for _, p := range m.plans {
		if keyword != "" && !containsStr(p.Name, keyword) && !containsStr(p.Code, keyword) {
			continue
		}
		if status != nil && p.Status != *status {
			continue
		}
		result = append(result, p)
	}
	return result, int64(len(result)), nil
}

func (m *MockPlanRepository) ListAllEnabled(_ context.Context) ([]*model.Plan, error) {
	var result []*model.Plan
	for _, p := range m.plans {
		if p.Status == model.StatusEnabled {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *MockPlanRepository) CountByID(_ context.Context, ids []string) (int64, error) {
	var count int64
	for _, id := range ids {
		for _, p := range m.plans {
			if p.ID == id {
				count++
				break
			}
		}
	}
	return count, nil
}

func containsStr(s, sub string) bool {
	if sub == "" {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
