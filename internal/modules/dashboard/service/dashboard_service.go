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

// GetTenantHealthDetail 获取单个租户健康度详情
func (s *DashboardService) GetTenantHealthDetail(ctx context.Context, tenantID string) (*dto.TenantHealthItemResp, error) {
	item, err := s.repo.GetTenantHealthDetail(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	resp := dto.ToTenantHealthItemResp(*item)
	return &resp, nil
}

// GetTrend 获取最近 days 天的增长趋势时间序列
func (s *DashboardService) GetTrend(ctx context.Context, days int) (*dto.GrowthTrendResp, error) {
	trend, err := s.repo.GetTrend(ctx, days)
	if err != nil {
		return nil, err
	}
	return dto.ToGrowthTrendResp(trend), nil
}

// GetTenantOverview 获取单个租户的总览聚合（成员/任务/Wiki/存储），供租户工作台首屏
func (s *DashboardService) GetTenantOverview(ctx context.Context, tenantID string) (*dto.TenantOverviewResp, error) {
	overview, err := s.repo.GetTenantOverview(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return dto.ToTenantOverviewResp(overview), nil
}
