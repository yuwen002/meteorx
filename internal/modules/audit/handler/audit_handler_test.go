package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/middleware"
	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/handler"
	"meteorx/internal/modules/audit/model"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureAudit 注入审计捕获钩子，返回捕获切片指针（t.Cleanup 自动还原）。
func captureAudit(t *testing.T) *[]dto.CreateAuditLogReq {
	t.Helper()
	var got []dto.CreateAuditLogReq
	restore := middleware.SetAuditCaptureForTest(func(req *dto.CreateAuditLogReq) {
		got = append(got, *req)
	})
	t.Cleanup(restore)
	return &got
}

func newAuditRouter(stub *stubAuditService) http.Handler {
	h := handler.NewAuditHandler(stub)
	r := chi.NewRouter()
	r.Post("/logs", h.CreateLog)
	r.Get("/logs", h.ListLogs)
	r.Get("/logs/{id}", h.GetLog)
	r.Get("/logs/export", h.ExportLogs)
	r.Get("/stats", h.GetStats)
	r.Get("/dashboard", h.GetDashboard)
	r.Delete("/cleanup", h.CleanupLogs)
	return r
}

func TestCreateLog_Success_ReturnsResp(t *testing.T) {
	stub := &stubAuditService{Created: &model.AuditLog{ID: "log-1"}}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodPost, "/logs", `{"module":"auth","action":"login","result":"success"}`, []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"log-1"`)
}

func TestCreateLog_ServiceError_Returns500(t *testing.T) {
	stub := &stubAuditService{Err: errors.New("insert failed")}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodPost, "/logs", `{"module":"auth"}`, []string{"admin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "创建审计日志失败")
}

func TestGetLog_MissingID_Returns400(t *testing.T) {
	stub := &stubAuditService{}
	h := handler.NewAuditHandler(stub)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.GetLog(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "日志ID不能为空")
}

func TestGetLog_Success_ForwardsID(t *testing.T) {
	stub := &stubAuditService{LogResp: &dto.AuditLogResp{ID: "log-1", Module: "auth"}}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodGet, "/logs/log-1", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "log-1", stub.GotID)
	assert.Contains(t, w.Body.String(), `"auth"`)
}

func TestGetLog_NotFound_Returns404(t *testing.T) {
	stub := &stubAuditService{Err: errors.New("record not found")}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodGet, "/logs/log-x", "", []string{"admin"})

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "日志不存在")
}

func TestListLogs_Success_ForwardsFilters(t *testing.T) {
	stub := &stubAuditService{ListResp: &dto.AuditLogListResp{
		Items: []*dto.AuditLogResp{{ID: "l1", Module: "auth"}},
		Total: 1,
	}}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodGet, "/logs?page=2&page_size=20&module=auth&action=login&user_id=u1", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stub.GotQuery)
	assert.Equal(t, "auth", stub.GotQuery.Module)
	assert.Equal(t, "login", stub.GotQuery.Action)
	assert.Equal(t, "u1", stub.GotQuery.UserID)
	assert.Equal(t, 20, stub.GotQuery.PageSize)
	assert.Contains(t, w.Body.String(), `"l1"`)
}

func TestListLogs_ServiceError_Returns500(t *testing.T) {
	stub := &stubAuditService{Err: errors.New("scan failed")}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodGet, "/logs", "", []string{"admin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetStats_Success(t *testing.T) {
	stub := &stubAuditService{Stats: &dto.AuditLogStatsResp{TotalCount: 10, TodayCount: 2}}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodGet, "/stats", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"total_count":10`)
}

func TestGetStats_ServiceError_Returns500(t *testing.T) {
	stub := &stubAuditService{Err: errors.New("count failed")}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodGet, "/stats", "", []string{"admin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCleanupLogs_NonSuperadmin_Returns403(t *testing.T) {
	stub := &stubAuditService{}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodDelete, "/cleanup?days=30", "", []string{"admin"})

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "超级管理员")
	assert.Empty(t, stub.GotDays, "非超管不应触达清理逻辑")
}

func TestCleanupLogs_Superadmin_Success_ForwardsDays(t *testing.T) {
	stub := &stubAuditService{Affected: 88}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodDelete, "/cleanup?days=60", "", []string{"superadmin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 60, stub.GotDays)
	assert.Contains(t, w.Body.String(), `"deleted_count":88`)
}

func TestCleanupLogs_Superadmin_DefaultDays30(t *testing.T) {
	stub := &stubAuditService{Affected: 1}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodDelete, "/cleanup", "", []string{"superadmin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 30, stub.GotDays)
}

func TestCleanupLogs_Superadmin_ServiceError_Returns500(t *testing.T) {
	stub := &stubAuditService{Err: errors.New("cleanup failed")}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodDelete, "/cleanup?days=30", "", []string{"superadmin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestExportLogs_CSV_Success(t *testing.T) {
	stub := &stubAuditService{ListResp: &dto.AuditLogListResp{
		Items: []*dto.AuditLogResp{{ID: "l1", Module: "auth", Action: "login", Result: "failure", RiskLevel: "high", CreatedAt: "2026-09-07 10:00:00"}},
		Total: 1,
	}}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodGet, "/logs/export?format=csv", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	assert.Contains(t, w.Header().Get("Content-Disposition"), "attachment")
	body := w.Body.String()
	assert.Contains(t, body, "日志ID")
	assert.Contains(t, body, "失败", "failure 应导出为中文“失败”")
	assert.Contains(t, body, "高", "high 风险等级应导出为“高”")
}

func TestExportLogs_UnsupportedFormat_Returns400(t *testing.T) {
	stub := &stubAuditService{ListResp: &dto.AuditLogListResp{Items: []*dto.AuditLogResp{{ID: "l1"}}}}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodGet, "/logs/export?format=xls", "", []string{"admin"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "不支持的导出格式")
}

func TestGetDashboard_Success_ForwardsTenantAndDays(t *testing.T) {
	stub := &stubAuditService{Dashboard: &dto.DashboardResp{TotalCount: 5, Trend: []dto.TrendPoint{{Date: "2026-09-06", Count: 5}}}}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodGet, "/dashboard?days=5", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "tenant-1", stub.GotTenant, "优先使用登录态租户")
	assert.Equal(t, 5, stub.GotDays)
	assert.Contains(t, w.Body.String(), `"total_count":5`)
}

func TestGetDashboard_UsesQueryTenantWhenNoLoginCtx(t *testing.T) {
	stub := &stubAuditService{Dashboard: &dto.DashboardResp{TotalCount: 1}}
	router := newAuditRouter(stub)

	// 不带身份上下文但带 tenant_id 查询参数（内部调用场景）
	req := httptest.NewRequest(http.MethodGet, "/dashboard?tenant_id=t-query&days=3", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "t-query", stub.GotTenant)
}

func TestGetDashboard_ServiceError_Returns500(t *testing.T) {
	stub := &stubAuditService{Err: errors.New("stat failed")}
	router := newAuditRouter(stub)

	w := doReq(t, router, http.MethodGet, "/dashboard", "", []string{"admin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------- 审计系统自身高危操作留痕 ----------

func TestCleanupLogs_Superadmin_RecordsCriticalAudit(t *testing.T) {
	stub := &stubAuditService{Affected: 88}
	router := newAuditRouter(stub)
	got := captureAudit(t)

	doReq(t, router, http.MethodDelete, "/cleanup?days=60", "", []string{"superadmin"})

	require.Len(t, *got, 1)
	rec := (*got)[0]
	assert.Equal(t, "audit", rec.Module)
	assert.Equal(t, model.ActionTypeDelete, rec.Action)
	assert.Equal(t, model.RiskCritical, rec.RiskLevel)
	assert.Equal(t, model.ResultSuccess, rec.Result)
	assert.Equal(t, http.StatusOK, rec.StatusCode)
	assert.Equal(t, "user-1", rec.UserID)
	assert.Equal(t, "tenant-1", rec.TenantID)
	assert.Contains(t, rec.RequestBody, `"days":60`)
	assert.Contains(t, rec.ResponseBody, `"deleted_count":88`)
}

func TestCleanupLogs_NonSuperadmin_RecordsFailureAudit(t *testing.T) {
	stub := &stubAuditService{}
	router := newAuditRouter(stub)
	got := captureAudit(t)

	doReq(t, router, http.MethodDelete, "/cleanup?days=30", "", []string{"admin"})

	require.Len(t, *got, 1)
	rec := (*got)[0]
	assert.Equal(t, model.ActionTypeDelete, rec.Action)
	assert.Equal(t, model.RiskCritical, rec.RiskLevel)
	assert.Equal(t, model.ResultFailure, rec.Result)
	assert.Equal(t, http.StatusForbidden, rec.StatusCode)
}

func TestCleanupLogs_ServiceError_RecordsFailureAudit(t *testing.T) {
	stub := &stubAuditService{Err: errors.New("cleanup failed")}
	router := newAuditRouter(stub)
	got := captureAudit(t)

	doReq(t, router, http.MethodDelete, "/cleanup?days=30", "", []string{"superadmin"})

	require.Len(t, *got, 1)
	rec := (*got)[0]
	assert.Equal(t, model.ResultFailure, rec.Result)
	assert.Equal(t, http.StatusInternalServerError, rec.StatusCode)
	assert.Equal(t, "cleanup failed", rec.ErrorMessage)
}

func TestExportLogs_CSV_RecordsHighRiskAudit(t *testing.T) {
	stub := &stubAuditService{ListResp: &dto.AuditLogListResp{
		Items: []*dto.AuditLogResp{{ID: "l1"}, {ID: "l2"}},
		Total: 2,
	}}
	router := newAuditRouter(stub)
	got := captureAudit(t)

	doReq(t, router, http.MethodGet, "/logs/export?format=csv&module=auth", "", []string{"admin"})

	require.Len(t, *got, 1)
	rec := (*got)[0]
	assert.Equal(t, "audit", rec.Module)
	assert.Equal(t, "export", rec.Action)
	assert.Equal(t, model.RiskHigh, rec.RiskLevel)
	assert.Equal(t, model.ResultSuccess, rec.Result)
	assert.Contains(t, rec.RequestBody, `"module":"auth"`)
	assert.Contains(t, rec.ResponseBody, `"exported_count":2`)
}

func TestExportLogs_UnsupportedFormat_RecordsFailureAudit(t *testing.T) {
	stub := &stubAuditService{ListResp: &dto.AuditLogListResp{Items: []*dto.AuditLogResp{{ID: "l1"}}}}
	router := newAuditRouter(stub)
	got := captureAudit(t)

	doReq(t, router, http.MethodGet, "/logs/export?format=xls", "", []string{"admin"})

	require.Len(t, *got, 1)
	rec := (*got)[0]
	assert.Equal(t, "export", rec.Action)
	assert.Equal(t, model.ResultFailure, rec.Result)
	assert.Equal(t, http.StatusBadRequest, rec.StatusCode)
}
