// Package model 定义任务（Task）模块的领域模型。
// 任务同时支持"个人待办"与"租户团队协作"两种场景，
// 通过 Visibility 字段区分可见范围，统一存储于同一张表。
package model

import "time"

// 任务状态常量
const (
	// TaskStatusPending 待办，任务刚创建尚未开始
	TaskStatusPending = "pending"
	// TaskStatusInProgress 进行中，任务已开始处理
	TaskStatusInProgress = "in_progress"
	// TaskStatusCompleted 已完成，任务被标记完成
	TaskStatusCompleted = "completed"
)

// 任务优先级常量
const (
	// TaskPriorityLow 低优先级
	TaskPriorityLow = "low"
	// TaskPriorityNormal 普通优先级（默认）
	TaskPriorityNormal = "normal"
	// TaskPriorityHigh 高优先级
	TaskPriorityHigh = "high"
	// TaskPriorityUrgent 紧急
	TaskPriorityUrgent = "urgent"
)

// 任务可见范围常量
const (
	// TaskVisibilityPersonal 个人待办，仅创建人可见可管理
	TaskVisibilityPersonal = "personal"
	// TaskVisibilityTenant 租户协作，租户内成员共享可见、可指派
	TaskVisibilityTenant = "tenant"
)

// 任务重复周期常量（空字符串表示不重复）
const (
	// TaskRecurrenceDaily 每日重复
	TaskRecurrenceDaily = "daily"
	// TaskRecurrenceWeekly 每周重复
	TaskRecurrenceWeekly = "weekly"
	// TaskRecurrenceMonthly 每月重复
	TaskRecurrenceMonthly = "monthly"
)

// Task 任务领域模型。
// 记录一条待办/协作任务的完整信息：内容、状态、优先级、截止日、标签、归属与可见范围。
type Task struct {
	ID          string     // 任务唯一标识（ULID）
	TenantID    string     // 所属租户 ID（多租户隔离）
	CreatorID   string     // 创建人用户 ID
	AssigneeID  string     // 负责人用户 ID，为空表示由创建人自己负责
	Title       string     // 任务标题
	Description string     // 任务详细描述
	Status      string     // 状态：pending / in_progress / completed
	Priority    string     // 优先级：low / normal / high / urgent
	DueDate     *time.Time // 截止时间，可为空
	// RemindBefore 任务级自定义提醒提前量：非空时覆盖全局提醒窗口（horizon），
	// 定时扫描在 截止日 - RemindBefore 到点时触发提醒；为空则回落配置默认。
	RemindBefore *time.Duration
	Tags         []string // 分类标签
	Visibility   string   // 可见范围：personal / tenant
	// Recurrence 重复周期：空表示不重复；daily/weekly/monthly 时，
	// 完成该任务会自动按周期推进截止日生成下一条 pending 任务。
	Recurrence string
	// StartedAt 首次进入 in_progress 时自动记录，一旦开始不清空（退回 pending/完成均保留），
	// 用于计算处理耗时（完成-开始）
	StartedAt   *time.Time // 开始时间
	CompletedAt *time.Time // 完成时间，仅 completed 状态有值
	// ReminderSentAt 到期提醒已发送时间，为空表示尚未提醒；
	// 定时任务据此对同一任务只提醒一次，修改截止日时会被重置
	ReminderSentAt *time.Time // 到期提醒发送时间
	CreatedAt      time.Time  // 创建时间
	UpdatedAt      time.Time  // 更新时间
	DeletedAt      *time.Time // 软删除时间，为空表示未删除

	// CreatorName/AssigneeName 为非持久化的展示字段，仅在读取响应时由服务层
	// 依据用户目录批量富化，不参与仓储读写。
	CreatorName  string // 创建人展示名（富化字段，不持久化）
	AssigneeName string // 负责人展示名（富化字段，不持久化）
}

// IsValidStatus 判断给定字符串是否为合法的任务状态。
func IsValidStatus(s string) bool {
	switch s {
	case TaskStatusPending, TaskStatusInProgress, TaskStatusCompleted:
		return true
	}
	return false
}

// IsValidPriority 判断给定字符串是否为合法的优先级。
func IsValidPriority(p string) bool {
	switch p {
	case TaskPriorityLow, TaskPriorityNormal, TaskPriorityHigh, TaskPriorityUrgent:
		return true
	}
	return false
}

// IsValidVisibility 判断给定字符串是否为合法的可见范围。
func IsValidVisibility(v string) bool {
	switch v {
	case TaskVisibilityPersonal, TaskVisibilityTenant:
		return true
	}
	return false
}

// IsValidRecurrence 判断给定字符串是否为合法的重复周期（空字符串表示不重复，亦视为合法）。
func IsValidRecurrence(r string) bool {
	switch r {
	case "", TaskRecurrenceDaily, TaskRecurrenceWeekly, TaskRecurrenceMonthly:
		return true
	}
	return false
}

// NextOccurrence 依据重复周期从 base 时间推算下一次截止/参考时间；
// 周期非法或为空时返回 false，调用方据此不生成下一实例。
func NextOccurrence(recurrence string, base time.Time) (time.Time, bool) {
	switch recurrence {
	case TaskRecurrenceDaily:
		return base.AddDate(0, 0, 1), true
	case TaskRecurrenceWeekly:
		return base.AddDate(0, 0, 7), true
	case TaskRecurrenceMonthly:
		return base.AddDate(0, 1, 0), true
	default:
		return base, false
	}
}
