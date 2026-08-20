package tenant

import (
	"context"
	"log"
	"time"

	"meteorx/internal/modules/tenant/repository"
	"meteorx/internal/modules/tenant/service"

	"gorm.io/gorm"
)

// CancelCleanupJob 租户注销定时执行任务
type CancelCleanupJob struct {
	svc    *service.TenantService
	logger *log.Logger
}

// NewCancelCleanupJob 创建注销执行任务实例
func NewCancelCleanupJob(db *gorm.DB) *CancelCleanupJob {
	// 复用 tenant 模块的 repo 与 service
	tenantRepo := repository.NewTenantRepository(db)
	svc := service.NewTenantService(
		tenantRepo,
		nil, // user repo 由 bootstrap 注入，注销流程不需要
		nil,
		nil,
	)
	return &CancelCleanupJob{
		svc:    svc,
		logger: log.Default(),
	}
}

// Start 启动定时任务（后台 goroutine），interval 为扫描间隔
func (j *CancelCleanupJob) Start(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := j.runOnce(ctx); err != nil {
					j.logger.Printf("[CancelCleanupJob] 扫描失败: %v", err)
				}
			}
		}
	}()
	j.logger.Printf("[CancelCleanupJob] 租户注销任务已启动，间隔 %s", interval)
}

// runOnce 执行一次扫描：执行所有已到期（effective_at <= now）的已通过注销申请
func (j *CancelCleanupJob) runOnce(ctx context.Context) error {
	executed, err := j.svc.ExecuteDueCancellations(ctx)
	if err != nil {
		return err
	}
	if executed > 0 {
		j.logger.Printf("[CancelCleanupJob] 本次执行了 %d 个租户注销", executed)
	}
	return nil
}

// RunOnce 手动执行一次扫描（供测试或运维触发）
func (j *CancelCleanupJob) RunOnce(ctx context.Context) error {
	return j.runOnce(ctx)
}
