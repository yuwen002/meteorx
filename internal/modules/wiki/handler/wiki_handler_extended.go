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

type WikiHandlerExtended struct {
	*WikiHandler
	svc service.WikiServiceExtended
}

func NewWikiHandlerExtended(svc service.WikiServiceExtended) *WikiHandlerExtended {
	return &WikiHandlerExtended{
		WikiHandler: NewWikiHandler(svc),
		svc:         svc,
	}
}

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

func (h *WikiHandlerExtended) ListTags(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListTags(r.Context())
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

func (h *WikiHandlerExtended) DeleteTag(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteTag(r.Context(), id); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

func (h *WikiHandlerExtended) AddDocumentTag(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	tagID := chi.URLParam(r, "tagId")

	if err := h.svc.AddDocumentTag(r.Context(), documentID, tagID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

func (h *WikiHandlerExtended) RemoveDocumentTag(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	tagID := chi.URLParam(r, "tagId")

	if err := h.svc.RemoveDocumentTag(r.Context(), documentID, tagID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

func (h *WikiHandlerExtended) ListDocumentTags(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.ListDocumentTags(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

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

func (h *WikiHandlerExtended) ListComments(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.ListComments(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

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

func (h *WikiHandlerExtended) DeleteComment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteComment(r.Context(), id); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

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

func (h *WikiHandlerExtended) ListShareLinks(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.ListShareLinks(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

func (h *WikiHandlerExtended) DeleteShareLink(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteShareLink(r.Context(), id); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

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

func (h *WikiHandlerExtended) ListTemplates(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	resp, err := h.svc.ListTemplates(r.Context(), category)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

func (h *WikiHandlerExtended) GetTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	resp, err := h.svc.GetTemplate(r.Context(), id)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

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

func (h *WikiHandlerExtended) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.DeleteTemplate(r.Context(), id); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

func (h *WikiHandlerExtended) GetDocumentStats(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.GetDocumentStats(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

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

func (h *WikiHandlerExtended) UnsubscribeDocument(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	if err := h.svc.UnsubscribeDocument(r.Context(), documentID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

func (h *WikiHandlerExtended) ListUserSubscriptions(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListUserSubscriptions(r.Context())
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

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

func (h *WikiHandlerExtended) MarkNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.MarkNotificationAsRead(r.Context(), id); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

func (h *WikiHandlerExtended) MarkAllNotificationsAsRead(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MarkAllNotificationsAsRead(r.Context()); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

func (h *WikiHandlerExtended) GetUnreadNotificationCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.GetUnreadNotificationCount(r.Context())
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, map[string]interface{}{"count": count})
}

func (h *WikiHandlerExtended) AcquireEditLock(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.AcquireEditLock(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

func (h *WikiHandlerExtended) ReleaseEditLock(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	if err := h.svc.ReleaseEditLock(r.Context(), documentID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

func (h *WikiHandlerExtended) RefreshEditLock(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")
	if err := h.svc.RefreshEditLock(r.Context(), documentID); err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, nil)
}

func (h *WikiHandlerExtended) GetEditLock(w http.ResponseWriter, r *http.Request) {
	documentID := chi.URLParam(r, "id")

	resp, err := h.svc.GetEditLock(r.Context(), documentID)
	if err != nil {
		response.FailError(w, err)
		return
	}
	response.Success(w, resp)
}

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
		})
	})
}
