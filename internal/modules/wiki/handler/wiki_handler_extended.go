package handler

import (
	"io"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/service"
	"meteorx/pkg/pagination"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// WikiHandlerExtended 内嵌 WikiHandler，提供标签/评论/分享/模板/统计/订阅/通知/编辑锁/批量/导入导出/审核等扩展 HTTP 接口。
type WikiHandlerExtended struct {
	*WikiHandler
	svc service.WikiServiceExtended
}

// NewWikiHandlerExtended 创建 Wiki 扩展处理器实例。
func NewWikiHandlerExtended(svc service.WikiServiceExtended) *WikiHandlerExtended {
	return &WikiHandlerExtended{
		WikiHandler: NewWikiHandler(svc),
		svc:         svc,
	}
}

// CreateTag 创建标签 POST /wiki/spaces/tags
func (h *WikiHandlerExtended) CreateTag(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTagReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	resp, err := h.svc.CreateTag(r.Context(), &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// ListTags 列出当前租户标签 GET /wiki/spaces/tags
func (h *WikiHandlerExtended) ListTags(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListTags(r.Context())
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// DeleteTag 删除标签 DELETE /wiki/spaces/tags/{id}
func (h *WikiHandlerExtended) DeleteTag(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteTag(r.Context(), id); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// AddDocumentTag 为文档添加标签 POST /wiki/documents/{id}/tags/{tagId}
func (h *WikiHandlerExtended) AddDocumentTag(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	tagID := chi.URLParam(r, "tagId")

	if err := h.svc.AddDocumentTag(r.Context(), documentID, tagID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// RemoveDocumentTag 移除文档标签 DELETE /wiki/documents/{id}/tags/{tagId}
func (h *WikiHandlerExtended) RemoveDocumentTag(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	tagID := chi.URLParam(r, "tagId")

	if err := h.svc.RemoveDocumentTag(r.Context(), documentID, tagID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// ListDocumentTags 列出文档标签 GET /wiki/documents/{id}/tags
func (h *WikiHandlerExtended) ListDocumentTags(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.ListDocumentTags(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// CreateComment 新建文档评论 POST /wiki/documents/{id}/comments
func (h *WikiHandlerExtended) CreateComment(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateCommentReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	documentID := chi.URLParam(r, "id")
	req.DocumentID = documentID

	resp, err := h.svc.CreateComment(r.Context(), &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// ListComments 列出文档评论 GET /wiki/documents/{id}/comments
func (h *WikiHandlerExtended) ListComments(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.ListComments(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// UpdateComment 修改评论 PUT /wiki/documents/comments/{id}
func (h *WikiHandlerExtended) UpdateComment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.UpdateCommentReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	if err := h.svc.UpdateComment(r.Context(), id, &req); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// DeleteComment 删除评论 DELETE /wiki/documents/comments/{id}
func (h *WikiHandlerExtended) DeleteComment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteComment(r.Context(), id); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// CreateShareLink 创建分享链接 POST /wiki/documents/{id}/share
func (h *WikiHandlerExtended) CreateShareLink(w http.ResponseWriter, r *http.Request) {
	var req dto.ShareLinkReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	documentID := chi.URLParam(r, "id")
	req.DocumentID = documentID

	resp, err := h.svc.CreateShareLink(r.Context(), &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// GetShareLink 按令牌获取分享链接信息 GET /wiki/share/{token}?password=xxx
func (h *WikiHandlerExtended) GetShareLink(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	password := r.URL.Query().Get("password")

	resp, err := h.svc.GetShareLink(r.Context(), token, password)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// AccessSharedDocument 免登录公开分享落地：GET /wiki/share/{token}?password=xxx
func (h *WikiHandlerExtended) AccessSharedDocument(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	password := r.URL.Query().Get("password")

	resp, err := h.svc.AccessSharedDocument(r.Context(), token, password)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// ListShareLinks 列出文档分享链接 GET /wiki/documents/{id}/shares
func (h *WikiHandlerExtended) ListShareLinks(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.ListShareLinks(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// DeleteShareLink 删除分享链接 DELETE /wiki/documents/shares/{id}
func (h *WikiHandlerExtended) DeleteShareLink(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteShareLink(r.Context(), id); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// CreateTemplate 创建文档模板 POST /wiki/spaces/templates
func (h *WikiHandlerExtended) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTemplateReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	resp, err := h.svc.CreateTemplate(r.Context(), &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// ListTemplates 按分类列出模板 GET /wiki/spaces/templates
func (h *WikiHandlerExtended) ListTemplates(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	resp, err := h.svc.ListTemplates(r.Context(), category)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// GetTemplate 获取模板详情 GET /wiki/spaces/templates/{id}
func (h *WikiHandlerExtended) GetTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resp, err := h.svc.GetTemplate(r.Context(), id)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// UpdateTemplate 更新模板 PUT /wiki/spaces/templates/{id}
func (h *WikiHandlerExtended) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.UpdateTemplateReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	resp, err := h.svc.UpdateTemplate(r.Context(), id, &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// DeleteTemplate 删除模板 DELETE /wiki/spaces/templates/{id}
func (h *WikiHandlerExtended) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteTemplate(r.Context(), id); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// GetDocumentStats 获取文档统计 GET /wiki/documents/{id}/stats
func (h *WikiHandlerExtended) GetDocumentStats(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.GetDocumentStats(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// ListAccessLogs 分页列出文档访问日志 GET /wiki/documents/{id}/access-logs
func (h *WikiHandlerExtended) ListAccessLogs(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	pg := pagination.NewPagination(page, pageSize)
	resp, total, err := h.svc.ListAccessLogs(r.Context(), documentID, pg.Page, pg.PageSize)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.SuccessWithPagination(w, resp, pg.Page, pg.PageSize, total)
}

// SubscribeDocument 订阅文档变更通知 POST /wiki/documents/{id}/subscribe
func (h *WikiHandlerExtended) SubscribeDocument(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	notifyType := r.URL.Query().Get("notify_type")
	if notifyType == "" {
		notifyType = "all"
	}

	resp, err := h.svc.SubscribeDocument(r.Context(), documentID, notifyType)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// UnsubscribeDocument 取消订阅文档 DELETE /wiki/documents/{id}/subscribe
func (h *WikiHandlerExtended) UnsubscribeDocument(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	if err := h.svc.UnsubscribeDocument(r.Context(), documentID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// ListUserSubscriptions 列出当前用户订阅 GET /wiki/spaces/subscriptions
func (h *WikiHandlerExtended) ListUserSubscriptions(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListUserSubscriptions(r.Context())
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// ListNotifications 分页列出站内通知 GET /wiki/spaces/notifications
func (h *WikiHandlerExtended) ListNotifications(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	pg := pagination.NewPagination(page, pageSize)
	resp, total, err := h.svc.ListNotifications(r.Context(), pg.Page, pg.PageSize)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.SuccessWithPagination(w, resp, pg.Page, pg.PageSize, total)
}

// MarkNotificationAsRead 标记通知已读 PUT /wiki/spaces/notifications/{id}/read
func (h *WikiHandlerExtended) MarkNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.MarkNotificationAsRead(r.Context(), id); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// MarkAllNotificationsAsRead 全部通知标记已读 PUT /wiki/spaces/notifications/read-all
func (h *WikiHandlerExtended) MarkAllNotificationsAsRead(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MarkAllNotificationsAsRead(r.Context()); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// GetUnreadNotificationCount 获取未读通知数量 GET /wiki/spaces/notifications/unread-count
func (h *WikiHandlerExtended) GetUnreadNotificationCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.GetUnreadNotificationCount(r.Context())
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, map[string]interface{}{"count": count})
}

// AcquireEditLock 获取文档编辑锁 POST /wiki/documents/{id}/edit-lock
func (h *WikiHandlerExtended) AcquireEditLock(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.AcquireEditLock(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// ReleaseEditLock 释放文档编辑锁 DELETE /wiki/documents/{id}/edit-lock
func (h *WikiHandlerExtended) ReleaseEditLock(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	if err := h.svc.ReleaseEditLock(r.Context(), documentID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// RefreshEditLock 刷新编辑锁有效期 PUT /wiki/documents/{id}/edit-lock
func (h *WikiHandlerExtended) RefreshEditLock(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	if err := h.svc.RefreshEditLock(r.Context(), documentID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// GetEditLock 查询编辑锁状态 GET /wiki/documents/{id}/edit-lock
func (h *WikiHandlerExtended) GetEditLock(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.GetEditLock(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// BatchOperation 批量操作节点（删除/移动） POST /wiki/spaces/nodes/batch
func (h *WikiHandlerExtended) BatchOperation(w http.ResponseWriter, r *http.Request) {
	var req dto.BatchOperationReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	var err error
	switch req.Action {
	case "delete":
		err = h.svc.BatchDeleteNodes(r.Context(), req.NodeIDs)
	case "move":
		err = h.svc.BatchMoveNodes(r.Context(), req.NodeIDs, req.Target)
	default:
		response.BadRequest(w, "invalid action")
		return
	}

	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

// CompareRevisions 对比两个修订版本 GET /wiki/documents/{id}/revisions/compare
func (h *WikiHandlerExtended) CompareRevisions(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	version1Str := r.URL.Query().Get("version1")
	version2Str := r.URL.Query().Get("version2")

	if version1Str == "" || version2Str == "" {
		response.BadRequest(w, "version1 and version2 are required")
		return
	}

	version1, _ := strconv.Atoi(version1Str)
	version2, _ := strconv.Atoi(version2Str)

	if version1 == 0 || version2 == 0 {
		response.BadRequest(w, "version1 and version2 must be valid integers")
		return
	}

	resp, err := h.svc.CompareRevisions(r.Context(), documentID, version1, version2)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// ExportDocument 导出文档为文件流 POST /wiki/documents/{id}/export
func (h *WikiHandlerExtended) ExportDocument(w http.ResponseWriter, r *http.Request) {
	var req dto.ExportDocumentReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	documentID := chi.URLParam(r, "id")
	content, filename, err := h.svc.ExportDocument(r.Context(), documentID, req.Format)
	if err != nil {
		response.FailError(w, err)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

// ImportDocument 上传文件导入为新修订 POST /wiki/documents/{id}/import（multipart）
func (h *WikiHandlerExtended) ImportDocument(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20) // 32MB

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "file is required")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		response.FailError(w, err)
		return
	}

	format := r.FormValue("format")
	if format == "" {
		format = "markdown"
	}

	documentID := chi.URLParam(r, "id")
	if documentID == "" {
		response.BadRequest(w, "document id is required")
		return
	}

	resp, err := h.svc.ImportDocument(r.Context(), documentID, content, format)
	if err != nil {
		response.FailError(w, err)
		return
	}

	response.Success(w, map[string]interface{}{
		"document_id": resp.ID,
		"node_id":     resp.NodeID,
		"filename":    header.Filename,
		"size":        header.Size,
		"format":      format,
	})
}

// SubmitForReview 提交文档进入审核 POST /wiki/documents/{id}/submit-review
func (h *WikiHandlerExtended) SubmitForReview(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	var req dto.SubmitForReviewReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	resp, err := h.svc.SubmitForReview(r.Context(), documentID, &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// ApproveDocument 审核通过文档 POST /wiki/documents/{id}/approve
func (h *WikiHandlerExtended) ApproveDocument(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	var req dto.ReviewActionReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	resp, err := h.svc.ApproveDocument(r.Context(), documentID, &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// RejectDocument 驳回文档审核 POST /wiki/documents/{id}/reject
func (h *WikiHandlerExtended) RejectDocument(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	var req dto.ReviewActionReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	resp, err := h.svc.RejectDocument(r.Context(), documentID, &req)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// PublishDocument 直接发布文档 POST /wiki/documents/{id}/publish
func (h *WikiHandlerExtended) PublishDocument(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.PublishDocument(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// UnpublishDocument 取消发布文档 POST /wiki/documents/{id}/unpublish
func (h *WikiHandlerExtended) UnpublishDocument(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.UnpublishDocument(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// ArchiveDocument 归档文档 POST /wiki/documents/{id}/archive
func (h *WikiHandlerExtended) ArchiveDocument(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.ArchiveDocument(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// ListReviewComments 列出文档审核评论 GET /wiki/documents/{id}/review-comments
func (h *WikiHandlerExtended) ListReviewComments(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.ListReviewComments(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

// ListPendingReviews 分页列出待审核文档 GET /wiki/pending-reviews
func (h *WikiHandlerExtended) ListPendingReviews(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	resp, total, err := h.svc.ListPendingReviews(r.Context(), page, pageSize)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, map[string]interface{}{
		"items":     resp,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// RegisterExtendedRoutes 将扩展接口挂载到 /wiki 路由分组（需登录，免登录分享已单独注册）。
func (h *WikiHandlerExtended) RegisterExtendedRoutes(r chi.Router) {
	r.Route("/wiki", func(r chi.Router) {
		// 注意：免登录分享 GET /wiki/share/{token} 已在公开分组注册（见 public_routes.go），
		// 此处不再重复注册，避免与公共路由冲突。
		r.Route("/spaces", func(r chi.Router) {
			r.Post("/tags", h.CreateTag)
			r.Get("/tags", h.ListTags)
			r.Delete("/tags/{id}", h.DeleteTag)

			r.Post("/nodes/batch", h.BatchOperation)

			r.Post("/templates", h.CreateTemplate)
			r.Get("/templates", h.ListTemplates)
			r.Get("/templates/{id}", h.GetTemplate)
			r.Put("/templates/{id}", h.UpdateTemplate)
			r.Delete("/templates/{id}", h.DeleteTemplate)

			r.Get("/notifications", h.ListNotifications)
			r.Put("/notifications/read-all", h.MarkAllNotificationsAsRead)
			r.Get("/notifications/unread-count", h.GetUnreadNotificationCount)
			r.Put("/notifications/{id}/read", h.MarkNotificationAsRead)

			r.Get("/subscriptions", h.ListUserSubscriptions)
		})

		r.Route("/documents", func(r chi.Router) {
			r.Post("/{id}/tags/{tagId}", h.AddDocumentTag)
			r.Delete("/{id}/tags/{tagId}", h.RemoveDocumentTag)
			r.Get("/{id}/tags", h.ListDocumentTags)

			r.Post("/{id}/comments", h.CreateComment)
			r.Get("/{id}/comments", h.ListComments)
			r.Put("/comments/{id}", h.UpdateComment)
			r.Delete("/comments/{id}", h.DeleteComment)

			r.Post("/{id}/share", h.CreateShareLink)
			r.Get("/{id}/shares", h.ListShareLinks)
			r.Delete("/shares/{id}", h.DeleteShareLink)

			r.Get("/{id}/stats", h.GetDocumentStats)
			r.Get("/{id}/access-logs", h.ListAccessLogs)

			r.Post("/{id}/subscribe", h.SubscribeDocument)
			r.Delete("/{id}/subscribe", h.UnsubscribeDocument)

			r.Post("/{id}/edit-lock", h.AcquireEditLock)
			r.Delete("/{id}/edit-lock", h.ReleaseEditLock)
			r.Put("/{id}/edit-lock", h.RefreshEditLock)
			r.Get("/{id}/edit-lock", h.GetEditLock)

			r.Post("/{id}/export", h.ExportDocument)
			r.Post("/{id}/import", h.ImportDocument)

			r.Get("/{id}/revisions/compare", h.CompareRevisions)

			r.Post("/{id}/submit-review", h.SubmitForReview)
			r.Post("/{id}/approve", h.ApproveDocument)
			r.Post("/{id}/reject", h.RejectDocument)
			r.Post("/{id}/publish", h.PublishDocument)
			r.Post("/{id}/unpublish", h.UnpublishDocument)
			r.Post("/{id}/archive", h.ArchiveDocument)
			r.Get("/{id}/review-comments", h.ListReviewComments)
		})

		r.Get("/pending-reviews", h.ListPendingReviews)
	})
}
