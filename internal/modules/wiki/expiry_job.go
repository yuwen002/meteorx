package wiki

import (
	"context"
	"fmt"
	"log"
	"time"

	dbpkg "meteorx/internal/pkg/db"
	"meteorx/internal/modules/wiki/repository"
	"meteorx/internal/modules/wiki/service"
	"meteorx/internal/notify"

	"gorm.io/gorm"
)

// WikiTrashExpiryJob 回收站过期清理定时任务。
// 周期性扫描 expires_at 已过期的回收站项目，物理清除其关联实体（文档/节点树/空间树）并移除回收站条目。
type WikiTrashExpiryJob struct {
	svc    service.WikiService
	logger *log.Logger
}

// NewWikiTrashExpiryJob 创建回收站过期清理任务实例（内部基于 db 组装仓储与业务服务）。
// opts 用于注入跨副本分布式锁等可选依赖（如 service.WithExpireCacheLock(rdb)），不传时仅进程内互斥。
func NewWikiTrashExpiryJob(db *gorm.DB, opts ...service.Option) *WikiTrashExpiryJob {
	repo := repository.NewWikiRepository(db)
	txManager := dbpkg.NewTxManager(db)
	svc := service.NewWikiService(repo, txManager, opts...)
	return &WikiTrashExpiryJob{
		svc:    svc,
		logger: log.Default(),
	}
}

// Start 启动后台定时扫描（绑定 ctx，应用关闭时随 ctx 取消）。
func (j *WikiTrashExpiryJob) Start(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// RunOnce 内部已包含成功/失败日志，此处无需重复记录
				_, _ = j.RunOnce(ctx)
			}
		}
	}()
	j.logger.Printf("[WikiTrashExpiryJob] 回收站过期清理任务已启动，间隔 %s", interval)
}

// runOnce 执行一次过期清理（委托业务服务跨租户全局扫描），返回清理数量。
func (j *WikiTrashExpiryJob) runOnce(ctx context.Context) (int, error) {
	return j.svc.ExpireTrashItems(ctx)
}

// RunOnce 手动触发一次过期清理（供运维或测试调用）。
// 成功时记录清理数量与耗时；失败时记录错误（作为告警信号）。
func (j *WikiTrashExpiryJob) RunOnce(ctx context.Context) (int, error) {
	start := time.Now()
	n, err := j.runOnce(ctx)
	if err != nil {
		dur := time.Since(start)
		j.logger.Printf("[WikiTrashExpiryJob] 过期清理失败 (耗时 %s): %v", dur, err)
		// 接入真实告警通道：若已初始化全局通知管理器，则推送高优先级运维告警；
		// 未配置任何渠道（email/webhook/ws）时为 no-op，不影响定时任务主流程。
		// 不传 UserIDs：运维自检告警没有「应该收到它的账号」，落收件箱无所指向。
		if gm := notify.GetGlobalManager(); gm != nil {
			gm.NotifyAlert(ctx, notify.AlertNotice{
				RuleName: "回收站过期清理失败",
				Content:  fmt.Sprintf("Wiki 回收站过期清理定时任务执行失败（耗时 %s）：%v", dur, err),
				Priority: notify.PriorityHigh,
			})
		}
		return n, err
	}
	if n > 0 {
		j.logger.Printf("[WikiTrashExpiryJob] 过期清理完成: 清理 %d 个回收站项目, 耗时 %s", n, time.Since(start))
	}
	return n, nil
}
