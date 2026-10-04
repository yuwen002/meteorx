// Package service 实现任务模块的业务逻辑。
// 提供任务的创建、查询、更新、完成/重开、删除、列表与统计，
// 并在个人待办与租户协作两种场景下执行归属与可见范围校验。
package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/task/dto"
	"meteorx/internal/modules/task/model"
	"meteorx/internal/modules/task/repository"
	"meteorx/internal/notify"
	"meteorx/pkg/idgen"

	"gorm.io/gorm"
)

// 业务错误定义
var (
	// ErrTaskNotFound 任务不存在或无权访问
	ErrTaskNotFound = errors.New("task not found")
	// ErrTaskForbidden 无权操作该任务
	ErrTaskForbidden = errors.New("no permission to operate this task")
	// ErrInvalidDueDate 截止时间格式不合法
	ErrInvalidDueDate = errors.New("invalid due date format")
)

// TaskServiceInterface 任务服务接口，供 handler 层依赖反转。
type TaskServiceInterface interface {
	Create(ctx context.Context, tenantID, userID string, req dto.CreateTaskReq) (*model.Task, error)
	Get(ctx context.Context, tenantID, userID, id string) (*model.Task, error)
	Update(ctx context.Context, tenantID, userID, id string, req dto.UpdateTaskReq) (*model.Task, error)
	Complete(ctx context.Context, tenantID, userID, id string) (*model.Task, error)
	Reopen(ctx context.Context, tenantID, userID, id string, status string) (*model.Task, error)
	Delete(ctx context.Context, tenantID, userID, id string) error
	List(ctx context.Context, f repository.ListFilter) ([]*model.Task, int64, error)
	Stats(ctx context.Context, f repository.ListFilter) (*repository.TaskStatusStats, error)
	// ListTrash 分页查询当前用户回收站（已软删除）任务
	ListTrash(ctx context.Context, f repository.ListFilter) ([]*model.Task, int64, error)
	// Restore 恢复已软删除任务，仅创建人（或超管）可操作
	Restore(ctx context.Context, tenantID, userID, id string) (*model.Task, error)
	// PermanentDelete 永久删除任务，仅创建人（或超管）可操作
	PermanentDelete(ctx context.Context, tenantID, userID, id string) error
	// BatchComplete 批量完成任务，返回实际生效数
	BatchComplete(ctx context.Context, tenantID, userID string, ids []string) (int, error)
	// BatchDelete 批量软删除任务，返回实际生效数
	BatchDelete(ctx context.Context, tenantID, userID string, ids []string) (int, error)
}

// TaskService 任务模块业务服务。
type TaskService struct {
	repo repository.TaskRepository // 任务仓储
}

// NewTaskService 创建任务业务服务实例。
func NewTaskService(repo repository.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

// Create 创建任务。缺省状态为 pending、优先级 normal、可见范围 personal；
// 个人任务强制负责人为创建人，租户任务可将负责人指派为其他成员。
func (s *TaskService) Create(ctx context.Context, tenantID, userID string, req dto.CreateTaskReq) (*model.Task, error) {
	now := time.Now()

	status := req.Status
	if status == "" {
		status = model.TaskStatusPending
	}
	priority := req.Priority
	if priority == "" {
		priority = model.TaskPriorityNormal
	}
	visibility := req.Visibility
	if visibility == "" {
		visibility = model.TaskVisibilityPersonal
	}

	dueDate, err := parseDueDate(req.DueDate)
	if err != nil {
		return nil, err
	}

	assignee := req.AssigneeID
	// 个人任务负责人始终为创建人本人
	if visibility == model.TaskVisibilityPersonal {
		assignee = userID
	}
	if assignee == "" {
		assignee = userID
	}

	task := &model.Task{
		ID:          idgen.New(),
		TenantID:    tenantID,
		CreatorID:   userID,
		AssigneeID:  assignee,
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		Status:      status,
		Priority:    priority,
		DueDate:     dueDate,
		Tags:        normalizeTags(req.Tags),
		Visibility:  visibility,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if status == model.TaskStatusCompleted {
		task.CompletedAt = &now
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	// 指派给他人时发送站内提醒（自己给自己的任务不提醒）
	if task.AssigneeID != task.CreatorID {
		notifyTaskAssigned(ctx, task)
	}
	return task, nil
}

// Get 查询任务详情，并校验当前用户是否有权查看。
func (s *TaskService) Get(ctx context.Context, tenantID, userID, id string) (*model.Task, error) {
	task, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canView(task, tenantID, userID) {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

// Update 更新任务字段。仅创建人或（租户任务）负责人可修改。
func (s *TaskService) Update(ctx context.Context, tenantID, userID, id string, req dto.UpdateTaskReq) (*model.Task, error) {
	task, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if task.TenantID != tenantID {
		return nil, ErrTaskNotFound
	}
	if !canEdit(task, userID) {
		return nil, ErrTaskForbidden
	}

	oldAssignee := task.AssigneeID

	if req.Title != nil {
		task.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Priority != nil && *req.Priority != "" {
		task.Priority = *req.Priority
	}
	if req.Visibility != nil && *req.Visibility != "" {
		task.Visibility = *req.Visibility
	}
	if req.AssigneeID != nil {
		task.AssigneeID = *req.AssigneeID
		if task.Visibility == model.TaskVisibilityPersonal {
			task.AssigneeID = task.CreatorID
		}
	}
	if req.Tags != nil {
		task.Tags = normalizeTags(*req.Tags)
	}
	if req.DueDate != nil {
		dueDate, err := parseDueDate(*req.DueDate)
		if err != nil {
			return nil, err
		}
		task.DueDate = dueDate
		// 截止日变化后重置提醒标记，使新截止日可再次触发到期提醒
		task.ReminderSentAt = nil
	}
	if req.Status != nil && *req.Status != "" {
		task.Status = *req.Status
		if task.Status == model.TaskStatusCompleted {
			now := time.Now()
			task.CompletedAt = &now
		} else {
			task.CompletedAt = nil
		}
	}
	task.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, err
	}
	// 负责人发生变更且指向他人时，向新负责人发送站内提醒
	if req.AssigneeID != nil && task.AssigneeID != oldAssignee && task.AssigneeID != task.CreatorID {
		notifyTaskAssigned(ctx, task)
	}
	return task, nil
}

// Complete 将任务标记为已完成。仅创建人或（租户任务）负责人可操作。
func (s *TaskService) Complete(ctx context.Context, tenantID, userID, id string) (*model.Task, error) {
	return s.changeStatus(ctx, tenantID, userID, id, model.TaskStatusCompleted)
}

// Reopen 将任务重新打开为 pending 或 in_progress。仅创建人或（租户任务）负责人可操作。
func (s *TaskService) Reopen(ctx context.Context, tenantID, userID, id string, status string) (*model.Task, error) {
	if status != model.TaskStatusInProgress {
		status = model.TaskStatusPending
	}
	return s.changeStatus(ctx, tenantID, userID, id, status)
}

// changeStatus 变更任务状态并执行归属校验。
func (s *TaskService) changeStatus(ctx context.Context, tenantID, userID, id, status string) (*model.Task, error) {
	task, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if task.TenantID != tenantID {
		return nil, ErrTaskNotFound
	}
	if !canEdit(task, userID) {
		return nil, ErrTaskForbidden
	}
	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return nil, err
	}
	return s.load(ctx, id)
}

// Delete 软删除任务。仅创建人可删除。
func (s *TaskService) Delete(ctx context.Context, tenantID, userID, id string) error {
	task, err := s.load(ctx, id)
	if err != nil {
		return err
	}
	if task.TenantID != tenantID {
		return ErrTaskNotFound
	}
	// 删除权限更严格：仅创建人（或超级管理员）可删除
	if task.CreatorID != userID && !contextx.HasRole(ctx, "superadmin") {
		return ErrTaskForbidden
	}
	return s.repo.Delete(ctx, id)
}

// List 分页查询任务列表。
func (s *TaskService) List(ctx context.Context, f repository.ListFilter) ([]*model.Task, int64, error) {
	return s.repo.List(ctx, f)
}

// Stats 统计各状态任务数量。
func (s *TaskService) Stats(ctx context.Context, f repository.ListFilter) (*repository.TaskStatusStats, error) {
	return s.repo.Stats(ctx, f)
}

// ListTrash 分页查询当前用户的回收站任务。
func (s *TaskService) ListTrash(ctx context.Context, f repository.ListFilter) ([]*model.Task, int64, error) {
	return s.repo.ListDeleted(ctx, f)
}

// Restore 恢复已软删除任务。仅创建人或超管可操作，跨租户视为不存在。
func (s *TaskService) Restore(ctx context.Context, tenantID, userID, id string) (*model.Task, error) {
	task, err := s.loadUnscoped(ctx, id)
	if err != nil {
		return nil, err
	}
	if task.TenantID != tenantID {
		return nil, ErrTaskNotFound
	}
	if task.CreatorID != userID && !contextx.HasRole(ctx, "superadmin") {
		return nil, ErrTaskForbidden
	}
	if err := s.repo.Restore(ctx, id); err != nil {
		return nil, err
	}
	return s.load(ctx, id)
}

// PermanentDelete 永久删除任务。仅创建人或超管可操作。
func (s *TaskService) PermanentDelete(ctx context.Context, tenantID, userID, id string) error {
	task, err := s.loadUnscoped(ctx, id)
	if err != nil {
		return err
	}
	if task.TenantID != tenantID {
		return ErrTaskNotFound
	}
	if task.CreatorID != userID && !contextx.HasRole(ctx, "superadmin") {
		return ErrTaskForbidden
	}
	return s.repo.PermanentDelete(ctx, id)
}

// BatchComplete 批量完成任务，逐条执行归属校验，跳过无权/不存在的项，返回成功数。
func (s *TaskService) BatchComplete(ctx context.Context, tenantID, userID string, ids []string) (int, error) {
	affected := 0
	for _, id := range ids {
		if _, err := s.changeStatus(ctx, tenantID, userID, id, model.TaskStatusCompleted); err == nil {
			affected++
		}
	}
	return affected, nil
}

// BatchDelete 批量软删除任务，逐条执行归属校验，返回成功数。
func (s *TaskService) BatchDelete(ctx context.Context, tenantID, userID string, ids []string) (int, error) {
	affected := 0
	for _, id := range ids {
		if err := s.Delete(ctx, tenantID, userID, id); err == nil {
			affected++
		}
	}
	return affected, nil
}

// SendDueReminders 扫描截止时间不晚于 horizon 且尚未提醒的未完成任务，
// 逐条向负责人（无负责人时向创建人）发送站内提醒并标记已提醒，返回实际发送数。
// 供定时提醒任务调用，不受多租户/可见范围限制（系统级扫描）。
func (s *TaskService) SendDueReminders(ctx context.Context, horizon time.Time) (int, error) {
	tasks, err := s.repo.FindDueForReminder(ctx, horizon, reminderBatchLimit)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	sent := 0
	for _, t := range tasks {
		recipient := t.AssigneeID
		if recipient == "" {
			recipient = t.CreatorID
		}
		title := "任务截止提醒"
		content := "您负责的任务「" + t.Title + "」即将到期（截止：" + formatDue(t.DueDate) + "），请及时处理。"
		if t.DueDate != nil && t.DueDate.Before(now) {
			title = "任务逾期提醒"
			content = "您负责的任务「" + t.Title + "」已逾期（截止：" + formatDue(t.DueDate) + "），请尽快处理。"
		}
		if mgr := notify.GetGlobalManager(); mgr != nil && recipient != "" {
			mgr.NotifyTaskReminder(ctx, recipient, title, content, reminderTypeDue,
				map[string]interface{}{"task_id": t.ID, "tenant_id": t.TenantID})
		}
		// 无论站内推送成否均标记已提醒，避免扫描风暴下重复打扰
		if err := s.repo.MarkReminderSent(ctx, t.ID, now); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}

// 提醒相关常量
const (
	reminderBatchLimit = 200             // 单次扫描发送提醒的最大任务数
	reminderTypeDue    = "task_due"      // 到期/逾期提醒类型标识
	reminderTypeAssign = "task_assigned" // 指派提醒类型标识
)

// formatDue 格式化截止时间供提醒正文使用。
func formatDue(t *time.Time) string {
	if t == nil {
		return "未设置"
	}
	return t.Format("2006-01-02 15:04")
}

// notifyTaskAssigned 向被指派的任务负责人发送站内提醒；
// 通知管理器未初始化时静默跳过，不影响主流程。
func notifyTaskAssigned(ctx context.Context, task *model.Task) {
	mgr := notify.GetGlobalManager()
	if mgr == nil || task.AssigneeID == "" {
		return
	}
	title := "新任务指派"
	content := "您被指派了一个新任务「" + task.Title + "」，请及时关注。"
	mgr.NotifyTaskReminder(ctx, task.AssigneeID, title, content, reminderTypeAssign,
		map[string]interface{}{"task_id": task.ID, "tenant_id": task.TenantID})
}

// loadUnscoped 按 ID 读取任务（含已软删除），将未找到错误归一化为 ErrTaskNotFound。
func (s *TaskService) loadUnscoped(ctx context.Context, id string) (*model.Task, error) {
	task, err := s.repo.GetByIDUnscoped(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return task, nil
}

// load 按 ID 读取任务，将 gorm 未找到错误归一化为 ErrTaskNotFound。
func (s *TaskService) load(ctx context.Context, id string) (*model.Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return task, nil
}

// canView 判断用户能否查看任务：租户协作任务对同租户成员开放，个人任务仅创建人可见。
func canView(task *model.Task, tenantID, userID string) bool {
	if task.TenantID != tenantID {
		return false
	}
	if task.Visibility == model.TaskVisibilityTenant {
		return true
	}
	return task.CreatorID == userID || task.AssigneeID == userID
}

// canEdit 判断用户能否编辑/完成：创建人恒可，租户协作任务的负责人亦可。
func canEdit(task *model.Task, userID string) bool {
	if task.CreatorID == userID {
		return true
	}
	return task.Visibility == model.TaskVisibilityTenant && task.AssigneeID == userID
}

// parseDueDate 宽松解析截止时间：支持空值、RFC3339、"2006-01-02 15:04:05" 与 "2006-01-02"。
func parseDueDate(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, ErrInvalidDueDate
}

// normalizeTags 去除空白标签并返回，空切片归一化为 nil。
func normalizeTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t != "" {
			result = append(result, t)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// ToResp 将任务领域模型转换为响应 DTO。
func ToResp(task *model.Task) *dto.TaskResp {
	if task == nil {
		return nil
	}
	resp := &dto.TaskResp{
		ID:          task.ID,
		TenantID:    task.TenantID,
		CreatorID:   task.CreatorID,
		AssigneeID:  task.AssigneeID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		Priority:    task.Priority,
		Tags:        task.Tags,
		Visibility:  task.Visibility,
		CreatedAt:   task.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   task.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if resp.Tags == nil {
		resp.Tags = []string{}
	}
	if task.DueDate != nil {
		resp.DueDate = task.DueDate.Format("2006-01-02 15:04:05")
	}
	if task.CompletedAt != nil {
		resp.CompletedAt = task.CompletedAt.Format("2006-01-02 15:04:05")
	}
	if task.DeletedAt != nil {
		resp.DeletedAt = task.DeletedAt.Format("2006-01-02 15:04:05")
	}
	return resp
}
