// Package handler 提供审计日志 HTTP 处理器
package handler

import (
	"context"
	"encoding/csv"
	"fmt"
	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/pkg/pagination"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// AuditService 审计服务接口（handler 依赖的最小业务面；*service.AuditService 完整实现）
type AuditService interface {
	CreateLog(ctx context.Context, req dto.CreateAuditLogReq) (*model.AuditLog, error)
	GetLog(ctx context.Context, id string) (*dto.AuditLogResp, error)
	ListLogs(ctx context.Context, query *dto.ListAuditLogsQuery) (*dto.AuditLogListResp, error)
	GetStats(ctx context.Context) (*dto.AuditLogStatsResp, error)
	CleanupLogs(ctx context.Context, days int) (int64, error)
	GetDashboard(ctx context.Context, tenantID string, days int) (*dto.DashboardResp, error)
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

// CleanupLogs 清理过期审计日志（超级管理员）
// DELETE /api/v1/audit/logs/cleanup
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

// ExportLogs 导出审计日志（CSV 格式）
// GET /api/v1/audit/logs/export
func (h *AuditHandler) ExportLogs(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

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

	result, err := h.svc.ListLogs(r.Context(), query)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取审计日志失败")
		return
	}

	switch format {
	case "csv":
		h.exportCSV(w, result.Items)
	default:
		response.Fail(w, http.StatusBadRequest, "不支持的导出格式")
	}
}

// GetDashboard 获取审计仪表盘数据（按天统计）
// GET /api/v1/audit/dashboard
func (h *AuditHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days, _ := strconv.Atoi(daysStr)

	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}

	data, err := h.svc.GetDashboard(r.Context(), tenantID, days)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取仪表盘数据失败")
		return
	}

	response.Success(w, data)
}

// exportCSV 导出为 CSV 格式
func (h *AuditHandler) exportCSV(w http.ResponseWriter, logs []*dto.AuditLogResp) {
	filename := fmt.Sprintf("audit_logs_%s.csv", time.Now().Format("20060102_150405"))

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// 添加 BOM 以支持 Excel 中文显示
	w.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// 写入表头
	headers := []string{"日志ID", "请求ID", "会话ID", "链路ID", "用户ID", "用户名", "租户ID", "模块", "操作", "风险等级", "资源", "资源ID",
		"HTTP方法", "请求路径", "来源页面", "状态码", "结果", "客户端IP", "IP位置", "设备信息", "用户代理", "耗时(ms)", "标签", "错误信息", "操作时间"}
	writer.Write(headers)

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
		writer.Write(record)
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
