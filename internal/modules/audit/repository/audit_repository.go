package repository

import (
	"context"
	"meteorx/internal/modules/audit/model"
	"strings"
	"time"

	"gorm.io/gorm"
)

// AuditLogPO 审计日志数据库模型
type AuditLogPO struct {
	ID           string    `gorm:"primaryKey;size:26;comment:日志ID"`
	UserID       string    `gorm:"index;size:26;comment:操作用户ID"`
	Username     string    `gorm:"index;size:50;comment:操作用户名"`
	TenantID     string    `gorm:"index;size:26;comment:租户ID"`
	Module       string    `gorm:"index;size:50;comment:操作模块"`
	Action       string    `gorm:"index;size:20;comment:操作类型"`
	Resource     string    `gorm:"size:100;comment:操作资源"`
	ResourceID   string    `gorm:"size:26;comment:被操作资源ID"`
	Method       string    `gorm:"size:10;comment:HTTP方法"`
	Path         string    `gorm:"size:255;comment:请求路径"`
	RequestBody  string    `gorm:"type:text;comment:请求参数"`
	ResponseBody string    `gorm:"type:text;comment:响应结果"`
	StatusCode   int       `gorm:"comment:HTTP状态码"`
	Result       string    `gorm:"index;size:10;comment:操作结果"`
	ErrorMessage string    `gorm:"type:text;comment:错误信息"`
	ClientIP     string    `gorm:"size:50;comment:客户端IP"`
	IPLocation   string    `gorm:"size:100;comment:IP地理位置"`
	UserAgent    string    `gorm:"size:255;comment:用户代理"`
	DeviceInfo   string    `gorm:"size:255;comment:设备信息"`
	Duration     int64     `gorm:"comment:请求耗时(毫秒)"`
	SessionID    string    `gorm:"index;size:50;comment:会话ID"`
	RequestID    string    `gorm:"index;size:50;comment:请求ID"`
	TraceID      string    `gorm:"index;size:50;comment:链路追踪ID"`
	Referer      string    `gorm:"size:500;comment:来源页面"`
	RiskLevel    string    `gorm:"index;size:20;comment:风险等级(low/medium/high/critical)"`
	Tags         string    `gorm:"size:500;comment:标签(JSON数组)"`
	CreatedAt    time.Time `gorm:"index;autoCreateTime;comment:创建时间"`
}

func (AuditLogPO) TableName() string {
	return "audit_logs"
}

// toDomain 转换为领域模型
func (po AuditLogPO) toDomain() *model.AuditLog {
	return &model.AuditLog{
		ID:           po.ID,
		UserID:       po.UserID,
		Username:     po.Username,
		TenantID:     po.TenantID,
		Module:       po.Module,
		Action:       po.Action,
		Resource:     po.Resource,
		ResourceID:   po.ResourceID,
		Method:       po.Method,
		Path:         po.Path,
		RequestBody:  po.RequestBody,
		ResponseBody: po.ResponseBody,
		StatusCode:   po.StatusCode,
		Result:       po.Result,
		ErrorMessage: po.ErrorMessage,
		ClientIP:     po.ClientIP,
		IPLocation:   po.IPLocation,
		UserAgent:    po.UserAgent,
		DeviceInfo:   po.DeviceInfo,
		Duration:     po.Duration,
		SessionID:    po.SessionID,
		RequestID:    po.RequestID,
		TraceID:      po.TraceID,
		Referer:      po.Referer,
		RiskLevel:    po.RiskLevel,
		Tags:         po.Tags,
		CreatedAt:    po.CreatedAt,
	}
}

// fromDomain 从领域模型转换
func auditLogFromDomain(l *model.AuditLog) *AuditLogPO {
	return &AuditLogPO{
		ID:           l.ID,
		UserID:       l.UserID,
		Username:     l.Username,
		TenantID:     l.TenantID,
		Module:       l.Module,
		Action:       l.Action,
		Resource:     l.Resource,
		ResourceID:   l.ResourceID,
		Method:       l.Method,
		Path:         l.Path,
		RequestBody:  l.RequestBody,
		ResponseBody: l.ResponseBody,
		StatusCode:   l.StatusCode,
		Result:       l.Result,
		ErrorMessage: l.ErrorMessage,
		ClientIP:     l.ClientIP,
		IPLocation:   l.IPLocation,
		UserAgent:    l.UserAgent,
		DeviceInfo:   l.DeviceInfo,
		Duration:     l.Duration,
		SessionID:    l.SessionID,
		RequestID:    l.RequestID,
		TraceID:      l.TraceID,
		Referer:      l.Referer,
		RiskLevel:    l.RiskLevel,
		Tags:         l.Tags,
		CreatedAt:    l.CreatedAt,
	}
}

// auditLogRepository 审计日志仓库实现
type auditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository 创建审计日志仓库
func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

// AutoMigrate 自动迁移表结构
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&AuditLogPO{}, &AlertRulePO{}, &AuditAlertPO{}); err != nil {
		return err
	}
	return nil
}

func (r *auditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	po := auditLogFromDomain(log)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *auditLogRepository) BatchCreate(ctx context.Context, logs []*model.AuditLog) error {
	if len(logs) == 0 {
		return nil
	}
	pos := make([]*AuditLogPO, len(logs))
	for i, log := range logs {
		pos[i] = auditLogFromDomain(log)
	}
	return r.db.WithContext(ctx).Create(&pos).Error
}

func (r *auditLogRepository) GetByID(ctx context.Context, id string) (*model.AuditLog, error) {
	var po AuditLogPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *auditLogRepository) List(ctx context.Context, query *AuditLogQuery) ([]*model.AuditLog, int64, error) {
	db := r.db.WithContext(ctx).Model(&AuditLogPO{})

	// 动态条件
	if query.UserID != "" {
		db = db.Where("user_id = ?", query.UserID)
	}
	if query.Username != "" {
		db = db.Where("username LIKE ?", "%"+query.Username+"%")
	}
	if query.TenantID != "" {
		db = db.Where("tenant_id = ?", query.TenantID)
	}
	if query.Module != "" {
		db = db.Where("module = ?", query.Module)
	}
	if query.Action != "" {
		actions := strings.Split(query.Action, ",")
		if len(actions) > 1 {
			db = db.Where("action IN ?", actions)
		} else {
			db = db.Where("action = ?", query.Action)
		}
	}
	if query.Resource != "" {
		db = db.Where("resource LIKE ?", "%"+query.Resource+"%")
	}
	if query.Result != "" {
		db = db.Where("result = ?", query.Result)
	}
	if query.RiskLevel != "" {
		db = db.Where("risk_level = ?", query.RiskLevel)
	}
	if query.StartTime != "" {
		db = db.Where("created_at >= ?", query.StartTime)
	}
	if query.EndTime != "" {
		db = db.Where("created_at <= ?", query.EndTime)
	}
	if query.Keyword != "" {
		db = db.Where("path LIKE ? OR username LIKE ? OR resource LIKE ?",
			"%"+query.Keyword+"%", "%"+query.Keyword+"%", "%"+query.Keyword+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var pos []AuditLogPO
	if err := db.Order("created_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	logs := make([]*model.AuditLog, len(pos))
	for i, po := range pos {
		logs[i] = po.toDomain()
	}
	return logs, total, nil
}

func (r *auditLogRepository) GetStats(ctx context.Context) (*model.AuditLogStats, error) {
	var totalCount int64
	if err := r.db.WithContext(ctx).Model(&AuditLogPO{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}

	var todayCount int64
	today := time.Now().Format("2006-01-02")
	if err := r.db.WithContext(ctx).Model(&AuditLogPO{}).Where("DATE(created_at) = ?", today).Count(&todayCount).Error; err != nil {
		return nil, err
	}

	actionStats, err := r.GetActionStats(ctx)
	if err != nil {
		return nil, err
	}

	moduleStats, err := r.GetModuleStats(ctx)
	if err != nil {
		return nil, err
	}

	var resultStats = make(map[string]int64)
	var successCount, failureCount int64
	r.db.WithContext(ctx).Model(&AuditLogPO{}).Where("result = ?", model.ResultSuccess).Count(&successCount)
	r.db.WithContext(ctx).Model(&AuditLogPO{}).Where("result = ?", model.ResultFailure).Count(&failureCount)
	resultStats[model.ResultSuccess] = successCount
	resultStats[model.ResultFailure] = failureCount

	return &model.AuditLogStats{
		TotalCount:  totalCount,
		TodayCount:  todayCount,
		ActionStats: actionStats,
		ModuleStats: moduleStats,
		ResultStats: resultStats,
	}, nil
}

func (r *auditLogRepository) GetActionStats(ctx context.Context) (map[string]int64, error) {
	type result struct {
		Action string
		Count  int64
	}
	var results []result
	if err := r.db.WithContext(ctx).Model(&AuditLogPO{}).
		Select("action, COUNT(*) as count").
		Group("action").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, r := range results {
		stats[r.Action] = r.Count
	}
	return stats, nil
}

func (r *auditLogRepository) GetModuleStats(ctx context.Context) (map[string]int64, error) {
	type result struct {
		Module string
		Count  int64
	}
	var results []result
	if err := r.db.WithContext(ctx).Model(&AuditLogPO{}).
		Select("module, COUNT(*) as count").
		Group("module").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, r := range results {
		stats[r.Module] = r.Count
	}
	return stats, nil
}

func (r *auditLogRepository) GetTrendStats(ctx context.Context, tenantID string, days int) ([]model.AuditTrendPoint, error) {
	type Result struct {
		Date    string
		Count   int64
		Success int64
		Failure int64
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	db := r.db.WithContext(ctx).Model(&AuditLogPO{})

	if tenantID != "" {
		db = db.Where("tenant_id = ?", tenantID)
	}

	var results []Result
	err := db.Select("DATE(created_at) as date, COUNT(*) as count, SUM(CASE WHEN result = 'success' THEN 1 ELSE 0 END) as success, SUM(CASE WHEN result = 'failure' THEN 1 ELSE 0 END) as failure").
		Where("created_at >= ?", cutoff).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	points := make([]model.AuditTrendPoint, len(results))
	for i, r := range results {
		points[i] = model.AuditTrendPoint{
			Date:    r.Date,
			Count:   r.Count,
			Success: r.Success,
			Failure: r.Failure,
		}
	}
	return points, nil
}

func (r *auditLogRepository) GetTopModules(ctx context.Context, tenantID string, limit int) ([]model.ModuleCount, error) {
	type Result struct {
		Module string
		Count  int64
	}

	db := r.db.WithContext(ctx).Model(&AuditLogPO{})
	if tenantID != "" {
		db = db.Where("tenant_id = ?", tenantID)
	}

	var results []Result
	err := db.Select("module, COUNT(*) as count").
		Group("module").
		Order("count DESC").
		Limit(limit).
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	modules := make([]model.ModuleCount, len(results))
	for i, r := range results {
		modules[i] = model.ModuleCount{
			Module: r.Module,
			Count:  r.Count,
		}
	}
	return modules, nil
}

func (r *auditLogRepository) GetDashboardData(ctx context.Context, tenantID string, days int) (*model.AuditDashboardData, error) {
	db := r.db.WithContext(ctx).Model(&AuditLogPO{})

	if tenantID != "" {
		db = db.Where("tenant_id = ?", tenantID)
	}

	var totalCount int64
	if err := db.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	var todayCount int64
	today := time.Now().Format("2006-01-02")
	db2 := r.db.WithContext(ctx).Model(&AuditLogPO{})
	if tenantID != "" {
		db2 = db2.Where("tenant_id = ?", tenantID)
	}
	if err := db2.Where("DATE(created_at) = ?", today).Count(&todayCount).Error; err != nil {
		return nil, err
	}

	actionStats, err := r.GetActionStats(ctx)
	if err != nil {
		return nil, err
	}

	moduleStats, err := r.GetModuleStats(ctx)
	if err != nil {
		return nil, err
	}

	var resultStats = make(map[string]int64)
	var successCount, failureCount int64
	db3 := r.db.WithContext(ctx).Model(&AuditLogPO{})
	if tenantID != "" {
		db3 = db3.Where("tenant_id = ?", tenantID)
	}
	db3.Where("result = ?", model.ResultSuccess).Count(&successCount)
	db4 := r.db.WithContext(ctx).Model(&AuditLogPO{})
	if tenantID != "" {
		db4 = db4.Where("tenant_id = ?", tenantID)
	}
	db4.Where("result = ?", model.ResultFailure).Count(&failureCount)
	resultStats[model.ResultSuccess] = successCount
	resultStats[model.ResultFailure] = failureCount

	trend, err := r.GetTrendStats(ctx, tenantID, days)
	if err != nil {
		return nil, err
	}

	topModules, err := r.GetTopModules(ctx, tenantID, 10)
	if err != nil {
		return nil, err
	}

	return &model.AuditDashboardData{
		TotalCount:  totalCount,
		TodayCount:  todayCount,
		ActionStats: actionStats,
		ModuleStats: moduleStats,
		ResultStats: resultStats,
		Trend:       trend,
		TopModules:  topModules,
	}, nil
}

func (r *auditLogRepository) Cleanup(ctx context.Context, days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	result := r.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&AuditLogPO{})
	return result.RowsAffected, result.Error
}

// ListBySessionID 根据会话ID查询日志
func (r *auditLogRepository) ListBySessionID(ctx context.Context, sessionID string) ([]*model.AuditLog, error) {
	var pos []AuditLogPO
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&pos).Error; err != nil {
		return nil, err
	}

	logs := make([]*model.AuditLog, len(pos))
	for i, po := range pos {
		logs[i] = po.toDomain()
	}
	return logs, nil
}

// ListSessions 获取会话摘要列表
func (r *auditLogRepository) ListSessions(ctx context.Context, page, pageSize int, userID string) ([]model.SessionSummary, int64, error) {
	type SessionRow struct {
		SessionID string
		UserID    string
		Username  string
		TotalOps  int64
		MinTime   time.Time
		MaxTime   time.Time
		MaxRisk   string
	}

	db := r.db.WithContext(ctx).Table("audit_logs").
		Select("session_id, user_id, username, COUNT(*) as total_ops, MIN(created_at) as min_time, MAX(created_at) as max_time").
		Where("session_id != '' AND session_id IS NOT NULL").
		Group("session_id, user_id, username")

	if userID != "" {
		db = db.Where("user_id = ?", userID)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []SessionRow
	if err := db.Order("max_time DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	summaries := make([]model.SessionSummary, len(rows))
	for i, row := range rows {
		summaries[i] = model.SessionSummary{
			SessionID: row.SessionID,
			UserID:    row.UserID,
			Username:  row.Username,
			TotalOps:  row.TotalOps,
			StartTime: row.MinTime,
			EndTime:   row.MaxTime,
		}
	}

	return summaries, total, nil
}

// GetUserTimeline 获取用户操作时间线
func (r *auditLogRepository) GetUserTimeline(ctx context.Context, userID string, page, pageSize int, startTime, endTime string) ([]*model.AuditLog, int64, error) {
	db := r.db.WithContext(ctx).Model(&AuditLogPO{}).Where("user_id = ?", userID)

	if startTime != "" {
		db = db.Where("created_at >= ?", startTime)
	}
	if endTime != "" {
		db = db.Where("created_at <= ?", endTime)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var pos []AuditLogPO
	if err := db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	logs := make([]*model.AuditLog, len(pos))
	for i, po := range pos {
		logs[i] = po.toDomain()
	}
	return logs, total, nil
}

// GetHourlyStats 获取小时级统计
func (r *auditLogRepository) GetHourlyStats(ctx context.Context, days int) (map[string]int64, error) {
	type result struct {
		Hour  string
		Count int64
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	var results []result
	if err := r.db.WithContext(ctx).Model(&AuditLogPO{}).
		Select("DATE_FORMAT(created_at, '%Y-%m-%d %H:00') as hour, COUNT(*) as count").
		Where("created_at >= ?", cutoff).
		Group("hour").
		Order("hour ASC").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, r := range results {
		stats[r.Hour] = r.Count
	}
	return stats, nil
}

// GetUserActivityStats 获取用户活跃度统计
func (r *auditLogRepository) GetUserActivityStats(ctx context.Context, days, limit int) ([]model.UserActivityStat, error) {
	type result struct {
		UserID   string
		Username string
		Count    int64
		Failures int64
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	var results []result
	if err := r.db.WithContext(ctx).Model(&AuditLogPO{}).
		Select("user_id, username, COUNT(*) as count, SUM(CASE WHEN result = 'failure' THEN 1 ELSE 0 END) as failures").
		Where("created_at >= ? AND user_id != ''", cutoff).
		Group("user_id, username").
		Order("count DESC").
		Limit(limit).
		Scan(&results).Error; err != nil {
		return nil, err
	}

	stats := make([]model.UserActivityStat, len(results))
	for i, r := range results {
		stats[i] = model.UserActivityStat{
			UserID:   r.UserID,
			Username: r.Username,
			Count:    r.Count,
			Failures: r.Failures,
		}
	}
	return stats, nil
}

// GetRiskLevelStats 获取风险等级分布
func (r *auditLogRepository) GetRiskLevelStats(ctx context.Context, days int) (map[string]int64, error) {
	type result struct {
		RiskLevel string
		Count     int64
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	var results []result
	if err := r.db.WithContext(ctx).Model(&AuditLogPO{}).
		Select("risk_level, COUNT(*) as count").
		Where("created_at >= ? AND risk_level != ''", cutoff).
		Group("risk_level").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, r := range results {
		stats[r.RiskLevel] = r.Count
	}
	return stats, nil
}

// GetAnomalyLogs 检测异常日志
func (r *auditLogRepository) GetAnomalyLogs(ctx context.Context, threshold int, windowMinutes int) ([]*model.AnomalyLog, error) {
	// 1. 高频失败检测：在指定时间窗口内，同一用户失败次数超过阈值
	type failureResult struct {
		UserID   string
		Username string
		Count    int64
		Total    int64
		MinTime  time.Time
		MaxTime  time.Time
	}

	cutoff := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)
	var failures []failureResult
	if err := r.db.WithContext(ctx).Model(&AuditLogPO{}).
		Select("user_id, username, COUNT(*) as count, SUM(CASE WHEN result = 'failure' THEN 1 ELSE 0 END) as total, MIN(created_at) as min_time, MAX(created_at) as max_time").
		Where("created_at >= ? AND user_id != ''", cutoff).
		Group("user_id, username").
		Having("total >= ?", threshold).
		Scan(&failures).Error; err != nil {
		return nil, err
	}

	var anomalies []*model.AnomalyLog
	for _, f := range failures {
		riskLevel := model.RiskMedium
		if f.Total >= int64(threshold*3) {
			riskLevel = model.RiskHigh
		}
		anomalies = append(anomalies, &model.AnomalyLog{
			UserID:        f.UserID,
			Username:      f.Username,
			AnomalyType:   model.AnomalyTypeHighFailure,
			FailureCount:  f.Total,
			TotalCount:    f.Count,
			WindowMinutes: windowMinutes,
			FirstSeen:     f.MinTime,
			LastSeen:      f.MaxTime,
			RiskLevel:     riskLevel,
			Details:       "",
		})
	}

	return anomalies, nil
}