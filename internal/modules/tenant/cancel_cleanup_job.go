package tenant

import (
	"context"
	"log"
	"time"

	planrepo "meteorx/internal/modules/plan/repository"
	"meteorx/internal/modules/tenant/repository"
	"meteorx/internal/modules/tenant/service"
	"meteorx/internal/pkg/scheduler"

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
		nil, // role repo 由 bootstrap 注入，注销流程不需要
	)
	// 注销执行需要取消租户的生效订阅（ExecuteCancellation 依赖 subRepo）。
	// 此前此处未注入 subRepo，导致定时任务执行的到期注销会静默跳过订阅取消。
	subRepo := planrepo.NewSubscriptionRepository(db)
	svc.SetSubscriptionRepository(subRepo)
	return &CancelCleanupJob{
		svc:    svc,
		logger: log.Default(),
	}
}

// Start 启动定时任务（后台 goroutine），interval 为扫描间隔
func (j *CancelCleanupJob) Start(ctx context.Context, interval time.Duration) {
	scheduler.Ticker(ctx, "CancelCleanupJob", interval, j.runOnce)
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