package repository

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/modules/task/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// newTaskTestDB 基于 go-sqlmock 构造 MySQL 方言的测试连接。
func newTaskTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
		_ = sqlDB.Close()
	})
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	return gormDB, mock
}

func sampleTask() *model.Task {
	now := time.Now()
	return &model.Task{
		ID:         "id1",
		TenantID:   "t1",
		CreatorID:  "u1",
		AssigneeID: "u1",
		Title:      "写周报",
		Status:     model.TaskStatusPending,
		Priority:   model.TaskPriorityNormal,
		Tags:       []string{"工作"},
		Visibility: model.TaskVisibilityPersonal,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func TestTaskRepo_Create(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	mock.ExpectExec("INSERT INTO `tasks`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.Create(context.Background(), sampleTask()))
}

func TestTaskRepo_GetByID(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "creator_id", "assignee_id", "title", "description",
		"status", "priority", "due_date", "tags", "visibility", "completed_at",
		"created_at", "updated_at", "deleted_at",
	}).AddRow(
		"id1", "t1", "u1", "u1", "写周报", "",
		"pending", "normal", nil, `["工作"]`, "personal", nil,
		now, now, nil,
	)
	mock.ExpectQuery("SELECT .* FROM `tasks`").WillReturnRows(rows)

	task, err := repo.GetByID(context.Background(), "id1")
	require.NoError(t, err)
	assert.Equal(t, "id1", task.ID)
	assert.Equal(t, []string{"工作"}, task.Tags)
	assert.Nil(t, task.DueDate)
	assert.Nil(t, task.DeletedAt)
}

func TestTaskRepo_UpdateStatus_Completed(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	mock.ExpectExec("UPDATE `tasks` SET").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.UpdateStatus(context.Background(), "id1", model.TaskStatusCompleted))
}

func TestTaskRepo_Delete_Soft(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	// GORM 软删除以 UPDATE deleted_at 实现
	mock.ExpectExec("UPDATE `tasks` SET `deleted_at`").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Delete(context.Background(), "id1"))
}

func TestTaskRepo_List_Personal(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	now := time.Now()
	// Count 查询
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `tasks`").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	// 数据查询
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "creator_id", "assignee_id", "title", "description",
		"status", "priority", "due_date", "tags", "visibility", "completed_at",
		"created_at", "updated_at", "deleted_at",
	}).AddRow(
		"id1", "t1", "u1", "u1", "写周报", "",
		"pending", "normal", nil, "[]", "personal", nil,
		now, now, nil,
	)
	mock.ExpectQuery("SELECT .* FROM `tasks`").WillReturnRows(rows)

	tasks, total, err := repo.List(context.Background(), ListFilter{
		TenantID:   "t1",
		UserID:     "u1",
		Visibility: model.TaskVisibilityPersonal,
		Page:       1,
		PageSize:   20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, tasks, 1)
	assert.Equal(t, "id1", tasks[0].ID)
}

func TestTaskRepo_Stats(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	// Stats 依次执行 Pending / InProgress / Completed / Overdue / Total 五个 count 查询
	counts := []int64{3, 2, 5, 1, 10}
	for _, c := range counts {
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `tasks`").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(c))
	}

	stats, err := repo.Stats(context.Background(), ListFilter{TenantID: "t1", UserID: "u1"})
	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.Pending)
	assert.Equal(t, int64(2), stats.InProgress)
	assert.Equal(t, int64(5), stats.Completed)
	assert.Equal(t, int64(1), stats.Overdue)
	assert.Equal(t, int64(10), stats.Total)
}

func TestTaskRepo_GetByIDUnscoped_IncludesDeleted(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "creator_id", "assignee_id", "title", "description",
		"status", "priority", "due_date", "tags", "visibility", "completed_at",
		"created_at", "updated_at", "deleted_at",
	}).AddRow(
		"id1", "t1", "u1", "u1", "写周报", "",
		"pending", "normal", nil, `[]`, "personal", nil,
		now, now, now,
	)
	mock.ExpectQuery("SELECT .* FROM `tasks`").WillReturnRows(rows)

	task, err := repo.GetByIDUnscoped(context.Background(), "id1")
	require.NoError(t, err)
	assert.Equal(t, "id1", task.ID)
	assert.NotNil(t, task.DeletedAt)
}

func TestTaskRepo_ListDeleted(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	now := time.Now()
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `tasks`").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "creator_id", "assignee_id", "title", "description",
		"status", "priority", "due_date", "tags", "visibility", "completed_at",
		"created_at", "updated_at", "deleted_at",
	}).AddRow(
		"d1", "t1", "u1", "u1", "已删任务", "",
		"pending", "normal", nil, `[]`, "personal", nil,
		now, now, now,
	)
	mock.ExpectQuery("SELECT .* FROM `tasks`").WillReturnRows(rows)

	tasks, total, err := repo.ListDeleted(context.Background(), ListFilter{
		TenantID: "t1", UserID: "u1", Page: 1, PageSize: 20,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, tasks, 1)
	assert.Equal(t, "d1", tasks[0].ID)
	assert.NotNil(t, tasks[0].DeletedAt)
}

func TestTaskRepo_Restore(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	mock.ExpectExec("UPDATE `tasks` SET").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Restore(context.Background(), "id1"))
}

func TestTaskRepo_PermanentDelete(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	mock.ExpectExec("DELETE FROM `tasks`").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.PermanentDelete(context.Background(), "id1"))
}

func TestTaskRepo_FindDueForReminder(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	now := time.Now()
	due := now.Add(-time.Hour)
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "creator_id", "assignee_id", "title", "description",
		"status", "priority", "due_date", "tags", "visibility", "completed_at",
		"reminder_sent_at", "created_at", "updated_at", "deleted_at",
	}).AddRow(
		"id1", "t1", "u1", "u2", "写周报", "",
		"pending", "normal", due, `[]`, "tenant", nil,
		nil, now, now, nil,
	)
	mock.ExpectQuery("SELECT .* FROM `tasks`").WillReturnRows(rows)

	tasks, err := repo.FindDueForReminder(context.Background(), now.Add(24*time.Hour), 100)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "id1", tasks[0].ID)
	assert.NotNil(t, tasks[0].DueDate)
	assert.Nil(t, tasks[0].ReminderSentAt)
}

func TestTaskRepo_MarkReminderSent(t *testing.T) {
	gormDB, mock := newTaskTestDB(t)
	repo := NewTaskRepository(gormDB)

	mock.ExpectExec("UPDATE `tasks` SET `reminder_sent_at`").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.MarkReminderSent(context.Background(), "id1", time.Now()))
}
