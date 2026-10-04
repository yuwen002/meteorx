// Package handler 实现任务模块的 HTTP 接口处理器。
// 提供任务的创建、查询、更新、完成/重开、删除、列表与统计等 RESTful API。
// 所有接口均需登录，访问控制以数据归属（个人/租户）为准，不做细粒度 RBAC 拦截。
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/task/dto"
	"meteorx/internal/modules/task/model"
	"meteorx/internal/modules/task/repository"
	"meteorx/internal/modules/task/service"
	"meteorx/pkg/pagination"

	"github.com/go-chi/chi/v5"
)

// TaskHandler 任务模块 HTTP 处理器。
type TaskHandler struct {
	svc service.TaskServiceInterface // 任务业务服务
}

// NewTaskHandler 创建任务模块 HTTP 处理器实例。
func NewTaskHandler(svc service.TaskServiceInterface) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// Create 创建任务。POST /api/v1/tasks
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	if tenantID == "" || userID == "" {
		response.Unauthorized(w, "未获取到用户信息")
		return
	}

	var req dto.CreateTaskReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	task, err := h.svc.Create(r.Context(), tenantID, userID, req)
	if err != nil {
		writeServiceError(w, err, "创建任务失败")
		return
	}
	response.Success(w, service.ToResp(task))
}

// List 查询任务列表。GET /api/v1/tasks
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	if tenantID == "" || userID == "" {
		response.Unauthorized(w, "未获取到用户信息")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	pg := pagination.NewPagination(page, pageSize)

	f := repository.ListFilter{
		TenantID:   tenantID,
		UserID:     userID,
		Visibility: q.Get("visibility"),
		Status:     q.Get("status"),
		Priority:   q.Get("priority"),
		AssigneeID: q.Get("assignee_id"),
		Keyword:    q.Get("keyword"),
		Page:       pg.Page,
		PageSize:   pg.PageSize,
		SortBy:     q.Get("sort_by"),
		SortOrder:  q.Get("sort_order"),
	}

	tasks, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取任务列表失败")
		return
	}

	items := make([]*dto.TaskResp, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, service.ToResp(t))
	}
	result := pagination.NewPaginatedResult(items, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// Get 查询任务详情。GET /api/v1/tasks/{id}
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}

	task, err := h.svc.Get(r.Context(), tenantID, userID, id)
	if err != nil {
		if errors.Is(err, service.ErrTaskNotFound) {
			response.NotFound(w, "任务不存在")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "获取任务失败")
		return
	}
	response.Success(w, service.ToResp(task))
}

// Update 更新任务。PUT /api/v1/tasks/{id}
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}

	var req dto.UpdateTaskReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	task, err := h.svc.Update(r.Context(), tenantID, userID, id, req)
	if err != nil {
		if errors.Is(err, service.ErrTaskNotFound) {
			response.NotFound(w, "任务不存在")
			return
		}
		if errors.Is(err, service.ErrTaskForbidden) {
			response.Forbidden(w, "无权修改该任务")
			return
		}
		writeServiceError(w, err, "更新任务失败")
		return
	}
	response.Success(w, service.ToResp(task))
}

// Complete 完成任务。PUT /api/v1/tasks/{id}/complete
func (h *TaskHandler) Complete(w http.ResponseWriter, r *http.Request) {
	h.doChange(w, r, true)
}

// Reopen 重新打开任务。PUT /api/v1/tasks/{id}/reopen
// 请求体可选 {"status":"pending"|"in_progress"}，默认回到 pending。
func (h *TaskHandler) Reopen(w http.ResponseWriter, r *http.Request) {
	h.doChange(w, r, false)
}

// doChange 完成/重开的公共处理。
func (h *TaskHandler) doChange(w http.ResponseWriter, r *http.Request, complete bool) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}

	var task *model.Task
	var err error
	if complete {
		task, err = h.svc.Complete(r.Context(), tenantID, userID, id)
	} else {
		var body struct {
			Status string `json:"status"`
		}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		task, err = h.svc.Reopen(r.Context(), tenantID, userID, id, body.Status)
	}
	if err != nil {
		if errors.Is(err, service.ErrTaskNotFound) {
			response.NotFound(w, "任务不存在")
			return
		}
		if errors.Is(err, service.ErrTaskForbidden) {
			response.Forbidden(w, "无权操作该任务")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "操作任务失败")
		return
	}
	response.Success(w, service.ToResp(task))
}

// Delete 删除任务（软删除）。DELETE /api/v1/tasks/{id}
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}

	if err := h.svc.Delete(r.Context(), tenantID, userID, id); err != nil {
		if errors.Is(err, service.ErrTaskNotFound) {
			response.NotFound(w, "任务不存在")
			return
		}
		if errors.Is(err, service.ErrTaskForbidden) {
			response.Forbidden(w, "无权删除该任务")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "删除任务失败")
		return
	}
	response.Success(w, nil)
}

// ListTrash 回收站列表。GET /api/v1/tasks/deleted
func (h *TaskHandler) ListTrash(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	if tenantID == "" || userID == "" {
		response.Unauthorized(w, "未获取到用户信息")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	pg := pagination.NewPagination(page, pageSize)

	f := repository.ListFilter{
		TenantID: tenantID,
		UserID:   userID,
		Keyword:  q.Get("keyword"),
		Page:     pg.Page,
		PageSize: pg.PageSize,
	}
	tasks, total, err := h.svc.ListTrash(r.Context(), f)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取回收站列表失败")
		return
	}
	items := make([]*dto.TaskResp, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, service.ToResp(t))
	}
	response.Success(w, pagination.NewPaginatedResult(items, pg.Page, pg.PageSize, int(total)))
}

// Restore 恢复已删除任务。PUT /api/v1/tasks/{id}/restore
func (h *TaskHandler) Restore(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}
	task, err := h.svc.Restore(r.Context(), tenantID, userID, id)
	if err != nil {
		writeServiceError(w, err, "恢复任务失败")
		return
	}
	response.Success(w, service.ToResp(task))
}

// PermanentDelete 永久删除任务。DELETE /api/v1/tasks/{id}/permanent
func (h *TaskHandler) PermanentDelete(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}
	if err := h.svc.PermanentDelete(r.Context(), tenantID, userID, id); err != nil {
		writeServiceError(w, err, "永久删除任务失败")
		return
	}
	response.Success(w, nil)
}

// BatchComplete 批量完成。POST /api/v1/tasks/batch/complete
func (h *TaskHandler) BatchComplete(w http.ResponseWriter, r *http.Request) {
	h.doBatch(w, r, true)
}

// BatchDelete 批量删除。POST /api/v1/tasks/batch/delete
func (h *TaskHandler) BatchDelete(w http.ResponseWriter, r *http.Request) {
	h.doBatch(w, r, false)
}

// doBatch 批量完成/删除的公共处理。
func (h *TaskHandler) doBatch(w http.ResponseWriter, r *http.Request, complete bool) {
	tenantID, userID := currentIdentity(r)
	if tenantID == "" || userID == "" {
		response.Unauthorized(w, "未获取到用户信息")
		return
	}
	var req dto.BatchTaskReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	var (
		affected int
		err      error
	)
	if complete {
		affected, err = h.svc.BatchComplete(r.Context(), tenantID, userID, req.IDs)
	} else {
		affected, err = h.svc.BatchDelete(r.Context(), tenantID, userID, req.IDs)
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "批量操作失败")
		return
	}
	response.Success(w, dto.BatchTaskResp{Affected: affected})
}

// BatchStatus 批量修改状态。POST /api/v1/tasks/batch/status
func (h *TaskHandler) BatchStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	if tenantID == "" || userID == "" {
		response.Unauthorized(w, "未获取到用户信息")
		return
	}
	var req dto.BatchStatusReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	affected, err := h.svc.BatchUpdateStatus(r.Context(), tenantID, userID, req.IDs, req.Status)
	if err != nil {
		if errors.Is(err, service.ErrInvalidStatus) {
			response.BadRequest(w, "状态值不合法")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "批量修改状态失败")
		return
	}
	response.Success(w, dto.BatchTaskResp{Affected: affected})
}

// BatchAssign 批量指派负责人。POST /api/v1/tasks/batch/assign
func (h *TaskHandler) BatchAssign(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	if tenantID == "" || userID == "" {
		response.Unauthorized(w, "未获取到用户信息")
		return
	}
	var req dto.BatchAssignReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	affected, err := h.svc.BatchAssign(r.Context(), tenantID, userID, req.IDs, req.AssigneeID)
	if err != nil {
		if errors.Is(err, service.ErrAssigneeInvalid) {
			response.BadRequest(w, "负责人不存在或不属于当前租户")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "批量指派失败")
		return
	}
	response.Success(w, dto.BatchTaskResp{Affected: affected})
}

// Stats 任务统计。GET /api/v1/tasks/stats
func (h *TaskHandler) Stats(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	if tenantID == "" || userID == "" {
		response.Unauthorized(w, "未获取到用户信息")
		return
	}

	f := repository.ListFilter{
		TenantID:   tenantID,
		UserID:     userID,
		Visibility: r.URL.Query().Get("visibility"),
	}
	stats, err := h.svc.Stats(r.Context(), f)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取任务统计失败")
		return
	}
	response.Success(w, dto.TaskStatsResp{
		Pending:          stats.Pending,
		InProgress:       stats.InProgress,
		Completed:        stats.Completed,
		Overdue:          stats.Overdue,
		Total:            stats.Total,
		CompletionRate:   completionRate(stats.Completed, stats.Total),
		AvgHandleSeconds: stats.AvgHandleSeconds,
	})
}

// completionRate 计算完成率（0~1，保留四位小数）；总数为 0 时返回 0。
func completionRate(completed, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(int64(float64(completed)/float64(total)*10000+0.5)) / 10000
}

// currentIdentity 从上下文提取当前租户与用户 ID。
func currentIdentity(r *http.Request) (tenantID, userID string) {
	return contextx.GetTenantID(r.Context()), contextx.GetUserID(r.Context())
}

// writeServiceError 将业务错误映射为合适的 HTTP 响应。
func writeServiceError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrInvalidDueDate):
		response.BadRequest(w, "截止时间格式不正确")
	case errors.Is(err, service.ErrInvalidRemindLead):
		response.BadRequest(w, "提醒提前量不合法（形如 2h/30m，最长 31 天）")
	case errors.Is(err, service.ErrAssigneeInvalid):
		response.BadRequest(w, "负责人不存在或不属于当前租户")
	case errors.Is(err, service.ErrInvalidRecurrence):
		response.BadRequest(w, "重复周期不合法（仅支持 daily/weekly/monthly）")
	case errors.Is(err, service.ErrRecurrenceNeedsDue):
		response.BadRequest(w, "周期性任务必须设置截止时间")
	case errors.Is(err, service.ErrTaskNotFound):
		response.NotFound(w, "任务不存在")
	case errors.Is(err, service.ErrTaskForbidden):
		response.Forbidden(w, "无权操作该任务")
	default:
		response.Fail(w, http.StatusInternalServerError, fallback+": "+err.Error())
	}
}
