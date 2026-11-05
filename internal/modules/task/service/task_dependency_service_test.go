package service

import (
	"context"
	"testing"

	"meteorx/internal/modules/task/dto"
	"meteorx/internal/modules/task/model"
	"meteorx/internal/notify"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreate_WithBlockedBy 建锁成功后，任务读取回显前置集合，且前置反向显示后继（Blocks）。
func TestCreate_WithBlockedBy(t *testing.T) {
	svc, repo := newService()
	// 前置任务：租户可见，便于 canView
	repo.tasks["b1"] = &model.Task{ID: "b1", TenantID: "t1", CreatorID: "u1", AssigneeID: "u1", Title: "前置一", Status: model.TaskStatusPending, Visibility: model.TaskVisibilityTenant}

	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title:     "后继任务",
		BlockedBy: []string{"b1"},
	})
	require.NoError(t, err)

	// 读取后继：BlockedBy 含 b1（带标题/状态）
	got, err := svc.Get(context.Background(), "t1", "u1", task.ID)
	require.NoError(t, err)
	require.Len(t, got.BlockedBy, 1)
	assert.Equal(t, "b1", got.BlockedBy[0].ID)
	assert.Equal(t, "前置一", got.BlockedBy[0].Title)

	// 读取前置：Blocks 反向含新任务
	gotB, err := svc.Get(context.Background(), "t1", "u1", "b1")
	require.NoError(t, err)
	require.Len(t, gotB.Blocks, 1)
	assert.Equal(t, task.ID, gotB.Blocks[0].ID)
}

// TestCreate_BlockedByNotFound 前置不存在视为不可见，拒绝。
func TestCreate_BlockedByNotFound(t *testing.T) {
	svc, _ := newService()
	_, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title:     "x",
		BlockedBy: []string{"nope"},
	})
	assert.ErrorIs(t, err, ErrDependencyNotFound)
}

// TestCreate_BlockedByNotViewable 他人个人任务不可见，拒绝为 ErrDependencyNotFound。
func TestCreate_BlockedByNotViewable(t *testing.T) {
	svc, repo := newService()
	repo.tasks["b1"] = &model.Task{ID: "b1", TenantID: "t1", CreatorID: "u2", AssigneeID: "u2", Title: "他人私密", Visibility: model.TaskVisibilityPersonal}
	_, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{
		Title:     "x",
		BlockedBy: []string{"b1"},
	})
	assert.ErrorIs(t, err, ErrDependencyNotFound)
}

// TestUpdate_BlockedBySelf 不能依赖自身。
func TestUpdate_BlockedBySelf(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityPersonal}
	_, err := svc.Update(context.Background(), "t1", "u1", "id1", dto.UpdateTaskReq{BlockedBy: &[]string{"id1"}})
	assert.ErrorIs(t, err, ErrDependencySelf)
}

// TestUpdate_BlockedByCycle 先建 id1->id2，再尝试 id2->id1 形成环，应拒绝。
func TestUpdate_BlockedByCycle(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	repo.tasks["id2"] = &model.Task{ID: "id2", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}

	_, err := svc.Update(context.Background(), "t1", "u1", "id1", dto.UpdateTaskReq{BlockedBy: &[]string{"id2"}})
	require.NoError(t, err)

	_, err = svc.Update(context.Background(), "t1", "u1", "id2", dto.UpdateTaskReq{BlockedBy: &[]string{"id1"}})
	assert.ErrorIs(t, err, ErrDependencyCycle)
}

// TestUpdate_BlockedByNilKeepsDeps blocked_by 为 nil 时不改动既有依赖。
func TestUpdate_BlockedByNilKeepsDeps(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	repo.tasks["b1"] = &model.Task{ID: "b1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}

	_, err := svc.Update(context.Background(), "t1", "u1", "id1", dto.UpdateTaskReq{BlockedBy: &[]string{"b1"}})
	require.NoError(t, err)

	// 仅改标题，不带 blocked_by
	_, err = svc.Update(context.Background(), "t1", "u1", "id1", dto.UpdateTaskReq{Title: ptr("改名")})
	require.NoError(t, err)

	got, err := svc.Get(context.Background(), "t1", "u1", "id1")
	require.NoError(t, err)
	assert.Len(t, got.BlockedBy, 1, "nil blocked_by 不应清除既有依赖")
}

// TestUpdate_BlockedByClear 传空数组清空依赖。
func TestUpdate_BlockedByClear(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	repo.tasks["b1"] = &model.Task{ID: "b1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}

	_, err := svc.Update(context.Background(), "t1", "u1", "id1", dto.UpdateTaskReq{BlockedBy: &[]string{"b1"}})
	require.NoError(t, err)

	_, err = svc.Update(context.Background(), "t1", "u1", "id1", dto.UpdateTaskReq{BlockedBy: &[]string{}})
	require.NoError(t, err)

	got, err := svc.Get(context.Background(), "t1", "u1", "id1")
	require.NoError(t, err)
	assert.Empty(t, got.BlockedBy)
}

// TestComplete_PrunningDependent_RunsUnlockSafely 完成前置会扫描后继并走解锁提醒路径；
// 通知管理器为 nil 时应静默跳过且不报错（与本模块 notify 调用方既有 nil-safe 约定一致）。
func TestComplete_PrunningDependent_RunsUnlockSafely(t *testing.T) {
	svc, repo := newService()
	repo.tasks["pre"] = &model.Task{ID: "pre", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant, Status: model.TaskStatusPending}
	repo.tasks["dep"] = &model.Task{ID: "dep", TenantID: "t1", CreatorID: "u1", AssigneeID: "u1", Visibility: model.TaskVisibilityTenant, Status: model.TaskStatusPending}
	// dep 依赖 pre
	repo.deps["dep"] = []string{"pre"}

	got, err := svc.Complete(context.Background(), "t1", "u1", "pre")
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusCompleted, got.Status)

	// 后继仍为 pending（仅提醒、不自动推进状态）
	dep, err := svc.Get(context.Background(), "t1", "u1", "dep")
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusPending, dep.Status)
	require.Len(t, dep.BlockedBy, 1)
}

// TestTaskRefRespMapping ToResp 将 BlockedBy/Blocks 映射为响应体，未完成前置原样透出状态。
func TestTaskRefRespMapping(t *testing.T) {
	task := &model.Task{
		ID:        "id1",
		TenantID:  "t1",
		CreatorID: "u1",
		Status:    model.TaskStatusPending,
		BlockedBy: []model.TaskRef{{ID: "b1", Title: "前置一", Status: model.TaskStatusPending}},
		Blocks:    []model.TaskRef{{ID: "d1", Title: "后继一", Status: model.TaskStatusCompleted}},
	}
	resp := ToResp(task)
	require.Len(t, resp.BlockedBy, 1)
	assert.Equal(t, "b1", resp.BlockedBy[0].ID)
	assert.Equal(t, model.TaskStatusPending, resp.BlockedBy[0].Status)
	require.Len(t, resp.Blocks, 1)
	assert.Equal(t, "d1", resp.Blocks[0].ID)
}

// TestReachesDetectsCycle 直接验证 reaches 图可达（BFS）判定，覆盖自环、直连与深链。
func TestReachesDetectsCycle(t *testing.T) {
	edges := map[string][]string{
		"a": {"b"},
		"b": {"c"},
	}
	assert.True(t, reaches(edges, "a", "c")) // a->b->c
	assert.True(t, reaches(edges, "a", "b"))
	assert.False(t, reaches(edges, "a", "z"))
	assert.True(t, reaches(edges, "a", "a")) // from==target
}

// ===== 解锁提醒计算（notifyUnblocked gating）：接管全局通知管理器以断言真实发出 =====

// unlockRecorder 是录制型 WS 渠道，捕获 Manager 派发的消息供断言解锁提醒的收件人与载荷。
type unlockRecorder struct {
	msgs []*notify.Message
}

func (r *unlockRecorder) Type() notify.ChannelType { return notify.ChannelWebSocket }
func (r *unlockRecorder) Name() string             { return "unlock-recorder" }
func (r *unlockRecorder) Send(_ context.Context, m *notify.Message) error {
	r.msgs = append(r.msgs, m)
	return nil
}

// installUnlockNotifyManager 用录制渠道接管全局通知管理器，结束即恢复；
// notifyUnblocked 走的是真实派发路径（非 nil 短路），故可断言是否/向谁发出提醒。
func installUnlockNotifyManager(t *testing.T) *unlockRecorder {
	t.Helper()
	rec := &unlockRecorder{}
	mgr := notify.NewManager(nil, "", "", nil)
	mgr.AddChannel(rec)
	prev := notify.GetGlobalManager()
	notify.SetGlobalManager(mgr)
	t.Cleanup(func() { notify.SetGlobalManager(prev) })
	return rec
}

// TestNotifyUnblocked_AllPrereqsDone_SendsUnlockReminder 完成“最后一个未完成前置”后，
// 后继的全部前置均已完成 → 向其负责人发出一次 task_unblocked 提醒。
func TestNotifyUnblocked_AllPrereqsDone_SendsUnlockReminder(t *testing.T) {
	svc, repo := newService()
	rec := installUnlockNotifyManager(t)
	repo.tasks["pre"] = &model.Task{ID: "pre", TenantID: "t1", CreatorID: "u1", Status: model.TaskStatusPending, Visibility: model.TaskVisibilityTenant, Title: "前置"}
	repo.tasks["dep"] = &model.Task{ID: "dep", TenantID: "t1", CreatorID: "u1", AssigneeID: "a", Status: model.TaskStatusPending, Visibility: model.TaskVisibilityTenant, Title: "后继"}
	repo.assignees["dep"] = []string{"a"}
	repo.deps["dep"] = []string{"pre"}

	_, err := svc.Complete(context.Background(), "t1", "u1", "pre")
	require.NoError(t, err)
	require.Len(t, rec.msgs, 1, "全部前置完成应触发一次解锁提醒")
	assert.Equal(t, []string{"a"}, rec.msgs[0].Recipients)
	assert.Equal(t, reminderTypeUnlock, rec.msgs[0].Extra["type"])
	assert.Equal(t, "dep", rec.msgs[0].Extra["task_id"])
}

// TestNotifyUnblocked_PartialPrereqs_NoReminder 后继仍有其它未完成前置时，
// 完成其中一条不得触发解锁提醒（软约束：全部前置完成才解锁），且后继状态保持不变。
func TestNotifyUnblocked_PartialPrereqs_NoReminder(t *testing.T) {
	svc, repo := newService()
	rec := installUnlockNotifyManager(t)
	repo.tasks["pre1"] = &model.Task{ID: "pre1", TenantID: "t1", CreatorID: "u1", Status: model.TaskStatusPending, Visibility: model.TaskVisibilityTenant}
	repo.tasks["pre2"] = &model.Task{ID: "pre2", TenantID: "t1", CreatorID: "u1", Status: model.TaskStatusPending, Visibility: model.TaskVisibilityTenant}
	repo.tasks["dep"] = &model.Task{ID: "dep", TenantID: "t1", CreatorID: "u1", AssigneeID: "a", Status: model.TaskStatusPending, Visibility: model.TaskVisibilityTenant}
	repo.assignees["dep"] = []string{"a"}
	repo.deps["dep"] = []string{"pre1", "pre2"}

	_, err := svc.Complete(context.Background(), "t1", "u1", "pre1")
	require.NoError(t, err)
	assert.Empty(t, rec.msgs, "仍有未完成前置（pre2）时不得发解锁提醒")

	d, err := svc.Get(context.Background(), "t1", "u1", "dep")
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusPending, d.Status, "仅提醒，不自动推进后继状态")
}

// TestNotifyUnblocked_MultiAssignee_Deduped 后继多负责人（含重复）时，
// 解锁提醒按负责人去重后各发一条。
func TestNotifyUnblocked_MultiAssignee_Deduped(t *testing.T) {
	svc, repo := newService()
	rec := installUnlockNotifyManager(t)
	repo.tasks["pre"] = &model.Task{ID: "pre", TenantID: "t1", CreatorID: "u1", Status: model.TaskStatusPending, Visibility: model.TaskVisibilityTenant}
	repo.tasks["dep"] = &model.Task{ID: "dep", TenantID: "t1", CreatorID: "u1", AssigneeID: "a", Status: model.TaskStatusPending, Visibility: model.TaskVisibilityTenant}
	repo.assignees["dep"] = []string{"a", "b", "a"} // 含重复 a
	repo.deps["dep"] = []string{"pre"}

	_, err := svc.Complete(context.Background(), "t1", "u1", "pre")
	require.NoError(t, err)
	var got []string
	for _, m := range rec.msgs {
		require.Len(t, m.Recipients, 1)
		got = append(got, m.Recipients[0])
	}
	assert.ElementsMatch(t, []string{"a", "b"}, got, "去重后 a、b 各收一条解锁提醒")
}

// ===== 依赖写入的安全/边界 =====

// TestCreate_BlockedByCrossTenant_Rejected 前置属于其它租户时，即便按其 ID 能取到实体，
// 也须按租户边界拒绝为 ErrDependencyNotFound（与“同租户他人私密不可见”是不同分支）。
func TestCreate_BlockedByCrossTenant_Rejected(t *testing.T) {
	svc, repo := newService()
	repo.tasks["x"] = &model.Task{ID: "x", TenantID: "t2", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	_, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{Title: "x", BlockedBy: []string{"x"}})
	assert.ErrorIs(t, err, ErrDependencyNotFound)
}

// TestUpdate_BlockedByMultiHopCycle_Rejected 深链 A→B→C 已存在，再让 C→A 会构成环，须拒绝。
func TestUpdate_BlockedByMultiHopCycle_Rejected(t *testing.T) {
	svc, repo := newService()
	for _, id := range []string{"A", "B", "C"} {
		repo.tasks[id] = &model.Task{ID: id, TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	}
	repo.deps["A"] = []string{"B"} // A 依赖 B
	repo.deps["B"] = []string{"C"} // B 依赖 C
	_, err := svc.Update(context.Background(), "t1", "u1", "C", dto.UpdateTaskReq{BlockedBy: &[]string{"A"}})
	assert.ErrorIs(t, err, ErrDependencyCycle)
}

// TestUpdate_BlockedBySelfNotFirst_Rejected 自引用出现在列表非首位时同样须拒绝。
func TestUpdate_BlockedBySelfNotFirst_Rejected(t *testing.T) {
	svc, repo := newService()
	repo.tasks["id1"] = &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	repo.tasks["b1"] = &model.Task{ID: "b1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	_, err := svc.Update(context.Background(), "t1", "u1", "id1", dto.UpdateTaskReq{BlockedBy: &[]string{"b1", "id1"}})
	assert.ErrorIs(t, err, ErrDependencySelf)
}

// TestGet_MultipleBlockers_SortedStable 多前置富化按 ID 稳定升序返回，供前端计数与展示保持确定性。
func TestGet_MultipleBlockers_SortedStable(t *testing.T) {
	svc, repo := newService()
	repo.tasks["b2"] = &model.Task{ID: "b2", TenantID: "t1", CreatorID: "u1", Title: "二", Visibility: model.TaskVisibilityTenant}
	repo.tasks["b1"] = &model.Task{ID: "b1", TenantID: "t1", CreatorID: "u1", Title: "一", Visibility: model.TaskVisibilityTenant}
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{Title: "后继", BlockedBy: []string{"b2", "b1"}})
	require.NoError(t, err)

	got, err := svc.Get(context.Background(), "t1", "u1", task.ID)
	require.NoError(t, err)
	require.Len(t, got.BlockedBy, 2)
	assert.Equal(t, "b1", got.BlockedBy[0].ID)
	assert.Equal(t, "b2", got.BlockedBy[1].ID)
}

// TestCreate_BlockedByDedupStored 入参含重复与空串时清洗为一条有效前置后落库。
func TestCreate_BlockedByDedupStored(t *testing.T) {
	svc, repo := newService()
	repo.tasks["b1"] = &model.Task{ID: "b1", TenantID: "t1", CreatorID: "u1", Visibility: model.TaskVisibilityTenant}
	task, err := svc.Create(context.Background(), "t1", "u1", dto.CreateTaskReq{Title: "后继", BlockedBy: []string{"b1", "b1", ""}})
	require.NoError(t, err)

	got, err := svc.Get(context.Background(), "t1", "u1", task.ID)
	require.NoError(t, err)
	require.Len(t, got.BlockedBy, 1, "重复/空前置应清洗为一条")
	assert.Equal(t, "b1", got.BlockedBy[0].ID)
}
