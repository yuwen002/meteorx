// Package handler 提供通知公告 HTTP 处理器
package handler

import (
	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/notification/dto"
	"meteorx/internal/modules/notification/service"
	"meteorx/pkg/pagination"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// AnnouncementHandler 公告处理器
type AnnouncementHandler struct {
	svc *service.AnnouncementService
}

// NewAnnouncementHandler 创建公告处理器
func NewAnnouncementHandler(svc *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{svc: svc}
}

// Create 创建公告
// POST /api/v1/admin/announcements
func (h *AnnouncementHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAnnouncementReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	resp, err := h.svc.Create(r.Context(), contextx.GetUserID(r.Context()), req)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "创建公告失败")
		return
	}
	response.Success(w, resp)
}

// Get 获取公告详情
// GET /api/v1/admin/announcements/{id}
func (h *AnnouncementHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "公告ID不能为空")
		return
	}

	resp, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Fail(w, http.StatusNotFound, "公告不存在")
		return
	}
	response.Success(w, resp)
}

// List 分页查询公告
// GET /api/v1/admin/announcements?page=1&page_size=10&keyword=&status=&scope=
func (h *AnnouncementHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	pg := pagination.NewPagination(page, pageSize)

	status, _ := strconv.Atoi(r.URL.Query().Get("status"))
	// status 为空字符串时 Atoi 返回 0（草稿），需显式判断：未传则按全部处理
	if r.URL.Query().Get("status") == "" {
		status = -1
	}

	query := &dto.ListAnnouncementsQuery{
		Page:     pg.Page,
		PageSize: pg.PageSize,
		Keyword:  r.URL.Query().Get("keyword"),
		Status:   status,
		Scope:    r.URL.Query().Get("scope"),
	}

	result, err := h.svc.List(r.Context(), query)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取公告列表失败")
		return
	}
	response.Success(w, pagination.NewPaginatedResult(result.Items, pg.Page, pg.PageSize, int(result.Total)))
}

// Update 更新公告
// PUT /api/v1/admin/announcements/{id}
func (h *AnnouncementHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "公告ID不能为空")
		return
	}

	var req dto.UpdateAnnouncementReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	resp, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "更新公告失败")
		return
	}
	response.Success(w, resp)
}

// UpdateStatus 发布/下架公告
// PUT /api/v1/admin/announcements/{id}/status
func (h *AnnouncementHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "公告ID不能为空")
		return
	}

	var req dto.UpdateAnnouncementStatusReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	resp, err := h.svc.UpdateStatus(r.Context(), id, req.Status)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "更新公告状态失败")
		return
	}
	response.Success(w, resp)
}

// Delete 删除公告
// DELETE /api/v1/admin/announcements/{id}
func (h *AnnouncementHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "公告ID不能为空")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		response.Fail(w, http.StatusInternalServerError, "删除公告失败")
		return
	}
	response.Success(w, map[string]string{"id": id})
}

// ListForTenant 租户获取可见公告列表
// GET /api/v1/announcements?page=1&page_size=10
func (h *AnnouncementHandler) ListForTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "租户信息缺失")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	pg := pagination.NewPagination(page, pageSize)

	result, err := h.svc.ListForTenant(r.Context(), tenantID, pg.Page, pg.PageSize)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取公告列表失败")
		return
	}
	response.Success(w, pagination.NewPaginatedResult(result.Items, pg.Page, pg.PageSize, int(result.Total)))
}