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

func newTaskRouter(stub *stubTaskService) http.Handler {
	h := handler.NewTaskHandler(stub)
	r := chi.NewRouter()
	r.Route("/tasks", func(r chi.Router) {
		r.Get("/stats", h.Stats)
		r.Get("/deleted", h.ListTrash)
		r.Post("/batch/complete", h.BatchComplete)
		r.Post("/batch/delete", h.BatchDelete)
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

func TestTaskStats_Success(t *testing.T) {
	stub := &stubTaskService{statsFn: func(context.Context, repository.ListFilter) (*repository.TaskStatusStats, error) {
		return &repository.TaskStatusStats{Pending: 2, Total: 3}, nil
	}}
	router := newTaskRouter(stub)
	req := withCtx(httptest.NewRequest(http.MethodGet, "/tasks/stats", nil), "t1", "u1")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
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

// helpers
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }

func strPtr(s string) *string { return &s }
