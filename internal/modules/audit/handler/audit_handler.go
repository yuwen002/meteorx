package handler

import (
	"net/http"
	"strconv"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/service"
	"meteorx/pkg/pagination"

	"github.com/go-chi/chi/v5"
)

// AuditHandler 审计日志处理器
type AuditHandler struct {
	svc *service.AuditService
}

// NewAuditHandler 创建审计日志处理器
func NewAuditHandler(svc *service.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// CreateLog 创建审计日志（内部API，通常由中间件调用）
// POST /api/v1/audit/logs
func (h *AuditHandler) CreateLog(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAuditLogReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	log, err := h.svc.CreateLog(r.Context(), req)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "创建审计日志失败")
		return
	}

	response.Success(w, dto.ToAuditLogResp(log))
}

// GetLog 获取审计日志详情
// GET /api/v1/audit/logs/{id}
func (h *AuditHandler) GetLog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "日志ID不能为空")
		return
	}

	log, err := h.svc.GetLog(r.Context(), id)
	if err != nil {
		response.Fail(w, http.StatusNotFound, "日志不存在")
		return
	}

	response.Success(w, log)
}

// ListLogs 分页查询审计日志
// GET /api/v1/audit/logs?page=1&page_size=10&user_id=&username=&module=&action=&result=&start_time=&end_time=&keyword=
func (h *AuditHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	pg := pagination.NewPagination(page, pageSize)

	query := &dto.ListAuditLogsQuery{
		Page:      pg.Page,
		PageSize:  pg.PageSize,
		UserID:    r.URL.Query().Get("user_id"),
		Username:  r.URL.Query().Get("username"),
		TenantID:  r.URL.Query().Get("tenant_id"),
		Module:    r.URL.Query().Get("module"),
		Action:    r.URL.Query().Get("action"),
		Resource:  r.URL.Query().Get("resource"),
		Result:    r.URL.Query().Get("result"),
		StartTime: r.URL.Query().Get("start_time"),
		EndTime:   r.URL.Query().Get("end_time"),
		Keyword:   r.URL.Query().Get("keyword"),
	}

	result, err := h.svc.ListLogs(r.Context(), query)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取审计日志列表失败")
		return
	}

	paginatedResult := pagination.NewPaginatedResult(result.Items, pg.Page, pg.PageSize, int(result.Total))
	response.Success(w, paginatedResult)
}

// GetStats 获取审计日志统计
// GET /api/v1/audit/stats
func (h *AuditHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetStats(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取审计日志统计失败")
		return
	}

	response.Success(w, stats)
}

// CleanupLogs 清理审计日志
// DELETE /api/v1/audit/logs/cleanup?days=30
func (h *AuditHandler) CleanupLogs(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 30 // 默认清理30天前的日志
	}

	// 只有超级管理员可以清理日志
	if !contextx.HasRole(r.Context(), "superadmin") {
		response.Fail(w, http.StatusForbidden, "只有超级管理员可以清理审计日志")
		return
	}

	affected, err := h.svc.CleanupLogs(r.Context(), days)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "清理审计日志失败")
		return
	}

	response.Success(w, map[string]int64{"deleted_count": affected})
}