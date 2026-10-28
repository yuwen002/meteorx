// Package repository 定义协作即时消息（Chat）模块的仓储接口。
package repository

import (
	"context"
	"time"

	"meteorx/internal/modules/chat/model"
)

// ChatRepository 聊天仓储接口，定义会话与消息的持久化操作。
type ChatRepository interface {
	// EnsureDirectConversation 获取或创建租户内 userA、userB 的 1:1 私聊会话，
	// 并保证双方成员记录存在；返回规范化后的会话（含去重 key）。
	EnsureDirectConversation(ctx context.Context, tenantID, userA, userB string) (*model.Conversation, error)
	// CreateGroupConversation 创建群聊会话：创建人记为 owner，初始成员记为 member；
	// memberIDs 不得包含 creatorID（去重由服务层保证）。
	CreateGroupConversation(ctx context.Context, tenantID, creatorID, name string, memberIDs []string) (*model.Conversation, error)
	// GetConversation 按 ID 查询未删除会话。
	GetConversation(ctx context.Context, id string) (*model.Conversation, error)
	// UpdateConversationName 重命名会话（群改名）。
	UpdateConversationName(ctx context.Context, conversationID, name string) error
	// DeleteConversation 软删除会话（已无任何成员的群）。
	DeleteConversation(ctx context.Context, conversationID string) error
	// IsMember 判断用户是否为指定会话的成员（租户内）。
	IsMember(ctx context.Context, conversationID, tenantID, userID string) (bool, error)
	// GetMemberRole 返回用户在指定会话中的角色；非成员时 ok=false。
	GetMemberRole(ctx context.Context, conversationID, userID string) (role string, ok bool, err error)
	// ListMembers 列出会话全部成员。
	ListMembers(ctx context.Context, conversationID string) ([]*model.ConversationMember, error)
	// CountMembers 统计会话成员数。
	CountMembers(ctx context.Context, conversationID string) (int64, error)
	// AddMembers 向会话添加成员（已存在的跳过），返回实际新增人数。
	AddMembers(ctx context.Context, conversationID, tenantID string, userIDs []string) (int64, error)
	// RemoveMember 从会话移除指定成员。
	RemoveMember(ctx context.Context, conversationID, userID string) error
	// ListConversations 列出某用户的全部会话，并富化对方、最后消息与未读数。
	ListConversations(ctx context.Context, tenantID, userID string) ([]*model.Conversation, error)
	// SendMessage 落库一条消息。
	SendMessage(ctx context.Context, msg *model.ChatMessage) error
	// GetMessage 按会话与消息 ID 查询一条消息；不存在或已软删除时返回 gorm.ErrRecordNotFound。
	GetMessage(ctx context.Context, conversationID, messageID string) (*model.ChatMessage, error)
	// RecallMessage 软撤回消息：仅对未撤回的消息置 deleted 标记并清空正文。
	RecallMessage(ctx context.Context, conversationID, messageID string, at time.Time) error
	// ListMessages 分页查询会话历史消息（按时间升序返回，便于前端顺序渲染）。
	ListMessages(ctx context.Context, conversationID string, page, pageSize int) ([]*model.ChatMessage, int64, error)
	// MarkRead 将会话中某成员的已读水位更新为 at。
	MarkRead(ctx context.Context, conversationID, userID string, at time.Time) error
}
