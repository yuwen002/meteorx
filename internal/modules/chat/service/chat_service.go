// Package service 实现协作即时消息（Chat）模块的业务逻辑。
// 首版聚焦同租户内 1:1 私聊：发起/复用会话、发送消息（落库后经 WS 实时投递给对方）、
// 列出会话与历史消息、标记已读。访问控制以会话成员归属为准，仅要求登录，不做 RBAC 拦截。
package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"meteorx/internal/modules/chat/dto"
	"meteorx/internal/modules/chat/model"
	"meteorx/internal/modules/chat/repository"
	"meteorx/internal/ws"
	"meteorx/pkg/idgen"
)

// 业务错误定义
var (
	// ErrConversationNotFound 会话不存在或无权访问
	ErrConversationNotFound = errors.New("conversation not found")
	// ErrSelfChatForbidden 不允许与自己发起私聊
	ErrSelfChatForbidden = errors.New("cannot chat with yourself")
	// ErrParticipantInvalid 对方用户不存在或不属于当前租户
	ErrParticipantInvalid = errors.New("participant is not a member of this tenant")
	// ErrEmptyContent 消息内容为空
	ErrEmptyContent = errors.New("message content is empty")
)

// UserNameResolver 批量解析用户展示名，供会话/消息响应富化姓名。
type UserNameResolver interface {
	ResolveUserNames(ctx context.Context, ids []string) (map[string]string, error)
}

// TenantMemberChecker 校验用户是否为指定租户成员。
type TenantMemberChecker interface {
	IsTenantMember(ctx context.Context, tenantID, userID string) (bool, error)
}

// ChatServiceInterface 聊天服务接口，供 handler 层依赖反转。
type ChatServiceInterface interface {
	OpenDirect(ctx context.Context, tenantID, userID, participantID string) (*model.Conversation, error)
	ListConversations(ctx context.Context, tenantID, userID string) ([]*model.Conversation, error)
	SendMessage(ctx context.Context, tenantID, userID, conversationID, content string) (*model.ChatMessage, error)
	ListMessages(ctx context.Context, tenantID, userID, conversationID string, page, pageSize int) ([]*model.ChatMessage, int64, error)
	MarkRead(ctx context.Context, tenantID, userID, conversationID string) error
}

// ChatService 聊天模块业务服务。
type ChatService struct {
	repo    repository.ChatRepository
	dir     UserNameResolver
	members TenantMemberChecker
	pusher  *ws.Hub // 可为 nil：WS 未启用时仅落库不实时推送
}

// NewChatService 创建聊天业务服务实例。
// names 用于姓名富化；若其同时实现 TenantMemberChecker，则自动启用发起私聊前的租户成员校验。
// hub 可为 nil（WS 未启用时发送仅落库、不实时推送）。
func NewChatService(repo repository.ChatRepository, names UserNameResolver, hub *ws.Hub) *ChatService {
	s := &ChatService{repo: repo, dir: names, pusher: hub}
	if checker, ok := names.(TenantMemberChecker); ok {
		s.members = checker
	}
	return s
}

// OpenDirect 发起/复用与 participantID 的 1:1 私聊会话。
func (s *ChatService) OpenDirect(ctx context.Context, tenantID, userID, participantID string) (*model.Conversation, error) {
	participantID = strings.TrimSpace(participantID)
	if participantID == "" {
		return nil, ErrParticipantInvalid
	}
	if participantID == userID {
		return nil, ErrSelfChatForbidden
	}
	if s.members != nil {
		ok, err := s.members.IsTenantMember(ctx, tenantID, participantID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrParticipantInvalid
		}
	}
	conv, err := s.repo.EnsureDirectConversation(ctx, tenantID, userID, participantID)
	if err != nil {
		return nil, err
	}
	conv.PeerUserID = participantID
	s.enrichConversationNames(ctx, []*model.Conversation{conv})
	return conv, nil
}

// ListConversations 列出当前用户的全部会话。
func (s *ChatService) ListConversations(ctx context.Context, tenantID, userID string) ([]*model.Conversation, error) {
	convs, err := s.repo.ListConversations(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	s.enrichConversationNames(ctx, convs)
	return convs, nil
}

// SendMessage 发送一条消息：校验成员归属后落库，并经 WS 实时投递给会话其他成员。
func (s *ChatService) SendMessage(ctx context.Context, tenantID, userID, conversationID, content string) (*model.ChatMessage, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrEmptyContent
	}
	if err := s.ensureMember(ctx, conversationID, tenantID, userID); err != nil {
		return nil, err
	}

	now := time.Now()
	msg := &model.ChatMessage{
		ID:             idgen.New(),
		ConversationID: conversationID,
		TenantID:       tenantID,
		SenderID:       userID,
		Content:        content,
		Type:           model.MessageTypeText,
		CreatedAt:      now,
	}
	if err := s.repo.SendMessage(ctx, msg); err != nil {
		return nil, err
	}

	// 富化发送人姓名后再返回并推送
	s.enrichMessageNames(ctx, []*model.ChatMessage{msg})

	// 实时投递给会话其他成员（排除发送人本人）；hub 为 nil 时跳过（离线用户下次拉历史即可）
	if s.pusher != nil {
		members, err := s.repo.ListMembers(ctx, conversationID)
		if err == nil {
			resp := ToMessageResp(msg)
			for _, m := range members {
				if m.UserID == userID {
					continue
				}
				s.pusher.SendToUser(m.UserID, ws.NewMessage(ws.MsgTypeChat, resp))
			}
		}
	}
	return msg, nil
}

// ListMessages 分页查询会话历史消息，并校验调用者为会话成员。
func (s *ChatService) ListMessages(ctx context.Context, tenantID, userID, conversationID string, page, pageSize int) ([]*model.ChatMessage, int64, error) {
	if err := s.ensureMember(ctx, conversationID, tenantID, userID); err != nil {
		return nil, 0, err
	}
	msgs, total, err := s.repo.ListMessages(ctx, conversationID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	s.enrichMessageNames(ctx, msgs)
	return msgs, total, nil
}

// MarkRead 将当前用户在该会话的已读水位推进到现在。
func (s *ChatService) MarkRead(ctx context.Context, tenantID, userID, conversationID string) error {
	if err := s.ensureMember(ctx, conversationID, tenantID, userID); err != nil {
		return err
	}
	return s.repo.MarkRead(ctx, conversationID, userID, time.Now())
}

// ensureMember 校验用户为指定会话（同租户）成员，否则返回 ErrConversationNotFound。
func (s *ChatService) ensureMember(ctx context.Context, conversationID, tenantID, userID string) error {
	ok, err := s.repo.IsMember(ctx, conversationID, tenantID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrConversationNotFound
	}
	return nil
}

// enrichConversationNames 批量富化会话对方展示名。
func (s *ChatService) enrichConversationNames(ctx context.Context, convs []*model.Conversation) {
	if s.dir == nil || len(convs) == 0 {
		return
	}
	ids := make([]string, 0, len(convs))
	for _, c := range convs {
		if c.PeerUserID != "" {
			ids = append(ids, c.PeerUserID)
		}
	}
	names, err := s.dir.ResolveUserNames(ctx, ids)
	if err != nil {
		return
	}
	for _, c := range convs {
		c.PeerName = names[c.PeerUserID]
	}
}

// enrichMessageNames 批量富化消息发送人展示名。
func (s *ChatService) enrichMessageNames(ctx context.Context, msgs []*model.ChatMessage) {
	if s.dir == nil || len(msgs) == 0 {
		return
	}
	ids := make([]string, 0, len(msgs))
	for _, m := range msgs {
		ids = append(ids, m.SenderID)
	}
	names, err := s.dir.ResolveUserNames(ctx, ids)
	if err != nil {
		return
	}
	for _, m := range msgs {
		m.SenderName = names[m.SenderID]
	}
}

// ToConversationResp 将领域会话转换为响应对象。
func ToConversationResp(c *model.Conversation) *dto.ConversationResp {
	resp := &dto.ConversationResp{
		ID:          c.ID,
		Type:        c.Type,
		PeerUserID:  c.PeerUserID,
		PeerName:    c.PeerName,
		LastMessage: c.LastMessage,
		UnreadCount: c.UnreadCount,
		CreatedAt:   c.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if c.LastAt != nil {
		resp.LastAt = c.LastAt.Format("2006-01-02 15:04:05")
	}
	return resp
}

// ToMessageResp 将领域消息转换为响应对象。
func ToMessageResp(m *model.ChatMessage) *dto.ChatMessageResp {
	return &dto.ChatMessageResp{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		SenderName:     m.SenderName,
		Content:        m.Content,
		Type:           m.Type,
		CreatedAt:      m.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
