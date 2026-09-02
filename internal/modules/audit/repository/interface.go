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
	StartTime string
	EndTime   string
	Keyword   string
}