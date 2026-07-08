package repository

import (
	"context"
	"meteorx/internal/modules/audit/model"
)

// AuditLogRepository 审计日志数据访问接口
type AuditLogRepository interface {
	// Create 创建审计日志
	Create(ctx context.Context, log *model.AuditLog) error

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

	// Cleanup 清理指定天数之前的日志
	Cleanup(ctx context.Context, days int) (int64, error)
}

// AuditLogQuery 审计日志查询条件
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