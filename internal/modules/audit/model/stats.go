package model

// AuditLogStats 审计日志统计
type AuditLogStats struct {
	TotalCount  int64
	TodayCount  int64
	ActionStats map[string]int64
	ModuleStats map[string]int64
	ResultStats map[string]int64
}