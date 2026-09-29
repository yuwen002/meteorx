package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/handler"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSessionService struct {
	Err      error
	Result   *dto.SessionAnalysisResp
	Sessions []dto.SessionSummaryResp
	Total    int64

	GotSessionID string
	GotUserID    string
	GotPage      int
	GotPageSize  int
}

func (s *stubSessionService) GetSessionLogs(_ context.Context, sessionID string) (*dto.SessionAnalysisResp, error) {
	s.GotSessionID = sessionID
	return s.Result, s.Err
}

func (s *stubSessionService) ListSessions(_ context.Context, page, pageSize int, userID string) ([]dto.SessionSummaryResp, int64, error) {
	s.GotPage, s.GotPageSize, s.GotUserID = page, pageSize, userID
	return s.Sessions, s.Total, s.Err
}

func newSessionRouter(stub *stubSessionService) http.Handler {
	h := handler.NewSessionHandler(stub)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sessions/{id}", h.GetSessionLogs)
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

func TestGetSessionLogs_Success_ForwardsID(t *testing.T) {
	stub := &stubSessionService{
		Result: &dto.SessionAnalysisResp{
			SessionID:     "sess-001",
			UserID:        "u-001",
			Username:      "alice",
			TotalRequests: 10,
			SuccessCount:  8,
			FailureCount:  2,
		},
	}
	router := newSessionRouter(stub)

	w := doReq(t, router, http.MethodGet, "/sessions/sess-001", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "sess-001", stub.GotSessionID)
	assert.Contains(t, w.Body.String(), `"sess-001"`)
	assert.Contains(t, w.Body.String(), `"alice"`)
}

func TestGetSessionLogs_ServiceError_Returns500(t *testing.T) {
	stub := &stubSessionService{Err: errors.New("query failed")}
	router := newSessionRouter(stub)

	w := doReq(t, router, http.MethodGet, "/sessions/sess-001", "", []string{"admin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取会话日志失败")
}

func TestListSessions_Success_DefaultsPagination(t *testing.T) {
	stub := &stubSessionService{
		Sessions: []dto.SessionSummaryResp{
			{SessionID: "sess-001", UserID: "u-001", Username: "alice", TotalRequests: 5},
		},
		Total: 1,
	}
	router := newSessionRouter(stub)

	w := doReq(t, router, http.MethodGet, "/sessions", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, stub.GotPage)
	assert.Equal(t, 20, stub.GotPageSize)
	assert.Contains(t, w.Body.String(), `"sess-001"`)
}

func TestListSessions_WithParams_ForwardsPagination(t *testing.T) {
	stub := &stubSessionService{
		Sessions: []dto.SessionSummaryResp{},
		Total:    0,
	}
	router := newSessionRouter(stub)

	w := doReq(t, router, http.MethodGet, "/sessions?page=2&page_size=50&user_id=u-001", "", []string{"admin"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 2, stub.GotPage)
	assert.Equal(t, 50, stub.GotPageSize)
	assert.Equal(t, "u-001", stub.GotUserID)
}

func TestListSessions_ServiceError_Returns500(t *testing.T) {
	stub := &stubSessionService{Err: errors.New("query failed")}
	router := newSessionRouter(stub)

	w := doReq(t, router, http.MethodGet, "/sessions", "", []string{"admin"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "获取会话列表失败")
}