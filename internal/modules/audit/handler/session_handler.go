package handler

import (
	"context"
	"net/http"
	"strconv"

	"meteorx/internal/common/response"
	"meteorx/internal/modules/audit/dto"
)

// SessionService 会话分析服务接口（handler 依赖的最小业务面；*service.SessionService 完整实现）
type SessionService interface {
	GetSessionLogs(ctx context.Context, sessionID string) (*dto.SessionAnalysisResp, error)
	ListSessions(ctx context.Context, page, pageSize int, userID string) ([]dto.SessionSummaryResp, int64, error)
}

// SessionHandler 会话分析 HTTP 处理器，提供会话列表与会话详情接口。
type SessionHandler struct {
	sessionSvc SessionService
}

// NewSessionHandler 创建会话分析处理器实例。
func NewSessionHandler(sessionSvc SessionService) *SessionHandler {
	return &SessionHandler{sessionSvc: sessionSvc}
}

// GetSessionLogs 获取指定会话的日志分析详情 GET /api/v1/audit/sessions/{id}
func (h *SessionHandler) GetSessionLogs(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		response.Fail(w, http.StatusBadRequest, "会话ID不能为空")
		return
	}

	result, err := h.sessionSvc.GetSessionLogs(r.Context(), sessionID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取会话日志失败")
		return
	}

	response.Success(w, result)
}

// ListSessions 分页查询会话摘要列表 GET /api/v1/audit/sessions
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	sessions, total, err := h.sessionSvc.ListSessions(
		r.Context(),
		page,
		pageSize,
		r.URL.Query().Get("user_id"),
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取会话列表失败")
		return
	}

	result := map[string]interface{}{
		"items": sessions,
		"total": total,
	}
	response.Success(w, result)
}
