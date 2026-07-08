package repository

import (
	"context"
	"meteorx/internal/modules/audit/model"
)

// MockAuditLogRepository 审计日志仓库的内存实现（用于测试）
type MockAuditLogRepository struct {
	logs  []*model.AuditLog
	stats *model.AuditLogStats
}

// NewMockAuditLogRepository 创建 Mock 仓库
func NewMockAuditLogRepository() *MockAuditLogRepository {
	return &MockAuditLogRepository{
		logs: make([]*model.AuditLog, 0),
		stats: &model.AuditLogStats{
			TotalCount:  0,
			TodayCount:  0,
			ActionStats: make(map[string]int64),
			ModuleStats: make(map[string]int64),
			ResultStats: make(map[string]int64),
		},
	}
}

func (m *MockAuditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	m.logs = append(m.logs, log)
	m.stats.TotalCount++
	m.stats.ActionStats[log.Action]++
	m.stats.ModuleStats[log.Module]++
	m.stats.ResultStats[log.Result]++
	return nil
}

func (m *MockAuditLogRepository) GetByID(ctx context.Context, id string) (*model.AuditLog, error) {
	for _, log := range m.logs {
		if log.ID == id {
			return log, nil
		}
	}
	return nil, nil
}

func (m *MockAuditLogRepository) List(ctx context.Context, query *AuditLogQuery) ([]*model.AuditLog, int64, error) {
	var result []*model.AuditLog
	for _, log := range m.logs {
		if query.UserID != "" && log.UserID != query.UserID {
			continue
		}
		if query.Module != "" && log.Module != query.Module {
			continue
		}
		if query.Action != "" && log.Action != query.Action {
			continue
		}
		if query.Result != "" && log.Result != query.Result {
			continue
		}
		result = append(result, log)
	}

	// 分页
	start := (query.Page - 1) * query.PageSize
	if start > len(result) {
		start = len(result)
	}
	end := start + query.PageSize
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], int64(len(result)), nil
}

func (m *MockAuditLogRepository) GetStats(ctx context.Context) (*model.AuditLogStats, error) {
	return m.stats, nil
}

func (m *MockAuditLogRepository) GetActionStats(ctx context.Context) (map[string]int64, error) {
	return m.stats.ActionStats, nil
}

func (m *MockAuditLogRepository) GetModuleStats(ctx context.Context) (map[string]int64, error) {
	return m.stats.ModuleStats, nil
}

func (m *MockAuditLogRepository) Cleanup(ctx context.Context, days int) (int64, error) {
	deleted := int64(len(m.logs))
	m.logs = make([]*model.AuditLog, 0)
	m.stats.TotalCount = 0
	m.stats.ActionStats = make(map[string]int64)
	m.stats.ModuleStats = make(map[string]int64)
	m.stats.ResultStats = make(map[string]int64)
	return deleted, nil
}