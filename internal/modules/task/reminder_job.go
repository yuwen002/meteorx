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
// 周期性扫描截止时间进入提醒窗口或已逾期的未完成任务，
// 向负责人推送站内提醒；即将到期首次提醒一次，已逾期则按冷却期重复提醒。
// 提醒窗口/扫描间隔/冷却期/单轮上限均由配置注入，零值回落内置默认。
type TaskReminderJob struct {
	svc      *service.TaskService
	horizon  time.Duration           // 提醒窗口：截止日早于 now+horizon 的任务纳入提醒
	interval time.Duration           // 扫描间隔
	reminder service.ReminderOptions // 传递给服务层的扫描参数
	logger   *log.Logger
}

// ReminderJobOptions 任务提醒定时任务的可配置参数；零值字段回落内置默认。
type ReminderJobOptions struct {
	Horizon         time.Duration // 提醒窗口（提前量）
	Interval        time.Duration // 扫描间隔
	OverdueCooldown time.Duration // 逾期任务重复提醒的最小间隔
	BatchLimit      int           // 单次扫描发送提醒的最大任务数
}

// NewTaskReminderJob 创建任务提醒定时任务实例。
// 仅依赖任务仓储，通知通过全局 notify.Manager 发送；opts 零值项使用默认参数。
func NewTaskReminderJob(db *gorm.DB, opts ReminderJobOptions) *TaskReminderJob {
	if opts.Horizon <= 0 {
		opts.Horizon = 24 * time.Hour
	}
	if opts.Interval <= 0 {
		opts.Interval = 30 * time.Minute
	}
	if opts.OverdueCooldown <= 0 {
		opts.OverdueCooldown = 24 * time.Hour
	}
	if opts.BatchLimit <= 0 {
		opts.BatchLimit = 200
	}
	repo := repository.NewTaskRepository(db)
	return &TaskReminderJob{
		svc:      service.NewTaskService(repo),
		horizon:  opts.Horizon,
		interval: opts.Interval,
		reminder: service.ReminderOptions{OverdueCooldown: opts.OverdueCooldown, BatchLimit: opts.BatchLimit},
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
	j.logger.Printf("[TaskReminderJob] 任务到期提醒任务已启动，间隔 %s，窗口 %s，逾期冷却 %s，单轮上限 %d",
		j.interval, j.horizon, j.reminder.OverdueCooldown, j.reminder.BatchLimit)
}

// runOnce 执行一轮扫描：发送到期/逾期提醒。
func (j *TaskReminderJob) runOnce(ctx context.Context) error {
	n, err := j.svc.SendDueReminders(ctx, time.Now().Add(j.horizon), j.reminder)
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
