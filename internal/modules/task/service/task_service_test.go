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
	tasks   map[string]*model.Task
	deleted map[string]*model.Task
	marked  []string
}

func newStubTaskRepo() *stubTaskRepo {
	return &stubTaskRepo{tasks: make(map[string]*model.Task), deleted: make(map[string]*model.Task)}
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
	n, err := svc.BatchAssign(context.Background(), "t1", "u1", []string{"a", "x"}, "u3")
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
