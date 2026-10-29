// Package handler 提供审计日志 HTTP 处理器
package handler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/middleware"
	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/pkg/pagination"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// auditActionExport 审计导出动作名：model 无对应常量，导出既非增删改查，
// 单列一类便于在审计列表中按动作检索这类数据外流行为。
const auditActionExport = "export"

// AuditService 审计服务接口（handler 依赖的最小业务面；*service.AuditService 完整实现）
type AuditService interface {
	CreateLog(ctx context.Context, req dto.CreateAuditLogReq) (*model.AuditLog, error)
	GetLog(ctx context.Context, id string) (*dto.AuditLogResp, error)
	ListLogs(ctx context.Context, query *dto.ListAuditLogsQuery) (*dto.AuditLogListResp, error)
	GetStats(ctx context.Context, tenantID string) (*dto.AuditLogStatsResp, error)
	CleanupLogs(ctx context.Context, days int) (int64, error)
	GetDashboard(ctx context.Context, tenantID string, days int) (*dto.DashboardResp, error)
	GetUserTimeline(ctx context.Context, req *dto.UserTimelineReq) (*dto.UserTimelineResp, error)
	GetDetailedStats(ctx context.Context, tenantID string, days int) (*dto.DetailedStatsResp, error)
	GetAnomalyLogs(ctx context.Context, tenantID string, threshold int, windowMinutes int) ([]*dto.AnomalyLogResp, error)
}

// AuditHandler 审计日志处理器
type AuditHandler struct {
	svc AuditService
}

// NewAuditHandler 创建审计日志处理器
func NewAuditHandler(svc AuditService) *AuditHandler {
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

// ListLogs 分页查询审计日志（支持多条件筛选）
// GET /api/v1/audit/logs
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
		RiskLevel: r.URL.Query().Get("risk_level"),
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

// resolveTenantScope 依据登录上下文推导本次查询的租户范围：
// 上下文带租户 → 强制锁定到该租户（无法通过查询参数越权切换）；
// 上下文无租户（超管 / 上帝账号）→ 允许用 ?tenant_id 指定目标租户，仍为空则全局。
func (h *AuditHandler) resolveTenantScope(r *http.Request) string {
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	return tenantID
}

// GetStats 获取审计日志统计
// GET /api/v1/audit/stats
func (h *AuditHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetStats(r.Context(), h.resolveTenantScope(r))
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取审计日志统计失败")
		return
	}

	response.Success(w, stats)
}

// CleanupLogs 清理过期审计日志（超级管理员）
// DELETE /api/v1/audit/logs/cleanup
//
// 该端点路径命中 shouldSkipAudit 的 /api/v1/audit/logs 前缀，不会被自动审计；
// 而“删除审计历史”本身是最需留痕的高危操作，故在此手动补记，成功/失败均入库。
func (h *AuditHandler) CleanupLogs(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 30 // 默认清理30天前的日志
	}

	start := time.Now()
	reqBody, _ := json.Marshal(map[string]int{"days": days})

	// 只有超级管理员可以清理日志
	if !contextx.HasRole(r.Context(), "superadmin") {
		h.recordSelfOpAudit(r, start, model.ActionTypeDelete, model.RiskCritical,
			string(reqBody), "", http.StatusForbidden, model.ResultFailure, "非超级管理员尝试清理审计日志")
		response.Fail(w, http.StatusForbidden, "只有超级管理员可以清理审计日志")
		return
	}

	affected, err := h.svc.CleanupLogs(r.Context(), days)
	if err != nil {
		h.recordSelfOpAudit(r, start, model.ActionTypeDelete, model.RiskCritical,
			string(reqBody), "", http.StatusInternalServerError, model.ResultFailure, err.Error())
		response.Fail(w, http.StatusInternalServerError, "清理审计日志失败")
		return
	}

	respBody, _ := json.Marshal(map[string]int64{"deleted_count": affected})
	h.recordSelfOpAudit(r, start, model.ActionTypeDelete, model.RiskCritical,
		string(reqBody), string(respBody), http.StatusOK, model.ResultSuccess, "")
	response.Success(w, map[string]int64{"deleted_count": affected})
}

// recordSelfOpAudit 记录“针对审计系统自身”的高危操作（清理/导出）。
// 这些端点被自动审计的跳过规则排除，若不在此手动补记，删除/导出审计数据的行为将毫无痕迹。
// 只记录操作者、请求参数摘要与结果摘要，不落任何审计数据正文。
func (h *AuditHandler) recordSelfOpAudit(r *http.Request, start time.Time, action, risk, requestBody, responseBody string, statusCode int, result, errMsg string) {
	req := middleware.NewAuditLogReq(r)
	req.Module = "audit"
	req.Action = action
	req.Resource = r.URL.Path
	req.RiskLevel = risk
	req.UserID = contextx.GetUserID(r.Context())
	req.Username = req.UserID
	req.TenantID = contextx.GetTenantID(r.Context())
	req.RequestBody = requestBody
	req.ResponseBody = responseBody
	req.StatusCode = statusCode
	req.Result = result
	req.ErrorMessage = errMsg
	req.Duration = time.Since(start).Milliseconds()
	req.Tags = fmt.Sprintf(`["audit",%q,%q]`, action, result)
	middleware.RecordAuditLog(r.Context(), &req)
}

// ExportLogs 导出审计日志（CSV 格式）
// GET /api/v1/audit/logs/export
//
// 批量导出审计数据属于数据外流，同样命中自动审计的跳过前缀，故手动留痕：
// 只记录导出筛选条件与条数，不落 CSV 正文。
func (h *AuditHandler) ExportLogs(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	start := time.Now()
	query := &dto.ListAuditLogsQuery{
		Page:      1,
		PageSize:  10000, // 最多导出10000条
		Module:    r.URL.Query().Get("module"),
		Action:    r.URL.Query().Get("action"),
		Result:    r.URL.Query().Get("result"),
		StartTime: r.URL.Query().Get("start_time"),
		EndTime:   r.URL.Query().Get("end_time"),
		Keyword:   r.URL.Query().Get("keyword"),
	}
	filters, _ := json.Marshal(map[string]string{
		"format":     format,
		"module":     query.Module,
		"action":     query.Action,
		"result":     query.Result,
		"start_time": query.StartTime,
		"end_time":   query.EndTime,
		"keyword":    query.Keyword,
	})

	result, err := h.svc.ListLogs(r.Context(), query)
	if err != nil {
		h.recordSelfOpAudit(r, start, auditActionExport, model.RiskHigh, string(filters), "", http.StatusInternalServerError, model.ResultFailure, err.Error())
		response.Fail(w, http.StatusInternalServerError, "获取审计日志失败")
		return
	}

	switch format {
	case "csv":
		respBody, _ := json.Marshal(map[string]int{"exported_count": len(result.Items)})
		h.recordSelfOpAudit(r, start, auditActionExport, model.RiskHigh, string(filters), string(respBody), http.StatusOK, model.ResultSuccess, "")
		h.exportCSV(w, result.Items)
	default:
		h.recordSelfOpAudit(r, start, auditActionExport, model.RiskHigh, string(filters), "", http.StatusBadRequest, model.ResultFailure, "不支持的导出格式: "+format)
		response.Fail(w, http.StatusBadRequest, "不支持的导出格式")
	}
}

// GetDashboard 获取审计仪表盘数据（按天统计）
// GET /api/v1/audit/dashboard
func (h *AuditHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days, _ := strconv.Atoi(daysStr)

	data, err := h.svc.GetDashboard(r.Context(), h.resolveTenantScope(r), days)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取仪表盘数据失败")
		return
	}

	response.Success(w, data)
}

// GetUserTimeline 获取用户操作时间线
// GET /api/v1/audit/user-timeline?user_id=xxx&page=1&page_size=20
func (h *AuditHandler) GetUserTimeline(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		response.Fail(w, http.StatusBadRequest, "用户ID不能为空")
		return
	}

	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	req := &dto.UserTimelineReq{
		Page:      page,
		PageSize:  pageSize,
		UserID:    userID,
		StartTime: r.URL.Query().Get("start_time"),
		EndTime:   r.URL.Query().Get("end_time"),
		TenantID:  h.resolveTenantScope(r),
	}

	result, err := h.svc.GetUserTimeline(r.Context(), req)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取用户时间线失败")
		return
	}

	response.Success(w, result)
}

// GetDetailedStats 获取详细统计
// GET /api/v1/audit/detailed-stats?days=7
func (h *AuditHandler) GetDetailedStats(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days, _ := strconv.Atoi(daysStr)

	result, err := h.svc.GetDetailedStats(r.Context(), h.resolveTenantScope(r), days)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取详细统计数据失败")
		return
	}

	response.Success(w, result)
}

// GetAnomalyLogs 获取异常日志
// GET /api/v1/audit/anomalies?threshold=5&window_minutes=30
func (h *AuditHandler) GetAnomalyLogs(w http.ResponseWriter, r *http.Request) {
	thresholdStr := r.URL.Query().Get("threshold")
	windowStr := r.URL.Query().Get("window_minutes")
	threshold, _ := strconv.Atoi(thresholdStr)
	windowMinutes, _ := strconv.Atoi(windowStr)

	result, err := h.svc.GetAnomalyLogs(r.Context(), h.resolveTenantScope(r), threshold, windowMinutes)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取异常日志失败")
		return
	}

	response.Success(w, result)
}

// exportCSV 导出为 CSV 格式
func (h *AuditHandler) exportCSV(w http.ResponseWriter, logs []*dto.AuditLogResp) {
	filename := fmt.Sprintf("audit_logs_%s.csv", time.Now().Format("20060102_150405"))

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// 添加 BOM 以支持 Excel 中文显示（导出流已部分写出，失败无法回退，显式忽略）
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// 写入表头
	headers := []string{"日志ID", "请求ID", "会话ID", "链路ID", "用户ID", "用户名", "租户ID", "模块", "操作", "风险等级", "资源", "资源ID",
		"HTTP方法", "请求路径", "来源页面", "状态码", "结果", "客户端IP", "IP位置", "设备信息", "用户代理", "耗时(ms)", "标签", "错误信息", "操作时间"}
	_ = writer.Write(headers)

	// 写入数据
	for _, log := range logs {
		result := "成功"
		if log.Result == "failure" {
			result = "失败"
		}

		riskLevel := getRiskLevelLabel(log.RiskLevel)

		record := []string{
			log.ID,
			log.RequestID,
			log.SessionID,
			log.TraceID,
			log.UserID,
			log.Username,
			log.TenantID,
			log.Module,
			log.Action,
			riskLevel,
			log.Resource,
			log.ResourceID,
			log.Method,
			log.Path,
			log.Referer,
			strconv.Itoa(log.StatusCode),
			result,
			log.ClientIP,
			log.IPLocation,
			log.DeviceInfo,
			log.UserAgent,
			strconv.FormatInt(log.Duration, 10),
			log.Tags,
			log.ErrorMessage,
			log.CreatedAt,
		}
		_ = writer.Write(record)
	}
}

// getRiskLevelLabel 获取风险等级中文标签
func getRiskLevelLabel(level string) string {
	switch level {
	case "low":
		return "低"
	case "medium":
		return "中"
	case "high":
		return "高"
	case "critical":
		return "严重"
	default:
		return level
	}
}
