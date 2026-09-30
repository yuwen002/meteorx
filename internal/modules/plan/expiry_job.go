package plan

import (
	"context"
	"log"
	"time"

	"meteorx/internal/modules/plan/repository"
	"meteorx/internal/modules/plan/service"
	"meteorx/internal/modules/tenant/model"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	"meteorx/internal/notify"
	userRepo "meteorx/internal/modules/user/repository"

	"gorm.io/gorm"
)

// expiryRemindWindow 到期提醒窗口：订阅在此时间内到期将发送提醒
const expiryRemindWindow = 7 * 24 * time.Hour

// ExpiryJob 到期自动禁用租户的任务实例
type ExpiryJob struct {
	planSvc    *service.PlanService
	subRepo    repository.SubscriptionRepository
	tenantRepo tenantRepo.TenantRepository
	logger     *log.Logger
	notified   map[string]struct{} // 已提醒的订阅+到期日，避免周期任务重复通知
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
		subRepo:    subRepo,
		tenantRepo: tenantRepo,
		logger:     log.Default(),
		notified:   make(map[string]struct{}),
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

	// 对即将到期（尚在有效期内）的订阅发送提醒
	j.remindExpiringSoon(ctx)
	return nil
}

// remindExpiringSoon 扫描即将到期的生效订阅，向租户管理员发送到期提醒
// 通过进程内 notified 集合去重，避免同一订阅在同一到期日内被反复提醒
func (j *ExpiryJob) remindExpiringSoon(ctx context.Context) {
	m := notify.GetGlobalManager()
	if m == nil {
		return
	}
	subs, err := j.subRepo.FindExpiringSoon(ctx, expiryRemindWindow)
	if err != nil {
		j.logger.Printf("[ExpiryJob] 查询即将到期订阅失败: %v", err)
		return
	}
	for _, sub := range subs {
		if sub.ExpiresAt == nil {
			continue
		}
		key := sub.ID + ":" + sub.ExpiresAt.Format("2006-01-02")
		if _, ok := j.notified[key]; ok {
			continue
		}
		tenant, err := j.tenantRepo.GetByID(ctx, sub.TenantID)
		if err != nil || tenant == nil {
			continue
		}
		daysLeft := int(time.Until(*sub.ExpiresAt).Hours() / 24)
		if daysLeft < 0 {
			daysLeft = 0
		}
		m.NotifySubscriptionExpiry(ctx, tenant.Name, daysLeft, tenant.ContactEmail)
		j.notified[key] = struct{}{}
	}
}

// RunOnce 手动执行一次扫描（供测试或运维触发）
func (j *ExpiryJob) RunOnce(ctx context.Context) error {
	return j.runOnce(ctx)
}
