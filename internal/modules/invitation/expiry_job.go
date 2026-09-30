package invitation

import (
	"context"
	"log"
	"time"

	"meteorx/internal/config"
	"meteorx/internal/modules/invitation/repository"
	"meteorx/internal/modules/invitation/service"

	"gorm.io/gorm"
)

// InvitationExpiryJob 邀请过期清理定时任务
// 周期性扫描超过有效期仍为 pending 的邀请，将其状态置为 expired
type InvitationExpiryJob struct {
	svc    *service.InvitationService
	logger *log.Logger
}

// NewInvitationExpiryJob 创建邀请过期任务实例
// 仅依赖邀请仓储，其余仓储/邮件配置对过期扫描无意义，故传空配置
func NewInvitationExpiryJob(db *gorm.DB) *InvitationExpiryJob {
	invRepo := repository.NewInvitationRepository(db)
	svc := service.NewInvitationService(
		invRepo, nil, nil, nil, nil,
		config.EmailConfig{}, config.ClientConfig{}, config.SecurityConfig{},
	)
	return &InvitationExpiryJob{
		svc:    svc,
		logger: log.Default(),
	}
}

// Start 启动定时任务（后台 goroutine），interval 为扫描间隔
func (j *InvitationExpiryJob) Start(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := j.runOnce(ctx); err != nil {
					j.logger.Printf("[InvitationExpiryJob] 扫描失败: %v", err)
				}
			}
		}
	}()
	j.logger.Printf("[InvitationExpiryJob] 邀请过期清理任务已启动，间隔 %s", interval)
}

// runOnce 执行一次扫描：将过期邀请置为 expired
func (j *InvitationExpiryJob) runOnce(ctx context.Context) error {
	n, err := j.svc.ExpireOutdatedInvitations(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		j.logger.Printf("[InvitationExpiryJob] 已将 %d 个过期邀请置为 expired", n)
	}
	return nil
}

// RunOnce 手动执行一次扫描（供测试或运维触发）
func (j *InvitationExpiryJob) RunOnce(ctx context.Context) error {
	return j.runOnce(ctx)
}
