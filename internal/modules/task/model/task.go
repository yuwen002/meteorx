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
	Tags        []string   // 分类标签
	Visibility  string     // 可见范围：personal / tenant
	CompletedAt *time.Time // 完成时间，仅 completed 状态有值
	// ReminderSentAt 到期提醒已发送时间，为空表示尚未提醒；
	// 定时任务据此对同一任务只提醒一次，修改截止日时会被重置
	ReminderSentAt *time.Time // 到期提醒发送时间
	CreatedAt      time.Time  // 创建时间
	UpdatedAt      time.Time  // 更新时间
	DeletedAt      *time.Time // 软删除时间，为空表示未删除
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
