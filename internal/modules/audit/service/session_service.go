package service

import (
	"context"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"
)

type SessionService struct {
	repo repository.AuditLogRepository
}

func NewSessionService(repo repository.AuditLogRepository) *SessionService {
	return &SessionService{repo: repo}
}

// GetSessionLogs 获取会话的所有日志
func (s *SessionService) GetSessionLogs(ctx context.Context, sessionID string) (*dto.SessionAnalysisResp, error) {
	logs, err := s.repo.ListBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if len(logs) == 0 {
		return &dto.SessionAnalysisResp{
			SessionID: sessionID,
			Logs:      []*dto.AuditLogResp{},
			RiskStats: make(map[string]int64),
		}, nil
	}

	logResps := make([]*dto.AuditLogResp, len(logs))
	riskStats := make(map[string]int64)
	var userID, username string
	var startTime, endTime = logs[0].CreatedAt, logs[len(logs)-1].CreatedAt
	var totalDuration int64

	for i, log := range logs {
		logResps[i] = dto.ToAuditLogResp(log)
		if log.RiskLevel != "" {
			riskStats[log.RiskLevel]++
		}
		if userID == "" {
			userID = log.UserID
			username = log.Username
		}
		if log.CreatedAt.Before(startTime) {
			startTime = log.CreatedAt
		}
		if log.CreatedAt.After(endTime) {
			endTime = log.CreatedAt
		}
		totalDuration += log.Duration
	}

	return &dto.SessionAnalysisResp{
		SessionID: sessionID,
		UserID:    userID,
		Username:  username,
		TotalOps:  int64(len(logs)),
		Duration:  totalDuration,
		StartTime: startTime.Format("2006-01-02 15:04:05"),
		EndTime:   endTime.Format("2006-01-02 15:04:05"),
		Logs:      logResps,
		RiskStats: riskStats,
	}, nil
}

// ListSessions 获取会话摘要列表
func (s *SessionService) ListSessions(ctx context.Context, page, pageSize int, userID string) ([]dto.SessionSummaryResp, int64, error) {
	summaries, total, err := s.repo.ListSessions(ctx, page, pageSize, userID)
	if err != nil {
		return nil, 0, err
	}

	resps := make([]dto.SessionSummaryResp, len(summaries))
	for i, s := range summaries {
		duration := s.EndTime.Sub(s.StartTime).Milliseconds()
		resps[i] = dto.SessionSummaryResp{
			SessionID: s.SessionID,
			UserID:    s.UserID,
			Username:  s.Username,
			TotalOps:  s.TotalOps,
			Duration:  duration,
			StartTime: s.StartTime.Format("2006-01-02 15:04:05"),
			EndTime:   s.EndTime.Format("2006-01-02 15:04:05"),
		}
	}

	return resps, total, nil
}

// GetMaxRiskLevel 获取会话中的最高风险等级
func GetMaxRiskLevel(logs []*model.AuditLog) string {
	riskOrder := map[string]int{
		model.RiskLow:      1,
		model.RiskMedium:   2,
		model.RiskHigh:     3,
		model.RiskCritical: 4,
	}

	maxRisk := model.RiskLow
	maxLevel := 0

	for _, log := range logs {
		if level, ok := riskOrder[log.RiskLevel]; ok && level > maxLevel {
			maxLevel = level
			maxRisk = log.RiskLevel
		}
	}

	return maxRisk
}
