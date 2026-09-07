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

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDashboardService struct {
	resp *dto.DashboardOverviewResp
	err  error
}

func (s *stubDashboardService) GetOverview(_ context.Context) (*dto.DashboardOverviewResp, error) {
	return s.resp, s.err
}

func newDashboardRouter(stub *stubDashboardService) http.Handler {
	h := handler.NewDashboardHandler(stub)
	r := chi.NewRouter()
	r.Get("/admin/dashboard/overview", h.GetOverview)
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
