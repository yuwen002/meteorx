// Package dto 定义任务模块的请求/响应数据传输对象。
package dto

// CreateTaskReq 创建任务请求。
// 个人待办与租户协作任务共用此结构，通过 visibility 区分。
type CreateTaskReq struct {
	Title        string   `json:"title" validate:"required,max=200" label:"标题"` // 任务标题
	Description  string   `json:"description" validate:"max=2000" label:"描述"`   // 任务描述
	Status       string   `json:"status" validate:"omitempty,oneof=pending in_progress completed" label:"状态"`
	Priority     string   `json:"priority" validate:"omitempty,oneof=low normal high urgent" label:"优先级"`
	DueDate      string   `json:"due_date" validate:"omitempty" label:"截止时间"`              // 支持 RFC3339 或 "2006-01-02 15:04:05"
	RemindBefore string   `json:"remind_before" validate:"omitempty,max=16" label:"提醒提前量"` // Go Duration 格式如 2h/30m，留空使用全局默认
	Tags         []string `json:"tags" validate:"omitempty,max=10" label:"标签"`
	Visibility   string   `json:"visibility" validate:"omitempty,oneof=personal tenant" label:"可见范围"`
	AssigneeID   string   `json:"assignee_id" validate:"omitempty,max=26" label:"负责人"`
}

// UpdateTaskReq 更新任务请求。仅提交需要修改的字段，指针/空值语义表示"不改动"。
type UpdateTaskReq struct {
	Title        *string   `json:"title" validate:"omitempty,max=200" label:"标题"`
	Description  *string   `json:"description" validate:"omitempty,max=2000" label:"描述"`
	Status       *string   `json:"status" validate:"omitempty,oneof=pending in_progress completed" label:"状态"`
	Priority     *string   `json:"priority" validate:"omitempty,oneof=low normal high urgent" label:"优先级"`
	DueDate      *string   `json:"due_date" label:"截止时间"`                                   // 传空字符串表示清除截止时间
	RemindBefore *string   `json:"remind_before" validate:"omitempty,max=16" label:"提醒提前量"` // 传空字符串表示恢复使用全局默认
	Tags         *[]string `json:"tags" validate:"omitempty,max=10" label:"标签"`
	Visibility   *string   `json:"visibility" validate:"omitempty,oneof=personal tenant" label:"可见范围"`
	AssigneeID   *string   `json:"assignee_id" validate:"omitempty,max=26" label:"负责人"`
}

// TaskResp 任务响应，返回给前端的任务详情。
type TaskResp struct {
	ID           string   `json:"id"`
	TenantID     string   `json:"tenant_id"`
	CreatorID    string   `json:"creator_id"`
	CreatorName  string   `json:"creator_name,omitempty"` // 创建人展示名（昵称优先，回退用户名）
	AssigneeID   string   `json:"assignee_id"`
	AssigneeName string   `json:"assignee_name,omitempty"` // 负责人展示名（昵称优先，回退用户名）
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Status       string   `json:"status"`
	Priority     string   `json:"priority"`
	DueDate      string   `json:"due_date,omitempty"`
	RemindBefore string   `json:"remind_before,omitempty"` // 任务级提醒提前量（为空表示使用全局默认）
	Tags         []string `json:"tags"`
	Visibility   string   `json:"visibility"`
	StartedAt    string   `json:"started_at,omitempty"`
	CompletedAt  string   `json:"completed_at,omitempty"`
	DeletedAt    string   `json:"deleted_at,omitempty"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

// TaskStatsResp 任务统计响应。
type TaskStatsResp struct {
	Pending    int64 `json:"pending"`     // 待办数量
	InProgress int64 `json:"in_progress"` // 进行中数量
	Completed  int64 `json:"completed"`   // 已完成数量
	Overdue    int64 `json:"overdue"`     // 逾期数量
	Total      int64 `json:"total"`       // 总数量
}

// BatchTaskReq 批量操作任务请求。
type BatchTaskReq struct {
	IDs []string `json:"ids" validate:"required,min=1,max=100" label:"任务ID列表"`
}

// BatchStatusReq 批量修改状态请求。
type BatchStatusReq struct {
	IDs    []string `json:"ids" validate:"required,min=1,max=100" label:"任务ID列表"`
	Status string   `json:"status" validate:"required,oneof=pending in_progress completed" label:"状态"`
}

// BatchAssignReq 批量指派负责人请求。
type BatchAssignReq struct {
	IDs        []string `json:"ids" validate:"required,min=1,max=100" label:"任务ID列表"`
	AssigneeID string   `json:"assignee_id" validate:"required,max=26" label:"负责人"`
}

// BatchTaskResp 批量操作任务响应。
type BatchTaskResp struct {
	Affected int `json:"affected"` // 实际生效的任务数
}
