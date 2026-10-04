// Package repository 定义任务模块的仓储接口与查询过滤条件。
package repository

import (
	"context"
	"time"

	"meteorx/internal/modules/task/model"
)

// ListFilter 任务列表查询过滤条件。
// 通过 TenantID + UserID + Visibility 组合实现"个人待办"与"租户协作"的统一查询。
type ListFilter struct {
	TenantID   string // 租户 ID（必填，多租户隔离）
	UserID     string // 当前用户 ID（用于个人任务归属与"全部"视图）
	Visibility string // 可见范围筛选：空=全部可见，personal=仅个人，tenant=仅租户协作
	Status     string // 状态筛选：空=不限
	Priority   string // 优先级筛选：空=不限
	AssigneeID string // 负责人筛选：空=不限
	Keyword    string // 关键词（标题/描述模糊匹配）
	Page       int    // 页码
	PageSize   int    // 每页条数
	SortBy     string // 排序字段：created_at/due_date/priority/updated_at
	SortOrder  string // 排序方向：asc/desc
}

// TaskStatusStats 按状态统计的任务数量。
type TaskStatusStats struct {
	Pending    int64 // 待办数量
	InProgress int64 // 进行中数量
	Completed  int64 // 已完成数量
	Overdue    int64 // 已逾期（未完成且超过截止时间）数量
	Total      int64 // 总数量
	// AvgHandleSeconds 已完成任务平均处理耗时（秒），基于 started_at→completed_at，
	// 仅统计两者均有值的已完成任务；无可统计项时为 0。
	AvgHandleSeconds int64
}

// TaskRepository 任务仓储接口，定义任务数据的持久化操作。
type TaskRepository interface {
	// Create 新建任务记录
	Create(ctx context.Context, task *model.Task) error
	// GetByID 按主键查询任务（未删除）
	GetByID(ctx context.Context, id string) (*model.Task, error)
	// Update 更新任务的可变字段（标题/描述/状态/优先级/截止日/标签/负责人）
	Update(ctx context.Context, task *model.Task) error
	// Delete 软删除任务
	Delete(ctx context.Context, id string) error
	// GetByIDUnscoped 按主键查询任务（包含已软删除记录，用于恢复/永久删除场景）
	GetByIDUnscoped(ctx context.Context, id string) (*model.Task, error)
	// ListDeleted 分页查询当前用户的回收站（已软删除）任务
	ListDeleted(ctx context.Context, f ListFilter) ([]*model.Task, int64, error)
	// Restore 恢复已软删除的任务
	Restore(ctx context.Context, id string) error
	// PermanentDelete 永久删除任务（物理删除，不可恢复）
	PermanentDelete(ctx context.Context, id string) error
	// FindDueForReminder 查询需要发送到期/逾期提醒的未完成任务（供定时提醒任务使用）：
	//  - 提醒到点：任务自带 remind_before 时以 now+remind_before 为阈值，否则使用全局 horizon；
	//  - 尚未提醒过（reminder_sent_at 为空）；或已逾期（due_date < now）且距上次提醒已超过冷却期（reminder_sent_at <= lastReminderCutoff），实现逾期重复提醒。
	FindDueForReminder(ctx context.Context, horizon, now, lastReminderCutoff time.Time, limit int) ([]*model.Task, error)
	// MarkReminderSent 标记任务已发送到期提醒，避免重复提醒
	MarkReminderSent(ctx context.Context, id string, t time.Time) error
	// List 按过滤条件分页查询任务列表，返回任务切片与总数
	List(ctx context.Context, f ListFilter) ([]*model.Task, int64, error)
	// Stats 统计当前用户在指定可见范围下各状态任务数量
	Stats(ctx context.Context, f ListFilter) (*TaskStatusStats, error)
}
