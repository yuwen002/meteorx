package plan

import (
	"context"
	"log"
	"strings"
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
// expiryRemindCooldown 同一订阅两次提醒的最小间隔。去重靠 subscription.expiry_reminded_at
// 这一列而不是任务内存，因此跨进程重启与多副本都只会落回「每天最多一条」
const (
	expiryRemindWindow   = 7 * 24 * time.Hour
	expiryRemindCooldown = 24 * time.Hour

	// memberLookupLimit 单个租户一次取成员 ID 的上限：扫描每 5 分钟一轮，
	// 不为超大租户一次拉全表；超出部分由邮件渠道兜住
	memberLookupLimit = 200
)

// TenantDirectory 到期提醒文案所需的租户信息查询。
// 声明为窄接口：本任务只需按 ID 取名称/邮箱，tenantRepo.TenantRepository 天然满足，
// 既避免依赖含二十余个方法的胖接口，也使定时任务可脱离数据库单测。
type TenantDirectory interface {
	GetByID(ctx context.Context, id string) (*model.Tenant, error)
}

// MemberDirectory 租户成员用户 ID 查询（窄接口，与 plan 模块已有的 UserCounter 同族）。
// WS 渠道以 claims.UserID 索引在线连接，收件人必须是用户 ID：
// 旧实现把联系人邮箱当 userID 传入，永远 miss 且不报错，提醒静默空转。
type MemberDirectory interface {
	ListUserIDs(ctx context.Context, tenantID string) ([]string, error)
}

// tenantDisabler 将 tenant 仓储的 UpdateStatus 适配为 plan service 的到期禁用回调
type tenantDisabler struct{ repo tenantRepo.TenantRepository }

func (d tenantDisabler) DisableTenant(ctx context.Context, tenantID string) error {
	return d.repo.UpdateStatus(ctx, tenantID, model.StatusDisabled)
}

// tenantMembers 将 user 仓储适配为 MemberDirectory
type tenantMembers struct{ repo userRepo.UserRepository }

func (m tenantMembers) ListUserIDs(ctx context.Context, tenantID string) ([]string, error) {
	enabled := 1 // user.status 取值：1-正常 0-禁用；被禁用的成员不应收到续费提醒
	users, _, err := m.repo.ListByTenant(ctx, tenantID, 1, memberLookupLimit, "", &enabled)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(users))
	for _, u := range users {
		if u.ID != "" {
			ids = append(ids, u.ID)
		}
	}
	return ids, nil
}

// ExpiryJob 到期自动禁用租户的任务实例
type ExpiryJob struct {
	planSvc *service.PlanService
	subRepo repository.SubscriptionRepository
	tenants TenantDirectory
	members MemberDirectory
	logger  *log.Logger
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
		planSvc: planSvc,
		subRepo: subRepo,
		tenants: tRepo,
		members: tenantMembers{repo: uRepo},
		logger:  log.Default(),
	}
}

// newExpiryJobWith 测试装配入口，与 NewExpiryJob 同构字段绑定
func newExpiryJobWith(planSvc *service.PlanService, subRepo repository.SubscriptionRepository,
	tenants TenantDirectory, members MemberDirectory) *ExpiryJob {
	return &ExpiryJob{
		planSvc: planSvc,
		subRepo: subRepo,
		tenants: tenants,
		members: members,
		logger:  log.Default(),
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

// remindExpiringSoon 扫描即将到期的生效订阅，向租户发送邮件提醒并推送给在线成员。
// 去重靠 subscription.expiry_reminded_at（由仓储按冷却期在 SQL 里过滤），
// 并且只在确实送达后才销账：旧写法丢弃发送结果并无条件记入进程内集合，
// 一次渠道抖动或一个未注册的渠道就会把该到期日的提醒永久吞掉。
func (j *ExpiryJob) remindExpiringSoon(ctx context.Context) {
	m := notify.GetGlobalManager()
	if m == nil {
		return
	}
	subs, err := j.subRepo.FindExpiringSoon(ctx, expiryRemindWindow, expiryRemindCooldown)
	if err != nil {
		j.logger.Printf("[ExpiryJob] 查询即将到期订阅失败: %v", err)
		return
	}
	var undelivered []string
	for _, sub := range subs {
		if sub.ExpiresAt == nil {
			continue
		}
		tenant, err := j.tenants.GetByID(ctx, sub.TenantID)
		if err != nil || tenant == nil {
			continue
		}
		// 成员查询失败只降级掉 WS，不能连带吞掉邮件提醒
		memberIDs, err := j.members.ListUserIDs(ctx, sub.TenantID)
		if err != nil {
			j.logger.Printf("[ExpiryJob] 查询租户 %s 成员失败，本轮仅走邮件: %v", sub.TenantID, err)
			memberIDs = nil
		}
		daysLeft := int(time.Until(*sub.ExpiresAt).Hours() / 24)
		if daysLeft < 0 {
			daysLeft = 0
		}

		results := m.NotifySubscriptionExpiry(ctx, tenant.Name, daysLeft, tenant.ContactEmail, memberIDs)
		if !expiryRemindDelivered(results) {
			// 不回写提醒时间，下一轮（5 分钟后）仍会重试
			undelivered = append(undelivered, sub.ID)
			continue
		}
		if err := j.subRepo.MarkExpiryReminded(ctx, sub.ID, time.Now()); err != nil {
			j.logger.Printf("[ExpiryJob] 回写订阅 %s 提醒时间失败（已送达，下轮会重发一次）: %v", sub.ID, err)
		}
	}
	// 未送达按轮聚合：默认部署只开了 WS，而 WS 会如实上报「收件人均不在线」，
	// 逐条打日志会让同一个订阅每 5 分钟刷一行（七天窗口下单订阅可达两千行）。
	// 聚合成一行后，订阅名单仍在但不随轮数放大。
	if len(undelivered) > 0 {
		j.logger.Printf("[ExpiryJob] %d 条到期提醒未送达任何渠道，本轮不销账（下轮重试）: %s",
			len(undelivered), strings.Join(undelivered, ", "))
	}
}

// expiryRemindDelivered 判定本轮到期提醒是否算已送达。
// 邮件是唯一能在收件人不在线时仍然可达的渠道，只要结果里出现邮件渠道就必须邮件成功；
// 否则「WS 成功 + 邮件失败」会被误销账，管理员离线时这条提醒就此消失。
// 未配置邮件时（本项目默认 email.enabled=false）退化为看是否有任一渠道成功，
// 以免每轮反复推送骚扰在线用户。
func expiryRemindDelivered(results []notify.SendResult) bool {
	var sawEmail, emailOK, anyOK bool
	for _, r := range results {
		if r.Channel == notify.ChannelEmail {
			sawEmail = true
			if r.Success {
				emailOK = true
			}
		}
		if r.Success {
			anyOK = true
		}
	}
	if sawEmail {
		return emailOK
	}
	return anyOK
}

// RunOnce 手动执行一次扫描（供测试或运维触发）
func (j *ExpiryJob) RunOnce(ctx context.Context) error {
	return j.runOnce(ctx)
}
