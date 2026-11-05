// Package service 实现任务模块的业务逻辑。
// 提供任务的创建、查询、更新、完成/重开、删除、列表与统计，
// 并在个人待办与租户协作两种场景下执行归属与可见范围校验。
package service

import (
	"context"
	"errors"
	"sort"
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
	// ErrParentNotFound 父任务不存在、跨租户或当前用户不可见
	ErrParentNotFound = errors.New("parent task not found")
	// ErrInvalidParent 父任务不合法：自引用、或父任务自身已是子任务（仅允许单层）
	ErrInvalidParent = errors.New("invalid parent task (only one level is allowed)")
	// ErrDependencySelf 前置依赖不得为任务自身
	ErrDependencySelf = errors.New("task cannot depend on itself")
	// ErrDependencyNotFound 前置任务不存在、跨租户或当前用户不可见
	ErrDependencyNotFound = errors.New("prerequisite task not found or not viewable")
	// ErrDependencyCycle 设置该前置依赖会形成循环阻塞链
	ErrDependencyCycle = errors.New("task dependency would create a cycle")
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
	// Board 一次性拉取当前筛选范围内看板所需任务（上限 repository.BoardMax），返回条目、真实总数与是否被截断。
	Board(ctx context.Context, f repository.ListFilter) ([]*model.Task, int64, bool, error)
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
	// BatchAssign 批量指派负责人，逐条按归属校验并通知新负责人，返回实际生效数。
	// assignees 为完整负责人集合（主负责人=首位），个人任务会被回写为仅创建人。
	BatchAssign(ctx context.Context, tenantID, userID string, ids []string, assignees []string) (int, error)
	// Assignable 返回当前租户存活成员列表，供前端负责人选择器（登录即可，不依赖 user:list 权限）。
	Assignable(ctx context.Context, tenantID string) ([]repository.AssigneeBrief, error)
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

// AssigneeLister 列出指定租户的存活成员，供 /tasks/assignable 端点使用。
// 由仓储层（UserDirectory）实现，服务层可选依赖：未实现时 Assignable 返回空列表。
type AssigneeLister interface {
	ListAssignable(ctx context.Context, tenantID string) ([]repository.AssigneeBrief, error)
}

// TaskService 任务模块业务服务。
type TaskService struct {
	repo    repository.TaskRepository // 任务仓储
	names   UserNameResolver          // 可选的用户姓名解析器，为 nil 时不富化展示名
	members TenantMemberChecker       // 可选的租户成员校验器，为 nil 时不校验负责人归属
	lister  AssigneeLister            // 可选的成员列表器，为 nil 时 Assignable 返回空
}

// NewTaskService 创建任务业务服务实例（不富化人员姓名、不校验负责人归属）。
func NewTaskService(repo repository.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

// NewTaskServiceWithNames 创建带人员姓名富化能力的任务业务服务实例。
// names 为 nil 时行为等同 NewTaskService，响应中人员姓名为空。
// 若 names 同时实现 TenantMemberChecker / AssigneeLister，则自动启用负责人归属校验与成员列表能力。
func NewTaskServiceWithNames(repo repository.TaskRepository, names UserNameResolver) *TaskService {
	s := &TaskService{repo: repo, names: names}
	if checker, ok := names.(TenantMemberChecker); ok {
		s.members = checker
	}
	if lister, ok := names.(AssigneeLister); ok {
		s.lister = lister
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
	parentID := strings.TrimSpace(req.ParentID)
	if parentID != "" {
		parent, perr := s.validateParent(ctx, tenantID, userID, parentID, "")
		if perr != nil {
			return nil, perr
		}
		// 子任务可见范围继承父任务（个人子任务的 creator 仍为当前用户）
		visibility = parent.Visibility
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

	// 解析多负责人集合：给出 assignees 以集合为准（主负责人=首位），否则回落 assignee_id；
	// 个人任务强制仅创建人本人。集合恒非空（缺省回落创建人），供 assignees[0] 取主负责人。
	assignees := normalizeAssignees(req.Assignees, req.AssigneeID, userID, visibility)
	if err := s.validateAssignees(ctx, tenantID, userID, visibility, assignees); err != nil {
		return nil, err
	}

	task := &model.Task{
		ID:           idgen.New(),
		TenantID:     tenantID,
		CreatorID:    userID,
		AssigneeID:   assignees[0],
		AssigneeIDs:  assignees,
		ParentID:     parentID,
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

	// 前置依赖校验（存在/同租户/可查看/不自引用/不成环）在建库前完成，避免创建了孤儿任务后再拒绝。
	blockedIDs, berr := s.validateBlockedBy(ctx, tenantID, userID, task.ID, req.BlockedBy)
	if berr != nil {
		return nil, berr
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	// 负责人集合写入关联表（供反向过滤/多负责人富化）；主负责人已落 tasks.assignee_id。
	if err := s.repo.ReplaceAssignees(ctx, task.ID, assignees); err != nil {
		return nil, err
	}
	// 前置依赖集合落库（校验已在建库前完成）；为空时 ReplaceBlockedBy 对新任务为无效操作。
	if err := s.repo.ReplaceBlockedBy(ctx, task.ID, blockedIDs); err != nil {
		return nil, err
	}
	// 向集合中被指派为“他人”的负责人去重发送站内提醒（自己给自己的不提醒）。
	notifyTaskAssigned(ctx, task.CreatorID, assignees, task)
	s.enrichNames(ctx, task)
	s.fillDependencies(ctx, task)
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
	s.fillDependencies(ctx, task)
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

	wasCompleted := task.Status == model.TaskStatusCompleted
	// assigneesDirty 标记本次是否显式变更了负责人集合，仅变更时才回写关联表并提醒。
	assigneesDirty := false

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
	if req.ParentID != nil {
		pid := strings.TrimSpace(*req.ParentID)
		if pid != "" {
			parent, perr := s.validateParent(ctx, tenantID, userID, pid, task.ID)
			if perr != nil {
				return nil, perr
			}
			task.ParentID = pid
			// 子任务可见范围继承父任务
			task.Visibility = parent.Visibility
		} else {
			// 传空字符串解除父子关系，回退为根任务
			task.ParentID = ""
		}
	}
	if req.Recurrence != nil {
		r := strings.TrimSpace(*req.Recurrence)
		if !model.IsValidRecurrence(r) {
			return nil, ErrInvalidRecurrence
		}
		task.Recurrence = r
	}
	if req.Assignees != nil || req.AssigneeID != nil {
		// 集合优先：req.Assignees 非 nil 以集合为准（同时给了 assignee_id 则作为集合为空时的回落）；
		// 仅有 assignee_id 时退化为单元素集合。个人任务由 normalizeAssignees 强制回写为创建人。
		var set []string
		if req.Assignees != nil {
			single := ""
			if req.AssigneeID != nil {
				single = *req.AssigneeID
			}
			set = normalizeAssignees(*req.Assignees, single, task.CreatorID, task.Visibility)
		} else {
			set = normalizeAssignees(nil, *req.AssigneeID, task.CreatorID, task.Visibility)
		}
		if err := s.validateAssignees(ctx, tenantID, task.CreatorID, task.Visibility, set); err != nil {
			return nil, err
		}
		task.AssigneeIDs = set
		task.AssigneeID = set[0]
		assigneesDirty = true
	}
	blockedChanged := false
	var blockedIDs []string
	if req.BlockedBy != nil {
		// blocked_by 非 nil 即视为要重置前置集合（空数组=清空）；nil 保持不动。校验在建库前完成。
		ids, berr := s.validateBlockedBy(ctx, tenantID, userID, task.ID, *req.BlockedBy)
		if berr != nil {
			return nil, berr
		}
		blockedIDs = ids
		blockedChanged = true
	}
	if req.Tags != nil {
		task.Tags = normalizeTags(*req.Tags)
	}
	if req.DueDate != nil {
		dueDate, err := parseDueDate(*req.DueDate)
		if err != nil {
			return nil, err
		}
		// 截止日实质变化时才重置提醒标记，使新截止日可再次触发到期提醒；
		// 原值回传不能重置，否则前端整体保存会抹掉冷却基准导致重复骚扰。
		if !sameDue(task.DueDate, dueDate) {
			task.ReminderSentAt = nil
		}
		task.DueDate = dueDate
	}
	if req.RemindBefore != nil {
		remindBefore, err := parseRemindLead(*req.RemindBefore)
		if err != nil {
			return nil, err
		}
		// 同理：仅当提前量实质变化时才重置，使新提醒窗口可重新评估。
		if !sameLead(task.RemindBefore, remindBefore) {
			task.ReminderSentAt = nil
		}
		task.RemindBefore = remindBefore
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
	// 负责人集合变更时回写关联表，并向新集合中除创建人外的负责人去重发送提醒
	if assigneesDirty {
		if err := s.repo.ReplaceAssignees(ctx, task.ID, task.AssigneeIDs); err != nil {
			return nil, err
		}
		notifyTaskAssigned(ctx, task.CreatorID, task.AssigneeIDs, task)
	}
	// 前置依赖集合本次发生变更时回写关联表（含清空）。
	if blockedChanged {
		if err := s.repo.ReplaceBlockedBy(ctx, task.ID, blockedIDs); err != nil {
			return nil, err
		}
	}
	// 本次从非完成首次转为完成，且任务设了重复周期时，自动生成下一周期实例
	if task.Status == model.TaskStatusCompleted && !wasCompleted {
		s.spawnNextOccurrence(ctx, task)
		s.notifyUnblocked(ctx, task)
	}
	s.enrichNames(ctx, task)
	s.fillDependencies(ctx, task)
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
		s.notifyUnblocked(ctx, task)
	}
	s.enrichNames(ctx, task)
	s.fillDependencies(ctx, task)
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
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	// 级联软删其直接子任务（单层，无更深后代）。失败不阻断主删除但向上抛出供感知。
	return s.repo.SoftDeleteChildren(ctx, id)
}

// validateParent 校验待挂父任务合法，返回父任务实体。规则：
//   - parentID 为空时返回 (nil, nil)，调用方据此视为根任务；
//   - 不允许自引用（parentID == selfID）；
//   - 父任务须存在、同租户、当前用户可查看，否则 ErrParentNotFound；
//   - 仅允许单层：父任务自身 ParentID 非空（已是子任务）则 ErrInvalidParent。
func (s *TaskService) validateParent(ctx context.Context, tenantID, userID, parentID, selfID string) (*model.Task, error) {
	if parentID == "" {
		return nil, nil
	}
	if selfID != "" && parentID == selfID {
		return nil, ErrInvalidParent
	}
	parent, err := s.load(ctx, parentID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, ErrParentNotFound
		}
		return nil, err
	}
	if parent.TenantID != tenantID || !canView(parent, tenantID, userID) {
		return nil, ErrParentNotFound
	}
	if parent.ParentID != "" {
		return nil, ErrInvalidParent
	}
	return parent, nil
}

// List 分页查询任务列表，并对结果富化人员展示名。
func (s *TaskService) List(ctx context.Context, f repository.ListFilter) ([]*model.Task, int64, error) {
	tasks, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, err
	}
	s.fillAssignees(ctx, tasks...)
	s.enrichNames(ctx, tasks...)
	s.fillDependencies(ctx, tasks...)
	return tasks, total, nil
}

// Board 一次性拉取看板所需任务（复用当前筛选、不分页），富化展示名后返回条目、真实总数与是否被截断。
// truncated 为 true 表示匹配总数超过 repository.BoardMax（看板仅展示了其中一部分）。
func (s *TaskService) Board(ctx context.Context, f repository.ListFilter) ([]*model.Task, int64, bool, error) {
	tasks, total, err := s.repo.ListBoard(ctx, f)
	if err != nil {
		return nil, 0, false, err
	}
	s.fillAssignees(ctx, tasks...)
	s.enrichNames(ctx, tasks...)
	s.fillDependencies(ctx, tasks...)
	return tasks, total, total > int64(len(tasks)), nil
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

// BatchAssign 批量指派负责人，逐条复用更新逻辑（含归属校验与对新负责人集合的站内提醒），
// 跳过无权/不存在的项，返回成功数。个人任务会被回写为仅创建人，仍计入成功。
func (s *TaskService) BatchAssign(ctx context.Context, tenantID, userID string, ids []string, assignees []string) (int, error) {
	// 先校验集合合法（去重去空后非空），避免逐项 Update 静默吞错导致徒劳批量。
	set := normalizeAssignees(assignees, "", userID, model.TaskVisibilityTenant)
	if err := s.validateAssignees(ctx, tenantID, userID, model.TaskVisibilityTenant, set); err != nil {
		return 0, err
	}
	affected := 0
	for _, id := range ids {
		if _, err := s.Update(ctx, tenantID, userID, id, dto.UpdateTaskReq{Assignees: &set}); err == nil {
			affected++
		}
	}
	return affected, nil
}

// Assignable 返回当前租户存活成员列表，供前端负责人选择器使用。
// 未注入 lister 时返回空列表（登录仍可访问，仅无候选人）。
func (s *TaskService) Assignable(ctx context.Context, tenantID string) ([]repository.AssigneeBrief, error) {
	if s.lister == nil {
		return []repository.AssigneeBrief{}, nil
	}
	return s.lister.ListAssignable(ctx, tenantID)
}

// ReminderOptions 到期提醒扫描的可配置参数；零值字段回落内置默认。
// 由定时提醒任务从应用配置注入，使逾期重复提醒冷却期与单轮扫描上限可外部调整。
type ReminderOptions struct {
	OverdueCooldown time.Duration // 逾期任务两次重复提醒之间的最小间隔
	BatchLimit      int           // 单次扫描发送提醒的最大任务数
}

// SendDueReminders 扫描需要提醒的未完成任务（即将到期/首次逾期、以及距上次提醒超过冷却期的已逾期任务），
// 逐条向全部负责人（去重广播，无负责人时回退创建人）发送站内提醒并刷新提醒时间，返回实际提醒的任务数。
// 供定时提醒任务调用，不受多租户/可见范围限制（系统级扫描）。
// opts 为可选扫描参数（冷却期/单轮上限），缺省时使用内置默认值。
// 冷却基准 MarkReminderSent 按任务（而非收件人）记录，故多负责人广播不会放大重复提醒频率。
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
	// FindDueForReminder 不回填负责人集合，此处补齐以支持多负责人广播（仅依赖 LoadAssignees）。
	s.fillAssignees(ctx, tasks...)
	sent := 0
	for _, t := range tasks {
		title := "任务截止提醒"
		content := "您负责的任务「" + t.Title + "」即将到期（截止：" + formatDue(t.DueDate) + "），请及时处理。"
		if t.DueDate != nil && t.DueDate.Before(now) {
			title = "任务逾期提醒"
			content = "您负责的任务「" + t.Title + "」已逾期（截止：" + formatDue(t.DueDate) + "），请尽快处理。"
		}
		// 收件人与 notifyTaskUnblocked 对称：以负责人集合为基准去重广播，
		// 集合为空回退主负责人，再为空回退创建人。
		recipients := t.AssigneeIDs
		if len(recipients) == 0 && t.AssigneeID != "" {
			recipients = []string{t.AssigneeID}
		}
		if len(recipients) == 0 && t.CreatorID != "" {
			recipients = []string{t.CreatorID}
		}
		if mgr := notify.GetGlobalManager(); mgr != nil {
			seen := make(map[string]struct{}, len(recipients))
			for _, a := range recipients {
				if a == "" {
					continue
				}
				if _, ok := seen[a]; ok {
					continue
				}
				seen[a] = struct{}{}
				mgr.NotifyTaskReminder(ctx, a, title, content, reminderTypeDue,
					map[string]interface{}{"task_id": t.ID, "tenant_id": t.TenantID})
			}
		}
		// 无论站内推送成否均刷新提醒时间，作为下次重复提醒的冷却基准（每任务一次）
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
	reminderTypeUnlock = "task_unblocked" // 前置完成后解锁提醒类型标识
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

// notifyTaskAssigned 向负责人集合中除创建人外的每位去重发送站内指派提醒；
// 通知管理器未初始化时静默跳过，不影响主流程。
func notifyTaskAssigned(ctx context.Context, creatorID string, assignees []string, task *model.Task) {
	mgr := notify.GetGlobalManager()
	if mgr == nil {
		return
	}
	title := "新任务指派"
	content := "您被指派了一个新任务「" + task.Title + "」，请及时关注。"
	seen := make(map[string]struct{}, len(assignees))
	for _, a := range assignees {
		if a == "" || a == creatorID {
			continue
		}
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		mgr.NotifyTaskReminder(ctx, a, title, content, reminderTypeAssign,
			map[string]interface{}{"task_id": task.ID, "tenant_id": task.TenantID})
	}
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
// 未注入解析器或解析失败时静默跳过，不影响主流程。同时根据 AssigneeIDs 集合富化 AssigneeNames。
func (s *TaskService) enrichNames(ctx context.Context, tasks ...*model.Task) {
	if s.names == nil || len(tasks) == 0 {
		return
	}
	idSet := make(map[string]struct{}, len(tasks)*2)
	addID := func(id string) {
		if id != "" {
			idSet[id] = struct{}{}
		}
	}
	for _, t := range tasks {
		addID(t.CreatorID)
		addID(t.AssigneeID)
		for _, a := range t.AssigneeIDs {
			addID(a)
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
		if len(t.AssigneeIDs) > 0 {
			names := make([]string, 0, len(t.AssigneeIDs))
			for _, a := range t.AssigneeIDs {
				names = append(names, nameMap[a])
			}
			t.AssigneeNames = names
		}
	}
}

// fillAssignees 从 task_assignees 表批量回填各任务的负责人集合（主负责人排首位）。
// 仓储读取（GetByID/List/ListBoard）不携带集合，统一在此补齐，供鉴权与响应富化使用。
func (s *TaskService) fillAssignees(ctx context.Context, tasks ...*model.Task) {
	if len(tasks) == 0 {
		return
	}
	ids := make([]string, 0, len(tasks))
	for _, t := range tasks {
		if t.ID != "" {
			ids = append(ids, t.ID)
		}
	}
	m, err := s.repo.LoadAssignees(ctx, ids)
	if err != nil || m == nil {
		return
	}
	for _, t := range tasks {
		set := m[t.ID]
		// 主负责人恒排首位：若集合缺失主负责人（历史未回填）则前置补齐，否则保持主负责人在首。
		if t.AssigneeID != "" && !containsID(set, t.AssigneeID) {
			set = append([]string{t.AssigneeID}, set...)
		} else if t.AssigneeID != "" {
			ordered := make([]string, 0, len(set))
			ordered = append(ordered, t.AssigneeID)
			for _, id := range set {
				if id != t.AssigneeID {
					ordered = append(ordered, id)
				}
			}
			set = ordered
		}
		if len(set) > 0 {
			t.AssigneeIDs = set
		}
	}
}

// validateBlockedBy 校验并归一化前置依赖集合，返回去重去空后的合法 ID 列表（不落库，供调用方在校验通过后再写）。
// 规则：
//   - taskID 为空（新任务尚无 ID）时直接拒绝非空集合？——不会发生：调用方均先生成 ID；
//   - 任一前置 == taskID → ErrDependencySelf；
//   - 前置须存在、同租户、当前用户可查看 → 否则 ErrDependencyNotFound；
//   - 加入后不得成环（从候选前置沿现有 blocked_by 边可达 taskID 即为环）→ 否则 ErrDependencyCycle。
func (s *TaskService) validateBlockedBy(ctx context.Context, tenantID, userID, taskID string, blockedIDs []string) ([]string, error) {
	if len(blockedIDs) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(blockedIDs))
	seen := make(map[string]struct{}, len(blockedIDs))
	for _, raw := range blockedIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if id == taskID {
			return nil, ErrDependencySelf
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		dep, err := s.load(ctx, id)
		if err != nil {
			if errors.Is(err, ErrTaskNotFound) {
				return nil, ErrDependencyNotFound
			}
			return nil, err
		}
		if dep.TenantID != tenantID || !canView(dep, tenantID, userID) {
			return nil, ErrDependencyNotFound
		}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, nil
	}
	// 循环检测：基于当前租户已有依赖图，若任一候选前置沿 blocked_by 边可达 taskID，则新边会闭环。
	edges, err := s.repo.TenantDependencyEdges(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, d := range out {
		if reaches(edges, d, taskID) {
			return nil, ErrDependencyCycle
		}
	}
	return out, nil
}

// reaches 判断在 edges（node -> 其前置列表）构成的有向图中，从 from 出发沿前置边能否到达 target。采用 BFS + visited 防重入。
func reaches(edges map[string][]string, from, target string) bool {
	if from == target {
		return true
	}
	visited := make(map[string]struct{})
	queue := []string{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == target {
			return true
		}
		if _, ok := visited[cur]; ok {
			continue
		}
		visited[cur] = struct{}{}
		queue = append(queue, edges[cur]...)
	}
	return false
}

// fillDependencies 批量回填各任务的阻塞关系双向视图（BlockedBy 前置、Blocks 后继），携带标题/状态供前端徽标。
// 仓储读取不携带依赖，统一在此补齐。失败静默跳过（不影响主流程）。
func (s *TaskService) fillDependencies(ctx context.Context, tasks ...*model.Task) {
	if len(tasks) == 0 {
		return
	}
	ids := make([]string, 0, len(tasks))
	for _, t := range tasks {
		if t.ID != "" {
			ids = append(ids, t.ID)
		}
	}
	blockedBy, blocks, err := s.repo.LoadDependencies(ctx, ids)
	if err != nil {
		return
	}
	// 汇总双向引用的全部任务 ID，一次拉取标题/状态
	refSet := make(map[string]struct{})
	for _, list := range blockedBy {
		for _, id := range list {
			refSet[id] = struct{}{}
		}
	}
	for _, list := range blocks {
		for _, id := range list {
			refSet[id] = struct{}{}
		}
	}
	refs := make([]string, 0, len(refSet))
	for id := range refSet {
		refs = append(refs, id)
	}
	brief, err := s.repo.FindBrief(ctx, refs)
	if err != nil {
		brief = nil
	}
	for _, t := range tasks {
		t.BlockedBy = toTaskRefs(blockedBy[t.ID], brief)
		t.Blocks = toTaskRefs(blocks[t.ID], brief)
	}
}

// toTaskRefs 将一组依赖 ID 按 ID 稳定排序后映射为 model.TaskRef（跳过已删/不可见的引用）。
func toTaskRefs(idList []string, brief map[string]model.TaskRef) []model.TaskRef {
	if len(idList) == 0 {
		return nil
	}
	sorted := append([]string(nil), idList...)
	sort.Strings(sorted)
	out := make([]model.TaskRef, 0, len(sorted))
	for _, id := range sorted {
		if ref, ok := brief[id]; ok {
			out = append(out, ref)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// notifyUnblocked 在 task 刚转为完成后，扫描直接后继：若某后继的全部前置均已完成，则向其负责人发送解锁提醒。
// 仅提醒、不自动推进后继状态；通知管理器未初始化时静默跳过。
func (s *TaskService) notifyUnblocked(ctx context.Context, task *model.Task) {
	mgr := notify.GetGlobalManager()
	if mgr == nil {
		return
	}
	deps, err := s.repo.DependentsOf(ctx, task.ID)
	if err != nil || len(deps) == 0 {
		return
	}
	for _, depID := range deps {
		bb, _, err := s.repo.LoadDependencies(ctx, []string{depID})
		if err != nil {
			continue
		}
		prereq := bb[depID]
		if len(prereq) == 0 {
			continue
		}
		brief, err := s.repo.FindBrief(ctx, prereq)
		if err != nil {
			continue
		}
		allDone := true
		for _, p := range prereq {
			ref, ok := brief[p]
			if !ok || ref.Status != model.TaskStatusCompleted {
				allDone = false
				break
			}
		}
		if !allDone {
			continue
		}
		dep, err := s.load(ctx, depID)
		if err != nil {
			continue
		}
		s.notifyTaskUnblocked(ctx, task, dep)
	}
}

// notifyTaskUnblocked 向后继任务的全部负责人（去重）发送“前置已完成、可开始”提醒。
func (s *TaskService) notifyTaskUnblocked(ctx context.Context, done, dependent *model.Task) {
	mgr := notify.GetGlobalManager()
	if mgr == nil {
		return
	}
	title := "任务已解锁"
	content := "前置任务「" + done.Title + "」已完成，任务「" + dependent.Title + "」可以开始了。"
	recipients := dependent.AssigneeIDs
	if len(recipients) == 0 && dependent.AssigneeID != "" {
		recipients = []string{dependent.AssigneeID}
	}
	seen := make(map[string]struct{}, len(recipients))
	for _, a := range recipients {
		if a == "" {
			continue
		}
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		mgr.NotifyTaskReminder(ctx, a, title, content, reminderTypeUnlock,
			map[string]interface{}{"task_id": dependent.ID, "tenant_id": dependent.TenantID})
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
// 复制标题/描述/优先级/负责人（含多负责人集合）/可见范围/提醒提前量/标签，重置为 pending 并按周期从原截止日推进。
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
		AssigneeIDs:  append([]string(nil), task.AssigneeIDs...),
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
	if err := s.repo.Create(ctx, next); err != nil {
		return
	}
	// Create 仅落 tasks 主表，不写 task_assignees；副负责人集合需显式回写，与 Create 服务路径对称。
	if len(next.AssigneeIDs) > 0 {
		_ = s.repo.ReplaceAssignees(ctx, next.ID, next.AssigneeIDs)
	}
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
// 读取后回填负责人集合，使基于多负责人的 canView/canEdit 在单条操作（改/删/完成）时生效。
func (s *TaskService) load(ctx context.Context, id string) (*model.Task, error) {
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	s.fillAssignees(ctx, task)
	return task, nil
}

// canView 判断用户能否查看任务：租户协作任务对同租户成员开放，个人任务仅创建人/任一负责人（含副负责人）可见。
func canView(task *model.Task, tenantID, userID string) bool {
	if task.TenantID != tenantID {
		return false
	}
	if task.Visibility == model.TaskVisibilityTenant {
		return true
	}
	return task.CreatorID == userID || task.AssigneeID == userID || containsID(task.AssigneeIDs, userID)
}

// canEdit 判断用户能否编辑/完成：创建人恒可，租户协作任务的任一负责人（含副负责人）亦可。
func canEdit(task *model.Task, userID string) bool {
	if task.CreatorID == userID {
		return true
	}
	if task.Visibility != model.TaskVisibilityTenant {
		return false
	}
	return task.AssigneeID == userID || containsID(task.AssigneeIDs, userID)
}

// containsID 判断去空后的 ID 是否出现在集合中。
func containsID(ids []string, id string) bool {
	if id == "" {
		return false
	}
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// normalizeAssignees 归一化负责人集合：个人任务强制仅创建人；否则优先取 set（去重去空），
// 集合为空时回落 single，仍为空则回落 userID（缺省自己负责）。返回值恒非空，首位即主负责人。
func normalizeAssignees(set []string, single, userID, visibility string) []string {
	if visibility == model.TaskVisibilityPersonal {
		return []string{userID}
	}
	out := make([]string, 0, len(set)+1)
	seen := make(map[string]struct{}, len(set)+1)
	for _, id := range set {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		if s := strings.TrimSpace(single); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		out = append(out, userID)
	}
	return out
}

// validateAssignees 校验负责人集合：个人任务无需校验（仅创建人）；租户任务逐个校验为同租户成员。
func (s *TaskService) validateAssignees(ctx context.Context, tenantID, creatorID, visibility string, assignees []string) error {
	if visibility == model.TaskVisibilityPersonal {
		return nil
	}
	for _, a := range assignees {
		if err := s.validateAssignee(ctx, tenantID, creatorID, a); err != nil {
			return err
		}
	}
	return nil
}

// ParseTaskDate 导出给 handler 层复用的时间解析：宽松支持 RFC3339、
// "2006-01-02 15:04:05" 与 "2006-01-02"；空串返回 (nil, nil)，无法解析返回 ErrInvalidDueDate。
func ParseTaskDate(s string) (*time.Time, error) {
	return parseDueDate(s)
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

// sameDue 判断两个截止时间是否等价（均为空视为相等）。
// 按秒比较：接口层 ToResp 以秒级精度输出 due_date，前端原样回传时只会携带秒，
// 若按纳秒比较会把“未变更的回传”误判为改期，从而误重置提醒冷却基准。
func sameDue(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Truncate(time.Second).Equal(b.Truncate(time.Second))
}

// sameLead 判断两个提醒提前量是否等价（均为空视为相等）。
// 同样用于避免原值回传被误判为变更而重置提醒冷却。
func sameLead(a, b *time.Duration) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
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
		AssigneeID:    task.AssigneeID,
		AssigneeName:  task.AssigneeName,
		Assignees:     task.AssigneeIDs,
		AssigneeNames: task.AssigneeNames,
		ParentID:      task.ParentID,
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
	resp.BlockedBy = depBriefs(task.BlockedBy)
	resp.Blocks = depBriefs(task.Blocks)
	return resp
}

// depBriefs 将领域层依赖引用切片转为响应体；空集合返回 nil（配合 omitempty 不输出）。
func depBriefs(refs []model.TaskRef) []dto.DependencyBriefResp {
	if len(refs) == 0 {
		return nil
	}
	out := make([]dto.DependencyBriefResp, 0, len(refs))
	for _, r := range refs {
		out = append(out, dto.DependencyBriefResp{ID: r.ID, Title: r.Title, Status: r.Status})
	}
	return out
}
