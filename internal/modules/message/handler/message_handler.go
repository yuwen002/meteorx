// Package handler 提供站内消息（收件箱）HTTP 处理器。
package handler

import (
	"context"
	"errors"
	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/message/dto"
	"meteorx/internal/modules/message/model"
	messagerepo "meteorx/internal/modules/message/repository"
	"meteorx/internal/modules/message/service"
	"meteorx/pkg/pagination"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// MessageService 消息服务接口（handler 依赖的最小业务面，便于测试注入桩）。
type MessageService interface {
	List(ctx context.Context, f messagerepo.ListFilter) ([]*dto.MessageResp, int64, error)
	UnreadCount(ctx context.Context, userID string) (*dto.UnreadCountResp, error)
	Preferences(ctx context.Context, userID string) (*dto.PreferencesResp, error)
	UpdatePreference(ctx context.Context, userID string, req dto.UpdatePreferenceReq) (*dto.PreferencesResp, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID, category string) (int64, error)
	MarkReadBatch(ctx context.Context, userID string, ids []string) (int64, error)
	Delete(ctx context.Context, userID, id string) error
	DeleteBatch(ctx context.Context, userID string, ids []string) (int64, error)
}

// MessageHandler 站内消息处理器。
type MessageHandler struct {
	svc MessageService
}

// NewMessageHandler 创建站内消息处理器。
func NewMessageHandler(svc MessageService) *MessageHandler {
	return &MessageHandler{svc: svc}
}

// List 分页查询当前用户的收件箱。
// GET /api/v1/messages?page=1&page_size=20&category=task&unread=true&keyword=
func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "用户信息缺失")
		return
	}

	q := r.URL.Query()
	category := q.Get("category")
	// 非法分类返回 400 而非静默当作全部：否则前端页签拼错会得到「看起来正常但内容不对」的列表
	if category != "" && !model.IsValidCategory(category) {
		response.BadRequest(w, "非法消息分类")
		return
	}

	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	pg := pagination.NewPagination(page, pageSize)

	f := messagerepo.ListFilter{
		UserID:     userID,
		Category:   category,
		UnreadOnly: q.Get("unread") == "true" || q.Get("unread") == "1",
		Keyword:    q.Get("keyword"),
		Page:       pg.Page,
		PageSize:   pg.PageSize,
	}

	items, total, err := h.svc.List(r.Context(), f)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取消息列表失败")
		return
	}
	response.Success(w, pagination.NewPaginatedResult(items, pg.Page, pg.PageSize, int(total)))
}

// UnreadCount 查询未读数（铃铛角标 + 分类角标）。
// GET /api/v1/messages/unread-count
func (h *MessageHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "用户信息缺失")
		return
	}

	resp, err := h.svc.UnreadCount(r.Context(), userID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取未读数失败")
		return
	}
	response.Success(w, resp)
}

// Preferences 查询当前用户的消息偏好。
// GET /api/v1/messages/preferences
func (h *MessageHandler) Preferences(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "用户信息缺失")
		return
	}

	resp, err := h.svc.Preferences(r.Context(), userID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取消息偏好失败")
		return
	}
	response.Success(w, resp)
}

// UpdatePreference 修改单个分类的接收开关。
// PUT /api/v1/messages/preferences
func (h *MessageHandler) UpdatePreference(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "用户信息缺失")
		return
	}

	var req dto.UpdatePreferenceReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	resp, err := h.svc.UpdatePreference(r.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCategory):
			response.BadRequest(w, "非法消息分类")
		case errors.Is(err, service.ErrMandatoryCategory):
			response.BadRequest(w, "安全告警为强制接收，不可关闭")
		default:
			response.Fail(w, http.StatusInternalServerError, "更新消息偏好失败")
		}
		return
	}
	response.Success(w, resp)
}

// MarkRead 标记单条消息已读。
// PUT /api/v1/messages/{id}/read
func (h *MessageHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "用户信息缺失")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "消息ID不能为空")
		return
	}

	if err := h.svc.MarkRead(r.Context(), userID, id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.Fail(w, http.StatusNotFound, "消息不存在")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "标记已读失败")
		return
	}
	response.Success(w, map[string]string{"id": id})
}

// MarkAllRead 标记全部已读（可按分类）。
// PUT /api/v1/messages/read-all
func (h *MessageHandler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "用户信息缺失")
		return
	}

	// 允许空 body：全部标记已读是默认语义
	var req dto.MarkAllReadReq
	if r.ContentLength > 0 {
		if !validator.ValidateJSON(w, r, &req) {
			return
		}
	}
	if req.Category != "" && !model.IsValidCategory(req.Category) {
		response.BadRequest(w, "非法消息分类")
		return
	}

	n, err := h.svc.MarkAllRead(r.Context(), userID, req.Category)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "全部已读失败")
		return
	}
	response.Success(w, map[string]int64{"updated": n})
}

// Delete 删除单条消息。
// DELETE /api/v1/messages/{id}
func (h *MessageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "用户信息缺失")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		response.BadRequest(w, "消息ID不能为空")
		return
	}

	if err := h.svc.Delete(r.Context(), userID, id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.Fail(w, http.StatusNotFound, "消息不存在")
			return
		}
		response.Fail(w, http.StatusInternalServerError, "删除消息失败")
		return
	}
	response.Success(w, map[string]string{"id": id})
}

// MarkReadBatch 批量标记已读。
// PUT /api/v1/messages/batch/read
//
// 与单条接口的差异：个别 ID 不存在 / 不属于本人 / 已读时不报 404，
// 只把它们从 updated 里扣除——批量操作不应因选中项被别处删掉而整批失败。
func (h *MessageHandler) MarkReadBatch(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "用户信息缺失")
		return
	}

	var req dto.BatchIDsReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	n, err := h.svc.MarkReadBatch(r.Context(), userID, req.IDs)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "批量标记已读失败")
		return
	}
	response.Success(w, map[string]int64{"updated": n})
}

// DeleteBatch 批量删除。
// DELETE /api/v1/messages/batch/delete
func (h *MessageHandler) DeleteBatch(w http.ResponseWriter, r *http.Request) {
	userID := contextx.GetUserID(r.Context())
	if userID == "" {
		response.Fail(w, http.StatusUnauthorized, "用户信息缺失")
		return
	}

	var req dto.BatchIDsReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}

	n, err := h.svc.DeleteBatch(r.Context(), userID, req.IDs)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "批量删除消息失败")
		return
	}
	response.Success(w, map[string]int64{"deleted": n})
}
