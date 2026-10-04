package task

import (
	"context"
	"log"
	"time"

	"meteorx/internal/modules/task/repository"
	"meteorx/internal/modules/task/service"

	"gorm.io/gorm"
)

// TaskReminderJob 任务到期提醒定时任务
// 周期性扫描截止时间进入提醒窗口（默认未来 24 小时内）或已逾期、
// 且尚未提醒过的未完成任务，向负责人推送站内提醒并标记去重。
type TaskReminderJob struct {
	svc      *service.TaskService
	horizon  time.Duration // 提醒窗口：截止日早于 now+horizon 的任务纳入提醒
	interval time.Duration // 扫描间隔
	logger   *log.Logger
}

// NewTaskReminderJob 创建任务提醒定时任务实例。
// 仅依赖任务仓储，通知通过全局 notify.Manager 发送。
func NewTaskReminderJob(db *gorm.DB) *TaskReminderJob {
	repo := repository.NewTaskRepository(db)
	return &TaskReminderJob{
		svc:      service.NewTaskService(repo),
		horizon:  24 * time.Hour,
		interval: 30 * time.Minute,
		logger:   log.Default(),
	}
}

// Start 启动定时扫描（后台 goroutine），绑定 ctx 支持优雅退出。
func (j *TaskReminderJob) Start(ctx context.Context) {
	// 启动后先立即执行一次，避免等待首个 tick
	go func() {
		if err := j.runOnce(ctx); err != nil {
			j.logger.Printf("[TaskReminderJob] 扫描失败: %v", err)
		}
		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := j.runOnce(ctx); err != nil {
					j.logger.Printf("[TaskReminderJob] 扫描失败: %v", err)
				}
			}
		}
	}()
	j.logger.Printf("[TaskReminderJob] 任务到期提醒任务已启动，间隔 %s，窗口 %s", j.interval, j.horizon)
}

// runOnce 执行一轮扫描：发送到期/逾期提醒。
func (j *TaskReminderJob) runOnce(ctx context.Context) error {
	n, err := j.svc.SendDueReminders(ctx, time.Now().Add(j.horizon))
	if err != nil {
		return err
	}
	if n > 0 {
		j.logger.Printf("[TaskReminderJob] 本轮发送 %d 条任务到期提醒", n)
	}
	return nil
}

// RunOnce 手动触发一轮扫描（供测试或运维使用）。
func (j *TaskReminderJob) RunOnce(ctx context.Context) error {
	return j.runOnce(ctx)
}
