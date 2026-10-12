// Package handler 提供数据导出模块的 HTTP 处理层。
package handler

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/tenantctx"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/export/dto"
	"meteorx/internal/modules/export/model"
	"meteorx/internal/modules/export/service"
	"meteorx/pkg/pagination"

	"github.com/go-chi/chi/v5"
)

// ExportServiceAPI handler 依赖的服务接口（便于测试 mock）
type ExportServiceAPI interface {
	CreateExport(ctx context.Context, tenantID, userID string, req dto.CreateExportReq) (*model.ExportJob, error)
	GetByID(ctx context.Context, id, tenantID, userID string, isMaster bool) (*model.ExportJob, error)
	List(ctx context.Context, tenantID, userID string, req dto.ListExportReq) ([]*model.ExportJob, int64, error)
	Delete(ctx context.Context, id, tenantID, userID string, isMaster bool) (*model.ExportJob, error)
}

// ExportHandler 数据导出 HTTP 处理器
type ExportHandler struct {
	svc ExportServiceAPI
}

// NewExportHandler 构造 handler
func NewExportHandler(svc ExportServiceAPI) *ExportHandler {
	return &ExportHandler{svc: svc}
}

// Create 创建导出任务 POST /api/v1/exports
func (h *ExportHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateExportReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	tenantID := contextx.GetTenantID(r.Context())
	userID := contextx.GetUserID(r.Context())
	if tenantID == "" || userID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到用户信息")
		return
	}
	job, err := h.svc.CreateExport(r.Context(), tenantID, userID, req)
	if err != nil {
		writeExportErr(w, err)
		return
	}
	response.Success(w, dto.ToResp(job))
}

// List 分页列出当前用户的导出任务 GET /api/v1/exports
func (h *ExportHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	userID := contextx.GetUserID(r.Context())
	if tenantID == "" || userID == "" {
		response.Fail(w, http.StatusUnauthorized, "未获取到用户信息")
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	pg := pagination.NewPagination(page, pageSize)

	req := dto.ListExportReq{
		Page:     pg.Page,
		PageSize: pg.PageSize,
		Status:   r.URL.Query().Get("status"),
		Resource: r.URL.Query().Get("resource"),
	}
	items, total, err := h.svc.List(r.Context(), tenantID, userID, req)
	if err != nil {
		writeExportErr(w, err)
		return
	}
	resp := make([]*dto.ExportJobResp, 0, len(items))
	for _, it := range items {
		resp = append(resp, dto.ToResp(it))
	}
	response.Success(w, pagination.NewPaginatedResult(resp, pg.Page, pg.PageSize, int(total)))
}

// Get 查询单条任务详情 GET /api/v1/exports/{id}
func (h *ExportHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := h.fetchWithAccess(r, id)
	if err != nil {
		writeExportErr(w, err)
		return
	}
	response.Success(w, dto.ToResp(job))
}

// Download 下载已完成的 CSV 文件 GET /api/v1/exports/{id}/download
func (h *ExportHandler) Download(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := h.fetchWithAccess(r, id)
	if err != nil {
		writeExportErr(w, err)
		return
	}
	if job.Status != model.ExportStatusCompleted || job.FilePath == "" {
		response.Fail(w, http.StatusConflict, "导出任务尚未完成")
		return
	}
	if _, err := os.Stat(job.FilePath); err != nil {
		response.NotFound(w, "导出文件已过期或被删除")
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+id+`.csv"`)
	http.ServeFile(w, r, job.FilePath)
}

// Delete 删除导出任务（同时删除文件）DELETE /api/v1/exports/{id}
func (h *ExportHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenantID := contextx.GetTenantID(r.Context())
	userID := contextx.GetUserID(r.Context())
	info, _ := tenantctx.From(r.Context())
	isMaster := info != nil && info.IsMaster

	job, err := h.svc.Delete(r.Context(), id, tenantID, userID, isMaster)
	if err != nil {
		writeExportErr(w, err)
		return
	}
	if job.FilePath != "" {
		_ = os.Remove(job.FilePath)
	}
	response.Success(w, nil)
}

// fetchWithAccess 读取任务并做归属校验
func (h *ExportHandler) fetchWithAccess(r *http.Request, id string) (*model.ExportJob, error) {
	tenantID := contextx.GetTenantID(r.Context())
	userID := contextx.GetUserID(r.Context())
	info, _ := tenantctx.From(r.Context())
	isMaster := info != nil && info.IsMaster
	return h.svc.GetByID(r.Context(), id, tenantID, userID, isMaster)
}

// writeExportErr 将 service 层错误映射为 HTTP 响应
func writeExportErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrExportNotFound):
		response.Fail(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrExportForbidden):
		response.Fail(w, http.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrInvalidResource):
		response.Fail(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrExportQueueFull):
		response.Fail(w, http.StatusTooManyRequests, err.Error())
	default:
		response.Fail(w, http.StatusInternalServerError, "导出服务异常")
	}
}
