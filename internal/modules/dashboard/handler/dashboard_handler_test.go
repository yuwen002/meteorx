package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/modules/dashboard/dto"
	"meteorx/internal/modules/dashboard/handler"
	"meteorx/internal/modules/dashboard/model"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDashboardService struct {
	resp    *dto.DashboardOverviewResp
	health  *dto.TenantHealthListResp
	detail  *dto.TenantHealthItemResp
	trend   *dto.GrowthTrendResp
	gotDays int
	err     error
}

func (s *stubDashboardService) GetOverview(_ context.Context) (*dto.DashboardOverviewResp, error) {
	return s.resp, s.err
}

func (s *stubDashboardService) GetTrend(_ context.Context, days int) (*dto.GrowthTrendResp, error) {
	s.gotDays = days
	return s.trend, s.err
}

func (s *stubDashboardService) GetTenantHealth(_ context.Context, _ model.HealthQuery) (*dto.TenantHealthListResp, error) {
	return s.health, s.err
}

func (s *stubDashboardService) GetTenantHealthDetail(_ context.Context, _ string) (*dto.TenantHealthItemResp, error) {
	return s.detail, s.err
}

func newDashboardRouter(stub *stubDashboardService) http.Handler {
	h := handler.NewDashboardHandler(stub)
	r := chi.NewRouter()
	r.Get("/admin/dashboard/overview", h.GetOverview)
	r.Get("/admin/dashboard/trend", h.GetTrend)
	return r
}

func sampleOverview() *dto.DashboardOverviewResp {
	return &dto.DashboardOverviewResp{
		TenantStats: dto.TenantStatsResp{Total: 10, Enabled: 9, Disabled: 1, TodayNew: 2},
		UserStats:   dto.UserStatsResp{Total: 88, TodayNew: 5, WeekNew: 20, MonthNew: 60},
	}
}

func TestGetOverview_Success(t *testing.T) {
	router := newDashboardRouter(&stubDashboardService{resp: sampleOverview()})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/dashboard/overview", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	var out struct {
		Data dto.DashboardOverviewResp `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, int64(10), out.Data.TenantStats.Total)
	assert.Equal(t, int64(88), out.Data.UserStats.Total)
}

func TestGetOverview_ServiceError_Returns500(t *testing.T) {
	router := newDashboardRouter(&stubDashboardService{err: errors.New("stats broken")})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/dashboard/overview", nil))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取运营数据总览失败")
	assert.Contains(t, w.Body.String(), "stats broken")
}

func TestGetTrend_Success_DefaultDays(t *testing.T) {
	stub := &stubDashboardService{trend: &dto.GrowthTrendResp{Days: 30}}
	router := newDashboardRouter(stub)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/dashboard/trend", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 30, stub.gotDays, "未传 days 时应默认 30")
}

func TestGetTrend_Success_WithDays(t *testing.T) {
	stub := &stubDashboardService{trend: &dto.GrowthTrendResp{Days: 7}}
	router := newDashboardRouter(stub)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/dashboard/trend?days=7", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 7, stub.gotDays)
}

func TestGetTrend_InvalidDays_Returns400(t *testing.T) {
	router := newDashboardRouter(&stubDashboardService{})

	for _, q := range []string{"days=0", "days=abc", "days=400", "days=-1"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/dashboard/trend?"+q, nil))
		assert.Equal(t, http.StatusBadRequest, w.Code, q)
	}
}

func TestGetTrend_ServiceError_Returns500(t *testing.T) {
	router := newDashboardRouter(&stubDashboardService{err: errors.New("trend broken")})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/dashboard/trend?days=15", nil))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "trend broken")
}
