package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/config"
	"meteorx/internal/modules/file/dto"
	"meteorx/internal/modules/file/service"
	"meteorx/pkg/pagination"

	"github.com/go-chi/chi/v5"
)

// FileHandler 文件处理器
type FileHandler struct {
	service *service.FileService
	cfg     config.FileConfig
}

// NewFileHandler 创建文件处理器实例
func NewFileHandler(svc *service.FileService, cfg config.FileConfig) *FileHandler {
	return &FileHandler{service: svc, cfg: cfg}
}

// Upload 上传文件
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	maxSize := h.cfg.MaxFileSize
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	if err := r.ParseMultipartForm(maxSize); err != nil {
		response.Fail(w, http.StatusBadRequest, "文件过大或解析失败")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Fail(w, http.StatusBadRequest, "未找到上传文件")
		return
	}
	defer file.Close()

	// 验证文件类型
	mimeType := header.Header.Get("Content-Type")
	if !h.isValidFileType(mimeType) {
		response.Fail(w, http.StatusBadRequest, "不支持的文件类型: "+mimeType)
		return
	}

	// 验证文件大小
	if header.Size > maxSize {
		response.Fail(w, http.StatusBadRequest, "文件大小超过限制")
		return
	}

	// 获取租户ID和用户ID
	tenantID := contextx.GetTenantID(r.Context())
	userID := contextx.GetUserID(r.Context())

	if tenantID == "" || userID == "" {
		response.Fail(w, http.StatusUnauthorized, "未授权")
		return
	}

	// 调用服务上传文件
	result, err := h.service.Upload(r.Context(), header, tenantID, userID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "文件上传失败: "+err.Error())
		return
	}

	response.Success(w, result)
}

// isValidFileType 验证文件类型是否允许（基于配置）
func (h *FileHandler) isValidFileType(mimeType string) bool {
	if len(h.cfg.AllowedTypes) == 0 {
		// 未配置时使用默认白名单
		defaults := []string{
			"image/jpeg",
			"image/png",
			"image/gif",
			"image/webp",
			"application/pdf",
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.ms-excel",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"text/plain",
		}
		for _, allowed := range defaults {
			if mimeType == allowed {
				return true
			}
		}
		return false
	}
	for _, allowed := range h.cfg.AllowedTypes {
		if mimeType == allowed {
			return true
		}
	}
	return false
}

// GetByID 获取文件详情
func (h *FileHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "文件ID不能为空")
		return
	}

	file, err := h.service.GetByIDWithScope(r.Context(), id, contextx.GetTenantID(r.Context()))
	if err != nil {
		response.Fail(w, http.StatusNotFound, "文件不存在")
		return
	}

	response.Success(w, file)
}

// ListByTenant 获取租户文件列表
func (h *FileHandler) ListByTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未授权")
		return
	}

	req := h.parseListReq(r)

	files, total, err := h.service.ListByTenant(r.Context(), tenantID, req)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取文件列表失败")
		return
	}

	response.Success(w, pagination.NewPaginatedResult(files, req.Page, req.PageSize, int(total)))
}

// ListByUser 获取用户文件列表
func (h *FileHandler) ListByUser(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	userID := contextx.GetUserID(r.Context())

	if tenantID == "" || userID == "" {
		response.Fail(w, http.StatusUnauthorized, "未授权")
		return
	}

	req := h.parseListReq(r)

	files, total, err := h.service.ListByUser(r.Context(), tenantID, userID, req)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取文件列表失败")
		return
	}

	response.Success(w, pagination.NewPaginatedResult(files, req.Page, req.PageSize, int(total)))
}

// parseListReq 解析分页与过滤参数
func (h *FileHandler) parseListReq(r *http.Request) *dto.FileListReq {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	pg := pagination.NewPagination(page, pageSize)

	return &dto.FileListReq{
		Page:     pg.Page,
		PageSize: pg.PageSize,
		FileType: r.URL.Query().Get("file_type"),
		Keyword:  r.URL.Query().Get("keyword"),
	}
}

// Update 更新文件信息
func (h *FileHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "文件ID不能为空")
		return
	}

	var req dto.FileUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "参数错误")
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	if err := h.service.Update(r.Context(), id, tenantID, &req); err != nil {
		response.Fail(w, http.StatusInternalServerError, "更新文件失败")
		return
	}

	response.Success(w, nil)
}

// Delete 删除文件
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "文件ID不能为空")
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	if err := h.service.Delete(r.Context(), id, tenantID); err != nil {
		response.Fail(w, http.StatusInternalServerError, "删除文件失败")
		return
	}

	response.Success(w, nil)
}

// BatchDelete 批量删除文件
func (h *FileHandler) BatchDelete(w http.ResponseWriter, r *http.Request) {
	var req dto.BatchDeleteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "参数错误")
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	result, err := h.service.BatchDelete(r.Context(), tenantID, &req)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "批量删除失败")
		return
	}

	response.Success(w, result)
}

// GetDeletedList 获取已删除文件列表
func (h *FileHandler) GetDeletedList(w http.ResponseWriter, r *http.Request) {
	tenantID := contextx.GetTenantID(r.Context())
	if tenantID == "" {
		response.Fail(w, http.StatusUnauthorized, "未授权")
		return
	}

	req := h.parseListReq(r)

	files, total, err := h.service.GetDeletedList(r.Context(), tenantID, req.Page, req.PageSize)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取已删除文件列表失败")
		return
	}

	response.Success(w, pagination.NewPaginatedResult(files, req.Page, req.PageSize, int(total)))
}

// Restore 恢复已删除文件
func (h *FileHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "文件ID不能为空")
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	if err := h.service.Restore(r.Context(), id, tenantID); err != nil {
		response.Fail(w, http.StatusInternalServerError, "恢复文件失败")
		return
	}

	response.Success(w, nil)
}

// PermanentDelete 永久删除文件（从回收站物理删除）
func (h *FileHandler) PermanentDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "文件ID不能为空")
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	if err := h.service.PermanentDelete(r.Context(), id, tenantID); err != nil {
		response.Fail(w, http.StatusInternalServerError, "永久删除失败")
		return
	}

	response.Success(w, nil)
}

// Download 下载文件
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Fail(w, http.StatusBadRequest, "文件ID不能为空")
		return
	}

	tenantID := contextx.GetTenantID(r.Context())
	reader, filename, err := h.service.Download(r.Context(), id, tenantID)
	if err != nil {
		response.Fail(w, http.StatusNotFound, "文件不存在或下载失败")
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")

	if _, err := io.Copy(w, reader); err != nil {
		return
	}
}
