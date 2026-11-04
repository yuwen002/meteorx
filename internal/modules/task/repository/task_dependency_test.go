package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskRepo_ReplaceBlockedBy 验证前置依赖替换：事务内先删旧行再插入去重去空、剔除自引用后的新行。
func TestTaskRepo_ReplaceBlockedBy(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `task_dependencies` WHERE task_id = ?").
		WithArgs("id1").WillReturnResult(sqlmock.NewResult(0, 1))
	// 输入含自引用 id1、重复 b2、空串，期望仅插入 b1、b2 两行
	mock.ExpectExec("INSERT INTO `task_dependencies`").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	require.NoError(t, repo.ReplaceBlockedBy(context.Background(), "id1", []string{"b1", "b2", "b2", "id1", ""}))
}

// TestTaskRepo_ReplaceBlockedBy_Empty 清空集合：事务内删除旧行后无新行，不再 INSERT。
func TestTaskRepo_ReplaceBlockedBy_Empty(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM `task_dependencies` WHERE task_id = ?").
		WithArgs("id1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.ReplaceBlockedBy(context.Background(), "id1", nil))
}

// TestTaskRepo_LoadDependencies 验证双向读取：正向按 task_id 取 blocked_by，反向按 blocked_by_id 取后继 task。
func TestTaskRepo_LoadDependencies(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	// 正向：id1 -> b1,b2；id2 -> b3
	mock.ExpectQuery("SELECT .* FROM `task_dependencies` WHERE task_id IN").
		WillReturnRows(sqlmock.NewRows([]string{"task_id", "blocked_by_id"}).
			AddRow("id1", "b1").AddRow("id1", "b2").AddRow("id2", "b3"))
	// 反向：b1 被 id1 依赖；b4 被 id3 依赖（b4 是入参之一，其后继进入反向视图）
	mock.ExpectQuery("SELECT .* FROM `task_dependencies` WHERE blocked_by_id IN").
		WillReturnRows(sqlmock.NewRows([]string{"task_id", "blocked_by_id"}).
			AddRow("id1", "b1").AddRow("id3", "b4"))

	blockedBy, blocks, err := repo.LoadDependencies(context.Background(), []string{"id1", "id2", "b4"})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"b1", "b2"}, blockedBy["id1"])
	assert.Equal(t, []string{"b3"}, blockedBy["id2"])
	assert.Equal(t, []string{"id3"}, blocks["b4"])
}

// TestTaskRepo_LoadDependencies_EmptyIDs 空 ID 列表不触发查询，直接返回两个空映射。
func TestTaskRepo_LoadDependencies_EmptyIDs(t *testing.T) {
	gormDB, _ := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	blockedBy, blocks, err := repo.LoadDependencies(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, blockedBy)
	assert.Empty(t, blocks)
}

// TestTaskRepo_FindBrief 批量读取轻量引用（id/标题/状态），供依赖富化。
func TestTaskRepo_FindBrief(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	mock.ExpectQuery("SELECT .* FROM `tasks`").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "status"}).
			AddRow("b1", "前置一", "completed").
			AddRow("b2", "前置二", "pending"))

	m, err := repo.FindBrief(context.Background(), []string{"b1", "b2"})
	require.NoError(t, err)
	require.Len(t, m, 2)
	assert.Equal(t, "前置一", m["b1"].Title)
	assert.Equal(t, "completed", m["b1"].Status)
	assert.Equal(t, "pending", m["b2"].Status)
}

// TestTaskRepo_FindBrief_EmptyIDs 空 ID 列表直接返回空映射，不触发查询。
func TestTaskRepo_FindBrief_EmptyIDs(t *testing.T) {
	gormDB, _ := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	m, err := repo.FindBrief(context.Background(), []string{})
	require.NoError(t, err)
	assert.Empty(t, m)
}

// TestTaskRepo_DependentsOf 查询 blocked_by_id=blocked 的后继 task_id 列表。
func TestTaskRepo_DependentsOf(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	mock.ExpectQuery("SELECT .* FROM `task_dependencies` WHERE blocked_by_id = ?").
		WithArgs("b1").
		WillReturnRows(sqlmock.NewRows([]string{"task_id", "blocked_by_id"}).
			AddRow("id1", "b1").AddRow("id2", "b1"))

	ids, err := repo.DependentsOf(context.Background(), "b1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"id1", "id2"}, ids)
}

// TestTaskRepo_DependentsOf_Empty 空入参直接返回 nil，不触发查询。
func TestTaskRepo_DependentsOf_Empty(t *testing.T) {
	gormDB, _ := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	ids, err := repo.DependentsOf(context.Background(), "")
	require.NoError(t, err)
	assert.Empty(t, ids)
}

// TestTaskRepo_TenantDependencyEdges join tasks 限定本租户的依赖图查询能正确返回边。
func TestTaskRepo_TenantDependencyEdges(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	mock.ExpectQuery("SELECT .* FROM `task_dependencies`").
		WithArgs("t1").
		WillReturnRows(sqlmock.NewRows([]string{"task_id", "blocked_by_id"}).
			AddRow("id1", "b1").AddRow("id1", "b2").AddRow("id2", "b3"))

	edges, err := repo.TenantDependencyEdges(context.Background(), "t1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"b1", "b2"}, edges["id1"])
	assert.Equal(t, []string{"b3"}, edges["id2"])
}

// TestTaskRepo_TenantDependencyEdges_EmptyTenant 空租户直接返回空图，不触发查询。
func TestTaskRepo_TenantDependencyEdges_EmptyTenant(t *testing.T) {
	gormDB, _ := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	edges, err := repo.TenantDependencyEdges(context.Background(), "")
	require.NoError(t, err)
	assert.Empty(t, edges)
}
