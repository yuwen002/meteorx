package plan

import (
	"context"
	"log"
	"time"

	"meteorx/internal/modules/plan/repository"
	"meteorx/internal/modules/plan/service"
	"meteorx/internal/modules/tenant/model"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	userRepo "meteorx/internal/modules/user/repository"

	"gorm.io/gorm"
)

// ExpiryJob 到期自动禁用租户的任务实例
type ExpiryJob struct {
	planSvc    *service.PlanService
	tenantRepo tenantRepo.TenantRepository
	logger     *log.Logger
}

// NewExpiryJob 创建到期任务实例
func NewExpiryJob(db *gorm.DB) *ExpiryJob {
	// 复用 plan 模块的 repo 与 service
	planRepo := repository.NewPlanRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)
	uRepo := userRepo.NewUserRepository(db)
	tenantRepo := tenantRepo.NewTenantRepository(db)

	planSvc := service.NewPlanService(planRepo, subRepo, uRepo)
	return &ExpiryJob{
		planSvc:    planSvc,
		tenantRepo: tenantRepo,
		logger:     log.Default(),
	}
}

// Start 启动定时任务（后台 goroutine），interval 为扫描间隔
func (j *ExpiryJob) Start(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := j.runOnce(ctx); err != nil {
					j.logger.Printf("[ExpiryJob] 扫描失败: %v", err)
				}
			}
		}
	}()
	j.logger.Printf("[ExpiryJob] 订阅到期监控已启动，间隔 %s", interval)
}

// runOnce 执行一次扫描：发现过期订阅 → 禁用对应租户
func (j *ExpiryJob) runOnce(ctx context.Context) error {
	expiredTenantIDs, err := j.planSvc.ExpireSubscriptions(ctx)
	if err != nil {
		return err
	}
	if len(expiredTenantIDs) == 0 {
		return nil
	}
	for _, tid := range expiredTenantIDs {
		if err := j.tenantRepo.UpdateStatus(ctx, tid, model.StatusDisabled); err != nil {
			j.logger.Printf("[ExpiryJob] 禁用租户 %s 失败: %v", tid, err)
			continue
		}
		j.logger.Printf("[ExpiryJob] 套餐到期，已禁用租户 %s", tid)
	}
	return nil
}

// RunOnce 手动执行一次扫描（供测试或运维触发）
func (j *ExpiryJob) RunOnce(ctx context.Context) error {
	return j.runOnce(ctx)
}