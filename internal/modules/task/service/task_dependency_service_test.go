package service

import (
	"context"
	"testing"

	"meteorx/internal/modules/task/dto"
	"meteorx/internal/modules/task/model"

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
