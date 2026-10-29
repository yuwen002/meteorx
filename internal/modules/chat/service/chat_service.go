// Package service 实现协作即时消息（Chat）模块的业务逻辑。
// 首版聚焦同租户内 1:1 私聊：发起/复用会话、发送消息（落库后经 WS 实时投递给对方）、
// 列出会话与历史消息、标记已读。访问控制以会话成员归属为准，仅要求登录，不做 RBAC 拦截。
package service

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"meteorx/internal/common/signedurl"
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
	// ErrMessageNotFound 消息不存在或不属于该会话
	ErrMessageNotFound = errors.New("message not found")
	// ErrNotMessageOwner 只能撤回自己发送的消息
	ErrNotMessageOwner = errors.New("can only recall your own message")
	// ErrRecallWindowClosed 超出可撤回时限
	ErrRecallWindowClosed = errors.New("recall window has closed")
	// ErrAlreadyRecalled 消息已被撤回
	ErrAlreadyRecalled = errors.New("message already recalled")
	// ErrGroupNameEmpty 群名称为空
	ErrGroupNameEmpty = errors.New("group name is empty")
	// ErrGroupMembersEmpty 群初始成员为空（不含创建人）
	ErrGroupMembersEmpty = errors.New("group must have at least one member")
	// ErrGroupMemberLimit 群成员数超出上限
	ErrGroupMemberLimit = errors.New("group member limit exceeded")
	// ErrNotGroupConversation 目标会话不是群聊会话
	ErrNotGroupConversation = errors.New("conversation is not a group")
	// ErrOwnerOnly 仅群主可执行该操作
	ErrOwnerOnly = errors.New("only the group owner can do this")
	// ErrCannotRemoveSelf 不能移除自己，请使用退群
	ErrCannotRemoveSelf = errors.New("cannot remove yourself, use leave instead")
	// ErrNotGroupMember 目标用户不在群内
	ErrNotGroupMember = errors.New("user is not a member of the group")
	// ErrInvalidMessageType 不支持的消息类型
	ErrInvalidMessageType = errors.New("unsupported message type")
	// ErrInvalidMediaContent 媒体消息正文不是合法的结构化元信息
	ErrInvalidMediaContent = errors.New("invalid media message content")
)

// RecallWindow 发送人可撤回自身消息的时间窗口，超时后拒绝撤回。
const RecallWindow = 5 * time.Minute

// mediaFileRe 媒体消息引用的存储文件名：仅允许单段安全字符，杜绝路径穿越与任意名签名。
var mediaFileRe = regexp.MustCompile(`^[A-Za-z0-9._-]{1,120}$`)

// mediaPayload 图片/文件消息的正文结构（以 JSON 字符串持久化于 Content）：
// file 为文件模块返回的存储文件名（/uploads 下），name/size 仅供展示。
type mediaPayload struct {
	File string `json:"file"` // 存储文件名（不含目录）
	Name string `json:"name"` // 原始文件名（展示用）
	Size int64  `json:"size"` // 字节数（展示用）
}

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
	SendMessage(ctx context.Context, tenantID, userID, conversationID, content, mentionIDs, msgType string) (*model.ChatMessage, error)
	ListMessages(ctx context.Context, tenantID, userID, conversationID string, page, pageSize int) ([]*model.ChatMessage, int64, error)
	MarkRead(ctx context.Context, tenantID, userID, conversationID string) error
	RecallMessage(ctx context.Context, tenantID, userID, conversationID, messageID string) error
	NotifyTyping(ctx context.Context, tenantID, userID, conversationID string, typing bool) error
	PresenceUsers(ctx context.Context) []string
	// 群聊：创建/改名/加人/踢人/退群/成员列表
	CreateGroup(ctx context.Context, tenantID, userID, name string, memberIDs []string) (*model.Conversation, error)
	RenameGroup(ctx context.Context, tenantID, userID, conversationID, name string) error
	AddMembers(ctx context.Context, tenantID, userID, conversationID string, userIDs []string) error
	RemoveMember(ctx context.Context, tenantID, userID, conversationID, targetUserID string) error
	LeaveGroup(ctx context.Context, tenantID, userID, conversationID string) error
	ListMembers(ctx context.Context, tenantID, userID, conversationID string) ([]*model.ConversationMember, error)
	// ReadStatus 返回会话内除请求者外各成员的已读水位，供已读回执计算。
	ReadStatus(ctx context.Context, tenantID, userID, conversationID string) ([]*model.ConversationMember, error)
}

// ChatService 聊天模块业务服务。
type ChatService struct {
	repo    repository.ChatRepository
	dir     UserNameResolver
	members TenantMemberChecker
	pusher  *ws.Hub // 可为 nil：WS 未启用时仅落库不实时推送

	// /uploads 静态资源签名参数（媒体消息访问 URL 临时签发用），为空时不签发
	uploadBaseURL string
	uploadSignKey string
}

// SetUploadSigner 注入 /uploads 短时效签名参数（与 wiki 图片签名同源配置），仅模块初始化时调用。
func (s *ChatService) SetUploadSigner(uploadBaseURL, signKey string) {
	s.uploadBaseURL = strings.TrimSuffix(strings.TrimSpace(uploadBaseURL), "/")
	s.uploadSignKey = strings.TrimSpace(signKey)
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
// mentionIDs 为逗号分隔的 @提及用户 ID，仅保留属于本会话的有效成员（剔除自己与非成员）。
// msgType 缺省为 text；image/file 时 content 须为合法媒体 JSON（发送时校验，读取时现签访问 URL）。
func (s *ChatService) SendMessage(ctx context.Context, tenantID, userID, conversationID, content, mentionIDs, msgType string) (*model.ChatMessage, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrEmptyContent
	}
	if msgType == "" {
		msgType = model.MessageTypeText
	}
	switch msgType {
	case model.MessageTypeText, model.MessageTypeImage, model.MessageTypeFile:
	default:
		return nil, ErrInvalidMessageType
	}
	if msgType != model.MessageTypeText {
		if _, err := parseMediaPayload(content); err != nil {
			return nil, ErrInvalidMediaContent
		}
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
		Type:           msgType,
		MentionIDs:     s.filterMentionIDs(ctx, conversationID, userID, mentionIDs),
		CreatedAt:      now,
	}
	if err := s.repo.SendMessage(ctx, msg); err != nil {
		return nil, err
	}

	// 富化发送人姓名与媒体访问 URL 后再返回并推送
	s.enrichMessageNames(ctx, []*model.ChatMessage{msg})
	s.attachMediaURL(msg)

	// 实时投递给会话其他成员（排除发送人本人）；hub 为 nil 时跳过（离线用户下次拉历史即可）
	s.pushToOthers(ctx, conversationID, userID, ws.NewMessage(ws.MsgTypeChat, ToMessageResp(msg)))
	return msg, nil
}

// pushToOthers 向会话内除 excludeUserID 外的全部成员投递一条 WS 消息；hub 为 nil 时直接返回。
func (s *ChatService) pushToOthers(ctx context.Context, conversationID, excludeUserID string, msg *ws.Message) {
	if s.pusher == nil {
		return
	}
	members, err := s.repo.ListMembers(ctx, conversationID)
	if err != nil {
		return
	}
	for _, m := range members {
		if m.UserID == excludeUserID {
			continue
		}
		s.pusher.SendToUser(m.UserID, msg)
	}
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
	for _, m := range msgs {
		s.attachMediaURL(m)
	}
	return msgs, total, nil
}

// MarkRead 将当前用户在该会话的已读水位推进到现在，并实时告知其他成员（已读回执）。
func (s *ChatService) MarkRead(ctx context.Context, tenantID, userID, conversationID string) error {
	if err := s.ensureMember(ctx, conversationID, tenantID, userID); err != nil {
		return err
	}
	now := time.Now()
	if err := s.repo.MarkRead(ctx, conversationID, userID, now); err != nil {
		return err
	}
	// 推送失败不影响已读水位本身（hub 为 nil 时跳过，离线成员下次拉 read-status 即可）
	s.pushToOthers(ctx, conversationID, userID, ws.NewMessage(ws.MsgTypeChatRead, dto.ReadPush{
		ConversationID: conversationID,
		UserID:         userID,
		ReadAt:         now.Format("2006-01-02 15:04:05"),
	}))
	return nil
}

// RecallMessage 撤回自己发送的消息：软撤回（置标记、清空正文）后向其他成员投递撤回事件。
func (s *ChatService) RecallMessage(ctx context.Context, tenantID, userID, conversationID, messageID string) error {
	if err := s.ensureMember(ctx, conversationID, tenantID, userID); err != nil {
		return err
	}
	msg, err := s.repo.GetMessage(ctx, conversationID, messageID)
	if err != nil {
		return ErrMessageNotFound
	}
	if msg.Deleted {
		return ErrAlreadyRecalled
	}
	if msg.SenderID != userID {
		return ErrNotMessageOwner
	}
	if time.Since(msg.CreatedAt) > RecallWindow {
		return ErrRecallWindowClosed
	}
	if err := s.repo.RecallMessage(ctx, conversationID, messageID, time.Now()); err != nil {
		return err
	}
	s.pushToOthers(ctx, conversationID, userID, ws.NewMessage(ws.MsgTypeChatRecall, dto.RecallPush{
		ConversationID: conversationID,
		MessageID:      messageID,
	}))
	return nil
}

// NotifyTyping 输入中提示：无状态转发给会话其他成员，不落库、不存储。
func (s *ChatService) NotifyTyping(ctx context.Context, tenantID, userID, conversationID string, typing bool) error {
	if err := s.ensureMember(ctx, conversationID, tenantID, userID); err != nil {
		return err
	}
	s.pushToOthers(ctx, conversationID, userID, ws.NewMessage(ws.MsgTypeChatTyping, dto.TypingPush{
		ConversationID: conversationID,
		UserID:         userID,
		Typing:         typing,
	}))
	return nil
}

// PresenceUsers 返回当前持有 WS 连接的用户 ID；hub 未启用时返回空列表。
func (s *ChatService) PresenceUsers(ctx context.Context) []string {
	if s.pusher == nil {
		return []string{}
	}
	return s.pusher.OnlineUserIDs()
}

// ---- 群聊 ----

// CreateGroup 创建群聊会话：创建人不计入成员列表，群名必填，
// 成员（含创建人）不得超过上限；若启用租户校验则逐个确认同租户。
func (s *ChatService) CreateGroup(ctx context.Context, tenantID, userID, name string, memberIDs []string) (*model.Conversation, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrGroupNameEmpty
	}
	// 去重并剔除创建人（自己与自己建群无意义）
	seen := make(map[string]struct{}, len(memberIDs))
	ids := make([]string, 0, len(memberIDs))
	for _, id := range memberIDs {
		id = strings.TrimSpace(id)
		if id == "" || id == userID {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) < model.GroupMinMembers {
		return nil, ErrGroupMembersEmpty
	}
	if len(ids)+1 > model.GroupMaxMembers {
		return nil, ErrGroupMemberLimit
	}
	if err := s.ensureTenantMembers(ctx, tenantID, ids); err != nil {
		return nil, err
	}
	conv, err := s.repo.CreateGroupConversation(ctx, tenantID, userID, name, ids)
	if err != nil {
		return nil, err
	}
	conv.MyRole = model.MemberRoleOwner
	conv.MemberCount = int64(len(ids) + 1)
	conv.PeerName = name
	return conv, nil
}

// RenameGroup 群改名：仅群主可操作。
func (s *ChatService) RenameGroup(ctx context.Context, tenantID, userID, conversationID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrGroupNameEmpty
	}
	if _, err := s.ensureGroupOwner(ctx, tenantID, userID, conversationID); err != nil {
		return err
	}
	return s.repo.UpdateConversationName(ctx, conversationID, name)
}

// AddMembers 向群添加成员：任意成员可邀请；校验群上限与同租户后，
// 若确有新增则向全员投递最后一条消息摘要以刷新会话列表。
func (s *ChatService) AddMembers(ctx context.Context, tenantID, userID, conversationID string, userIDs []string) error {
	if err := s.ensureMember(ctx, conversationID, tenantID, userID); err != nil {
		return err
	}
	conv, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil || conv == nil {
		return ErrConversationNotFound
	}
	if conv.Type != model.ConversationTypeGroup {
		return ErrNotGroupConversation
	}
	seen := make(map[string]struct{}, len(userIDs))
	ids := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return ErrGroupMembersEmpty
	}
	cnt, err := s.repo.CountMembers(ctx, conversationID)
	if err != nil {
		return err
	}
	if cnt+int64(len(ids)) > model.GroupMaxMembers {
		return ErrGroupMemberLimit
	}
	if err := s.ensureTenantMembers(ctx, tenantID, ids); err != nil {
		return err
	}
	added, err := s.repo.AddMembers(ctx, conversationID, tenantID, ids)
	if err != nil {
		return err
	}
	if added > 0 {
		// 携带最新成员数推送，接收端可直接更新列表而无需回拉（人数为尽力而为，前端仍以列表接口为准）
		if cnt, err := s.repo.CountMembers(ctx, conversationID); err == nil {
			conv.MemberCount = cnt
		}
		s.pushConversationRefresh(ctx, conv, userID)
	}
	return nil
}

// RemoveMember 群主移除成员；不移除自己，移除后群只剩一人时删除会话。
func (s *ChatService) RemoveMember(ctx context.Context, tenantID, userID, conversationID, targetUserID string) error {
	conv, err := s.ensureGroupOwner(ctx, tenantID, userID, conversationID)
	if err != nil {
		return err
	}
	if targetUserID == userID {
		return ErrCannotRemoveSelf
	}
	if _, ok, err := s.repo.GetMemberRole(ctx, conversationID, targetUserID); err != nil {
		return err
	} else if !ok {
		return ErrNotGroupMember
	}
	if err := s.repo.RemoveMember(ctx, conversationID, targetUserID); err != nil {
		return err
	}
	return s.cleanupIfEmpty(ctx, conversationID, conv)
}

// LeaveGroup 退群：成员均可退；群主退群后群转为无主（仅保留成员角色不再可踢/改名），
// 退群后无人则删除会话。
func (s *ChatService) LeaveGroup(ctx context.Context, tenantID, userID, conversationID string) error {
	if err := s.ensureMember(ctx, conversationID, tenantID, userID); err != nil {
		return err
	}
	conv, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil || conv == nil {
		return ErrConversationNotFound
	}
	if conv.Type != model.ConversationTypeGroup {
		return ErrNotGroupConversation
	}
	if err := s.repo.RemoveMember(ctx, conversationID, userID); err != nil {
		return err
	}
	return s.cleanupIfEmpty(ctx, conversationID, conv)
}

// ListMembers 列出群（或任意会话）成员并富化展示名，调用者必须是成员。
func (s *ChatService) ListMembers(ctx context.Context, tenantID, userID, conversationID string) ([]*model.ConversationMember, error) {
	if err := s.ensureMember(ctx, conversationID, tenantID, userID); err != nil {
		return nil, err
	}
	members, err := s.repo.ListMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	s.enrichMemberNames(ctx, members)
	return members, nil
}

// ReadStatus 返回会话内除请求者外各成员的已读水位（含富化展示名），调用者必须是成员。
// 已读水位仅对同会话成员可见，供发送方计算自己消息的已读回执。
func (s *ChatService) ReadStatus(ctx context.Context, tenantID, userID, conversationID string) ([]*model.ConversationMember, error) {
	if err := s.ensureMember(ctx, conversationID, tenantID, userID); err != nil {
		return nil, err
	}
	members, err := s.repo.ListMembers(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	others := make([]*model.ConversationMember, 0, len(members))
	for _, m := range members {
		if m.UserID != userID {
			others = append(others, m)
		}
	}
	s.enrichMemberNames(ctx, others)
	return others, nil
}

// enrichMemberNames 批量富化成员展示名（解析失败静默跳过，属弱信息）。
func (s *ChatService) enrichMemberNames(ctx context.Context, members []*model.ConversationMember) {
	if s.dir == nil || len(members) == 0 {
		return
	}
	ids := make([]string, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.UserID)
	}
	if names, err := s.dir.ResolveUserNames(ctx, ids); err == nil {
		for _, m := range members {
			m.UserName = names[m.UserID]
		}
	}
}

// ensureGroupOwner 校验会话为群且调用者为群主，返回会话。
func (s *ChatService) ensureGroupOwner(ctx context.Context, tenantID, userID, conversationID string) (*model.Conversation, error) {
	if err := s.ensureMember(ctx, conversationID, tenantID, userID); err != nil {
		return nil, err
	}
	conv, err := s.repo.GetConversation(ctx, conversationID)
	if err != nil || conv == nil {
		return nil, ErrConversationNotFound
	}
	if conv.Type != model.ConversationTypeGroup {
		return nil, ErrNotGroupConversation
	}
	if role, _, err := s.repo.GetMemberRole(ctx, conversationID, userID); err != nil {
		return nil, err
	} else if role != model.MemberRoleOwner {
		return nil, ErrOwnerOnly
	}
	return conv, nil
}

// cleanupIfEmpty 群内已无成员时软删除会话；否则携带最新人数刷新全员会话列表。
func (s *ChatService) cleanupIfEmpty(ctx context.Context, conversationID string, conv *model.Conversation) error {
	cnt, err := s.repo.CountMembers(ctx, conversationID)
	if err != nil {
		return err
	}
	if cnt == 0 {
		return s.repo.DeleteConversation(ctx, conv.ID)
	}
	conv.MemberCount = cnt
	s.pushConversationRefresh(ctx, conv, "")
	return nil
}

// pushConversationRefresh 向全体成员投递会话摘要（带发送人占位），供会话列表/群名/成员数刷新；
// 排除 excludeUserID（操作发起者已同步拿到响应）。hub 为 nil 时跳过。
func (s *ChatService) pushConversationRefresh(ctx context.Context, conv *model.Conversation, excludeUserID string) {
	if s.pusher == nil {
		return
	}
	members, err := s.repo.ListMembers(ctx, conv.ID)
	if err != nil {
		return
	}
	resp := ToConversationResp(conv)
	for _, m := range members {
		if m.UserID == excludeUserID {
			continue
		}
		s.pusher.SendToUser(m.UserID, ws.NewMessage(ws.MsgTypeChatConversation, resp))
	}
}

// ensureTenantMembers 逐个校验目标用户属于当前租户（未启用校验器时跳过）。
func (s *ChatService) ensureTenantMembers(ctx context.Context, tenantID string, userIDs []string) error {
	if s.members == nil {
		return nil
	}
	for _, uid := range userIDs {
		ok, err := s.members.IsTenantMember(ctx, tenantID, uid)
		if err != nil {
			return err
		}
		if !ok {
			return ErrParticipantInvalid
		}
	}
	return nil
}

// filterMentionIDs 保留 @提及中属于本会话的有效成员：去重、剔除空白、发送人自己与非成员。
// 无有效提及或查询失败时返回空串（@提及为增强信息，不阻断发送）。
func (s *ChatService) filterMentionIDs(ctx context.Context, conversationID, senderID, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	members, err := s.repo.ListMembers(ctx, conversationID)
	if err != nil {
		return ""
	}
	valid := make(map[string]struct{}, len(members))
	for _, m := range members {
		valid[m.UserID] = struct{}{}
	}
	seen := make(map[string]struct{})
	kept := make([]string, 0, len(members))
	for _, id := range strings.Split(raw, ",") {
		id = strings.TrimSpace(id)
		if id == "" || id == senderID {
			continue
		}
		if _, ok := valid[id]; !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		kept = append(kept, id)
	}
	return strings.Join(kept, ",")
}

// parseMediaPayload 解析并校验媒体消息正文：file 须为单段安全文件名，name 非空，size 非负。
func parseMediaPayload(content string) (mediaPayload, error) {
	var p mediaPayload
	if err := json.Unmarshal([]byte(content), &p); err != nil {
		return p, err
	}
	if !mediaFileRe.MatchString(p.File) || strings.ContainsAny(p.File, `/\`) {
		return p, errors.New("invalid file name")
	}
	if strings.TrimSpace(p.Name) == "" || p.Size < 0 {
		return p, errors.New("invalid media metadata")
	}
	return p, nil
}

// attachMediaURL 为媒体消息现签短时效访问 URL（与 wiki 内嵌图片同源机制）；
// 未配置签名参数、非媒体类型或正文解析失败时保持为空，前端降级为不可预览展示。
func (s *ChatService) attachMediaURL(m *model.ChatMessage) {
	if s.uploadBaseURL == "" || s.uploadSignKey == "" || m.Deleted {
		return
	}
	if m.Type != model.MessageTypeImage && m.Type != model.MessageTypeFile {
		return
	}
	p, err := parseMediaPayload(m.Content)
	if err != nil {
		return
	}
	m.FileURL = signedurl.Build(s.uploadBaseURL, p.File, s.uploadSignKey, signedurl.Expires(signedurl.DefaultTTL))
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

// enrichConversationNames 批量富化会话展示名：direct 取对方昵称，group 取群名。
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
	if len(ids) == 0 {
		return
	}
	names, err := s.dir.ResolveUserNames(ctx, ids)
	if err != nil {
		return
	}
	for _, c := range convs {
		if c.Type == model.ConversationTypeGroup {
			c.PeerName = c.Name
			continue
		}
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
		ID:              c.ID,
		Type:            c.Type,
		PeerUserID:      c.PeerUserID,
		PeerName:        c.PeerName,
		LastMessage:     c.LastMessage,
		LastMessageType: c.LastMessageType,
		UnreadCount:     c.UnreadCount,
		MemberCount:     c.MemberCount,
		MyRole:          c.MyRole,
		CreatedAt:       c.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if c.LastAt != nil {
		resp.LastAt = c.LastAt.Format("2006-01-02 15:04:05")
	}
	return resp
}

// ToMemberResp 将领域成员转换为响应对象。
func ToMemberResp(m *model.ConversationMember) *dto.MemberResp {
	resp := &dto.MemberResp{
		UserID: m.UserID,
		Name:   m.UserName,
		Role:   m.Role,
	}
	if !m.JoinedAt.IsZero() {
		resp.JoinedAt = m.JoinedAt.Format("2006-01-02 15:04:05")
	}
	return resp
}

// ToMessageResp 将领域消息转换为响应对象。
func ToMessageResp(m *model.ChatMessage) *dto.ChatMessageResp {
	resp := &dto.ChatMessageResp{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		SenderName:     m.SenderName,
		Content:        m.Content,
		Type:           m.Type,
		MentionIDs:     m.MentionIDs,
		Recalled:       m.Deleted,
		URL:            m.FileURL,
		CreatedAt:      m.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	// 已撤回的消息不外泄正文与媒体访问 URL（正文已在软撤回时清空，此处再防一层存量脏数据）
	if resp.Recalled {
		resp.Content = ""
		resp.URL = ""
	}
	return resp
}
