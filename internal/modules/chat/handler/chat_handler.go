// Package handler 实现协作即时消息（Chat）模块的 HTTP 接口处理器。
// 提供发起/复用私聊会话、列出会话、发送消息、拉取历史与标记已读等 RESTful API。
// 所有接口均需登录，访问控制以会话成员归属为准，不做细粒度 RBAC 拦截。
package handler

import (
	"errors"
	"net/http"
	"strconv"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/common/validator"
	"meteorx/internal/modules/chat/dto"
	"meteorx/internal/modules/chat/service"
	"meteorx/pkg/pagination"

	"github.com/go-chi/chi/v5"
)

// ChatHandler 聊天模块 HTTP 处理器。
type ChatHandler struct {
	svc service.ChatServiceInterface
}

// NewChatHandler 创建聊天模块 HTTP 处理器实例。
func NewChatHandler(svc service.ChatServiceInterface) *ChatHandler {
	return &ChatHandler{svc: svc}
}

// OpenDirect 发起/复用 1:1 私聊会话。POST /api/v1/chat/conversations/direct
func (h *ChatHandler) OpenDirect(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	if tenantID == "" || userID == "" {
		response.Unauthorized(w, "未获取到用户信息")
		return
	}
	var req dto.CreateDirectReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	conv, err := h.svc.OpenDirect(r.Context(), tenantID, userID, req.ParticipantID)
	if err != nil {
		h.writeErr(w, err, "发起会话失败")
		return
	}
	response.Success(w, service.ToConversationResp(conv))
}

// ListConversations 列出当前用户的会话。GET /api/v1/chat/conversations
func (h *ChatHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	if tenantID == "" || userID == "" {
		response.Unauthorized(w, "未获取到用户信息")
		return
	}
	convs, err := h.svc.ListConversations(r.Context(), tenantID, userID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "获取会话列表失败")
		return
	}
	items := make([]*dto.ConversationResp, 0, len(convs))
	for _, c := range convs {
		items = append(items, service.ToConversationResp(c))
	}
	response.Success(w, items)
}

// SendMessage 发送消息。POST /api/v1/chat/conversations/{id}/messages
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}
	var req dto.SendMessageReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	msg, err := h.svc.SendMessage(r.Context(), tenantID, userID, id, req.Content, req.MentionIDs)
	if err != nil {
		h.writeErr(w, err, "发送消息失败")
		return
	}
	response.Success(w, service.ToMessageResp(msg))
}

// ListMessages 分页拉取会话历史消息。GET /api/v1/chat/conversations/{id}/messages
func (h *ChatHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	pg := pagination.NewPagination(page, pageSize)

	msgs, total, err := h.svc.ListMessages(r.Context(), tenantID, userID, id, pg.Page, pg.PageSize)
	if err != nil {
		h.writeErr(w, err, "获取消息失败")
		return
	}
	items := make([]*dto.ChatMessageResp, 0, len(msgs))
	for _, m := range msgs {
		items = append(items, service.ToMessageResp(m))
	}
	result := pagination.NewPaginatedResult(items, pg.Page, pg.PageSize, int(total))
	response.Success(w, result)
}

// MarkRead 标记会话已读。POST /api/v1/chat/conversations/{id}/read
func (h *ChatHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}
	if err := h.svc.MarkRead(r.Context(), tenantID, userID, id); err != nil {
		h.writeErr(w, err, "标记已读失败")
		return
	}
	response.Success(w, map[string]bool{"ok": true})
}

// RecallMessage 撤回自己发送的消息。DELETE /api/v1/chat/conversations/{id}/messages/{msgId}
func (h *ChatHandler) RecallMessage(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	msgID := chi.URLParam(r, "msgId")
	if tenantID == "" || userID == "" || id == "" || msgID == "" {
		response.BadRequest(w, "参数不完整")
		return
	}
	if err := h.svc.RecallMessage(r.Context(), tenantID, userID, id, msgID); err != nil {
		h.writeErr(w, err, "撤回消息失败")
		return
	}
	response.Success(w, map[string]bool{"ok": true})
}

// Typing 上报输入中状态（无状态转发）。POST /api/v1/chat/conversations/{id}/typing
func (h *ChatHandler) Typing(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}
	var req dto.TypingReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	if err := h.svc.NotifyTyping(r.Context(), tenantID, userID, id, req.Typing); err != nil {
		h.writeErr(w, err, "输入状态上报失败")
		return
	}
	response.Success(w, map[string]bool{"ok": true})
}

// Presence 返回当前在线用户 ID 集合。GET /api/v1/chat/presence
func (h *ChatHandler) Presence(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	if tenantID == "" || userID == "" {
		response.Unauthorized(w, "未获取到用户信息")
		return
	}
	response.Success(w, dto.PresenceResp{OnlineUserIDs: h.svc.PresenceUsers(r.Context())})
}

// ---- 群聊 ----

// CreateGroup 创建群聊会话。POST /api/v1/chat/conversations/group
func (h *ChatHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	if tenantID == "" || userID == "" {
		response.Unauthorized(w, "未获取到用户信息")
		return
	}
	var req dto.CreateGroupReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	conv, err := h.svc.CreateGroup(r.Context(), tenantID, userID, req.Name, req.MemberIDs)
	if err != nil {
		h.writeErr(w, err, "创建群聊失败")
		return
	}
	response.Success(w, service.ToConversationResp(conv))
}

// RenameGroup 群改名（仅群主）。PUT /api/v1/chat/conversations/{id}
func (h *ChatHandler) RenameGroup(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}
	var req dto.RenameGroupReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	if err := h.svc.RenameGroup(r.Context(), tenantID, userID, id, req.Name); err != nil {
		h.writeErr(w, err, "群改名失败")
		return
	}
	response.Success(w, map[string]bool{"ok": true})
}

// ListMembers 列出会话成员。GET /api/v1/chat/conversations/{id}/members
func (h *ChatHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}
	members, err := h.svc.ListMembers(r.Context(), tenantID, userID, id)
	if err != nil {
		h.writeErr(w, err, "获取成员列表失败")
		return
	}
	items := make([]*dto.MemberResp, 0, len(members))
	for _, m := range members {
		items = append(items, service.ToMemberResp(m))
	}
	response.Success(w, items)
}

// ReadStatus 获取会话内其他成员的已读水位（已读回执）。GET /api/v1/chat/conversations/{id}/read-status
func (h *ChatHandler) ReadStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}
	members, err := h.svc.ReadStatus(r.Context(), tenantID, userID, id)
	if err != nil {
		h.writeErr(w, err, "获取已读状态失败")
		return
	}
	items := make([]*dto.ReadStateResp, 0, len(members))
	for _, m := range members {
		item := &dto.ReadStateResp{UserID: m.UserID, Name: m.UserName}
		if m.LastReadAt != nil {
			item.LastReadAt = m.LastReadAt.Format("2006-01-02 15:04:05")
		}
		items = append(items, item)
	}
	response.Success(w, items)
}

// AddMembers 向群添加成员。POST /api/v1/chat/conversations/{id}/members
func (h *ChatHandler) AddMembers(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}
	var req dto.AddMembersReq
	if !validator.ValidateJSON(w, r, &req) {
		return
	}
	if err := h.svc.AddMembers(r.Context(), tenantID, userID, id, req.UserIDs); err != nil {
		h.writeErr(w, err, "添加成员失败")
		return
	}
	response.Success(w, map[string]bool{"ok": true})
}

// RemoveMember 群主移除成员。DELETE /api/v1/chat/conversations/{id}/members/{userId}
func (h *ChatHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	target := chi.URLParam(r, "userId")
	if tenantID == "" || userID == "" || id == "" || target == "" {
		response.BadRequest(w, "参数不完整")
		return
	}
	if err := h.svc.RemoveMember(r.Context(), tenantID, userID, id, target); err != nil {
		h.writeErr(w, err, "移除成员失败")
		return
	}
	response.Success(w, map[string]bool{"ok": true})
}

// LeaveGroup 退出群聊。POST /api/v1/chat/conversations/{id}/leave
func (h *ChatHandler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	tenantID, userID := currentIdentity(r)
	id := chi.URLParam(r, "id")
	if tenantID == "" || userID == "" || id == "" {
		response.BadRequest(w, "参数不完整")
		return
	}
	if err := h.svc.LeaveGroup(r.Context(), tenantID, userID, id); err != nil {
		h.writeErr(w, err, "退出群聊失败")
		return
	}
	response.Success(w, map[string]bool{"ok": true})
}

// writeErr 将业务错误映射为合适的 HTTP 响应。
func (h *ChatHandler) writeErr(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrConversationNotFound):
		response.NotFound(w, "会话不存在或无权访问")
	case errors.Is(err, service.ErrSelfChatForbidden):
		response.BadRequest(w, "不能与自己发起私聊")
	case errors.Is(err, service.ErrParticipantInvalid):
		response.BadRequest(w, "对方用户不存在或不属于当前租户")
	case errors.Is(err, service.ErrEmptyContent):
		response.BadRequest(w, "消息内容不能为空")
	case errors.Is(err, service.ErrMessageNotFound):
		response.NotFound(w, "消息不存在")
	case errors.Is(err, service.ErrNotMessageOwner):
		response.BadRequest(w, "只能撤回自己发送的消息")
	case errors.Is(err, service.ErrRecallWindowClosed):
		response.BadRequest(w, "已超过可撤回时限")
	case errors.Is(err, service.ErrAlreadyRecalled):
		response.BadRequest(w, "消息已被撤回")
	case errors.Is(err, service.ErrGroupNameEmpty):
		response.BadRequest(w, "群名称不能为空")
	case errors.Is(err, service.ErrGroupMembersEmpty):
		response.BadRequest(w, "群成员不能为空")
	case errors.Is(err, service.ErrGroupMemberLimit):
		response.BadRequest(w, "群成员数超出上限")
	case errors.Is(err, service.ErrNotGroupConversation):
		response.BadRequest(w, "该会话不是群聊会话")
	case errors.Is(err, service.ErrOwnerOnly):
		response.Fail(w, http.StatusForbidden, "仅群主可执行该操作")
	case errors.Is(err, service.ErrCannotRemoveSelf):
		response.BadRequest(w, "不能移除自己，请使用退群")
	case errors.Is(err, service.ErrNotGroupMember):
		response.BadRequest(w, "该用户不在群内")
	default:
		response.Fail(w, http.StatusInternalServerError, fallback)
	}
}

// currentIdentity 从请求上下文提取租户 ID 与用户 ID。
func currentIdentity(r *http.Request) (tenantID, userID string) {
	return contextx.GetTenantID(r.Context()), contextx.GetUserID(r.Context())
}
