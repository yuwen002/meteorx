package service_test

import (
	"context"
	"errors"
	"testing"

	"meteorx/internal/modules/dashboard/model"
	"meteorx/internal/modules/dashboard/service"

	"github.com/stretchr/testify/assert"
)

// mockDashboardRepo 看板仓库内存实现
type mockDashboardRepo struct {
	overview *model.DashboardOverview
	health   *model.TenantHealthList
	detail   *model.TenantHealthItem
	trend    *model.GrowthTrend
	tenantOv *model.TenantOverview
	gotTID   string
	err      error
}

func (m *mockDashboardRepo) GetOverview(_ context.Context) (*model.DashboardOverview, error) {
	return m.overview, m.err
}

func (m *mockDashboardRepo) GetTrend(_ context.Context, _ int) (*model.GrowthTrend, error) {
	return m.trend, m.err
}

func (m *mockDashboardRepo) GetTenantHealth(_ context.Context, _ model.HealthQuery) (*model.TenantHealthList, error) {
	return m.health, m.err
}

func (m *mockDashboardRepo) GetTenantHealthDetail(_ context.Context, _ string) (*model.TenantHealthItem, error) {
	return m.detail, m.err
}

func (m *mockDashboardRepo) GetTenantOverview(_ context.Context, tenantID string) (*model.TenantOverview, error) {
	m.gotTID = tenantID
	return m.tenantOv, m.err
}

func TestGetOverview_MapsAllStats(t *testing.T) {
	ov := &model.DashboardOverview{
		TenantStats:       model.TenantStats{Total: 10, Enabled: 9, Disabled: 1, TodayNew: 2},
		UserStats:         model.UserStats{Total: 88, TodayNew: 5, WeekNew: 20, MonthNew: 60},
		SubscriptionStats: model.SubscriptionStats{Total: 7, Active: 3, Expired: 2, Cancelled: 2, ActiveTenant: 3},
		AuditStats: model.AuditStats{
			Total: 99, Today: 4, Success: 90, Failure: 9,
			ActionStats: map[string]int64{"create": 40},
			ModuleStats: map[string]int64{"wiki": 55},
		},
	}
	svc := service.NewDashboardService(&mockDashboardRepo{overview: ov})

	resp, err := svc.GetOverview(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, int64(10), resp.TenantStats.Total)
	assert.Equal(t, int64(9), resp.TenantStats.Enabled)
	assert.Equal(t, int64(88), resp.UserStats.Total)
	assert.Equal(t, int64(3), resp.SubscriptionStats.Active)
	assert.Equal(t, int64(99), resp.AuditStats.Total)
	assert.Equal(t, map[string]int64{"create": 40}, resp.AuditStats.ActionStats)
	assert.Equal(t, map[string]int64{"wiki": 55}, resp.AuditStats.ModuleStats)
}

func TestGetOverview_PropagatesRepositoryError(t *testing.T) {
	svc := service.NewDashboardService(&mockDashboardRepo{err: errors.New("stats db down")})

	resp, err := svc.GetOverview(context.Background())

	assert.Nil(t, resp)
	assert.EqualError(t, err, "stats db down")
}

func TestGetOverview_EmptyStatsHandled(t *testing.T) {
	svc := service.NewDashboardService(&mockDashboardRepo{overview: &model.DashboardOverview{}})

	resp, err := svc.GetOverview(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	// 统计 map 为零值时应按“无数据”安全透传而非 panic/误造
	assert.Nil(t, resp.AuditStats.ActionStats)
	assert.Equal(t, int64(0), resp.TenantStats.Total)
}

func TestGetTrend_MapsDaysAndList(t *testing.T) {
	trend := &model.GrowthTrend{
		Days: 2,
		List: []model.TrendDay{
			{Date: "2026-10-13", NewTenants: 1, NewUsers: 3, AuditLogs: 20},
			{Date: "2026-10-14", NewTenants: 2, NewUsers: 5, AuditLogs: 40},
		},
	}
	svc := service.NewDashboardService(&mockDashboardRepo{trend: trend})

	resp, err := svc.GetTrend(context.Background(), 2)

	assert.NoError(t, err)
	assert.Equal(t, 2, resp.Days)
	assert.Len(t, resp.List, 2)
	assert.Equal(t, "2026-10-14", resp.List[1].Date)
	assert.Equal(t, int64(5), resp.List[1].NewUsers)
}

func TestGetTrend_PropagatesRepositoryError(t *testing.T) {
	svc := service.NewDashboardService(&mockDashboardRepo{err: errors.New("trend db down")})

	resp, err := svc.GetTrend(context.Background(), 30)

	assert.Nil(t, resp)
	assert.EqualError(t, err, "trend db down")
}

func TestGetTenantOverview_MapsAndPassesTenantID(t *testing.T) {
	repo := &mockDashboardRepo{tenantOv: &model.TenantOverview{
		Member:      model.TenantMemberOverview{Total: 5, Active: 3},
		Task:        model.TenantTaskOverview{Pending: 2, InProgress: 1, Completed: 4, Overdue: 1, Total: 7, CompletionRate: 0.5714},
		Wiki:        model.TenantWikiOverview{Spaces: 2, Documents: 9, Views: 100},
		StorageUsed: 4096,
	}}
	svc := service.NewDashboardService(repo)

	resp, err := svc.GetTenantOverview(context.Background(), "tA")

	assert.NoError(t, err)
	assert.Equal(t, "tA", repo.gotTID)
	assert.Equal(t, int64(5), resp.Member.Total)
	assert.Equal(t, int64(3), resp.Member.Active)
	assert.Equal(t, int64(1), resp.Task.Overdue)
	assert.InDelta(t, 0.5714, resp.Task.CompletionRate, 0.0001)
	assert.Equal(t, int64(9), resp.Wiki.Documents)
	assert.Equal(t, int64(4096), resp.StorageUsed)
}

func TestGetTenantOverview_PropagatesRepositoryError(t *testing.T) {
	svc := service.NewDashboardService(&mockDashboardRepo{err: errors.New("overview db down")})

	resp, err := svc.GetTenantOverview(context.Background(), "tA")

	assert.Nil(t, resp)
	assert.EqualError(t, err, "overview db down")
}
