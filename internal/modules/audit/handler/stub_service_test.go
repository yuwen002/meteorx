package handler_test

import (
	"context"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
)

// stubAuditService 桩实现 handler.AuditService
type stubAuditService struct {
	Err         error
	Created     *model.AuditLog
	LogResp     *dto.AuditLogResp
	ListResp    *dto.AuditLogListResp
	Stats       *dto.AuditLogStatsResp
	Dashboard   *dto.DashboardResp
	Affected    int64

	GotID      string
	GotTenant  string
	GotDays    int
	GotQuery   *dto.ListAuditLogsQuery
}

func (s *stubAuditService) CreateLog(_ context.Context, req dto.CreateAuditLogReq) (*model.AuditLog, error) {
	return s.Created, s.Err
}

func (s *stubAuditService) GetLog(_ context.Context, id string) (*dto.AuditLogResp, error) {
	s.GotID = id
	return s.LogResp, s.Err
}

func (s *stubAuditService) ListLogs(_ context.Context, query *dto.ListAuditLogsQuery) (*dto.AuditLogListResp, error) {
	s.GotQuery = query
	return s.ListResp, s.Err
}

func (s *stubAuditService) GetStats(_ context.Context) (*dto.AuditLogStatsResp, error) {
	return s.Stats, s.Err
}

func (s *stubAuditService) CleanupLogs(_ context.Context, days int) (int64, error) {
	s.GotDays = days
	return s.Affected, s.Err
}

func (s *stubAuditService) GetDashboard(_ context.Context, tenantID string, days int) (*dto.DashboardResp, error) {
	s.GotTenant, s.GotDays = tenantID, days
	return s.Dashboard, s.Err
}

func (s *stubAuditService) GetUserTimeline(_ context.Context, req *dto.UserTimelineReq) (*dto.UserTimelineResp, error) {
	return &dto.UserTimelineResp{Items: []*dto.TimelineItem{}, Total: 0, Pages: 0}, s.Err
}

func (s *stubAuditService) GetDetailedStats(_ context.Context, days int) (*dto.DetailedStatsResp, error) {
	return &dto.DetailedStatsResp{}, s.Err
}

func (s *stubAuditService) GetAnomalyLogs(_ context.Context, threshold int, windowMinutes int) ([]*dto.AnomalyLogResp, error) {
	return []*dto.AnomalyLogResp{}, s.Err
}

// stubAlertService 桩实现 handler.AlertService
type stubAlertService struct {
	Err      error
	Rule     *model.AlertRule
	Rules    []*model.AlertRule
	Alerts   []*model.AuditAlert
	Stats    *model.AlertStats
	Total    int64

	GotID         string
	GotPage       int
	GotPageSize   int
	GotRuleID     string
	GotUserID     string
	GotRiskLevel  string
	GotDays       int
	GotCreateReq  *dto.CreateAlertRuleReq
	GotUpdateReq  *dto.UpdateAlertRuleReq
}

func (s *stubAlertService) CreateRule(_ context.Context, req dto.CreateAlertRuleReq) (*model.AlertRule, error) {
	s.GotCreateReq = &req
	return s.Rule, s.Err
}

func (s *stubAlertService) UpdateRule(_ context.Context, id string, req dto.UpdateAlertRuleReq) (*model.AlertRule, error) {
	s.GotID, s.GotUpdateReq = id, &req
	return s.Rule, s.Err
}

func (s *stubAlertService) DeleteRule(_ context.Context, id string) error {
	s.GotID = id
	return s.Err
}

func (s *stubAlertService) ListRules(_ context.Context) ([]*model.AlertRule, error) {
	return s.Rules, s.Err
}

func (s *stubAlertService) ListAlerts(_ context.Context, page, pageSize int, ruleID, userID, riskLevel string) ([]*model.AuditAlert, int64, error) {
	s.GotPage, s.GotPageSize = page, pageSize
	s.GotRuleID, s.GotUserID, s.GotRiskLevel = ruleID, userID, riskLevel
	return s.Alerts, s.Total, s.Err
}

func (s *stubAlertService) GetAlertStats(_ context.Context, days int) (*model.AlertStats, error) {
	s.GotDays = days
	return s.Stats, s.Err
}

// stubSessionService 桩实现 handler.SessionService
type stubSessionService struct {
	Err      error
	Analysis *dto.SessionAnalysisResp
	Sessions []dto.SessionSummaryResp
	Total    int64

	GotSessionID string
	GotUserID    string
	GotPage      int
	GotPageSize  int
}

func (s *stubSessionService) GetSessionLogs(_ context.Context, sessionID string) (*dto.SessionAnalysisResp, error) {
	s.GotSessionID = sessionID
	return s.Analysis, s.Err
}

func (s *stubSessionService) ListSessions(_ context.Context, page, pageSize int, userID string) ([]dto.SessionSummaryResp, int64, error) {
	s.GotPage, s.GotPageSize, s.GotUserID = page, pageSize, userID
	return s.Sessions, s.Total, s.Err
}