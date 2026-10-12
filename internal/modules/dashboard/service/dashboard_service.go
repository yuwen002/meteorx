// Package service 提供数据看板业务逻辑
package service

import (
	"context"
	"meteorx/internal/modules/dashboard/dto"
	"meteorx/internal/modules/dashboard/model"
	"meteorx/internal/modules/dashboard/repository"
)

// DashboardService 数据看板业务服务
type DashboardService struct {
	repo repository.DashboardRepository
}

// NewDashboardService 创建数据看板服务
func NewDashboardService(repo repository.DashboardRepository) *DashboardService {
	return &DashboardService{repo: repo}
}

// GetOverview 获取运营数据总览
func (s *DashboardService) GetOverview(ctx context.Context) (*dto.DashboardOverviewResp, error) {
	overview, err := s.repo.GetOverview(ctx)
	if err != nil {
		return nil, err
	}
	return dto.ToDashboardOverviewResp(overview), nil
}

// GetTenantHealth 获取租户健康度列表（分页）
func (s *DashboardService) GetTenantHealth(ctx context.Context, query model.HealthQuery) (*dto.TenantHealthListResp, error) {
	list, err := s.repo.GetTenantHealth(ctx, query)
	if err != nil {
		return nil, err
	}
	return dto.ToTenantHealthListResp(list), nil
}
