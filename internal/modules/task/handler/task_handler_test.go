package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/task/dto"
	"meteorx/internal/modules/task/handler"
	"meteorx/internal/modules/task/model"
	"meteorx/internal/modules/task/repository"
	"meteorx/internal/modules/task/service"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubTaskService 实现 service.TaskServiceInterface，用于 handler 单测。
type stubTaskService struct {
	createFn          func(ctx context.Context, tenantID, userID string, req dto.CreateTaskReq) (*model.Task, error)
	getFn             func(ctx context.Context, tenantID, userID, id string) (*model.Task, error)
	updateFn          func(ctx context.Context, tenantID, userID, id string, req dto.UpdateTaskReq) (*model.Task, error)
	completeFn        func(ctx context.Context, tenantID, userID, id string) (*model.Task, error)
	reopenFn          func(ctx context.Context, tenantID, userID, id, status string) (*model.Task, error)
	deleteFn          func(ctx context.Context, tenantID, userID, id string) error
	listFn            func(ctx context.Context, f repository.ListFilter) ([]*model.Task, int64, error)
	statsFn           func(ctx context.Context, f repository.ListFilter) (*repository.TaskStatusStats, error)
	listTrashFn       func(ctx context.Context, f repository.ListFilter) ([]*model.Task, int64, error)
	restoreFn         func(ctx context.Context, tenantID, userID, id string) (*model.Task, error)
	permanentDeleteFn func(ctx context.Context, tenantID, userID, id string) error
	batchCompleteFn   func(ctx context.Context, tenantID, userID string, ids []string) (int, error)
	batchDeleteFn     func(ctx context.Context, tenantID, userID string, ids []string) (int, error)
	batchStatusFn     func(ctx context.Context, tenantID, userID string, ids []string, status string) (int, error)
	batchAssignFn     func(ctx context.Context, tenantID, userID string, ids []string, assigneeID string) (int, error)
}

func (s *stubTaskService) Create(ctx context.Context, t, u string, req dto.CreateTaskReq) (*model.Task, error) {
	return s.createFn(ctx, t, u, req)
}
func (s *stubTaskService) Get(ctx context.Context, t, u, id string) (*model.Task, error) {
	return s.getFn(ctx, t, u, id)
}
func (s *stubTaskService) Update(ctx context.Context, t, u, id string, req dto.UpdateTaskReq) (*model.Task, error) {
	return s.updateFn(ctx, t, u, id, req)
}
func (s *stubTaskService) Complete(ctx context.Context, t, u, id string) (*model.Task, error) {
	return s.completeFn(ctx, t, u, id)
}
func (s *stubTaskService) Reopen(ctx context.Context, t, u, id, status string) (*model.Task, error) {
	return s.reopenFn(ctx, t, u, id, status)
}
func (s *stubTaskService) Delete(ctx context.Context, t, u, id string) error {
	return s.deleteFn(ctx, t, u, id)
}
func (s *stubTaskService) List(ctx context.Context, f repository.ListFilter) ([]*model.Task, int64, error) {
	return s.listFn(ctx, f)
}
func (s *stubTaskService) Stats(ctx context.Context, f repository.ListFilter) (*repository.TaskStatusStats, error) {
	return s.statsFn(ctx, f)
}
func (s *stubTaskService) ListTrash(ctx context.Context, f repository.ListFilter) ([]*model.Task, int64, error) {
	return s.listTrashFn(ctx, f)
}
func (s *stubTaskService) Restore(ctx context.Context, t, u, id string) (*model.Task, error) {
	return s.restoreFn(ctx, t, u, id)
}
func (s *stubTaskService) PermanentDelete(ctx context.Context, t, u, id string) error {
	return s.permanentDeleteFn(ctx, t, u, id)
}
func (s *stubTaskService) BatchComplete(ctx context.Context, t, u string, ids []string) (int, error) {
	return s.batchCompleteFn(ctx, t, u, ids)
}
func (s *stubTaskService) BatchDelete(ctx context.Context, t, u string, ids []string) (int, error) {
	return s.batchDeleteFn(ctx, t, u, ids)
}
func (s *stubTaskService) BatchUpdateStatus(ctx context.Context, t, u string, ids []string, status string) (int, error) {
	return s.batchStatusFn(ctx, t, u, ids, status)
}
func (s *stubTaskService) BatchAssign(ctx context.Context, t, u string, ids []string, assigneeID string) (int, error) {
	return s.batchAssignFn(ctx, t, u, ids, assigneeID)
}

func newTaskRouter(stub *stubTaskService) http.Handler {
	h := handler.NewTaskHandler(stub)
	r := chi.NewRouter()
	r.Route("/tasks", func(r chi.Router) {
		r.Get("/stats", h.Stats)
		r.Get("/deleted", h.ListTrash)
		r.Post("/batch/complete", h.BatchComplete)
		r.Post("/batch/delete", h.BatchDelete)
		r.Post("/batch/status", h.BatchStatus)
		r.Post("/batch/assign", h.BatchAssign)
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Put("/{id}/restore", h.Restore)
		r.Delete("/{id}/permanent", h.PermanentDelete)
		r.Put("/{id}/complete", h.Complete)
		r.Put("/{id}/reopen", h.Reopen)
	})
	return r
}

func withCtx(r *http.Request, tenantID, userID string) *http.Request {
	return r.WithContext(contextx.SetVars(r.Context(), tenantID, userID, nil))
}

func sampleTask() *model.Task {
	now := time.Now()
	return &model.Task{ID: "id1", TenantID: "t1", CreatorID: "u1", Title: "任务", Status: model.TaskStatusPending, Priority: model.TaskPriorityNormal, Visibility: model.TaskVisibilityPersonal, CreatedAt: now, UpdatedAt: now}
}

func TestTaskCreate_Success(t *testing.T) {
	stub := &stubTaskService{createFn: func(context.Context, string, string, dto.CreateTaskReq) (*model.Task, error) {
		return sampleTask(), nil
	}}
	router := newTaskRouter(stub)
	body, _ := json.Marshal(dto.CreateTaskReq{Title: "任务"})
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytesReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withCtx(req, "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskCreate_Unauthorized(t *testing.T) {
	stub := &stubTaskService{createFn: func(context.Context, string, string, dto.CreateTaskReq) (*model.Task, error) {
		return sampleTask(), nil
	}}
	router := newTaskRouter(stub)
	body, _ := json.Marshal(dto.CreateTaskReq{Title: "任务"})
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytesReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestTaskCreate_ValidationError(t *testing.T) {
	stub := &stubTaskService{createFn: func(context.Context, string, string, dto.CreateTaskReq) (*model.Task, error) {
		return sampleTask(), nil
	}}
	router := newTaskRouter(stub)
	body, _ := json.Marshal(dto.CreateTaskReq{}) // 缺少必填 title
	req := httptest.NewRequest(http.MethodPost, "/tasks", bytesReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withCtx(req, "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskGet_Success(t *testing.T) {
	stub := &stubTaskService{getFn: func(context.Context, string, string, string) (*model.Task, error) {
		return sampleTask(), nil
	}}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodGet, "/tasks/id1", nil), "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskGet_NotFound(t *testing.T) {
	stub := &stubTaskService{getFn: func(context.Context, string, string, string) (*model.Task, error) {
		return nil, service.ErrTaskNotFound
	}}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodGet, "/tasks/missing", nil), "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTaskUpdate_Forbidden(t *testing.T) {
	stub := &stubTaskService{updateFn: func(context.Context, string, string, string, dto.UpdateTaskReq) (*model.Task, error) {
		return nil, service.ErrTaskForbidden
	}}
	router := newTaskRouter(stub)
	body, _ := json.Marshal(dto.UpdateTaskReq{Title: strPtr("改名")})
	req := withCtx(httptest.NewRequest(http.MethodPut, "/tasks/id1", bytesReader(body)), "t1", "u9")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestTaskUpdate_InvalidDueDate(t *testing.T) {
	stub := &stubTaskService{updateFn: func(context.Context, string, string, string, dto.UpdateTaskReq) (*model.Task, error) {
		return nil, service.ErrInvalidDueDate
	}}
	router := newTaskRouter(stub)
	body, _ := json.Marshal(dto.UpdateTaskReq{DueDate: strPtr("bad")})
	req := withCtx(httptest.NewRequest(http.MethodPut, "/tasks/id1", bytesReader(body)), "t1", "u1")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// TestTaskUpdate_EmptyRecurrenceAllowed 回归：前端编辑无重复任务时无条件提交 recurrence:""，
// 旧版 DTO 的 *string+omitempty,oneof 会把它拒成 400（保存不了）；
// 枚举合法性改由 service 把关后，空串应放行并透传到服务层。
func TestTaskUpdate_EmptyRecurrenceAllowed(t *testing.T) {
	var got dto.UpdateTaskReq
	stub := &stubTaskService{updateFn: func(_ context.Context, _, _, _ string, req dto.UpdateTaskReq) (*model.Task, error) {
		got = req
		return sampleTask(), nil
	}}
	router := newTaskRouter(stub)
	// 模拟前端真实提交体：recurrence 为空字符串
	req := withCtx(httptest.NewRequest(http.MethodPut, "/tasks/id1",
		bytesReader([]byte(`{"title":"改名","visibility":"personal","priority":"normal","recurrence":"","tags":[]}`))), "t1", "u1")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, got.Recurrence)
	assert.Equal(t, "", *got.Recurrence, "空串应原样透传，由 service 解释为取消重复")
}

// TestTaskUpdate_InvalidStatusStillRejectedAtBoundary 验证指针枚举字段的
// 非空非法值（status:"bogus"）仍被 DTO oneof 拦截为 400，不因 recurrence 放宽而退化。
func TestTaskUpdate_InvalidStatusStillRejectedAtBoundary(t *testing.T) {
	stub := &stubTaskService{updateFn: func(context.Context, string, string, string, dto.UpdateTaskReq) (*model.Task, error) {
		t.Fatal("校验失败时不应进入 service")
		return nil, nil
	}}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodPut, "/tasks/id1",
		bytesReader([]byte(`{"status":"bogus"}`))), "t1", "u1")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskComplete_Success(t *testing.T) {
	stub := &stubTaskService{completeFn: func(context.Context, string, string, string) (*model.Task, error) {
		tk := sampleTask()
		tk.Status = model.TaskStatusCompleted
		return tk, nil
	}}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodPut, "/tasks/id1/complete", nil), "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskReopen_Success(t *testing.T) {
	stub := &stubTaskService{reopenFn: func(context.Context, string, string, string, string) (*model.Task, error) {
		return sampleTask(), nil
	}}
	router := newTaskRouter(stub)
	body, _ := json.Marshal(map[string]string{"status": model.TaskStatusInProgress})
	req := withCtx(httptest.NewRequest(http.MethodPut, "/tasks/id1/reopen", bytesReader(body)), "t1", "u1")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskDelete_Success(t *testing.T) {
	stub := &stubTaskService{deleteFn: func(context.Context, string, string, string) error { return nil }}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodDelete, "/tasks/id1", nil), "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskList_Success(t *testing.T) {
	stub := &stubTaskService{listFn: func(context.Context, repository.ListFilter) ([]*model.Task, int64, error) {
		return []*model.Task{sampleTask()}, 1, nil
	}}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodGet, "/tasks?page=1&page_size=10", nil), "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestTaskList_OverduePseudoStatus 验证 status=overdue 被映射为 Overdue 过滤位而不污染 Status。
func TestTaskList_OverduePseudoStatus(t *testing.T) {
	var got repository.ListFilter
	stub := &stubTaskService{listFn: func(_ context.Context, f repository.ListFilter) ([]*model.Task, int64, error) {
		got = f
		return nil, 0, nil
	}}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodGet, "/tasks?status=overdue", nil), "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, got.Overdue)
	assert.Empty(t, got.Status)
}

// TestTaskList_TagAndDateRange 验证 tag 透传与 due_from/due_to 解析；
// 仅日期上界应补足到当天 23:59:59。
func TestTaskList_TagAndDateRange(t *testing.T) {
	var got repository.ListFilter
	stub := &stubTaskService{listFn: func(_ context.Context, f repository.ListFilter) ([]*model.Task, int64, error) {
		got = f
		return nil, 0, nil
	}}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodGet, "/tasks?tag=%E5%B7%A5%E4%BD%9C&due_from=2026-01-01&due_to=2026-01-31", nil), "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "工作", got.Tag)
	require.NotNil(t, got.DueFrom)
	require.NotNil(t, got.DueTo)
	expectedTo := time.Date(2026, 1, 31, 23, 59, 59, 0, time.Local)
	assert.True(t, got.DueTo.Equal(expectedTo), "due_to 应补足到当天末尾，got=%v want=%v", *got.DueTo, expectedTo)
}

// TestTaskList_InvalidDueRange 验证非法的 due_from/due_to 返回 400。
func TestTaskList_InvalidDueRange(t *testing.T) {
	stub := &stubTaskService{listFn: func(context.Context, repository.ListFilter) ([]*model.Task, int64, error) {
		return nil, 0, nil
	}}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodGet, "/tasks?due_from=not-a-date", nil), "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskStats_Success(t *testing.T) {
	stub := &stubTaskService{statsFn: func(context.Context, repository.ListFilter) (*repository.TaskStatusStats, error) {
		return &repository.TaskStatusStats{Pending: 2, Completed: 1, Total: 3, AvgHandleSeconds: 7200}, nil
	}}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodGet, "/tasks/stats", nil), "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data dto.TaskStatsResp `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.InDelta(t, 0.3333, body.Data.CompletionRate, 0.0001)
	assert.Equal(t, int64(7200), body.Data.AvgHandleSeconds)
}

func TestTaskListTrash_Success(t *testing.T) {
	stub := &stubTaskService{listTrashFn: func(context.Context, repository.ListFilter) ([]*model.Task, int64, error) {
		return []*model.Task{sampleTask()}, 1, nil
	}}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodGet, "/tasks/deleted?page=1&page_size=10", nil), "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskRestore_Success(t *testing.T) {
	stub := &stubTaskService{restoreFn: func(context.Context, string, string, string) (*model.Task, error) {
		return sampleTask(), nil
	}}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodPut, "/tasks/id1/restore", nil), "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskRestore_Forbidden(t *testing.T) {
	stub := &stubTaskService{restoreFn: func(context.Context, string, string, string) (*model.Task, error) {
		return nil, service.ErrTaskForbidden
	}}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodPut, "/tasks/id1/restore", nil), "t1", "u9")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestTaskPermanentDelete_Success(t *testing.T) {
	stub := &stubTaskService{permanentDeleteFn: func(context.Context, string, string, string) error { return nil }}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodDelete, "/tasks/id1/permanent", nil), "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskBatchComplete_Success(t *testing.T) {
	stub := &stubTaskService{batchCompleteFn: func(context.Context, string, string, []string) (int, error) {
		return 2, nil
	}}
	router := newTaskRouter(stub)
	body, _ := json.Marshal(dto.BatchTaskReq{IDs: []string{"a", "b"}})
	req := withCtx(httptest.NewRequest(http.MethodPost, "/tasks/batch/complete", bytesReader(body)), "t1", "u1")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskBatchDelete_ValidationError(t *testing.T) {
	stub := &stubTaskService{batchDeleteFn: func(context.Context, string, string, []string) (int, error) {
		return 0, nil
	}}
	router := newTaskRouter(stub)
	body, _ := json.Marshal(dto.BatchTaskReq{}) // ids 为空，校验失败
	req := withCtx(httptest.NewRequest(http.MethodPost, "/tasks/batch/delete", bytesReader(body)), "t1", "u1")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskBatchStatus_Success(t *testing.T) {
	stub := &stubTaskService{batchStatusFn: func(_ context.Context, _, _ string, ids []string, status string) (int, error) {
		assert.Equal(t, model.TaskStatusInProgress, status)
		return len(ids), nil
	}}
	router := newTaskRouter(stub)
	body, _ := json.Marshal(dto.BatchStatusReq{IDs: []string{"a", "b"}, Status: model.TaskStatusInProgress})
	req := withCtx(httptest.NewRequest(http.MethodPost, "/tasks/batch/status", bytesReader(body)), "t1", "u1")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTaskBatchStatus_ValidationError(t *testing.T) {
	stub := &stubTaskService{batchStatusFn: func(context.Context, string, string, []string, string) (int, error) {
		return 0, nil
	}}
	router := newTaskRouter(stub)
	// status 非法，校验失败
	body, _ := json.Marshal(dto.BatchStatusReq{IDs: []string{"a"}, Status: "bogus"})
	req := withCtx(httptest.NewRequest(http.MethodPost, "/tasks/batch/status", bytesReader(body)), "t1", "u1")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTaskBatchAssign_Success(t *testing.T) {
	stub := &stubTaskService{batchAssignFn: func(_ context.Context, _, _ string, ids []string, assignee string) (int, error) {
		assert.Equal(t, "u2", assignee)
		return len(ids), nil
	}}
	router := newTaskRouter(stub)
	body, _ := json.Marshal(dto.BatchAssignReq{IDs: []string{"a", "b"}, AssigneeID: "u2"})
	req := withCtx(httptest.NewRequest(http.MethodPost, "/tasks/batch/assign", bytesReader(body)), "t1", "u1")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// helpers
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }

func strPtr(s string) *string { return &s }
