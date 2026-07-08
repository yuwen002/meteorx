package dto

import (
	"meteorx/internal/modules/audit/model"
	"time"
)

// CreateAuditLogReq 创建审计日志请求（内部使用，不对外暴露为API）
type CreateAuditLogReq struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	TenantID     string `json:"tenant_id"`
	Module       string `json:"module"`
	Action       string `json:"action"`
	Resource     string `json:"resource"`
	ResourceID   string `json:"resource_id"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	RequestBody  string `json:"request_body"`
	ResponseBody string `json:"response_body"`
	StatusCode   int    `json:"status_code"`
	Result       string `json:"result"`
	ErrorMessage string `json:"error_message"`
	ClientIP     string `json:"client_ip"`
	UserAgent    string `json:"user_agent"`
	Duration     int64  `json:"duration"`
}

// AuditLogResp 审计日志响应
type AuditLogResp struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	TenantID     string `json:"tenant_id"`
	Module       string `json:"module"`
	Action       string `json:"action"`
	Resource     string `json:"resource"`
	ResourceID   string `json:"resource_id"`
	Method       string `json:"method"`
	Path         string `json:"path"`
	RequestBody  string `json:"request_body,omitempty"`
	ResponseBody string `json:"response_body,omitempty"`
	StatusCode   int    `json:"status_code"`
	Result       string `json:"result"`
	ErrorMessage string `json:"error_message,omitempty"`
	ClientIP     string `json:"client_ip"`
	UserAgent    string `json:"user_agent"`
	Duration     int64  `json:"duration"`
	CreatedAt    string `json:"created_at"`
}

// ToAuditLogResp 将 model.AuditLog 转为 AuditLogResp
func ToAuditLogResp(log *model.AuditLog) *AuditLogResp {
	return &AuditLogResp{
		ID:           log.ID,
		UserID:       log.UserID,
		Username:     log.Username,
		TenantID:     log.TenantID,
		Module:       log.Module,
		Action:       log.Action,
		Resource:     log.Resource,
		ResourceID:   log.ResourceID,
		Method:       log.Method,
		Path:         log.Path,
		RequestBody:  log.RequestBody,
		ResponseBody: log.ResponseBody,
		StatusCode:   log.StatusCode,
		Result:       log.Result,
		ErrorMessage: log.ErrorMessage,
		ClientIP:     log.ClientIP,
		UserAgent:    log.UserAgent,
		Duration:     log.Duration,
		CreatedAt:    log.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ListAuditLogsQuery 审计日志列表查询参数
type ListAuditLogsQuery struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	UserID   string `json:"user_id" form:"user_id"`
	Username string `json:"username" form:"username"`
	TenantID string `json:"tenant_id" form:"tenant_id"`
	Module   string `json:"module" form:"module"`
	Action   string `json:"action" form:"action"`
	Resource string `json:"resource" form:"resource"`
	Result   string `json:"result" form:"result"`
	StartTime string `json:"start_time" form:"start_time"`
	EndTime   string `json:"end_time" form:"end_time"`
	Keyword   string `json:"keyword" form:"keyword"`
}

// AuditLogListResp 审计日志列表响应
type AuditLogListResp struct {
	Items []*AuditLogResp `json:"items"`
	Total int64           `json:"total"`
}

// AuditLogStatsResp 审计日志统计响应
type AuditLogStatsResp struct {
	TotalCount  int64            `json:"total_count"`  // 总日志数
	TodayCount  int64            `json:"today_count"`  // 今日日志数
	ActionStats map[string]int64 `json:"action_stats"` // 按操作类型统计
	ModuleStats map[string]int64 `json:"module_stats"` // 按模块统计
	ResultStats map[string]int64 `json:"result_stats"` // 按结果统计
}