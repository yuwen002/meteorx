package message

import (
	"context"
	"time"

	"meteorx/internal/config"
	messagerepo "meteorx/internal/modules/message/repository"
	"meteorx/pkg/logger"

	"gorm.io/gorm"
)

// MessageExpiryJob 站内消息过期清理定时任务。
// 收件箱按「人 × 消息」写行，长期不清理会随用户量线性膨胀，因此按保留期物理删除。
type MessageExpiryJob struct {
	repo          messagerepo.MessageRepository
	retentionDays int
}

// NewMessageExpiryJob 创建站内消息过期清理任务实例。
func NewMessageExpiryJob(db *gorm.DB, cfg config.Config) *MessageExpiryJob {
	return &MessageExpiryJob{
		repo:          messagerepo.NewMessageRepository(db),
		retentionDays: cfg.Message.GetRetentionDays(),
	}
}

// Start 启动定时清理任务（后台 goroutine），interval 为扫描间隔。
// retentionDays 为负值时不启动定时器：显式关闭清理应完全无后台开销。
func (j *MessageExpiryJob) Start(ctx context.Context, interval time.Duration) {
	if j.retentionDays < 0 {
		logger.Info("[MessageExpiryJob] message.retention_days 为负值，站内消息清理已关闭")
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
					logger.Errorf("[MessageExpiryJob] 清理过期站内消息失败: %v", err)
				} else if n > 0 {
					logger.Infof("[MessageExpiryJob] 已清理 %d 条过期站内消息", n)
				}
			}
		}
	}()
	logger.Infof("[MessageExpiryJob] 站内消息清理任务已启动，保留 %d 天，间隔 %s", j.retentionDays, interval)
}

// runOnce 执行一轮清理，返回删除条数。
func (j *MessageExpiryJob) runOnce(ctx context.Context) (int64, error) {
	cut := time.Now().AddDate(0, 0, -j.retentionDays)
	return j.repo.DeleteOlderThan(ctx, cut)
}

// RunOnce 手动执行一轮清理（供测试或运维触发）。
func (j *MessageExpiryJob) RunOnce(ctx context.Context) (int64, error) {
	return j.runOnce(ctx)
}
