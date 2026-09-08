package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/handler"
	"meteorx/internal/modules/audit/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleAlertRule() *model.AlertRule {
	return &model.AlertRule{ID: "ar1", Name: "高风险登录", TriggerType: "risk_level", TriggerValue: "high", NotifyChannels: `["email"]`, NotifyTargets: `["a@x.com"]`, Enabled: true}
}

func sampleAuditAlert() *model.AuditAlert {
	return &model.AuditAlert{ID: "al1", RuleID: "ar1", RuleName: "高风险登录", Message: "检测到高风险登录", Notified: true}
}

func newAlertRouter(stub *stubAlertService) http.Handler {
	h := handler.NewAlertHandler(stub)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /rules", h.CreateRule)
	mux.HandleFunc("PUT /rules/{id}", h.UpdateRule)
	mux.HandleFunc("DELETE /rules/{id}", h.DeleteRule)
	mux.HandleFunc("GET /rules", h.ListRules)
	mux.HandleFunc("GET /alerts", h.ListAlerts)
	mux.HandleFunc("GET /alerts/stats", h.GetAlertStats)
	return mux
}

// ============ 告警规则 ============

func TestCreateRule_Success_ForwardsReq(t *testing.T) {
	stub := &stubAlertService{Rule: sampleAlertRule()}
	router := newAlertRouter(stub)

	body := `{"name":"高风险登录","trigger_type":"risk_level","trigger_value":"high","notify_channels":["email"],"notify_targets":["a@x.com"]}`
	w := doReq(t, router, http.MethodPost, "/rules", body, []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, stub.GotCreateReq)
	assert.Equal(t, "risk_level", stub.GotCreateReq.TriggerType)
	assert.Equal(t, []string{"email"}, stub.GotCreateReq.NotifyChannels)
	assert.Contains(t, w.Body.String(), `"ar1"`)
}

func TestCreateRule_InvalidJSON_Returns400(t *testing.T) {
	stub := &stubAlertService{}
	router := newAlertRouter(stub)

	w := doReq(t, router, http.MethodPost, "/rules", `{"name":`, []string{"admin"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "请求参数错误")
}

func TestCreateRule_ServiceError_Returns500(t *testing.T) {
	stub := &stubAlertService{Err: errors.New("create failed")}
	router := newAlertRouter(stub)

	w := doReq(t, router, http.MethodPost, "/rules", `{"name":"x"}`, []string{"admin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateRule_MissingID_Returns400(t *testing.T) {
	stub := &stubAlertService{}
	h := handler.NewAlertHandler(stub)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/", nil)
	h.UpdateRule(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "规则ID不能为空")
}

func TestUpdateRule_Success_ForwardsIDAndReq(t *testing.T) {
	stub := &stubAlertService{Rule: sampleAlertRule()}
	router := newAlertRouter(stub)

	w := doReq(t, router, http.MethodPut, "/rules/ar1", `{"name":"改名后的规则","enabled":false}`, []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "ar1", stub.GotID)
	require.NotNil(t, stub.GotUpdateReq)
	assert.Equal(t, "改名后的规则", stub.GotUpdateReq.Name)
	assert.False(t, stub.GotUpdateReq.Enabled)
}

func TestUpdateRule_ServiceError_Returns500(t *testing.T) {
	stub := &stubAlertService{Err: errors.New("update failed")}
	router := newAlertRouter(stub)

	w := doReq(t, router, http.MethodPut, "/rules/ar1", `{"name":"x"}`, []string{"admin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteRule_MissingID_Returns400(t *testing.T) {
	stub := &stubAlertService{}
	h := handler.NewAlertHandler(stub)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	h.DeleteRule(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteRule_Success_ForwardsID(t *testing.T) {
	stub := &stubAlertService{}
	router := newAlertRouter(stub)

	w := doReq(t, router, http.MethodDelete, "/rules/ar1", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "ar1", stub.GotID)
}

func TestDeleteRule_ServiceError_Returns500(t *testing.T) {
	stub := &stubAlertService{Err: errors.New("delete failed")}
	router := newAlertRouter(stub)

	w := doReq(t, router, http.MethodDelete, "/rules/ar1", "", []string{"admin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListRules_Success(t *testing.T) {
	stub := &stubAlertService{Rules: []*model.AlertRule{sampleAlertRule()}}
	router := newAlertRouter(stub)

	w := doReq(t, router, http.MethodGet, "/rules", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"ar1"`)
	assert.Contains(t, w.Body.String(), `"高风险登录"`)
}

func TestListRules_ServiceError_Returns500(t *testing.T) {
	stub := &stubAlertService{Err: errors.New("list failed")}
	router := newAlertRouter(stub)

	w := doReq(t, router, http.MethodGet, "/rules", "", []string{"admin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============ 告警记录与统计 ============

func TestListAlerts_Success_ForwardsFilters(t *testing.T) {
	stub := &stubAlertService{Alerts: []*model.AuditAlert{sampleAuditAlert()}, Total: 1}
	router := newAlertRouter(stub)

	w := doReq(t, router, http.MethodGet, "/alerts?page=1&page_size=15&rule_id=ar1&user_id=u1&risk_level=high", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, stub.GotPage)
	assert.Equal(t, 15, stub.GotPageSize)
	assert.Equal(t, "ar1", stub.GotRuleID)
	assert.Equal(t, "u1", stub.GotUserID)
	assert.Equal(t, "high", stub.GotRiskLevel)
	assert.Contains(t, w.Body.String(), `"al1"`)
}

func TestListAlerts_ServiceError_Returns500(t *testing.T) {
	stub := &stubAlertService{Err: errors.New("query failed")}
	router := newAlertRouter(stub)

	w := doReq(t, router, http.MethodGet, "/alerts", "", []string{"admin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetAlertStats_DefaultDays_Is7(t *testing.T) {
	stub := &stubAlertService{Stats: &model.AlertStats{TotalAlerts: 3, TodayAlerts: 1}}
	router := newAlertRouter(stub)

	w := doReq(t, router, http.MethodGet, "/alerts/stats", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 7, stub.GotDays)
	assert.Contains(t, w.Body.String(), `"total_alerts":3`)
}

func TestGetAlertStats_ForwardsDays(t *testing.T) {
	stub := &stubAlertService{Stats: &model.AlertStats{TotalAlerts: 3}}
	router := newAlertRouter(stub)

	w := doReq(t, router, http.MethodGet, "/alerts/stats?days=14", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 14, stub.GotDays)
}

func TestGetAlertStats_ServiceError_Returns500(t *testing.T) {
	stub := &stubAlertService{Err: errors.New("stat failed")}
	router := newAlertRouter(stub)

	w := doReq(t, router, http.MethodGet, "/alerts/stats", "", []string{"admin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ============ 会话分析 ============

func newSessionRouter(stub *stubSessionService) http.Handler {
	h := handler.NewSessionHandler(stub)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sessions/{id}/logs", h.GetSessionLogs)
	mux.HandleFunc("GET /sessions", h.ListSessions)
	return mux
}

func TestGetSessionLogs_MissingID_Returns400(t *testing.T) {
	stub := &stubSessionService{}
	h := handler.NewSessionHandler(stub)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.GetSessionLogs(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "会话ID不能为空")
}

func TestGetSessionLogs_Success_ForwardsSessionID(t *testing.T) {
	stub := &stubSessionService{Analysis: &dto.SessionAnalysisResp{SessionID: "s1", UserID: "u1", TotalOps: 3}}
	router := newSessionRouter(stub)

	w := doReq(t, router, http.MethodGet, "/sessions/s1/logs", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "s1", stub.GotSessionID)
	assert.Contains(t, w.Body.String(), `"session_id":"s1"`)
	assert.Contains(t, w.Body.String(), `"total_ops":3`)
}

func TestGetSessionLogs_ServiceError_Returns500(t *testing.T) {
	stub := &stubSessionService{Err: errors.New("query failed")}
	router := newSessionRouter(stub)

	w := doReq(t, router, http.MethodGet, "/sessions/s1/logs", "", []string{"admin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListSessions_DefaultsPageAndSize(t *testing.T) {
	stub := &stubSessionService{Sessions: []dto.SessionSummaryResp{{SessionID: "s1"}}, Total: 1}
	router := newSessionRouter(stub)

	w := doReq(t, router, http.MethodGet, "/sessions", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, stub.GotPage, "缺省页码应为 1")
	assert.Equal(t, 20, stub.GotPageSize, "缺省页大小应为 20")
	assert.Contains(t, w.Body.String(), `"s1"`)
}

func TestListSessions_ForwardsUserIDFilter(t *testing.T) {
	stub := &stubSessionService{Sessions: []dto.SessionSummaryResp{{SessionID: "s2", UserID: "u1"}}, Total: 1}
	router := newSessionRouter(stub)

	w := doReq(t, router, http.MethodGet, "/sessions?page=3&page_size=10&user_id=u1", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 3, stub.GotPage)
	assert.Equal(t, 10, stub.GotPageSize)
	assert.Equal(t, "u1", stub.GotUserID)
}

func TestListSessions_ServiceError_Returns500(t *testing.T) {
	stub := &stubSessionService{Err: errors.New("list failed")}
	router := newSessionRouter(stub)

	w := doReq(t, router, http.MethodGet, "/sessions", "", []string{"admin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
