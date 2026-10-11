package wiki

import (
	"context"
	"log"
	"time"

	dbpkg "meteorx/internal/pkg/db"
	"meteorx/internal/modules/wiki/repository"
	"meteorx/internal/modules/wiki/service"

	"gorm.io/gorm"
)

// WikiTrashExpiryJob 回收站过期清理定时任务。
// 周期性扫描 expires_at 已过期的回收站项目，物理清除其关联实体（文档/节点树/空间树）并移除回收站条目。
type WikiTrashExpiryJob struct {
	svc    service.WikiService
	logger *log.Logger
}

// NewWikiTrashExpiryJob 创建回收站过期清理任务实例（内部基于 db 组装仓储与业务服务）。
func NewWikiTrashExpiryJob(db *gorm.DB) *WikiTrashExpiryJob {
	repo := repository.NewWikiRepository(db)
	txManager := dbpkg.NewTxManager(db)
	svc := service.NewWikiService(repo, txManager)
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
		j.logger.Printf("[WikiTrashExpiryJob] 过期清理失败 (耗时 %s): %v", time.Since(start), err)
		return n, err
	}
	if n > 0 {
		j.logger.Printf("[WikiTrashExpiryJob] 过期清理完成: 清理 %d 个回收站项目, 耗时 %s", n, time.Since(start))
	}
	return n, nil
}
