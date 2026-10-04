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
	// ErrInvalidRemindLead 提醒提前量不合法（无法解析或超出允许范围）
	ErrInvalidRemindLead = errors.New("invalid remind lead time")
	// ErrAssigneeInvalid 负责人不存在或不属于当前租户
	ErrAssigneeInvalid = errors.New("assignee is not a member of this tenant")
	// ErrInvalidRecurrence 重复周期不合法
	ErrInvalidRecurrence = errors.New("invalid recurrence value")
	// ErrRecurrenceNeedsDue 设置了重复周期但未提供截止时间（无法推算下一周期）
	ErrRecurrenceNeedsDue = errors.New("recurring task requires a due date")
	// ErrInvalidStatus 任务状态不合法
	ErrInvalidStatus = errors.New("invalid task status")
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
	// BatchUpdateStatus 批量修改任务状态，逐条按归属校验，返回实际生效数
	BatchUpdateStatus(ctx context.Context, tenantID, userID string, ids []string, status string) (int, error)
	// BatchAssign 批量指派负责人，逐条按归属校验并通知新负责人，返回实际生效数
	BatchAssign(ctx context.Context, tenantID, userID string, ids []string, assigneeID string) (int, error)
}

// UserNameResolver 按用户 ID 批量解析展示名，供任务响应富化创建人/负责人姓名。
// 由仓储层基于用户表实现，服务层通过依赖注入使用，避免直接耦合用户模块。
type UserNameResolver interface {
	ResolveUserNames(ctx context.Context, ids []string) (map[string]string, error)
}

// TenantMemberChecker 校验某用户是否为指定租户的成员，用于指派前拦截不存在/跨租户的负责人。
// 由仓储层（UserDirectory）实现，服务层可选依赖：未实现时跳过校验，保持向后兼容。
type TenantMemberChecker interface {
	IsTenantMember(ctx context.Context, tenantID, userID string) (bool, error)
}

// TaskService 任务模块业务服务。
type TaskService struct {
	repo    repository.TaskRepository // 任务仓储
	names   UserNameResolver          // 可选的用户姓名解析器，为 nil 时不富化展示名
	members TenantMemberChecker       // 可选的租户成员校验器，为 nil 时不校验负责人归属
}

// NewTaskService 创建任务业务服务实例（不富化人员姓名、不校验负责人归属）。
func NewTaskService(repo repository.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

// NewTaskServiceWithNames 创建带人员姓名富化能力的任务业务服务实例。
// names 为 nil 时行为等同 NewTaskService，响应中人员姓名为空。
// 若 names 同时实现 TenantMemberChecker，则自动启用负责人同租户归属校验。
func NewTaskServiceWithNames(repo repository.TaskRepository, names UserNameResolver) *TaskService {
	s := &TaskService{repo: repo, names: names}
	if checker, ok := names.(TenantMemberChecker); ok {
		s.members = checker
	}
	return s
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
	recurrence := strings.TrimSpace(req.Recurrence)
	if !model.IsValidRecurrence(recurrence) {
		return nil, ErrInvalidRecurrence
	}

	dueDate, err := parseDueDate(req.DueDate)
	if err != nil {
		return nil, err
	}
	remindBefore, err := parseRemindLead(req.RemindBefore)
	if err != nil {
		return nil, err
	}

	if recurrence != "" && dueDate == nil {
		return nil, ErrRecurrenceNeedsDue
	}

	assignee := req.AssigneeID
	// 个人任务负责人始终为创建人本人
	if visibility == model.TaskVisibilityPersonal {
		assignee = userID
	}
	if assignee == "" {
		assignee = userID
	}
	if err := s.validateAssignee(ctx, tenantID, userID, assignee); err != nil {
		return nil, err
	}

	task := &model.Task{
		ID:           idgen.New(),
		TenantID:     tenantID,
		CreatorID:    userID,
		AssigneeID:   assignee,
		Title:        strings.TrimSpace(req.Title),
		Description:  req.Description,
		Status:       status,
		Priority:     priority,
		DueDate:      dueDate,
		RemindBefore: remindBefore,
		Tags:         normalizeTags(req.Tags),
		Visibility:   visibility,
		Recurrence:   recurrence,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	applyStatusChange(task, status, now)

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	// 指派给他人时发送站内提醒（自己给自己的任务不提醒）
	if task.AssigneeID != task.CreatorID {
		notifyTaskAssigned(ctx, task)
	}
	s.enrichNames(ctx, task)
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
	s.enrichNames(ctx, task)
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
	wasCompleted := task.Status == model.TaskStatusCompleted

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
	if req.Recurrence != nil {
		r := strings.TrimSpace(*req.Recurrence)
		if !model.IsValidRecurrence(r) {
			return nil, ErrInvalidRecurrence
		}
		task.Recurrence = r
	}
	if req.AssigneeID != nil {
		task.AssigneeID = *req.AssigneeID
		if task.Visibility == model.TaskVisibilityPersonal {
			task.AssigneeID = task.CreatorID
		}
		if err := s.validateAssignee(ctx, tenantID, task.CreatorID, task.AssigneeID); err != nil {
			return nil, err
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
	if req.RemindBefore != nil {
		remindBefore, err := parseRemindLead(*req.RemindBefore)
		if err != nil {
			return nil, err
		}
		task.RemindBefore = remindBefore
		// 提前量变化后重置提醒标记，使新提醒窗口可重新评估
		task.ReminderSentAt = nil
	}
	if req.Status != nil && *req.Status != "" {
		applyStatusChange(task, *req.Status, time.Now())
	}
	if task.Recurrence != "" && task.DueDate == nil {
		return nil, ErrRecurrenceNeedsDue
	}
	task.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, err
	}
	// 本次从非完成首次转为完成，且任务设了重复周期时，自动生成下一周期实例
	if task.Status == model.TaskStatusCompleted && !wasCompleted {
		s.spawnNextOccurrence(ctx, task)
	}
	// 负责人发生变更且指向他人时，向新负责人发送站内提醒
	if req.AssigneeID != nil && task.AssigneeID != oldAssignee && task.AssigneeID != task.CreatorID {
		notifyTaskAssigned(ctx, task)
	}
	s.enrichNames(ctx, task)
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
	now := time.Now()
	wasCompleted := task.Status == model.TaskStatusCompleted
	applyStatusChange(task, status, now)
	task.UpdatedAt = now
	if err := s.repo.Update(ctx, task); err != nil {
		return nil, err
	}
	// 首次从非完成转为完成时，若任务设了重复周期则自动生成下一周期实例
	if status == model.TaskStatusCompleted && !wasCompleted {
		s.spawnNextOccurrence(ctx, task)
	}
	s.enrichNames(ctx, task)
	return task, nil
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

// List 分页查询任务列表，并对结果富化人员展示名。
func (s *TaskService) List(ctx context.Context, f repository.ListFilter) ([]*model.Task, int64, error) {
	tasks, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	s.enrichNames(ctx, tasks...)
	return tasks, total, nil
}

// Stats 统计各状态任务数量。
func (s *TaskService) Stats(ctx context.Context, f repository.ListFilter) (*repository.TaskStatusStats, error) {
	return s.repo.Stats(ctx, f)
}

// ListTrash 分页查询当前用户的回收站任务，并对结果富化人员展示名。
func (s *TaskService) ListTrash(ctx context.Context, f repository.ListFilter) ([]*model.Task, int64, error) {
	tasks, total, err := s.repo.ListDeleted(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	s.enrichNames(ctx, tasks...)
	return tasks, total, nil
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
	task, err = s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	s.enrichNames(ctx, task)
	return task, nil
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

// BatchUpdateStatus 批量修改任务状态，逐条执行归属校验，跳过无权/不存在的项，返回成功数。
// status 必须为合法状态值，否则返回 ErrInvalidStatus。
func (s *TaskService) BatchUpdateStatus(ctx context.Context, tenantID, userID string, ids []string, status string) (int, error) {
	if !model.IsValidStatus(status) {
		return 0, ErrInvalidStatus
	}
	affected := 0
	for _, id := range ids {
		if _, err := s.changeStatus(ctx, tenantID, userID, id, status); err == nil {
			affected++
		}
	}
	return affected, nil
}

// BatchAssign 批量指派负责人，逐条复用更新逻辑（含归属校验与对新负责人的站内提醒），
// 跳过无权/不存在的项，返回成功数。个人任务会被回写为创建人，仍计入成功。
func (s *TaskService) BatchAssign(ctx context.Context, tenantID, userID string, ids []string, assigneeID string) (int, error) {
	// 先校验新负责人为同租户成员，避免逐项 Update 静默吞错导致徒劳批量
	if err := s.validateAssignee(ctx, tenantID, userID, assigneeID); err != nil {
		return 0, err
	}
	affected := 0
	aid := assigneeID
	for _, id := range ids {
		if _, err := s.Update(ctx, tenantID, userID, id, dto.UpdateTaskReq{AssigneeID: &aid}); err == nil {
			affected++
		}
	}
	return affected, nil
}

// ReminderOptions 到期提醒扫描的可配置参数；零值字段回落内置默认。
// 由定时提醒任务从应用配置注入，使逾期重复提醒冷却期与单轮扫描上限可外部调整。
type ReminderOptions struct {
	OverdueCooldown time.Duration // 逾期任务两次重复提醒之间的最小间隔
	BatchLimit      int           // 单次扫描发送提醒的最大任务数
}

// SendDueReminders 扫描需要提醒的未完成任务（即将到期/首次逾期、以及距上次提醒超过冷却期的已逾期任务），
// 逐条向负责人（无负责人时向创建人）发送站内提醒并刷新提醒时间，返回实际发送数。
// 供定时提醒任务调用，不受多租户/可见范围限制（系统级扫描）。
// opts 为可选扫描参数（冷却期/单轮上限），缺省时使用内置默认值。
func (s *TaskService) SendDueReminders(ctx context.Context, horizon time.Time, opts ...ReminderOptions) (int, error) {
	cooldown, limit := defaultReminderOverdueCooldown, defaultReminderBatchLimit
	if len(opts) > 0 {
		if opts[0].OverdueCooldown > 0 {
			cooldown = opts[0].OverdueCooldown
		}
		if opts[0].BatchLimit > 0 {
			limit = opts[0].BatchLimit
		}
	}
	now := time.Now()
	cutoff := now.Add(-cooldown)
	tasks, err := s.repo.FindDueForReminder(ctx, horizon, now, cutoff, limit)
	if err != nil {
		return 0, err
	}
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
		// 无论站内推送成否均刷新提醒时间，作为下次重复提醒的冷却基准
		if err := s.repo.MarkReminderSent(ctx, t.ID, now); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}

// 提醒相关类型标识与默认参数
const (
	reminderTypeDue    = "task_due"      // 到期/逾期提醒类型标识
	reminderTypeAssign = "task_assigned" // 指派提醒类型标识
	// 逾期重复提醒冷却期与单轮扫描上限的内置默认值，可被 ReminderOptions 覆盖
	defaultReminderBatchLimit      = 200
	defaultReminderOverdueCooldown = 24 * time.Hour
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

// applyStatusChange 统一维护状态变更相关时间戳，供 Create/Update/状态切换复用：
// 首次进入 in_progress 记录开始时间（一旦开始不再清空，用于计算处理耗时）；
// 进入 completed 记录完成时间，离开 completed 时清空完成时间。
func applyStatusChange(task *model.Task, status string, now time.Time) {
	task.Status = status
	if status == model.TaskStatusInProgress && task.StartedAt == nil {
		task.StartedAt = &now
	}
	if status == model.TaskStatusCompleted {
		task.CompletedAt = &now
	} else {
		task.CompletedAt = nil
	}
}

// enrichNames 依据创建人/负责人 ID 批量解析展示名并写回任务的富化字段。
// 未注入解析器或解析失败时静默跳过，不影响主流程。
func (s *TaskService) enrichNames(ctx context.Context, tasks ...*model.Task) {
	if s.names == nil || len(tasks) == 0 {
		return
	}
	idSet := make(map[string]struct{}, len(tasks)*2)
	for _, t := range tasks {
		if t.CreatorID != "" {
			idSet[t.CreatorID] = struct{}{}
		}
		if t.AssigneeID != "" {
			idSet[t.AssigneeID] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	nameMap, err := s.names.ResolveUserNames(ctx, ids)
	if err != nil || nameMap == nil {
		return
	}
	for _, t := range tasks {
		t.CreatorName = nameMap[t.CreatorID]
		t.AssigneeName = nameMap[t.AssigneeID]
	}
}

// validateAssignee 校验负责人为同租户成员：checker 未注入、负责人为空或即创建人时跳过；
// 非成员返回 ErrAssigneeInvalid，校验查询失败则向上抛出错误（避免误放行脏数据）。
func (s *TaskService) validateAssignee(ctx context.Context, tenantID, creatorID, assignee string) error {
	if s.members == nil || assignee == "" || assignee == creatorID {
		return nil
	}
	ok, err := s.members.IsTenantMember(ctx, tenantID, assignee)
	if err != nil {
		return err
	}
	if !ok {
		return ErrAssigneeInvalid
	}
	return nil
}

// spawnNextOccurrence 为设置了重复周期的任务生成下一周期实例：
// 复制标题/描述/优先级/负责人/可见范围/提醒提前量/标签，重置为 pending 并按周期从原截止日推进。
// 完成已落库后才调用，生成失败不阻断本次完成（静默跳过）。
func (s *TaskService) spawnNextOccurrence(ctx context.Context, task *model.Task) {
	if task.Recurrence == "" || task.DueDate == nil {
		return
	}
	nextDue, ok := model.NextOccurrence(task.Recurrence, *task.DueDate)
	if !ok {
		return
	}
	now := time.Now()
	due := nextDue
	next := &model.Task{
		ID:           idgen.New(),
		TenantID:     task.TenantID,
		CreatorID:    task.CreatorID,
		AssigneeID:   task.AssigneeID,
		Title:        task.Title,
		Description:  task.Description,
		Status:       model.TaskStatusPending,
		Priority:     task.Priority,
		DueDate:      &due,
		RemindBefore: task.RemindBefore,
		Tags:         append([]string(nil), task.Tags...),
		Visibility:   task.Visibility,
		Recurrence:   task.Recurrence,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_ = s.repo.Create(ctx, next)
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

// parseRemindLead 解析任务级提醒提前量：空字符串返回 nil（使用全局默认）；
// 否则按 Go Duration 格式（如 2h/30m/1h30m）解析，仅允许 [0, 31天] 区间。
func parseRemindLead(s string) (*time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 || d > 31*24*time.Hour {
		return nil, ErrInvalidRemindLead
	}
	return &d, nil
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
		ID:           task.ID,
		TenantID:     task.TenantID,
		CreatorID:    task.CreatorID,
		CreatorName:  task.CreatorName,
		AssigneeID:   task.AssigneeID,
		AssigneeName: task.AssigneeName,
		Title:        task.Title,
		Description:  task.Description,
		Status:       task.Status,
		Priority:     task.Priority,
		Tags:         task.Tags,
		Visibility:   task.Visibility,
		Recurrence:   task.Recurrence,
		CreatedAt:    task.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    task.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if resp.Tags == nil {
		resp.Tags = []string{}
	}
	if task.StartedAt != nil {
		resp.StartedAt = task.StartedAt.Format("2006-01-02 15:04:05")
	}
	if task.DueDate != nil {
		resp.DueDate = task.DueDate.Format("2006-01-02 15:04:05")
	}
	if task.RemindBefore != nil {
		resp.RemindBefore = task.RemindBefore.String()
	}
	if task.CompletedAt != nil {
		resp.CompletedAt = task.CompletedAt.Format("2006-01-02 15:04:05")
	}
	if task.DeletedAt != nil {
		resp.DeletedAt = task.DeletedAt.Format("2006-01-02 15:04:05")
	}
	return resp
}
