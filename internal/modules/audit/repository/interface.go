package repository

import (
	"context"
	"meteorx/internal/modules/audit/model"
)

// AuditLogRepository 审计日志数据访问接口
type AuditLogRepository interface {
	// Create 创建审计日志
	Create(ctx context.Context, log *model.AuditLog) error

	// BatchCreate 批量创建审计日志（性能优化）
	BatchCreate(ctx context.Context, logs []*model.AuditLog) error

	// GetByID 根据ID获取审计日志
	GetByID(ctx context.Context, id string) (*model.AuditLog, error)

	// List 分页查询审计日志
	List(ctx context.Context, query *AuditLogQuery) ([]*model.AuditLog, int64, error)

	// GetStats 获取审计日志统计信息
	GetStats(ctx context.Context) (*model.AuditLogStats, error)

	// GetActionStats 按操作类型统计
	GetActionStats(ctx context.Context) (map[string]int64, error)

	// GetModuleStats 按模块统计
	GetModuleStats(ctx context.Context) (map[string]int64, error)

	// GetTrendStats 获取趋势统计
	GetTrendStats(ctx context.Context, tenantID string, days int) ([]model.AuditTrendPoint, error)

	// GetTopModules 获取热门模块
	GetTopModules(ctx context.Context, tenantID string, limit int) ([]model.ModuleCount, error)

	// GetDashboardData 获取仪表盘数据
	GetDashboardData(ctx context.Context, tenantID string, days int) (*model.AuditDashboardData, error)

	// Cleanup 清理指定天数之前的日志
	Cleanup(ctx context.Context, days int) (int64, error)

	// ListBySessionID 根据会话ID查询日志
	ListBySessionID(ctx context.Context, sessionID string) ([]*model.AuditLog, error)

	// ListSessions 获取会话摘要列表
	ListSessions(ctx context.Context, page, pageSize int, userID string) ([]model.SessionSummary, int64, error)

	// GetUserTimeline 获取用户操作时间线（按天分组，按时间降序）
	GetUserTimeline(ctx context.Context, userID string, page, pageSize int, startTime, endTime string) ([]*model.AuditLog, int64, error)

	// GetHourlyStats 获取小时级统计
	GetHourlyStats(ctx context.Context, days int) (map[string]int64, error)

	// GetUserActivityStats 获取用户活跃度统计
	GetUserActivityStats(ctx context.Context, days, limit int) ([]model.UserActivityStat, error)

	// GetRiskLevelStats 获取风险等级分布
	GetRiskLevelStats(ctx context.Context, days int) (map[string]int64, error)

	// GetAnomalyLogs 检测异常日志（短时高频失败、异地登录等）
	GetAnomalyLogs(ctx context.Context, threshold int, windowMinutes int) ([]*model.AnomalyLog, error)
}

type AuditLogQuery struct {
	Page      int
	PageSize  int
	UserID    string
	Username  string
	TenantID  string
	Module    string
	Action    string
	Resource  string
	Result    string
	RiskLevel string
	StartTime string
	EndTime   string
	Keyword   string
}

// AlertRuleRepository 告警规则数据访问接口
type AlertRuleRepository interface {
	// Create 创建告警规则
	Create(ctx context.Context, rule *model.AlertRule) error

	// GetByID 根据ID获取告警规则
	GetByID(ctx context.Context, id string) (*model.AlertRule, error)

	// Update 更新告警规则
	Update(ctx context.Context, rule *model.AlertRule) error

	// Delete 删除告警规则
	Delete(ctx context.Context, id string) error

	// List 获取所有告警规则
	List(ctx context.Context) ([]*model.AlertRule, error)

	// GetEnabledRules 获取所有启用的告警规则
	GetEnabledRules(ctx context.Context) ([]*model.AlertRule, error)

	// CreateAlert 创建告警记录
	CreateAlert(ctx context.Context, alert *model.AuditAlert) error

	// ListAlerts 分页查询告警记录
	ListAlerts(ctx context.Context, page, pageSize int, ruleID, userID, riskLevel string) ([]*model.AuditAlert, int64, error)

	// IsInCooldown 检查规则是否在冷却期内
	IsInCooldown(ctx context.Context, ruleID string, cooldownMinutes int) (bool, error)

	// GetAlertStats 获取告警统计数据
	GetAlertStats(ctx context.Context, days int) (*model.AlertStats, error)
}