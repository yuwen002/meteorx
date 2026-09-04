package repository

import (
	"context"
	"sort"
	"time"

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

func (m *MockAuditLogRepository) BatchCreate(ctx context.Context, logs []*model.AuditLog) error {
	for _, log := range logs {
		m.logs = append(m.logs, log)
		m.stats.TotalCount++
		m.stats.ActionStats[log.Action]++
		m.stats.ModuleStats[log.Module]++
		m.stats.ResultStats[log.Result]++
	}
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

func (m *MockAuditLogRepository) GetTrendStats(ctx context.Context, tenantID string, days int) ([]model.AuditTrendPoint, error) {
	return []model.AuditTrendPoint{}, nil
}

func (m *MockAuditLogRepository) GetTopModules(ctx context.Context, tenantID string, limit int) ([]model.ModuleCount, error) {
	return []model.ModuleCount{}, nil
}

func (m *MockAuditLogRepository) GetDashboardData(ctx context.Context, tenantID string, days int) (*model.AuditDashboardData, error) {
	return &model.AuditDashboardData{
		TotalCount:  m.stats.TotalCount,
		TodayCount:  m.stats.TodayCount,
		ActionStats: m.stats.ActionStats,
		ModuleStats: m.stats.ModuleStats,
		ResultStats: m.stats.ResultStats,
		Trend:       []model.AuditTrendPoint{},
		TopModules:  []model.ModuleCount{},
	}, nil
}

func (m *MockAuditLogRepository) ListBySessionID(ctx context.Context, sessionID string) ([]*model.AuditLog, error) {
	var result []*model.AuditLog
	for _, log := range m.logs {
		if log.SessionID == sessionID {
			result = append(result, log)
		}
	}
	return result, nil
}

// ListSessions 获取会话摘要列表（模拟真实实现的按会话分组聚合）
func (m *MockAuditLogRepository) ListSessions(ctx context.Context, page, pageSize int, userID string) ([]model.SessionSummary, int64, error) {
	// 按 session_id 分组聚合
	type agg struct {
		sessionID, userID, username string
		totalOps                    int64
		minTime, maxTime            time.Time
	}
	aggMap := make(map[string]*agg)
	for _, log := range m.logs {
		if log.SessionID == "" {
			continue
		}
		if userID != "" && log.UserID != userID {
			continue
		}
		a, ok := aggMap[log.SessionID]
		if !ok {
			a = &agg{
				sessionID: log.SessionID,
				userID:    log.UserID,
				username:  log.Username,
				minTime:   log.CreatedAt,
				maxTime:   log.CreatedAt,
			}
			aggMap[log.SessionID] = a
		}
		a.totalOps++
		if log.CreatedAt.Before(a.minTime) {
			a.minTime = log.CreatedAt
		}
		if log.CreatedAt.After(a.maxTime) {
			a.maxTime = log.CreatedAt
		}
	}

	// 组装并按最后活跃时间倒序
	all := make([]*agg, 0, len(aggMap))
	for _, a := range aggMap {
		all = append(all, a)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].maxTime.After(all[j].maxTime)
	})

	total := int64(len(all))
	start := (page - 1) * pageSize
	if start > len(all) {
		start = len(all)
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}

	summaries := make([]model.SessionSummary, 0, end-start)
	for _, a := range all[start:end] {
		summaries = append(summaries, model.SessionSummary{
			SessionID: a.sessionID,
			UserID:    a.userID,
			Username:  a.username,
			TotalOps:  a.totalOps,
			StartTime: a.minTime,
			EndTime:   a.maxTime,
		})
	}
	return summaries, total, nil
}

// MockAlertRuleRepository 告警规则仓库的内存实现（用于测试）
type MockAlertRuleRepository struct {
	rules  []*model.AlertRule
	alerts []*model.AuditAlert
}

// NewMockAlertRuleRepository 创建 Mock 告警规则仓库
func NewMockAlertRuleRepository() *MockAlertRuleRepository {
	return &MockAlertRuleRepository{
		rules:  make([]*model.AlertRule, 0),
		alerts: make([]*model.AuditAlert, 0),
	}
}

func (m *MockAlertRuleRepository) Create(ctx context.Context, rule *model.AlertRule) error {
	m.rules = append(m.rules, rule)
	return nil
}

func (m *MockAlertRuleRepository) Update(ctx context.Context, rule *model.AlertRule) error {
	for i, r := range m.rules {
		if r.ID == rule.ID {
			m.rules[i] = rule
			return nil
		}
	}
	return nil
}

func (m *MockAlertRuleRepository) Delete(ctx context.Context, id string) error {
	for i, r := range m.rules {
		if r.ID == id {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *MockAlertRuleRepository) GetByID(ctx context.Context, id string) (*model.AlertRule, error) {
	for _, r := range m.rules {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, nil
}

func (m *MockAlertRuleRepository) List(ctx context.Context) ([]*model.AlertRule, error) {
	return m.rules, nil
}

func (m *MockAlertRuleRepository) GetEnabledRules(ctx context.Context) ([]*model.AlertRule, error) {
	var result []*model.AlertRule
	for _, r := range m.rules {
		if r.Enabled {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *MockAlertRuleRepository) CreateAlert(ctx context.Context, alert *model.AuditAlert) error {
	m.alerts = append(m.alerts, alert)
	return nil
}

func (m *MockAlertRuleRepository) ListAlerts(ctx context.Context, page, pageSize int, ruleID, userID, riskLevel string) ([]*model.AuditAlert, int64, error) {
	var result []*model.AuditAlert
	for _, a := range m.alerts {
		if ruleID != "" && a.RuleID != ruleID {
			continue
		}
		if userID != "" && a.UserID != userID {
			continue
		}
		if riskLevel != "" && a.RiskLevel != riskLevel {
			continue
		}
		result = append(result, a)
	}

	start := (page - 1) * pageSize
	if start > len(result) {
		start = len(result)
	}
	end := start + pageSize
	if end > len(result) {
		end = len(result)
	}

	return result[start:end], int64(len(result)), nil
}

func (m *MockAlertRuleRepository) GetAlertByID(ctx context.Context, id string) (*model.AuditAlert, error) {
	for _, a := range m.alerts {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, nil
}

func (m *MockAlertRuleRepository) IsInCooldown(ctx context.Context, ruleID string, cooldownMinutes int) (bool, error) {
	return false, nil
}

func (m *MockAlertRuleRepository) GetAlertStats(ctx context.Context, days int) (*model.AlertStats, error) {
	stats := &model.AlertStats{
		TotalAlerts:    int64(len(m.alerts)),
		TodayAlerts:    0,
		NotifiedCount:  0,
		PendingCount:   0,
		RiskLevelStats: make(map[string]int64),
		RuleStats:      make(map[string]int64),
		Trend:          []model.TrendPoint{},
		TopRules:       []model.RuleCount{},
	}

	for _, alert := range m.alerts {
		stats.RiskLevelStats[alert.RiskLevel]++
		stats.RuleStats[alert.RuleID]++
		if alert.Notified {
			stats.NotifiedCount++
		} else {
			stats.PendingCount++
		}
	}

	return stats, nil
}
