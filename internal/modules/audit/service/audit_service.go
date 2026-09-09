package service

import (
	"context"
	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"
	"meteorx/pkg/idgen"
	"sort"
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
		ID:           idgen.New(),
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
		IPLocation:   req.IPLocation,
		UserAgent:    req.UserAgent,
		DeviceInfo:   req.DeviceInfo,
		Duration:     req.Duration,
		SessionID:    req.SessionID,
		RequestID:    req.RequestID,
		TraceID:      req.TraceID,
		Referer:      req.Referer,
		RiskLevel:    req.RiskLevel,
		Tags:         req.Tags,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, log); err != nil {
		return nil, err
	}
	return log, nil
}

// BatchCreateLogs 批量创建审计日志（性能优化）
func (s *AuditService) BatchCreateLogs(ctx context.Context, reqs []dto.CreateAuditLogReq) error {
	if len(reqs) == 0 {
		return nil
	}

	logs := make([]*model.AuditLog, len(reqs))
	for i, req := range reqs {
		logs[i] = &model.AuditLog{
			ID:           idgen.New(),
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
			IPLocation:   req.IPLocation,
			UserAgent:    req.UserAgent,
			DeviceInfo:   req.DeviceInfo,
			Duration:     req.Duration,
			SessionID:    req.SessionID,
			RequestID:    req.RequestID,
			TraceID:      req.TraceID,
			Referer:      req.Referer,
			RiskLevel:    req.RiskLevel,
			Tags:         req.Tags,
			CreatedAt:    time.Now(),
		}
	}

	return s.repo.BatchCreate(ctx, logs)
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
		RiskLevel: query.RiskLevel,
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

// GetDashboard 获取仪表盘数据
func (s *AuditService) GetDashboard(ctx context.Context, tenantID string, days int) (*dto.DashboardResp, error) {
	if days <= 0 {
		days = 7
	}

	data, err := s.repo.GetDashboardData(ctx, tenantID, days)
	if err != nil {
		return nil, err
	}

	trend := make([]dto.TrendPoint, len(data.Trend))
	for i, t := range data.Trend {
		trend[i] = dto.TrendPoint{
			Date:    t.Date,
			Count:   t.Count,
			Success: t.Success,
			Failure: t.Failure,
		}
	}

	topModules := make([]dto.ModuleCount, len(data.TopModules))
	for i, m := range data.TopModules {
		topModules[i] = dto.ModuleCount{
			Module: m.Module,
			Count:  m.Count,
		}
	}

	return &dto.DashboardResp{
		TotalCount:  data.TotalCount,
		TodayCount:  data.TodayCount,
		ActionStats: data.ActionStats,
		ModuleStats: data.ModuleStats,
		ResultStats: data.ResultStats,
		Trend:       trend,
		TopModules:  topModules,
	}, nil
}

// GetUserTimeline 获取用户操作时间线
func (s *AuditService) GetUserTimeline(ctx context.Context, req *dto.UserTimelineReq) (*dto.UserTimelineResp, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	logs, total, err := s.repo.GetUserTimeline(ctx, req.UserID, req.Page, req.PageSize, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}

	// 按日期分组
	dateMap := make(map[string][]*dto.AuditLogResp)
	for _, log := range logs {
		date := log.CreatedAt.Format("2006-01-02")
		dateMap[date] = append(dateMap[date], dto.ToAuditLogResp(log))
	}

	// 按日期排序
	dates := make([]string, 0, len(dateMap))
	for date := range dateMap {
		dates = append(dates, date)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	items := make([]*dto.TimelineItem, 0, len(dates))
	for _, date := range dates {
		logs := dateMap[date]
		var success, failure int64
		for _, l := range logs {
			if l.Result == "success" {
				success++
			} else {
				failure++
			}
		}
		items = append(items, &dto.TimelineItem{
			Date:    date,
			Count:   int64(len(logs)),
			Success: success,
			Failure: failure,
			Logs:    logs,
		})
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &dto.UserTimelineResp{
		Items: items,
		Total: total,
		Pages: totalPages,
	}, nil
}

// GetDetailedStats 获取详细统计
func (s *AuditService) GetDetailedStats(ctx context.Context, days int) (*dto.DetailedStatsResp, error) {
	if days <= 0 {
		days = 7
	}

	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	hourlyStats, err := s.repo.GetHourlyStats(ctx, days)
	if err != nil {
		return nil, err
	}

	userActivity, err := s.repo.GetUserActivityStats(ctx, days, 20)
	if err != nil {
		return nil, err
	}

	riskLevelStats, err := s.repo.GetRiskLevelStats(ctx, days)
	if err != nil {
		return nil, err
	}

	trend, err := s.repo.GetTrendStats(ctx, "", days)
	if err != nil {
		return nil, err
	}

	trendPoints := make([]dto.TrendPoint, len(trend))
	for i, t := range trend {
		trendPoints[i] = dto.TrendPoint{
			Date:    t.Date,
			Count:   t.Count,
			Success: t.Success,
			Failure: t.Failure,
		}
	}

	activityDTOs := make([]*dto.UserActivityStatDTO, len(userActivity))
	for i, u := range userActivity {
		activityDTOs[i] = &dto.UserActivityStatDTO{
			UserID:   u.UserID,
			Username: u.Username,
			Count:    u.Count,
			Failures: u.Failures,
		}
	}

	return &dto.DetailedStatsResp{
		TotalCount:     stats.TotalCount,
		TodayCount:     stats.TodayCount,
		ActionStats:    stats.ActionStats,
		ModuleStats:    stats.ModuleStats,
		ResultStats:    stats.ResultStats,
		RiskLevelStats: riskLevelStats,
		HourlyStats:    hourlyStats,
		UserActivity:   activityDTOs,
		Trend:          trendPoints,
	}, nil
}

// GetAnomalyLogs 获取异常日志
func (s *AuditService) GetAnomalyLogs(ctx context.Context, threshold int, windowMinutes int) ([]*dto.AnomalyLogResp, error) {
	if threshold <= 0 {
		threshold = 5
	}
	if windowMinutes <= 0 {
		windowMinutes = 30
	}

	anomalies, err := s.repo.GetAnomalyLogs(ctx, threshold, windowMinutes)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.AnomalyLogResp, len(anomalies))
	for i, a := range anomalies {
		label := getAnomalyLabel(a.AnomalyType)
		result[i] = &dto.AnomalyLogResp{
			UserID:        a.UserID,
			Username:      a.Username,
			AnomalyType:   a.AnomalyType,
			AnomalyLabel:  label,
			FailureCount:  a.FailureCount,
			TotalCount:    a.TotalCount,
			WindowMinutes: a.WindowMinutes,
			FirstSeen:     a.FirstSeen.Format("2006-01-02 15:04:05"),
			LastSeen:      a.LastSeen.Format("2006-01-02 15:04:05"),
			RiskLevel:     a.RiskLevel,
			Details:       a.Details,
		}
	}
	return result, nil
}

// getAnomalyLabel 获取异常类型中文标签
func getAnomalyLabel(t string) string {
	switch t {
	case model.AnomalyTypeHighFailure:
		return "高频操作失败"
	case model.AnomalyTypeGeoAnomaly:
		return "异地登录异常"
	case model.AnomalyTypeBruteForce:
		return "暴力破解尝试"
	default:
		return t
	}
}