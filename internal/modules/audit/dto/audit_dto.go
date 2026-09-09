package dto

import (
	"meteorx/internal/modules/audit/model"
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
	IPLocation   string `json:"ip_location"`
	UserAgent    string `json:"user_agent"`
	DeviceInfo   string `json:"device_info"`
	Duration     int64  `json:"duration"`
	SessionID    string `json:"session_id"`
	RequestID    string `json:"request_id"`
	TraceID      string `json:"trace_id"`
	Referer      string `json:"referer"`
	RiskLevel    string `json:"risk_level"`
	Tags         string `json:"tags"`
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
	IPLocation   string `json:"ip_location,omitempty"`
	UserAgent    string `json:"user_agent"`
	DeviceInfo   string `json:"device_info,omitempty"`
	Duration     int64  `json:"duration"`
	SessionID    string `json:"session_id,omitempty"`
	RequestID    string `json:"request_id,omitempty"`
	TraceID      string `json:"trace_id,omitempty"`
	Referer      string `json:"referer,omitempty"`
	RiskLevel    string `json:"risk_level"`
	Tags         string `json:"tags,omitempty"`
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
		IPLocation:   log.IPLocation,
		UserAgent:    log.UserAgent,
		DeviceInfo:   log.DeviceInfo,
		Duration:     log.Duration,
		SessionID:    log.SessionID,
		RequestID:    log.RequestID,
		TraceID:      log.TraceID,
		Referer:      log.Referer,
		RiskLevel:    log.RiskLevel,
		Tags:         log.Tags,
		CreatedAt:    log.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ToAlertRuleResp 将 model.AlertRule 转为 AlertRuleResp
func ToAlertRuleResp(rule *model.AlertRule) *AlertRuleResp {
	return &AlertRuleResp{
		ID:              rule.ID,
		Name:            rule.Name,
		Description:     rule.Description,
		Enabled:         rule.Enabled,
		TriggerType:     rule.TriggerType,
		TriggerValue:    rule.TriggerValue,
		NotifyChannels:  rule.NotifyChannels,
		NotifyTargets:   rule.NotifyTargets,
		NotifyTemplate:  rule.NotifyTemplate,
		CooldownMinutes: rule.CooldownMinutes,
		CreatedAt:       rule.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:       rule.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ToAlertLogResp 将 model.AuditAlert 转为 AlertLogResp
func ToAlertLogResp(alert *model.AuditAlert) *AlertLogResp {
	notifyTime := ""
	if !alert.NotifyTime.IsZero() {
		notifyTime = alert.NotifyTime.Format("2006-01-02 15:04:05")
	}
	return &AlertLogResp{
		ID:         alert.ID,
		RuleID:     alert.RuleID,
		RuleName:   alert.RuleName,
		AuditLogID: alert.AuditLogID,
		UserID:     alert.UserID,
		Username:   alert.Username,
		RiskLevel:  alert.RiskLevel,
		Action:     alert.Action,
		Message:    alert.Message,
		Notified:   alert.Notified,
		NotifyTime: notifyTime,
		CreatedAt:  alert.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ListAuditLogsQuery 审计日志列表查询参数
type ListAuditLogsQuery struct {
	Page      int    `json:"page" form:"page"`
	PageSize  int    `json:"page_size" form:"page_size"`
	UserID    string `json:"user_id" form:"user_id"`
	Username  string `json:"username" form:"username"`
	TenantID  string `json:"tenant_id" form:"tenant_id"`
	Module    string `json:"module" form:"module"`
	Action    string `json:"action" form:"action"`
	Resource  string `json:"resource" form:"resource"`
	Result    string `json:"result" form:"result"`
	RiskLevel string `json:"risk_level" form:"risk_level"`
	StartTime string `json:"start_time" form:"start_time"`
	EndTime   string `json:"end_time" form:"end_time"`
	Keyword   string `json:"keyword" form:"keyword"`
}

// AuditLogListResp 审计日志列表响应
type AuditLogListResp struct {
	Items []*AuditLogResp `json:"items"`
	Total int64           `json:"total"`
}

// CreateAlertRuleReq 创建告警规则请求
type CreateAlertRuleReq struct {
	Name            string   `json:"name" binding:"required"`
	Description     string   `json:"description"`
	Enabled         bool     `json:"enabled"`
	TriggerType     string   `json:"trigger_type" binding:"required"`
	TriggerValue    string   `json:"trigger_value" binding:"required"`
	NotifyChannels  []string `json:"notify_channels" binding:"required"`
	NotifyTargets   []string `json:"notify_targets" binding:"required"`
	NotifyTemplate  string   `json:"notify_template"`
	CooldownMinutes int      `json:"cooldown_minutes"`
}

// UpdateAlertRuleReq 更新告警规则请求
type UpdateAlertRuleReq struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Enabled         bool     `json:"enabled"`
	TriggerType     string   `json:"trigger_type"`
	TriggerValue    string   `json:"trigger_value"`
	NotifyChannels  []string `json:"notify_channels"`
	NotifyTargets   []string `json:"notify_targets"`
	NotifyTemplate  string   `json:"notify_template"`
	CooldownMinutes int      `json:"cooldown_minutes"`
}

// AlertRuleResp 告警规则响应
type AlertRuleResp struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Enabled         bool   `json:"enabled"`
	TriggerType     string `json:"trigger_type"`
	TriggerValue    string `json:"trigger_value"`
	NotifyChannels  string `json:"notify_channels"`
	NotifyTargets   string `json:"notify_targets"`
	NotifyTemplate  string `json:"notify_template"`
	CooldownMinutes int    `json:"cooldown_minutes"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// AlertLogResp 告警记录响应
type AlertLogResp struct {
	ID         string `json:"id"`
	RuleID     string `json:"rule_id"`
	RuleName   string `json:"rule_name"`
	AuditLogID string `json:"audit_log_id"`
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	RiskLevel  string `json:"risk_level"`
	Action     string `json:"action"`
	Message    string `json:"message"`
	Notified   bool   `json:"notified"`
	NotifyTime string `json:"notify_time"`
	CreatedAt  string `json:"created_at"`
}

// AuditLogStatsResp 审计日志统计响应
type AuditLogStatsResp struct {
	TotalCount  int64            `json:"total_count"`  // 总日志数
	TodayCount  int64            `json:"today_count"`  // 今日日志数
	ActionStats map[string]int64 `json:"action_stats"` // 按操作类型统计
	ModuleStats map[string]int64 `json:"module_stats"` // 按模块统计
	ResultStats map[string]int64 `json:"result_stats"` // 按结果统计
}

type TrendPoint struct {
	Date    string `json:"date"`
	Count   int64  `json:"count"`
	Success int64  `json:"success"`
	Failure int64  `json:"failure"`
}

type ModuleCount struct {
	Module string `json:"module"`
	Count  int64  `json:"count"`
}

type DashboardResp struct {
	TotalCount  int64            `json:"total_count"`
	TodayCount  int64            `json:"today_count"`
	ActionStats map[string]int64 `json:"action_stats"`
	ModuleStats map[string]int64 `json:"module_stats"`
	ResultStats map[string]int64 `json:"result_stats"`
	Trend       []TrendPoint     `json:"trend"`
	TopModules  []ModuleCount    `json:"top_modules"`
}

// AlertStatsResp 告警统计响应
type AlertStatsResp struct {
	TotalAlerts    int64            `json:"total_alerts"`     // 总告警数
	TodayAlerts    int64            `json:"today_alerts"`     // 今日告警数
	NotifiedCount  int64            `json:"notified_count"`   // 已通知数
	PendingCount   int64            `json:"pending_count"`    // 未通知数
	RiskLevelStats map[string]int64 `json:"risk_level_stats"` // 按风险等级统计
	RuleStats      map[string]int64 `json:"rule_stats"`       // 按规则统计
	Trend          []TrendPoint     `json:"trend"`            // 告警趋势
	TopRules       []RuleCount      `json:"top_rules"`        // 触发最多的规则
}

// RuleCount 规则计数
type RuleCount struct {
	RuleID   string `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Count    int64  `json:"count"`
}

// --- 新增：用户时间线、详细统计、异常检测 ---

// UserTimelineReq 用户时间线查询参数
type UserTimelineReq struct {
	Page      int    `json:"page" form:"page"`
	PageSize  int    `json:"page_size" form:"page_size"`
	UserID    string `json:"user_id" form:"user_id" binding:"required"`
	StartTime string `json:"start_time" form:"start_time"`
	EndTime   string `json:"end_time" form:"end_time"`
}

// TimelineItem 时间线条目（按天聚合）
type TimelineItem struct {
	Date    string            `json:"date"`
	Count   int64             `json:"count"`
	Success int64             `json:"success"`
	Failure int64             `json:"failure"`
	Logs    []*AuditLogResp   `json:"logs"`
}

// UserTimelineResp 用户时间线响应
type UserTimelineResp struct {
	Items   []*TimelineItem   `json:"items"`
	Total   int64             `json:"total"`
	Pages   int               `json:"pages"`
}

// DetailedStatsResp 详细统计数据
type DetailedStatsResp struct {
	TotalCount     int64                  `json:"total_count"`
	TodayCount     int64                  `json:"today_count"`
	ActionStats    map[string]int64       `json:"action_stats"`
	ModuleStats    map[string]int64       `json:"module_stats"`
	ResultStats    map[string]int64       `json:"result_stats"`
	RiskLevelStats map[string]int64       `json:"risk_level_stats"`
	HourlyStats    map[string]int64       `json:"hourly_stats"`
	UserActivity   []*UserActivityStatDTO `json:"user_activity"`
	Trend          []TrendPoint           `json:"trend"`
}

// UserActivityStatDTO 用户活跃度
type UserActivityStatDTO struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Count    int64  `json:"count"`
	Failures int64  `json:"failures"`
}

// AnomalyLogResp 异常日志响应
type AnomalyLogResp struct {
	UserID        string `json:"user_id"`
	Username      string `json:"username"`
	AnomalyType   string `json:"anomaly_type"`
	AnomalyLabel  string `json:"anomaly_label"`
	FailureCount  int64  `json:"failure_count"`
	TotalCount    int64  `json:"total_count"`
	WindowMinutes int    `json:"window_minutes"`
	FirstSeen     string `json:"first_seen"`
	LastSeen      string `json:"last_seen"`
	RiskLevel     string `json:"risk_level"`
	Details       string `json:"details"`
}