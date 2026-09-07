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
	err      error
}

func (m *mockDashboardRepo) GetOverview(_ context.Context) (*model.DashboardOverview, error) {
	return m.overview, m.err
}

func TestGetOverview_MapsAllStats(t *testing.T) {
	ov := &model.DashboardOverview{
		TenantStats: model.TenantStats{Total: 10, Enabled: 9, Disabled: 1, TodayNew: 2},
		UserStats:   model.UserStats{Total: 88, TodayNew: 5, WeekNew: 20, MonthNew: 60},
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
