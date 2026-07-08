package service

import (
	"context"
	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"
	"meteorx/pkg/ulid"
	"time"
)

// AuditService 审计日志业务服务
type AuditService struct {
	repo repository.AuditLogRepository
}

// NewAuditService 创建审计日志服务
func NewAuditService(repo repository.AuditLogRepository) *AuditService {
	return &AuditService{repo: repo}
}

// CreateLog 创建审计日志
func (s *AuditService) CreateLog(ctx context.Context, req dto.CreateAuditLogReq) (*model.AuditLog, error) {
	log := &model.AuditLog{
		ID:           ulid.Generate(),
		UserID:       req.UserID,
		Username:     req.Username,
		TenantID:     req.TenantID,
		Module:       req.Module,
		Action:       req.Action,
		Resource:     req.Resource,
		ResourceID:   req.ResourceID,
		Method:       req.Method,
		Path:         req.Path,
		RequestBody:  req.RequestBody,
		ResponseBody: req.ResponseBody,
		StatusCode:   req.StatusCode,
		Result:       req.Result,
		ErrorMessage: req.ErrorMessage,
		ClientIP:     req.ClientIP,
		UserAgent:    req.UserAgent,
		Duration:     req.Duration,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, log); err != nil {
		return nil, err
	}
	return log, nil
}

// GetLog 获取审计日志详情
func (s *AuditService) GetLog(ctx context.Context, id string) (*dto.AuditLogResp, error) {
	log, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if log == nil {
		return nil, nil
	}
	return dto.ToAuditLogResp(log), nil
}

// ListLogs 分页查询审计日志
func (s *AuditService) ListLogs(ctx context.Context, query *dto.ListAuditLogsQuery) (*dto.AuditLogListResp, error) {
	repoQuery := &repository.AuditLogQuery{
		Page:      query.Page,
		PageSize:  query.PageSize,
		UserID:    query.UserID,
		Username:  query.Username,
		TenantID:  query.TenantID,
		Module:    query.Module,
		Action:    query.Action,
		Resource:  query.Resource,
		Result:    query.Result,
		StartTime: query.StartTime,
		EndTime:   query.EndTime,
		Keyword:   query.Keyword,
	}

	logs, total, err := s.repo.List(ctx, repoQuery)
	if err != nil {
		return nil, err
	}

	items := make([]*dto.AuditLogResp, len(logs))
	for i, log := range logs {
		items[i] = dto.ToAuditLogResp(log)
	}

	return &dto.AuditLogListResp{
		Items: items,
		Total: total,
	}, nil
}

// GetStats 获取审计日志统计
func (s *AuditService) GetStats(ctx context.Context) (*dto.AuditLogStatsResp, error) {
	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.AuditLogStatsResp{
		TotalCount:  stats.TotalCount,
		TodayCount:  stats.TodayCount,
		ActionStats: stats.ActionStats,
		ModuleStats: stats.ModuleStats,
		ResultStats: stats.ResultStats,
	}, nil
}

// CleanupLogs 清理指定天数之前的日志
func (s *AuditService) CleanupLogs(ctx context.Context, days int) (int64, error) {
	return s.repo.Cleanup(ctx, days)
}