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
	"meteorx/internal/notify"

	"gorm.io/gorm"
)

// expiryRemindWindow 到期提醒窗口：订阅在此时间内到期将发送提醒
const expiryRemindWindow = 7 * 24 * time.Hour

// TenantDirectory 到期提醒文案所需的租户信息查询。
// 声明为窄接口：本任务只需按 ID 取名称/邮箱，tenantRepo.TenantRepository 天然满足，
// 既避免依赖含二十余个方法的胖接口，也使定时任务可脱离数据库单测。
type TenantDirectory interface {
	GetByID(ctx context.Context, id string) (*model.Tenant, error)
}

// tenantDisabler 将 tenant 仓储的 UpdateStatus 适配为 plan service 的到期禁用回调
type tenantDisabler struct{ repo tenantRepo.TenantRepository }

func (d tenantDisabler) DisableTenant(ctx context.Context, tenantID string) error {
	return d.repo.UpdateStatus(ctx, tenantID, model.StatusDisabled)
}

// ExpiryJob 到期自动禁用租户的任务实例
type ExpiryJob struct {
	planSvc  *service.PlanService
	subRepo  repository.SubscriptionRepository
	tenants  TenantDirectory
	logger   *log.Logger
	notified map[string]struct{} // 已提醒的订阅+到期日，避免周期任务重复通知
}

// NewExpiryJob 创建到期任务实例
func NewExpiryJob(db *gorm.DB) *ExpiryJob {
	// 复用 plan 模块的 repo 与 service
	planRepo := repository.NewPlanRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)
	uRepo := userRepo.NewUserRepository(db)
	tRepo := tenantRepo.NewTenantRepository(db)

	planSvc := service.NewPlanService(planRepo, subRepo, uRepo)
	// 禁用租户与翻转订阅状态必须作为一个可重入的整体处理（见 ExpireSubscriptions 注释）
	planSvc.SetTenantDisabler(tenantDisabler{repo: tRepo})

	return &ExpiryJob{
		planSvc:  planSvc,
		subRepo:  subRepo,
		tenants:  tRepo,
		logger:   log.Default(),
		notified: make(map[string]struct{}),
	}
}

// newExpiryJobWith 测试装配入口，与 NewExpiryJob 同构字段绑定
func newExpiryJobWith(planSvc *service.PlanService, subRepo repository.SubscriptionRepository, tenants TenantDirectory) *ExpiryJob {
	return &ExpiryJob{
		planSvc:  planSvc,
		subRepo:  subRepo,
		tenants:  tenants,
		logger:   log.Default(),
		notified: make(map[string]struct{}),
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

// runOnce 执行一次扫描：到期订阅 → 禁用租户 → 即将到期提醒
func (j *ExpiryJob) runOnce(ctx context.Context) error {
	// ExpireSubscriptions 内部已保证「先禁用租户、后翻转订阅状态」，两步任一失败都能被下轮扫描重入
	expiredTenantIDs, err := j.planSvc.ExpireSubscriptions(ctx)
	if err != nil {
		return err
	}
	if len(expiredTenantIDs) > 0 {
		j.logger.Printf("[ExpiryJob] 套餐到期，已禁用 %d 个租户", len(expiredTenantIDs))
	}

	// 提醒必须无条件执行：原写法在本轮无到期订阅时提前 return，
	// 使「7 天内到期」提醒只有在恰好有别的订阅过期的那一轮才发得出，多数轮次一条也不发
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
		tenant, err := j.tenants.GetByID(ctx, sub.TenantID)
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
