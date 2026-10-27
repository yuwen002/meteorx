package audit

import (
	"context"
	"encoding/json"
	"time"

	"meteorx/internal/config"
	"meteorx/internal/middleware"
	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"
	"meteorx/internal/modules/audit/service"
	"meteorx/pkg/logger"

	"gorm.io/gorm"
)

// AuditRetentionJob 审计日志保留策略定时清理任务。
// 审计日志只追加、体量随操作量单调增长，长期不清理会使表无限膨胀并拖慢查询/导出，
// 故按保留天数物理删除更早的行。删除审计历史属敏感操作，默认关闭（retention_days ≤ 0）。
type AuditRetentionJob struct {
	svc  *service.AuditService
	days int
}

// NewAuditRetentionJob 创建审计保留清理任务实例（内部构造 repo + service）。
func NewAuditRetentionJob(db *gorm.DB, cfg config.Config) *AuditRetentionJob {
	repo := repository.NewAuditLogRepository(db)
	return &AuditRetentionJob{
		svc:  service.NewAuditService(repo),
		days: cfg.Audit.GetRetentionDays(),
	}
}

// Start 启动定时清理任务（后台 goroutine），interval 为扫描间隔（通常 1 小时）。
// days ≤ 0 表示关闭：不启动定时器，零后台开销，升级后不会静默删除历史审计。
func (j *AuditRetentionJob) Start(ctx context.Context, interval time.Duration) {
	if j.days <= 0 {
		logger.Info("[AuditRetentionJob] audit.retention_days ≤ 0，审计日志自动清理已关闭")
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := j.runOnce(ctx); err != nil {
					logger.Errorf("[AuditRetentionJob] 清理过期审计日志失败: %v", err)
				} else if n > 0 {
					logger.Infof("[AuditRetentionJob] 已清理 %d 条过期审计日志（保留 %d 天）", n, j.days)
				}
			}
		}
	}()
	logger.Infof("[AuditRetentionJob] 审计日志清理任务已启动，保留 %d 天，间隔 %s", j.days, interval)
}

// runOnce 执行一轮清理：删除早于保留期的审计日志，成功后补记一条自指审计。
// 走 service 直删不经过 handler 的自指留痕，故此处显式补记，避免“删审计历史”本身毫无痕迹。
func (j *AuditRetentionJob) runOnce(ctx context.Context) (int64, error) {
	// 防御：即便被直接调用 RunOnce（运维/测试），days≤0 也绝不删除，避免 cutoff 退化为“删全部”。
	if j.days <= 0 {
		return 0, nil
	}
	start := time.Now()
	deleted, err := j.svc.CleanupLogs(ctx, j.days)
	if err != nil {
		j.recordCleanupAudit(ctx, start, deleted, model.ResultFailure, err.Error())
		return deleted, err
	}
	j.recordCleanupAudit(ctx, start, deleted, model.ResultSuccess, "")
	return deleted, nil
}

// recordCleanupAudit 记录自动清理这一“针对审计系统自身”的高危操作。
// system 触发、无操作者，故 UserID/Username 标记为 system，不落任何审计数据正文，只记条数与结果。
func (j *AuditRetentionJob) recordCleanupAudit(ctx context.Context, start time.Time, deleted int64, result, errMsg string) {
	reqBody, _ := json.Marshal(map[string]interface{}{"days": j.days, "trigger": "system"})
	respBody, _ := json.Marshal(map[string]int64{"deleted_count": deleted})
	tags, _ := json.Marshal([]string{"audit", "retention", "system", model.ActionTypeDelete, result})
	req := dto.CreateAuditLogReq{
		UserID:       "system",
		Username:     "system",
		Module:       "audit",
		Action:       model.ActionTypeDelete,
		Resource:     "audit_logs",
		Path:         "/system/audit-retention",
		RequestBody:  string(reqBody),
		ResponseBody: string(respBody),
		StatusCode:   200,
		Result:       result,
		ErrorMessage: errMsg,
		RiskLevel:    model.RiskCritical,
		Duration:     time.Since(start).Milliseconds(),
		Tags:         string(tags),
	}
	middleware.RecordAuditLog(ctx, &req)
}

// RunOnce 手动执行一轮清理（供测试或运维触发）。
func (j *AuditRetentionJob) RunOnce(ctx context.Context) (int64, error) {
	return j.runOnce(ctx)
}
