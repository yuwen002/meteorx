package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/task/dto"
	"meteorx/internal/modules/task/model"
	"meteorx/internal/modules/task/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// stubTaskRepo 内存版任务仓储桩，用于 service 层单测。
// deleted 映射模拟软删除回收站，与 tasks（活跃数据）分离；
// marked 记录 MarkReminderSent 被调用的任务 ID，供提醒测试断言。
type stubTaskRepo struct {
	tasks     map[string]*model.Task
	deleted   map[string]*model.Task
	marked    []string
	assignees map[string][]string // taskID -> 负责人集合，模拟 task_assignees 表
	deps      map[string][]string // taskID -> 前置 blocked_by 集合，模拟 task_dependencies 表
}

func newStubTaskRepo() *stubTaskRepo {
	return &stubTaskRepo{tasks: make(map[string]*model.Task), deleted: make(map[string]*model.Task), assignees: make(map[string][]string), deps: make(map[string][]string)}
}

// ReplaceAssignees 覆盖式写入任务负责人集合（去重去空），模拟关联表原子替换。
func (r *stubTaskRepo) ReplaceAssignees(_ context.Context, taskID string, ids []string) error {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	r.assignees[taskID] = out
	return nil
}

// LoadAssignees 批量读取任务负责人集合。
func (r *stubTaskRepo) LoadAssignees(_ context.Context, ids []string) (map[string][]string, error) {
	out := make(map[string][]string, len(ids))
	for _, id := range ids {
		if set, ok := r.assignees[id]; ok {
			cp := append([]string(nil), set...)
			out[id] = cp
		}
	}
	return out, nil
}

// ReplaceBlockedBy 覆盖式写入任务前置依赖集合（去重去空、剔除自引用），模拟 task_dependencies 表原子替换。
func (r *stubTaskRepo) ReplaceBlockedBy(_ context.Context, taskID string, blockedIDs []string) error {
	if taskID == "" {
		return nil
	}
	seen := make(map[string]struct{}, len(blockedIDs))
	out := make([]string, 0, len(blockedIDs))
	for _, id := range blockedIDs {
		if id == "" || id == taskID {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		delete(r.deps, taskID)
		return nil
	}
	r.deps[taskID] = out
	return nil
}

// LoadDependencies 返回正向 blockedBy（从 r.deps）与反向 blocks（扫描 deps 值命中 ids 的 task）。
func (r *stubTaskRepo) LoadDependencies(_ context.Context, ids []string) (map[string][]string, map[string][]string, error) {
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	blockedBy := make(map[string][]string)
	blocks := make(map[string][]string)
	for _, id := range ids {
		if set, ok := r.deps[id]; ok {
			blockedBy[id] = append([]string(nil), set...)
		}
	}
	for taskID, prereq := range r.deps {
		for _, p := range prereq {
			if _, ok := idSet[p]; ok {
				blocks[p] = append(blocks[p], taskID)
			}
		}
	}
	return blockedBy, blocks, nil
}

// FindBrief 批量读取未删除任务的轻量引用（id -> TaskRef）。
func (r *stubTaskRepo) FindBrief(_ context.Context, ids []string) (map[string]model.TaskRef, error) {
	out := make(map[string]model.TaskRef, len(ids))
	for _, id := range ids {
		if t, ok := r.tasks[id]; ok {
			out[id] = model.TaskRef{ID: t.ID, Title: t.Title, Status: t.Status}
		}
	}
	return out, nil
}

// DependentsOf 扫描 deps 中哪些任务的 blocked_by 含 blockedByID（后继）。
func (r *stubTaskRepo) DependentsOf(_ context.Context, blockedByID string) ([]string, error) {
	var out []string
	for taskID, prereq := range r.deps {
		for _, p := range prereq {
			if p == blockedByID {
				out = append(out, taskID)
				break
			}
		}
	}
	return out, nil
}

// TenantDependencyEdges 返回本租户全部任务的依赖图（task -> blocked_by）。
func (r *stubTaskRepo) TenantDependencyEdges(_ context.Context, tenantID string) (map[string][]string, error) {
	edges := make(map[string][]string)
	for taskID, prereq := range r.deps {
		if t, ok := r.tasks[taskID]; ok && t.TenantID == tenantID {
			edges[taskID] = append([]string(nil), prereq...)
		}
	}
	return edges, nil
}

func (r *stubTaskRepo) Create(_ context.Context, t *model.Task) error {
	cp := *t
	r.tasks[t.ID] = &cp
	return nil
}

func (r *stubTaskRepo) GetByID(_ context.Context, id string) (*model.Task, error) {
	t, ok := r.tasks[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *stubTaskRepo) Update(_ context.Context, t *model.Task) error {
	cp := *t
	r.tasks[t.ID] = &cp
	return nil
}

func (r *stubTaskRepo) UpdateStatus(_ context.Context, id, status string) error {
	t, ok := r.tasks[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	t.Status = status
	if status == model.TaskStatusCompleted {
		now := time.Now()
		t.CompletedAt = &now
	} else {
		t.CompletedAt = nil
	}
	return nil
}

func (r *stubTaskRepo) Delete(_ context.Context, id string) error {
	delete(r.tasks, id)
	return nil
}

// SoftDeleteChildren 将 parentID 的直接子任务从活跃集移入回收站（模拟级联软删）。
func (r *stubTaskRepo) SoftDeleteChildren(_ context.Context, parentID string) error {
	if parentID == "" {
		return nil
	}
	for id, t := range r.tasks {
		if t.ParentID == parentID {
			cp := *t
			now := time.Now()
			cp.DeletedAt = &now
			r.deleted[id] = &cp
			delete(r.tasks, id)
		}
	}
	return nil
}

func (r *stubTaskRepo) GetByIDUnscoped(_ context.Context, id string) (*model.Task, error) {
	if t, ok := r.tasks[id]; ok {
		cp := *t
		return &cp, nil
	}
	if t, ok := r.deleted[id]; ok {
		cp := *t
		return &cp, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *stubTaskRepo) ListDeleted(_ context.Context, f repository.ListFilter) ([]*model.Task, int64, error) {
	var out []*model.Task
	for _, t := range r.deleted {
		if t.TenantID != f.TenantID {
			continue
		}
		if f.UserID != "" && t.CreatorID != f.UserID {
			continue
		}
		cp := *t
		out = append(out, &cp)
	}
	return out, int64(len(out)), nil
}

func (r *stubTaskRepo) Restore(_ context.Context, id string) error {
	if t, ok := r.deleted[id]; ok {
		cp := *t
		cp.DeletedAt = nil
		r.tasks[id] = &cp
		delete(r.deleted, id)
	}
	return nil
}

func (r *stubTaskRepo) PermanentDelete(_ context.Context, id string) error {
	delete(r.deleted, id)
	return nil
}

func (r *stubTaskRepo) List(_ context.Context, f repository.ListFilter) ([]*model.Task, int64, error) {
	var out []*model.Task
	for _, t := range r.tasks {
		if t.TenantID != f.TenantID {
			continue
		}
		if f.Status != "" && t.Status != f.Status {
			continue
		}
		cp := *t
		out = append(out, &cp)
	}
	return out, int64(len(out)), nil
}

// ListBoard 看板桩：与 List 同口径返回租户内匹配任务（不分页）。
func (r *stubTaskRepo) ListBoard(_ context.Context, f repository.ListFilter) ([]*model.Task, int64, error) {
	var out []*model.Task
	for _, t := range r.tasks {
		if t.TenantID != f.TenantID {
			continue
		}
		cp := *t
		out = append(out, &cp)
	}
	return out, int64(len(out)), nil
}

func (r *stubTaskRepo) Stats(_ context.Context, f repository.ListFilter) (*repository.TaskStatusStats, error) {
	s := &repository.TaskStatusStats{}
	for _, t := range r.tasks {
		if t.TenantID != f.TenantID {
			continue
		}
		s.Total++
		switch t.Status {
		case model.TaskStatusPending:
			s.Pending++
		case model.TaskStatusInProgress:
			s.InProgress++
		case model.TaskStatusCompleted:
			s.Completed++
		}
	}
	return s, nil
}

// FindDueForReminder 模拟到期扫描：未删除、未完成、截止日不晚于 horizon，且满足
// “未提醒过”或“已逾期(due<now)且距上次提醒超过冷却期(reminder_sent_at<=cutoff)”。
func (r *stubTaskRepo) FindDueForReminder(_ context.Context, horizon, now, cutoff time.Time, _ int) ([]*model.Task, error) {
	var out []*model.Task
	for _, t := range r.tasks {
		if t.Status != model.TaskStatusPending && t.Status != model.TaskStatusInProgress {
			continue
		}
		if t.DueDate == nil || t.DueDate.After(horizon) {
			continue
		}
		if t.ReminderSentAt == nil {
			cp := *t
			out = append(out, &cp)
			continue
		}
		// 已逾期且上次提醒早于冷却基准，应重复提醒
		if t.DueDate.Before(now) && !t.ReminderSentAt.After(cutoff) {
			cp := *t
			out = append(out, &cp)
		}
	}
	return out, nil
}

// MarkReminderSent 记录已提醒任务并写入标记时间。
func (r *stubTaskRepo) MarkReminderSent(_ context.Context, id string, tm time.Time) error {
	r.marked = append(r.marked, id)
	if t, ok := r.tasks[id]; ok {
		ts := tm
		t.ReminderSentAt = &ts
	}
	return nil
}

func newService() (*TaskService, *stubTaskRepo) {
	repo := newStubTaskRepo()
	return NewTaskService(repo), repo
}

func ptr[T any](v T) *T { return &v }

// ---- Create ----

func TestCreate_Defaults(t *testing.T) {
	svc, _ := newService()
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{Title: "写周报"})
	require.NoError(t, err)

	assert.Equal(t, "t1", task.TenantID)
	assert.Equal(t, "u1", task.CreatorID)
	assert.Equal(t, model.TaskStatusPending, task.Status)
	assert.Equal(t, model.TaskPriorityNormal, task.Priority)
	assert.Equal(t, model.TaskVisibilityPersonal, task.Visibility)
	// 个人任务负责人强制为创建人
	assert.Equal(t, "u1", task.AssigneeID)
	assert.Nil(t, task.DueDate)
}

func TestCreate_TenantVisibilityKeepsAssignee(t *testing.T) {
	svc, _ := newService()
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title:      "联调接口",
		Visibility: model.TaskVisibilityTenant,
		AssigneeID: "u2",
		Priority:   model.TaskPriorityHigh,
	})
	require.NoError(t, err)
	assert.Equal(t, model.TaskVisibilityTenant, task.Visibility)
	assert.Equal(t, "u2", task.AssigneeID)
	assert.Equal(t, model.TaskPriorityHigh, task.Priority)
}

func TestCreate_PersonalOverridesAssignee(t *testing.T) {
	svc, _ := newService()
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title:      "私密事项",
		Visibility: model.TaskVisibilityPersonal,
		AssigneeID: "u2", // 个人任务应被忽略
	})
	require.NoError(t, err)
	assert.Equal(t, "u1", task.AssigneeID)
}

func TestCreate_InvalidDueDate(t *testing.T) {
	svc, _ := newService()
	_, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title:   "x",
		DueDate: "not-a-date",
	})
	assert.ErrorIs(t, err, ErrInvalidDueDate)
}

func TestCreate_CompletedSetsCompletedAt(t *testing.T) {
	svc, _ := newService()
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title:  "已完成事项",
		Status: model.TaskStatusCompleted,
	})
	require.NoError(t, err)
	assert.NotNil(t, task.CompletedAt)
}

// ---- Get / 可见性 ----

func TestGet_PersonalByCreator(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", AssigneeID: "u1", Visibility: model.TaskVisibilityPersonal}
	task, err := svc.Get(context.Background(), "t1", "u1", "id1")
	require.NoError(t, err)
	assert.Equal(t, "id1", task.ID)
}

func TestGet_PersonalByOtherDenied(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityPersonal}
	_, err := svc.Get(context.Background(), "t1", "u9", "id1")
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestGet_TenantVisibleToMembers(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	task, err := svc.Get(context.Background(), "t1", "u2", "id1")
	require.NoError(t, err)
	assert.Equal(t, "id1", task.ID)
}

func TestGet_CrossTenantDenied(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	_, err := svc.Get(context.Background(), "t2", "u1", "id1")
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

// ---- Update / 归属 ----

func TestUpdate_ForbiddenForStranger(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	_, err := svc.Update(context.Background(), "t1", "u9", "id1", dto.UpdateTaskReq{Title: ptr("改名")})
	assert.ErrorIs(t, err, ErrTaskForbidden)
}

func TestUpdate_ByCreatorAppliesFields(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	task, err := svc.Update(context.Background(), "t1", "u1", "id1", dto.UpdateTaskReq{
		Title:    ptr("新标题"),
		Priority: ptr(model.TaskPriorityUrgent),
		Tags:     &[]string{" 重要 ", "", "工作"},
	})
	require.NoError(t, err)
	assert.Equal(t, "新标题", task.Title)
	assert.Equal(t, model.TaskPriorityUrgent, task.Priority)
	assert.Equal(t, []string{"重要", "工作"}, task.Tags)
}

func TestUpdate_TenantAssigneeCanEdit(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", AssigneeID: "u2", Visibility: model.TaskVisibilityTenant}
	_, err := svc.Update(context.Background(), "t1", "u2", "id1", dto.UpdateTaskReq{Description: ptr("负责人补充")})
	require.NoError(t, err)
}

func TestUpdate_StatusToCompletedSetsCompletedAt(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityPersonal}
	task, err := svc.Update(context.Background(), "t1", "u1", "id1", dto.UpdateTaskReq{Status: ptr(model.TaskStatusCompleted)})
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusCompleted, task.Status)
	assert.NotNil(t, task.CompletedAt)
}

// ---- Complete / Reopen ----

func TestComplete_ByCreator(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending}
	task, err := svc.Complete(context.Background(), "t1", "u1", "id1")
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusCompleted, task.Status)
	assert.NotNil(t, task.CompletedAt)
}

func TestComplete_ForbiddenForNonOwner(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityPersonal}
	_, err := svc.Complete(context.Background(), "t1", "u9", "id1")
	assert.ErrorIs(t, err, ErrTaskForbidden)
}

func TestReopen_DefaultPending(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusCompleted}
	task, err := svc.Reopen(context.Background(), "t1", "u1", "id1", "")
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusPending, task.Status)
	assert.Nil(t, task.CompletedAt)
}

func TestReopen_ToInProgress(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusCompleted}
	task, err := svc.Reopen(context.Background(), "t1", "u1", "id1", model.TaskStatusInProgress)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusInProgress, task.Status)
}

// ---- Delete / 更严格归属 ----

func TestDelete_ByCreator(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", AssigneeID: "u2", Visibility: model.TaskVisibilityTenant}
	require.NoError(t, svc.Delete(context.Background(), "t1", "u1", "id1"))
	assert.NotContains(t, repo.tasks, "id1")
}

func TestDelete_AsseeForbidden(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", AssigneeID: "u2", Visibility: model.TaskVisibilityTenant}
	err := svc.Delete(context.Background(), "t1", "u2", "id1")
	assert.ErrorIs(t, err, ErrTaskForbidden)
}

func TestDelete_SuperAdminAllowed(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	ctx := contextx.SetVars(context.Background(), "t1", "u9", []string{"superadmin"})
	require.NoError(t, svc.Delete(ctx, "t1", "u9", "id1"))
}

// ---- List / Stats 透传 ----

func TestList_Passthrough(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", Status: model.TaskStatusPending}
	tasks, total, err := svc.List(context.Background(), repository.ListFilter{TenantID: "t1", UserID: "u1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, tasks, 1)
}

func TestBoard_TenantOnlyNotTruncated(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", Status: model.TaskStatusPending}
	repo.tasks["id2"] = &model.Task{ID: "id2", TenantID: "t1", Status: model.TaskStatusCompleted}
	repo.tasks["other"] = &model.Task{ID: "other", TenantID: "t2", Status: model.TaskStatusPending}
	tasks, total, truncated, err := svc.Board(context.Background(), repository.ListFilter{TenantID: "t1", UserID: "u1"})
	require.NoError(t, err)
	// 仅返回本租户任务，跨租户不可见
	assert.Equal(t, int64(2), total)
	assert.Len(t, tasks, 2)
	// 桩内 total==len，未触及 BoardMax，故不截断
	assert.False(t, truncated)
}

func TestStats_CountsByStatus(t *testing.T) {
	svc, repo := newService()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", Status: model.TaskStatusPending}
	repo.tasks["b"] = &model.Task{ID: "b", TenantID: "t1", Status: model.TaskStatusCompleted}
	stats, err := svc.Stats(context.Background(), repository.ListFilter{TenantID: "t1", UserID: "u1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.Pending)
	assert.Equal(t, int64(1), stats.Completed)
	assert.Equal(t, int64(2), stats.Total)
}

// ---- 辅助函数 ----

func TestParseDueDate_Formats(t *testing.T) {
	empty, err := parseDueDate("")
	require.NoError(t, err)
	assert.Nil(t, empty)

	for _, layout := range []string{"2026-01-02T15:04:05Z", "2026-01-02 15:04:05", "2026-01-02"} {
		got, err := parseDueDate(layout)
		require.NoError(t, err, "layout=%s", layout)
		assert.NotNil(t, got)
	}

	_, err = parseDueDate("garbage")
	assert.ErrorIs(t, err, ErrInvalidDueDate)
}

func TestNormalizeTags(t *testing.T) {
	assert.Nil(t, normalizeTags([]string{"  ", ""}))
	assert.Equal(t, []string{"a", "b"}, normalizeTags([]string{" a", "b ", ""}))
}

func TestTaskNotFoundMapping(t *testing.T) {
	svc, _ := newService()
	_, err := svc.Get(context.Background(), "t1", "u1", "missing")
	assert.True(t, errors.Is(err, ErrTaskNotFound))
}

// ---- 回收站 / Restore / PermanentDelete ----

func TestListTrash_OnlyOwnDeleted(t *testing.T) {
	svc, repo := newService()
	now := time.Now()
	repo.deleted["d1"] = &model.Task{ID: "d1", TenantID: "t1", CreatorID: "u1", DeletedAt: &now}
	repo.deleted["d2"] = &model.Task{ID: "d2", TenantID: "t1", CreatorID: "u2", DeletedAt: &now}
	tasks, total, err := svc.ListTrash(context.Background(), repository.ListFilter{TenantID: "t1", UserID: "u1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "d1", tasks[0].ID)
}

func TestRestore_ByCreator(t *testing.T) {
	svc, repo := newService()
	now := time.Now()
	repo.deleted["d1"] = &model.Task{ID: "d1", TenantID: "t1", CreatorID: "u1", DeletedAt: &now}
	task, err := svc.Restore(context.Background(), "t1", "u1", "d1")
	require.NoError(t, err)
	assert.Equal(t, "d1", task.ID)
	assert.Nil(t, task.DeletedAt)
	assert.Contains(t, repo.tasks, "d1")
	assert.NotContains(t, repo.deleted, "d1")
}

func TestRestore_ForbiddenForNonCreator(t *testing.T) {
	svc, repo := newService()
	now := time.Now()
	repo.deleted["d1"] = &model.Task{ID: "d1", TenantID: "t1", CreatorID: "u1", DeletedAt: &now}
	_, err := svc.Restore(context.Background(), "t1", "u9", "d1")
	assert.ErrorIs(t, err, ErrTaskForbidden)
}

func TestRestore_NotFound(t *testing.T) {
	svc, _ := newService()
	_, err := svc.Restore(context.Background(), "t1", "u1", "missing")
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestPermanentDelete_ByCreator(t *testing.T) {
	svc, repo := newService()
	now := time.Now()
	repo.deleted["d1"] = &model.Task{ID: "d1", TenantID: "t1", CreatorID: "u1", DeletedAt: &now}
	require.NoError(t, svc.PermanentDelete(context.Background(), "t1", "u1", "d1"))
	assert.NotContains(t, repo.deleted, "d1")
}

func TestPermanentDelete_Forbidden(t *testing.T) {
	svc, repo := newService()
	now := time.Now()
	repo.deleted["d1"] = &model.Task{ID: "d1", TenantID: "t1", CreatorID: "u1", DeletedAt: &now}
	assert.ErrorIs(t, svc.PermanentDelete(context.Background(), "t1", "u9", "d1"), ErrTaskForbidden)
}

// ---- 批量操作 ----

func TestBatchComplete_CountsOnlyEditable(t *testing.T) {
	svc, repo := newService()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending}
	repo.tasks["b"] = &model.Task{ID: "b", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending}
	repo.tasks["x"] = &model.Task{ID: "x", TenantID: "t1", CreatorID: "u2", Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending}
	n, err := svc.BatchComplete(context.Background(), "t1", "u1", []string{"a", "b", "x", "missing"})
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, model.TaskStatusCompleted, repo.tasks["a"].Status)
	assert.Equal(t, model.TaskStatusCompleted, repo.tasks["b"].Status)
	assert.Equal(t, model.TaskStatusPending, repo.tasks["x"].Status)
}

func TestBatchDelete_CountsOnlyOwn(t *testing.T) {
	svc, repo := newService()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1"}
	repo.tasks["x"] = &model.Task{ID: "x", TenantID: "t1", CreatorID: "u2"}
	n, err := svc.BatchDelete(context.Background(), "t1", "u1", []string{"a", "x"})
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.NotContains(t, repo.tasks, "a")
	assert.Contains(t, repo.tasks, "x")
}

// ---- 到期提醒 ----

func TestSendDueReminders_OverdueAndDueSoon(t *testing.T) {
	svc, repo := newService()
	now := time.Now()
	// 已逾期（昨天）/ 即将到期（2小时后）均应提醒
	repo.tasks["overdue"] = &model.Task{ID: "overdue", TenantID: "t1", CreatorID: "u1", AssigneeID: "u2",
		Status: model.TaskStatusPending, DueDate: ptr(now.Add(-24 * time.Hour))}
	repo.tasks["soon"] = &model.Task{ID: "soon", TenantID: "t1", CreatorID: "u1",
		Status: model.TaskStatusInProgress, DueDate: ptr(now.Add(2 * time.Hour))}
	// 已完成不提醒；未来任务不提醒；已提醒过的不重复提醒
	repo.tasks["done"] = &model.Task{ID: "done", TenantID: "t1", CreatorID: "u1",
		Status: model.TaskStatusCompleted, DueDate: ptr(now.Add(-time.Hour))}
	repo.tasks["future"] = &model.Task{ID: "future", TenantID: "t1", CreatorID: "u1",
		Status: model.TaskStatusPending, DueDate: ptr(now.Add(48 * time.Hour))}
	repo.tasks["reminded"] = &model.Task{ID: "reminded", TenantID: "t1", CreatorID: "u1",
		Status: model.TaskStatusPending, DueDate: ptr(now.Add(time.Hour)), ReminderSentAt: ptr(now.Add(-time.Minute))}

	n, err := svc.SendDueReminders(context.Background(), now.Add(24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.ElementsMatch(t, []string{"overdue", "soon"}, repo.marked)
	// 发送后应标记已提醒，下轮扫描不再重复
	n2, err := svc.SendDueReminders(context.Background(), now.Add(24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 0, n2)
}

func TestSendDueReminders_NoneDue(t *testing.T) {
	svc, repo := newService()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1", Status: model.TaskStatusPending}
	n, err := svc.SendDueReminders(context.Background(), time.Now().Add(24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Empty(t, repo.marked)
}

func TestUpdate_DueDateChangeResetsReminder(t *testing.T) {
	svc, repo := newService()
	now := time.Now()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1", AssigneeID: "u1",
		Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending,
		DueDate: ptr(now.Add(-time.Hour)), ReminderSentAt: ptr(now.Add(-2 * time.Hour))}
	newDue := now.Add(72 * time.Hour).Format("2006-01-02 15:04:05")
	task, err := svc.Update(context.Background(), "t1", "u1", "a", dto.UpdateTaskReq{DueDate: &newDue})
	require.NoError(t, err)
	// 截止日变更后提醒标记被重置，新截止日可再次触发提醒
	assert.Nil(t, task.ReminderSentAt)
	assert.Nil(t, repo.tasks["a"].ReminderSentAt)
}

func TestCreate_TenantAssigneeNotifyNilSafe(t *testing.T) {
	// 全局通知管理器未初始化（测试环境），指派给他人不应报错
	svc, _ := newService()
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title:      "协作任务",
		Visibility: model.TaskVisibilityTenant,
		AssigneeID: "u2",
	})
	require.NoError(t, err)
	assert.Equal(t, "u2", task.AssigneeID)
}

// ---- 子任务 ParentID ----

func TestCreate_ChildInheritsParentVisibility(t *testing.T) {
	svc, repo := newService()
	repo.tasks["p"] = &model.Task{ID: "p", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	child, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "子任务", ParentID: "p",
	})
	require.NoError(t, err)
	assert.Equal(t, "p", child.ParentID)
	// 可见范围继承父任务
	assert.Equal(t, model.TaskVisibilityTenant, child.Visibility)
}

func TestCreate_ChildPersonalKeepsCreatorAssignee(t *testing.T) {
	svc, repo := newService()
	repo.tasks["p"] = &model.Task{ID: "p", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityPersonal}
	child, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "私密子项", ParentID: "p",
	})
	require.NoError(t, err)
	assert.Equal(t, model.TaskVisibilityPersonal, child.Visibility)
	// 个人子任务 creator/assignee 仍为当前用户
	assert.Equal(t, "u1", child.CreatorID)
	assert.Equal(t, "u1", child.AssigneeID)
}

func TestCreate_ChildParentNotFound(t *testing.T) {
	svc, _ := newService()
	_, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "x", ParentID: "missing",
	})
	assert.ErrorIs(t, err, ErrParentNotFound)
}

func TestCreate_ChildCrossTenantParentDenied(t *testing.T) {
	svc, repo := newService()
	repo.tasks["p"] = &model.Task{ID: "p", TenantID: "t2", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	_, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "x", ParentID: "p",
	})
	assert.ErrorIs(t, err, ErrParentNotFound)
}

func TestCreate_ChildRejectsParentBeingChild(t *testing.T) {
	// 仅允许单层：父任务自身已是子任务时拒绝
	svc, repo := newService()
	repo.tasks["gp"] = &model.Task{ID: "gp", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	repo.tasks["p"] = &model.Task{ID: "p", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant, ParentID: "gp"}
	_, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "x", ParentID: "p",
	})
	assert.ErrorIs(t, err, ErrInvalidParent)
}

func TestUpdate_ReparentSelfReferenceRejected(t *testing.T) {
	svc, repo := newService()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	_, err := svc.Update(context.Background(), "t1", "u1", "a", dto.UpdateTaskReq{ParentID: ptr("a")})
	assert.ErrorIs(t, err, ErrInvalidParent)
}

func TestUpdate_ReparentToGrandparentRejected(t *testing.T) {
	// 目标父自身是子任务 → 会形成两层，拒绝
	svc, repo := newService()
	repo.tasks["p"] = &model.Task{ID: "p", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant, ParentID: "gp"}
	repo.tasks["gp"] = &model.Task{ID: "gp", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	_, err := svc.Update(context.Background(), "t1", "u1", "a", dto.UpdateTaskReq{ParentID: ptr("p")})
	assert.ErrorIs(t, err, ErrInvalidParent)
}

func TestUpdate_ClearParentMakesRoot(t *testing.T) {
	svc, repo := newService()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant, ParentID: "p"}
	task, err := svc.Update(context.Background(), "t1", "u1", "a", dto.UpdateTaskReq{ParentID: ptr("")})
	require.NoError(t, err)
	assert.Equal(t, "", task.ParentID)
}

func TestDelete_CascadesChildren(t *testing.T) {
	svc, repo := newService()
	repo.tasks["p"] = &model.Task{ID: "p", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityPersonal}
	repo.tasks["c1"] = &model.Task{ID: "c1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityPersonal, ParentID: "p"}
	repo.tasks["c2"] = &model.Task{ID: "c2", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityPersonal, ParentID: "p"}
	require.NoError(t, svc.Delete(context.Background(), "t1", "u1", "p"))
	// 父与其全部子任务均进入回收站
	assert.NotContains(t, repo.tasks, "p")
	assert.NotContains(t, repo.tasks, "c1")
	assert.NotContains(t, repo.tasks, "c2")
	assert.Contains(t, repo.deleted, "c1")
	assert.Contains(t, repo.deleted, "c2")
}

// ---- 开始时间 StartedAt ----

func TestCreate_InProgressSetsStartedAt(t *testing.T) {
	svc, _ := newService()
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "立即开始", Status: model.TaskStatusInProgress,
	})
	require.NoError(t, err)
	assert.NotNil(t, task.StartedAt)
}

func TestCreate_PendingHasNoStartedAt(t *testing.T) {
	svc, _ := newService()
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{Title: "待办"})
	require.NoError(t, err)
	assert.Nil(t, task.StartedAt)
}

func TestUpdate_PendingToInProgressSetsStartedAt(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1",
		Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending}
	task, err := svc.Update(context.Background(), "t1", "u1", "id1",
		dto.UpdateTaskReq{Status: ptr(model.TaskStatusInProgress)})
	require.NoError(t, err)
	assert.NotNil(t, task.StartedAt)
}

func TestStatusChange_StartedAtPreservedAcrossReopen(t *testing.T) {
	// 已开始的任务完成后重新打开为进行中，应保留最初开始时间，不被覆盖
	svc, repo := newService()
	started := time.Now().Add(-5 * time.Hour)
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1",
		Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusCompleted, StartedAt: &started}
	task, err := svc.Reopen(context.Background(), "t1", "u1", "id1", model.TaskStatusInProgress)
	require.NoError(t, err)
	require.NotNil(t, task.StartedAt)
	assert.True(t, task.StartedAt.Equal(started), "开始时间应保持不变")
}

// ---- 姓名富化 ----

// stubNames 内存版用户名解析器，实现 UserNameResolver。
type stubNames struct{ m map[string]string }

func (s stubNames) ResolveUserNames(_ context.Context, ids []string) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	for _, id := range ids {
		if n, ok := s.m[id]; ok {
			out[id] = n
		}
	}
	return out, nil
}

func TestGet_EnrichesNames(t *testing.T) {
	repo := newStubTaskRepo()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", AssigneeID: "u2",
		Visibility: model.TaskVisibilityTenant, Status: model.TaskStatusPending}
	svc := NewTaskServiceWithNames(repo, stubNames{map[string]string{"u1": "张三", "u2": "李四"}})
	task, err := svc.Get(context.Background(), "t1", "u1", "id1")
	require.NoError(t, err)
	assert.Equal(t, "张三", task.CreatorName)
	assert.Equal(t, "李四", task.AssigneeName)
}

func TestList_EnrichesNames(t *testing.T) {
	repo := newStubTaskRepo()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", AssigneeID: "u1", Status: model.TaskStatusPending}
	svc := NewTaskServiceWithNames(repo, stubNames{map[string]string{"u1": "王五"}})
	tasks, _, err := svc.List(context.Background(), repository.ListFilter{TenantID: "t1", UserID: "u1"})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "王五", tasks[0].CreatorName)
}

func TestNewTaskService_NamesNilSafe(t *testing.T) {
	// 未注入解析器时，富化应为空且不报错
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", AssigneeID: "u1",
		Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending}
	task, err := svc.Get(context.Background(), "t1", "u1", "id1")
	require.NoError(t, err)
	assert.Empty(t, task.CreatorName)
}

func TestToResp_IncludesNamesAndStartedAt(t *testing.T) {
	now := time.Now()
	task := &model.Task{ID: "x", TenantID: "t1", CreatorID: "u1", CreatorName: "张三",
		AssigneeID: "u2", AssigneeName: "李四", Visibility: model.TaskVisibilityTenant,
		Status: model.TaskStatusInProgress, StartedAt: &now, CreatedAt: now, UpdatedAt: now}
	resp := ToResp(task)
	assert.Equal(t, "张三", resp.CreatorName)
	assert.Equal(t, "李四", resp.AssigneeName)
	assert.NotEmpty(t, resp.StartedAt)
}

// ---- 批量改状态 / 批量指派 ----

func TestBatchUpdateStatus_InProgress(t *testing.T) {
	svc, repo := newService()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending}
	repo.tasks["x"] = &model.Task{ID: "x", TenantID: "t1", CreatorID: "u2", Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending}
	n, err := svc.BatchUpdateStatus(context.Background(), "t1", "u1", []string{"a", "x", "missing"}, model.TaskStatusInProgress)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, model.TaskStatusInProgress, repo.tasks["a"].Status)
	assert.NotNil(t, repo.tasks["a"].StartedAt) // 进入进行中应记录开始时间
}

func TestBatchUpdateStatus_InvalidStatus(t *testing.T) {
	svc, _ := newService()
	_, err := svc.BatchUpdateStatus(context.Background(), "t1", "u1", []string{"a"}, "bogus")
	assert.ErrorIs(t, err, ErrInvalidStatus)
}

func TestBatchAssign_CountsOnlyEditable(t *testing.T) {
	svc, repo := newService()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1", AssigneeID: "u1", Visibility: model.TaskVisibilityTenant, Status: model.TaskStatusPending}
	repo.tasks["x"] = &model.Task{ID: "x", TenantID: "t1", CreatorID: "u2", AssigneeID: "u2", Visibility: model.TaskVisibilityTenant, Status: model.TaskStatusPending}
	n, err := svc.BatchAssign(context.Background(), "t1", "u1", []string{"a", "x"}, []string{"u3"})
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, "u3", repo.tasks["a"].AssigneeID)
	assert.Equal(t, "u2", repo.tasks["x"].AssigneeID) // 无权项保持不变
}

// ---- 逾期重复提醒 ----

func TestSendDueReminders_OverdueReReminderAfterCooldown(t *testing.T) {
	svc, repo := newService()
	now := time.Now()
	// 已逾期且上次提醒超过 24h 冷却 → 再次提醒
	repo.tasks["stale"] = &model.Task{ID: "stale", TenantID: "t1", CreatorID: "u1", AssigneeID: "u2",
		Status: model.TaskStatusPending, DueDate: ptr(now.Add(-time.Hour)), ReminderSentAt: ptr(now.Add(-25 * time.Hour))}
	// 已逾期但刚提醒过（冷却内）→ 不重复
	repo.tasks["fresh"] = &model.Task{ID: "fresh", TenantID: "t1", CreatorID: "u1",
		Status: model.TaskStatusPending, DueDate: ptr(now.Add(-time.Hour)), ReminderSentAt: ptr(now.Add(-time.Minute))}
	n, err := svc.SendDueReminders(context.Background(), now.Add(24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.ElementsMatch(t, []string{"stale"}, repo.marked)
}

// 冷却期可配：配置为 1h 时，距上次提醒 2h 的逾期任务应重复提醒（默认 24h 下则不会）
func TestSendDueReminders_ConfigurableCooldown(t *testing.T) {
	svc, repo := newService()
	now := time.Now()
	repo.tasks["overdue"] = &model.Task{ID: "overdue", TenantID: "t1", CreatorID: "u1", AssigneeID: "u2",
		Status: model.TaskStatusPending, DueDate: ptr(now.Add(-time.Hour)), ReminderSentAt: ptr(now.Add(-2 * time.Hour))}
	// 默认冷却 24h：2h 前的提醒仍应处于冷却内，不重复
	n0, err := svc.SendDueReminders(context.Background(), now.Add(24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 0, n0)
	// 自定义冷却 1h：超过冷却应重复提醒
	n1, err := svc.SendDueReminders(context.Background(), now.Add(24*time.Hour), ReminderOptions{OverdueCooldown: time.Hour})
	require.NoError(t, err)
	assert.Equal(t, 1, n1)
	assert.ElementsMatch(t, []string{"overdue"}, repo.marked)
}

// TestUpdate_DueDateChangeResetsReminder_EndToEnd 改期后应重置提醒标记：首次提醒已发出的任务若被
// 推延到新截止日，不重置则 reminder_sent_at 非空使新截止日永远发不出首提醒（只有再次逾期过冷却才会提醒）。
// 重置语义与修改 remind_before 时保持对称。基础重置行为另见 TestUpdate_DueDateChangeResetsReminder。
func TestUpdate_DueDateChangeResetsReminder_EndToEnd(t *testing.T) {
	svc, repo := newService()
	now := time.Now()
	oldDue := now.Add(24 * time.Hour)
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1", AssigneeID: "u2",
		Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending,
		DueDate: &oldDue, ReminderSentAt: ptr(now.Add(-time.Minute))}

	newDue := now.Add(72 * time.Hour)
	task, err := svc.Update(context.Background(), "t1", "u1", "a",
		dto.UpdateTaskReq{DueDate: ptr(newDue.Format("2006-01-02 15:04:05"))})
	require.NoError(t, err)

	assert.Nil(t, task.ReminderSentAt)
	// 端到端：下一个扫描周期应能向新截止日发出首提醒
	n, err := svc.SendDueReminders(context.Background(), now.Add(168*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, n, "改期后的新截止日应重新进入提醒范围")
	assert.ElementsMatch(t, []string{"a"}, repo.marked)
}

// TestUpdate_UnchangedDueDateKeepsReminderCooldown 冷却不被整体提交绕过：
// 前端保存往往原样回传 dueDate，若因此抹掉 reminder_sent_at，下一轮扫描会立即重发提醒，
// 冷却形同虚设（反复保存 = 反复骚扰）。
func TestUpdate_UnchangedDueDateKeepsReminderCooldown(t *testing.T) {
	svc, repo := newService()
	now := time.Now()
	// 截止日带毫秒精度，而接口回传会被截断到秒（ToResp 使用秒级格式）
	due := now.Add(24 * time.Hour).Truncate(time.Millisecond)
	remindedAt := now.Add(-time.Minute)
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1", AssigneeID: "u2",
		Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending,
		DueDate: &due, ReminderSentAt: &remindedAt}

	// 原样回传同一截止日（秒精度）+ 只改标题
	task, err := svc.Update(context.Background(), "t1", "u1", "a",
		dto.UpdateTaskReq{Title: ptr("只改了标题"), DueDate: ptr(due.Format("2006-01-02 15:04:05"))})
	require.NoError(t, err)

	require.NotNil(t, task.ReminderSentAt, "截止日未实质变化时不得重置已提醒标记")
	assert.True(t, task.ReminderSentAt.Equal(remindedAt))

	// 关键用户可见后果：冷却内不得重复发出提醒
	n, err := svc.SendDueReminders(context.Background(), now.Add(168*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 0, n, "原样回传 dueDate 不得导致冷却被绕过而重复提醒")
}

// TestUpdate_UnchangedRemindBeforeKeepsReminderCooldown 同理：remind_before 原值回传也不得重置冷却。
func TestUpdate_UnchangedRemindBeforeKeepsReminderCooldown(t *testing.T) {
	svc, repo := newService()
	now := time.Now()
	due := now.Add(24 * time.Hour)
	lead := 30 * time.Minute
	remindedAt := now.Add(-time.Minute)
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1", AssigneeID: "u2",
		Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending,
		DueDate: &due, RemindBefore: &lead, ReminderSentAt: &remindedAt}

	task, err := svc.Update(context.Background(), "t1", "u1", "a",
		dto.UpdateTaskReq{RemindBefore: ptr("30m")})
	require.NoError(t, err)

	require.NotNil(t, task.ReminderSentAt, "提前量未实质变化时不得重置已提醒标记")
	n, err := svc.SendDueReminders(context.Background(), now.Add(168*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 0, n, "原样回传 remind_before 不得重复提醒")

	// 对照：真的修改提前量时仍应重置（保持原有语义）
	task2, err := svc.Update(context.Background(), "t1", "u1", "a",
		dto.UpdateTaskReq{RemindBefore: ptr("2h")})
	require.NoError(t, err)
	assert.Nil(t, task2.ReminderSentAt, "提前量实质变化时应重置提醒标记")
}

// ---- 任务级提醒提前量 remind_before ----

func TestParseRemindLead(t *testing.T) {
	// 空值→nil（使用全局默认）
	d, err := parseRemindLead("")
	require.NoError(t, err)
	assert.Nil(t, d)
	// 合法 duration
	d, err = parseRemindLead("2h30m")
	require.NoError(t, err)
	require.NotNil(t, d)
	assert.Equal(t, 150*time.Minute, *d)
	// 非法格式/负值/超范围均报错
	for _, bad := range []string{"abc", "-1h", "32d", "1000h"} {
		_, err := parseRemindLead(bad)
		assert.ErrorIs(t, err, ErrInvalidRemindLead, bad)
	}
}

func TestCreate_WithRemindLead(t *testing.T) {
	svc, _ := newService()
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "带提前量", DueDate: "2026-12-31 10:00:00", RemindBefore: "3h",
	})
	require.NoError(t, err)
	require.NotNil(t, task.RemindBefore)
	assert.Equal(t, 3*time.Hour, *task.RemindBefore)
	resp := ToResp(task)
	assert.Equal(t, "3h0m0s", resp.RemindBefore)
}

func TestCreate_InvalidRemindLead(t *testing.T) {
	svc, _ := newService()
	_, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "非法提前量", DueDate: "2026-12-31 10:00:00", RemindBefore: "bogus",
	})
	assert.ErrorIs(t, err, ErrInvalidRemindLead)
}

func TestUpdate_SetAndClearRemindLead(t *testing.T) {
	svc, repo := newService()
	now := time.Now()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1",
		Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending,
		DueDate: ptr(now.Add(time.Hour)), ReminderSentAt: ptr(now.Add(-time.Minute))}
	// 设置提前量应写回并重置提醒标记
	task, err := svc.Update(context.Background(), "t1", "u1", "a",
		dto.UpdateTaskReq{RemindBefore: ptr("30m")})
	require.NoError(t, err)
	require.NotNil(t, task.RemindBefore)
	assert.Equal(t, 30*time.Minute, *task.RemindBefore)
	assert.Nil(t, task.ReminderSentAt)
	// 传空字符串清除自定义提前量，回落全局默认
	task2, err := svc.Update(context.Background(), "t1", "u1", "a",
		dto.UpdateTaskReq{RemindBefore: ptr("")})
	require.NoError(t, err)
	assert.Nil(t, task2.RemindBefore)
}

// ---- 负责人同租户校验 ----

// stubDirectory 同时实现 UserNameResolver 与 TenantMemberChecker，模拟用户目录。
// members 为租户成员集；不在集内且非创建人的用户视为非法负责人。
type stubDirectory struct {
	names   map[string]string
	members map[string]bool
}

func (d stubDirectory) ResolveUserNames(_ context.Context, ids []string) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	for _, id := range ids {
		if n, ok := d.names[id]; ok {
			out[id] = n
		}
	}
	return out, nil
}

func (d stubDirectory) IsTenantMember(_ context.Context, _, userID string) (bool, error) {
	return d.members[userID], nil
}

func newServiceWithDirectory(members map[string]bool) (*TaskService, *stubTaskRepo) {
	repo := newStubTaskRepo()
	return NewTaskServiceWithNames(repo, stubDirectory{names: map[string]string{}, members: members}), repo
}

func TestCreate_TenantAssigneeNotMember(t *testing.T) {
	svc, _ := newServiceWithDirectory(map[string]bool{"u1": true})
	_, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "协作", Visibility: model.TaskVisibilityTenant, AssigneeID: "u9",
	})
	assert.ErrorIs(t, err, ErrAssigneeInvalid)
}

func TestCreate_TenantAssigneeMember(t *testing.T) {
	svc, _ := newServiceWithDirectory(map[string]bool{"u1": true, "u2": true})
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "协作", Visibility: model.TaskVisibilityTenant, AssigneeID: "u2",
	})
	require.NoError(t, err)
	assert.Equal(t, "u2", task.AssigneeID)
}

func TestUpdate_AssigneeNotMember(t *testing.T) {
	svc, repo := newServiceWithDirectory(map[string]bool{"u1": true})
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1",
		Visibility: model.TaskVisibilityTenant, Status: model.TaskStatusPending, AssigneeID: "u1"}
	_, err := svc.Update(context.Background(), "t1", "u1", "a",
		dto.UpdateTaskReq{AssigneeID: ptr("u9")})
	assert.ErrorIs(t, err, ErrAssigneeInvalid)
}

func TestBatchAssign_InvalidAssigneeShortCircuits(t *testing.T) {
	svc, repo := newServiceWithDirectory(map[string]bool{"u1": true})
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1",
		Visibility: model.TaskVisibilityTenant, Status: model.TaskStatusPending, AssigneeID: "u1"}
	n, err := svc.BatchAssign(context.Background(), "t1", "u1", []string{"a"}, []string{"u9"})
	assert.ErrorIs(t, err, ErrAssigneeInvalid)
	assert.Equal(t, 0, n)
	assert.Equal(t, "u1", repo.tasks["a"].AssigneeID) // 非法指派不生效
}

// ---- B4 多负责人指派 ----

// TestCreate_MultipleAssignees 集合为准：Assignees 非空时主负责人=首位、集合写入关联表，
// 且响应同时携带 assignees 与富化后的 assignee_names。
func TestCreate_MultipleAssignees(t *testing.T) {
	repo := newStubTaskRepo()
	svc := NewTaskServiceWithNames(repo, stubDirectory{
		names:   map[string]string{"u2": "李四", "u3": "王五"},
		members: map[string]bool{"u1": true, "u2": true, "u3": true},
	})
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "协作", Visibility: model.TaskVisibilityTenant, Assignees: []string{"u2", "u3", "u2"},
	})
	require.NoError(t, err)
	assert.Equal(t, "u2", task.AssigneeID)                    // 主负责人=首位
	assert.Equal(t, []string{"u2", "u3"}, task.AssigneeIDs)   // 去重后集合
	assert.Equal(t, []string{"u2", "u3"}, repo.assignees[task.ID]) // 已写入关联表
	resp := ToResp(task)
	assert.Equal(t, []string{"u2", "u3"}, resp.Assignees)
	assert.Equal(t, []string{"李四", "王五"}, resp.AssigneeNames)
}

// TestCreate_PersonalForcesCreator 个人任务忽略传入集合，强制仅创建人本人。
func TestCreate_PersonalForcesCreator(t *testing.T) {
	svc, repo := newService()
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "个人", Visibility: model.TaskVisibilityPersonal, Assignees: []string{"u2", "u3"},
	})
	require.NoError(t, err)
	assert.Equal(t, "u1", task.AssigneeID)
	assert.Equal(t, []string{"u1"}, repo.assignees[task.ID])
}

// TestCreate_AssigneesFallsBackToAssigneeID 未给集合时回落单个 assignee_id。
func TestCreate_AssigneesFallsBackToAssigneeID(t *testing.T) {
	svc, _ := newService()
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "协作", Visibility: model.TaskVisibilityTenant, AssigneeID: "u2",
	})
	require.NoError(t, err)
	assert.Equal(t, "u2", task.AssigneeID)
	assert.Equal(t, []string{"u2"}, task.AssigneeIDs)
}

// TestCanView_CanEdit_SecondaryAssignee 命中非主负责人：副负责人可查看/编辑租户任务，
// 但个人任务不因副负责人跨租户泄露。
func TestCanView_CanEdit_SecondaryAssignee(t *testing.T) {
	tenantTask := &model.Task{TenantID: "t1", CreatorID: "u1", AssigneeID: "u2",
		AssigneeIDs: []string{"u2", "u3"}, Visibility: model.TaskVisibilityTenant}
	assert.True(t, canView(tenantTask, "t1", "u3"))
	assert.True(t, canEdit(tenantTask, "u3"))
	assert.False(t, canEdit(tenantTask, "u9"))

	personalTask := &model.Task{TenantID: "t1", CreatorID: "u1", AssigneeID: "u2",
		AssigneeIDs: []string{"u2", "u3"}, Visibility: model.TaskVisibilityPersonal}
	assert.True(t, canView(personalTask, "t1", "u3"))   // 任一负责人可见自己的个人任务
	assert.False(t, canView(personalTask, "t1", "u9"))  // 非集合成员不可见
	assert.False(t, canEdit(personalTask, "u3"))        // 个人任务仅创建人可编辑
}

// TestGet_FillsAssigneesFromRepo 单条读取应从关联表回填集合，使基于副负责人的鉴权生效。
func TestGet_FillsAssigneesFromRepo(t *testing.T) {
	svc, repo := newService()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1", AssigneeID: "u2",
		Visibility: model.TaskVisibilityTenant, Status: model.TaskStatusPending}
	repo.assignees["a"] = []string{"u2", "u3"}
	task, err := svc.Get(context.Background(), "t1", "u3", "a")
	require.NoError(t, err)
	assert.Equal(t, []string{"u2", "u3"}, task.AssigneeIDs)
}

// TestUpdate_ReplaceAssigneesCollection 以集合为准回写关联表，主负责人随集合首位更新。
func TestUpdate_ReplaceAssigneesCollection(t *testing.T) {
	svc, repo := newService()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1", AssigneeID: "u2",
		AssigneeIDs: []string{"u2"}, Visibility: model.TaskVisibilityTenant, Status: model.TaskStatusPending}
	repo.assignees["a"] = []string{"u2"}
	task, err := svc.Update(context.Background(), "t1", "u1", "a",
		dto.UpdateTaskReq{Assignees: &[]string{"u3", "u4"}})
	require.NoError(t, err)
	assert.Equal(t, "u3", task.AssigneeID)
	assert.Equal(t, []string{"u3", "u4"}, repo.assignees["a"])
}

func TestNormalizeAssignees(t *testing.T) {
	// 个人任务强制创建人
	assert.Equal(t, []string{"u1"}, normalizeAssignees([]string{"u2", "u3"}, "", "u1", model.TaskVisibilityPersonal))
	// 去重去空，保持顺序
	assert.Equal(t, []string{"u2", "u3"}, normalizeAssignees([]string{"u2", "", "u2", "u3 "}, "u9", "u1", model.TaskVisibilityTenant))
	// 集合为空回落单个
	assert.Equal(t, []string{"u9"}, normalizeAssignees(nil, "u9", "u1", model.TaskVisibilityTenant))
	// 均空回落 userID
	assert.Equal(t, []string{"u1"}, normalizeAssignees([]string{"", ""}, "", "u1", model.TaskVisibilityTenant))
}

// ---- 周期/重复任务 ----

func TestNextOccurrence(t *testing.T) {
	base := time.Date(2026, 1, 10, 9, 0, 0, 0, time.Local)
	d, ok := model.NextOccurrence(model.TaskRecurrenceDaily, base)
	require.True(t, ok)
	assert.Equal(t, base.AddDate(0, 0, 1), d)
	d, ok = model.NextOccurrence(model.TaskRecurrenceWeekly, base)
	require.True(t, ok)
	assert.Equal(t, base.AddDate(0, 0, 7), d)
	d, ok = model.NextOccurrence(model.TaskRecurrenceMonthly, base)
	require.True(t, ok)
	assert.Equal(t, base.AddDate(0, 1, 0), d)
	// 空/非法周期不推进
	_, ok = model.NextOccurrence("", base)
	assert.False(t, ok)
}

// TestNextOccurrence_MonthEndClamp 锁定月末修复：1/31 之类的锚定日不得溢出到再下个月，
// 应钳制到目标月最后一天（与主流日历“每月最后一天”语义一致）。
func TestNextOccurrence_MonthEndClamp(t *testing.T) {
	loc := time.Local
	cases := []struct {
		name  string
		base  time.Time
		wantY int
		wantM time.Month
		wantD int
	}{
		// 2026 非闰年：1/31 → 2/28（而非旧逻辑的 3/3）
		{"Jan31_to_Feb28", time.Date(2026, 1, 31, 10, 30, 0, 0, loc), 2026, 2, 28},
		// 闰年：2028 1/31 → 2/29
		{"LeapJan31_to_Feb29", time.Date(2028, 1, 31, 10, 30, 0, 0, loc), 2028, 2, 29},
		// 3/31 → 4/30（4 月无 31 号）
		{"Mar31_to_Apr30", time.Date(2026, 3, 31, 0, 0, 0, 0, loc), 2026, 4, 30},
		// 1/30 → 2/28（而非旧逻辑的 3/2）
		{"Jan30_to_Feb28", time.Date(2026, 1, 30, 8, 0, 0, 0, loc), 2026, 2, 28},
		// 年份进位：2026/12/31 → 2027/1/31（1 月有 31 天，不钳制）
		{"Dec31_yearRollover", time.Date(2026, 12, 31, 23, 59, 59, 0, loc), 2027, 1, 31},
		// 普通日不受影响：1/10 → 2/10
		{"NormalDay", time.Date(2026, 1, 10, 9, 0, 0, 0, loc), 2026, 2, 10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, ok := model.NextOccurrence(model.TaskRecurrenceMonthly, c.base)
			require.True(t, ok)
			assert.Equal(t, c.wantY, d.Year())
			assert.Equal(t, c.wantM, d.Month())
			assert.Equal(t, c.wantD, d.Day())
			// 时分秒应原样保留（仅日期部分受钳制）
			assert.Equal(t, c.base.Hour(), d.Hour())
			assert.Equal(t, c.base.Minute(), d.Minute())
			assert.Equal(t, c.base.Second(), d.Second())
		})
	}
}

func TestCreate_RecurrenceInvalid(t *testing.T) {
	svc, _ := newService()
	_, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "周期", DueDate: "2026-12-31 10:00:00", Recurrence: "yearly",
	})
	assert.ErrorIs(t, err, ErrInvalidRecurrence)
}

func TestCreate_RecurrenceRequiresDueDate(t *testing.T) {
	svc, _ := newService()
	_, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "周期无截止", Recurrence: model.TaskRecurrenceDaily,
	})
	assert.ErrorIs(t, err, ErrRecurrenceNeedsDue)
}

func findPendingCopy(repo *stubTaskRepo, excludeID string) *model.Task {
	for id, t := range repo.tasks {
		if id != excludeID && t.Status == model.TaskStatusPending {
			return t
		}
	}
	return nil
}

func TestComplete_RecurrenceSpawnsNextDaily(t *testing.T) {
	svc, repo := newService()
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "每日站会", DueDate: "2026-12-31 10:00:00", Recurrence: model.TaskRecurrenceDaily,
	})
	require.NoError(t, err)
	origDue := *task.DueDate
	_, err = svc.Complete(context.Background(), "t1", "u1", task.ID)
	require.NoError(t, err)
	require.Len(t, repo.tasks, 2)
	next := findPendingCopy(repo, task.ID)
	require.NotNil(t, next)
	assert.Equal(t, "每日站会", next.Title)
	assert.Equal(t, model.TaskRecurrenceDaily, next.Recurrence)
	require.NotNil(t, next.DueDate)
	assert.True(t, next.DueDate.Equal(origDue.AddDate(0, 0, 1)))
}

func TestComplete_AlreadyCompletedDoesNotRespawn(t *testing.T) {
	svc, repo := newService()
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "周报", DueDate: "2026-12-31 10:00:00", Recurrence: model.TaskRecurrenceWeekly,
	})
	require.NoError(t, err)
	_, err = svc.Complete(context.Background(), "t1", "u1", task.ID)
	require.NoError(t, err)
	require.Len(t, repo.tasks, 2) // 首次完成生成 1 条
	_, err = svc.Complete(context.Background(), "t1", "u1", task.ID)
	require.NoError(t, err)
	assert.Len(t, repo.tasks, 2) // 重复完成不再生成
}

func TestUpdate_TransitionToCompletedSpawnsNext(t *testing.T) {
	svc, repo := newService()
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title: "月报", DueDate: "2026-01-31 10:00:00", Recurrence: model.TaskRecurrenceMonthly,
	})
	require.NoError(t, err)
	_, err = svc.Update(context.Background(), "t1", "u1", task.ID,
		dto.UpdateTaskReq{Status: ptr(model.TaskStatusCompleted)})
	require.NoError(t, err)
	require.Len(t, repo.tasks, 2)
	next := findPendingCopy(repo, task.ID)
	require.NotNil(t, next)
	require.NotNil(t, next.DueDate)
	// 2026-01-31 月重复：应钳制到 2 月末 2026-02-28（而非旧逻辑溢出的 03-03），时分秒保留
	assert.True(t, next.DueDate.Equal(time.Date(2026, 2, 28, 10, 0, 0, 0, time.Local)),
		"expect clamped Feb 28, got %s", next.DueDate.Format("2006-01-02 15:04:05"))
}

func TestUpdate_SetRecurrenceWithoutDueRejected(t *testing.T) {
	svc, repo := newService()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1",
		Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending}
	_, err := svc.Update(context.Background(), "t1", "u1", "a",
		dto.UpdateTaskReq{Recurrence: ptr(model.TaskRecurrenceDaily)})
	assert.ErrorIs(t, err, ErrRecurrenceNeedsDue)
}

// ---- Update / 前端整表回传回归（编辑保存链路） ----

// TestUpdate_FrontendFullPayloadNoDue 模拟前端编辑普通无重复任务的完整提交体：
// recurrence 为指针指向空串、due_date 未提交（nil）。旧版在 DTO 层就被 oneof 拒成 400，
// 此后 service 层也曾误报 ErrRecurrenceNeedsDue：空串 recurrence 必须解释为“不重复”而非非法。
func TestUpdate_FrontendFullPayloadNoDue(t *testing.T) {
	svc, repo := newService()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1",
		Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending}
	task, err := svc.Update(context.Background(), "t1", "u1", "a", dto.UpdateTaskReq{
		Title:      ptr("改名"),
		Visibility: ptr(model.TaskVisibilityPersonal),
		Priority:   ptr(model.TaskPriorityNormal),
		Recurrence: ptr(""),
		Tags:       &[]string{},
	})
	require.NoError(t, err)
	assert.Equal(t, "改名", task.Title)
	assert.Equal(t, "", task.Recurrence)
}

// TestUpdate_RecurringTaskEchoDueDateRoundTrip 回归：“截止时间 + 重复周期”任务改标题保存时，
// 前端会原样回传接口输出的 "2006-01-02 15:04:05" 日期，必须解析成功且不触发 ErrRecurrenceNeedsDue。
func TestUpdate_RecurringTaskEchoDueDateRoundTrip(t *testing.T) {
	svc, repo := newService()
	due := time.Date(2026, 3, 15, 9, 30, 0, 0, time.Local)
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1",
		Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending,
		DueDate: &due, Recurrence: model.TaskRecurrenceWeekly}
	task, err := svc.Update(context.Background(), "t1", "u1", "a", dto.UpdateTaskReq{
		Title:        ptr("改标题"),
		DueDate:      ptr(due.Format("2006-01-02 15:04:05")), // 接口输出格式原样回传
		RemindBefore: ptr("2h0m0s"),                          // Go Duration 回环
		Recurrence:   ptr(model.TaskRecurrenceWeekly),
	})
	require.NoError(t, err)
	assert.Equal(t, "改标题", task.Title)
	require.NotNil(t, task.DueDate)
	assert.True(t, task.DueDate.Equal(due), "原值回传不应改变截止日，got=%v", *task.DueDate)
	require.NotNil(t, task.RemindBefore)
	assert.Equal(t, 2*time.Hour, *task.RemindBefore)
}

// TestUpdate_CancelRecurrenceByEmptyString 验证“取消重复”语义：重复任务回传 recurrence:"" 且保留截止日，
// 应成功清空 Recurrence（DTO 注释承诺的空串语义）。
func TestUpdate_CancelRecurrenceByEmptyString(t *testing.T) {
	svc, repo := newService()
	due := time.Date(2026, 3, 15, 9, 30, 0, 0, time.Local)
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1",
		Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending,
		DueDate: &due, Recurrence: model.TaskRecurrenceDaily}
	task, err := svc.Update(context.Background(), "t1", "u1", "a", dto.UpdateTaskReq{
		Recurrence: ptr(""),
		DueDate:    ptr(due.Format("2006-01-02 15:04:05")),
	})
	require.NoError(t, err)
	assert.Equal(t, "", task.Recurrence, "空串应清空重复周期")
}

// TestUpdate_InvalidRecurrenceValueRejected 验证枚举合法性由 service 把关：
// 非空非法值（yearly）返回 ErrInvalidRecurrence（handler 映射 400）。
func TestUpdate_InvalidRecurrenceValueRejected(t *testing.T) {
	svc, repo := newService()
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1",
		Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending}
	_, err := svc.Update(context.Background(), "t1", "u1", "a",
		dto.UpdateTaskReq{Recurrence: ptr("yearly")})
	assert.ErrorIs(t, err, ErrInvalidRecurrence)
}

// TestUpdate_ClearDueDateOnRecurringRejected 验证守住“重复必带截止日”不变量：
// 重复任务清空截止日应被 ErrRecurrenceNeedsDue 拒绝，避免后续完成时无法推算下一周期。
func TestUpdate_ClearDueDateOnRecurringRejected(t *testing.T) {
	svc, repo := newService()
	due := time.Date(2026, 3, 15, 9, 30, 0, 0, time.Local)
	repo.tasks["a"] = &model.Task{ID: "a", TenantID: "t1", CreatorID: "u1",
		Visibility: model.TaskVisibilityPersonal, Status: model.TaskStatusPending,
		DueDate: &due, Recurrence: model.TaskRecurrenceMonthly}
	_, err := svc.Update(context.Background(), "t1", "u1", "a",
		dto.UpdateTaskReq{DueDate: ptr("")})
	assert.ErrorIs(t, err, ErrRecurrenceNeedsDue)
}
