package export

import (
	"context"
	"time"

	"meteorx/internal/config"
	"meteorx/internal/modules/export/repository"
	"meteorx/internal/modules/export/service"
	"meteorx/pkg/logger"

	"gorm.io/gorm"
)

// expiryBatchLimit 单轮清理的最大过期任务数
const expiryBatchLimit = 200

// ExportExpiryJob 导出文件过期清理定时任务。
// 周期性扫描超过 expires_at 的导出任务，删除其 CSV 文件与数据库记录。
type ExportExpiryJob struct {
	svc *service.ExportService
}

// NewExportExpiryJob 创建导出过期清理任务实例
func NewExportExpiryJob(db *gorm.DB, cfg config.Config) *ExportExpiryJob {
	repo := repository.NewExportRepository(db)
	svc := service.NewExportService(repo, buildServiceConfig(cfg))
	return &ExportExpiryJob{svc: svc}
}

// Start 启动定时清理任务（后台 goroutine），interval 为扫描间隔
func (j *ExportExpiryJob) Start(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if n, err := j.runOnce(ctx); err != nil {
					logger.Errorf("[ExportExpiryJob] 清理过期导出任务失败: %v", err)
				} else if n > 0 {
					logger.Infof("[ExportExpiryJob] 已清理 %d 个过期导出任务", n)
				}
			}
		}
	}()
	logger.Infof("[ExportExpiryJob] 导出过期清理任务已启动，间隔 %s", interval)
}

// runOnce 执行一轮清理，返回清理条数
func (j *ExportExpiryJob) runOnce(ctx context.Context) (int64, error) {
	return j.svc.CleanupExpired(ctx, time.Now(), expiryBatchLimit)
}

// RunOnce 手动执行一轮清理（供测试或运维触发）
func (j *ExportExpiryJob) RunOnce(ctx context.Context) (int64, error) {
	return j.runOnce(ctx)
}
