package service

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/modules/chat/model"
	"meteorx/internal/modules/chat/repository"

	"gorm.io/gorm"
)

// ---- 假仓储：仅记录调用，满足接口，便于聚焦服务层逻辑 ----

type fakeRepo struct {
	convs      map[string]*model.Conversation
	memberOK   bool   // IsMember 返回值
	role       string // GetMemberRole 返回的角色（配合 memberOK）
	sent       []*model.ChatMessage
	markRead   []string
	ensureKey  string                        // EnsureDirectConversation 生成/复用会话记录的 key
	msgs       map[string]*model.ChatMessage // messageID -> 消息，供撤回用例定位
	recalled   []string                      // 已调用 RecallMessage 的 "convID:msgID"
	created    *model.Conversation           // CreateGroupConversation 记录的会话
	addedIDs   []string                      // AddMembers 收到的用户 ID
	removed    []string                      // RemoveMember 收到的用户 ID
	deleted    []string                      // DeleteConversation 收到的会话 ID
	renames    map[string]string             // conversationID -> 最新群名
	count      int64                         // CountMembers 返回值
	memberList []*model.ConversationMember   // ListMembers 返回值（非 nil 时优先）
	memberIDs  []string                      // 非空时成员判定改为白名单（IsMember/GetMemberRole）
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		convs:   map[string]*model.Conversation{},
		msgs:    map[string]*model.ChatMessage{},
		renames: map[string]string{},
		count:   2,
	}
}

func (f *fakeRepo) EnsureDirectConversation(ctx context.Context, tenantID, userA, userB string) (*model.Conversation, error) {
	key := model.DirectKey(userA, userB)
	f.ensureKey = key
	conv := &model.Conversation{ID: "conv-" + key, TenantID: tenantID, Type: model.ConversationTypeDirect, Key: key, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.convs[conv.ID] = conv
	return conv, nil
}

func (f *fakeRepo) GetConversation(ctx context.Context, id string) (*model.Conversation, error) {
	return f.convs[id], nil
}

func (f *fakeRepo) IsMember(ctx context.Context, conversationID, tenantID, userID string) (bool, error) {
	if f.memberIDs != nil {
		for _, id := range f.memberIDs {
			if id == userID {
				return true, nil
			}
		}
		return false, nil
	}
	return f.memberOK, nil
}

func (f *fakeRepo) ListMembers(ctx context.Context, conversationID string) ([]*model.ConversationMember, error) {
	if f.memberList != nil {
		return f.memberList, nil
	}
	return []*model.ConversationMember{{UserID: "u1"}, {UserID: "u2"}}, nil
}

func (f *fakeRepo) CreateGroupConversation(ctx context.Context, tenantID, creatorID, name string, memberIDs []string) (*model.Conversation, error) {
	conv := &model.Conversation{ID: "conv-group", TenantID: tenantID, Type: model.ConversationTypeGroup, Name: name, CreatedAt: time.Now()}
	f.convs[conv.ID] = conv
	f.created = conv
	return conv, nil
}

func (f *fakeRepo) UpdateConversationName(ctx context.Context, conversationID, name string) error {
	f.renames[conversationID] = name
	if c, ok := f.convs[conversationID]; ok {
		c.Name = name
	}
	return nil
}

func (f *fakeRepo) DeleteConversation(ctx context.Context, conversationID string) error {
	f.deleted = append(f.deleted, conversationID)
	return nil
}

func (f *fakeRepo) GetMemberRole(ctx context.Context, conversationID, userID string) (string, bool, error) {
	if f.memberIDs != nil {
		for _, id := range f.memberIDs {
			if id == userID {
				role := f.role
				if role == "" {
					role = model.MemberRoleMember
				}
				return role, true, nil
			}
		}
		return "", false, nil
	}
	if !f.memberOK {
		return "", false, nil
	}
	role := f.role
	if role == "" {
		role = model.MemberRoleMember
	}
	return role, true, nil
}

func (f *fakeRepo) CountMembers(ctx context.Context, conversationID string) (int64, error) {
	return f.count, nil
}

func (f *fakeRepo) AddMembers(ctx context.Context, conversationID, tenantID string, userIDs []string) (int64, error) {
	f.addedIDs = append(f.addedIDs, userIDs...)
	return int64(len(userIDs)), nil
}

func (f *fakeRepo) RemoveMember(ctx context.Context, conversationID, userID string) error {
	f.removed = append(f.removed, userID)
	return nil
}

func (f *fakeRepo) ListConversations(ctx context.Context, tenantID, userID string) ([]*model.Conversation, error) {
	return []*model.Conversation{}, nil
}

func (f *fakeRepo) SendMessage(ctx context.Context, msg *model.ChatMessage) error {
	f.sent = append(f.sent, msg)
	return nil
}

func (f *fakeRepo) ListMessages(ctx context.Context, conversationID string, page, pageSize int) ([]*model.ChatMessage, int64, error) {
	return nil, 0, nil
}

func (f *fakeRepo) GetMessage(ctx context.Context, conversationID, messageID string) (*model.ChatMessage, error) {
	m, ok := f.msgs[messageID]
	if !ok || m.ConversationID != conversationID {
		return nil, gorm.ErrRecordNotFound
	}
	return m, nil
}

func (f *fakeRepo) RecallMessage(ctx context.Context, conversationID, messageID string, at time.Time) error {
	f.recalled = append(f.recalled, conversationID+":"+messageID)
	if m, ok := f.msgs[messageID]; ok {
		m.Deleted = true
		m.Content = ""
	}
	return nil
}

func (f *fakeRepo) MarkRead(ctx context.Context, conversationID, userID string, at time.Time) error {
	f.markRead = append(f.markRead, conversationID+":"+userID)
	return nil
}

// ---- 假目录：可解析姓名并校验租户成员 ----

type fakeDir struct {
	member map[string]bool // tenantID|userID -> 是否成员
}

func (d *fakeDir) ResolveUserNames(ctx context.Context, ids []string) (map[string]string, error) {
	m := map[string]string{}
	for _, id := range ids {
		m[id] = "name-" + id
	}
	return m, nil
}

func (d *fakeDir) IsTenantMember(ctx context.Context, tenantID, userID string) (bool, error) {
	return d.member[tenantID+"|"+userID], nil
}

// 编译期断言：假类型满足真实接口
var (
	_ repository.ChatRepository = (*fakeRepo)(nil)
	_ UserNameResolver          = (*fakeDir)(nil)
	_ TenantMemberChecker       = (*fakeDir)(nil)
)

func newService(hubOpt ...bool) (*ChatService, *fakeRepo, *fakeDir) {
	repo := newFakeRepo()
	dir := &fakeDir{member: map[string]bool{"t1|u2": true}}
	// hubOpt 预留（首版服务测试不传真实 hub，pusher 为 nil 亦不应 panic）
	svc := NewChatService(repo, dir, nil)
	return svc, repo, dir
}

func TestOpenDirect_SelfForbidden(t *testing.T) {
	svc, _, _ := newService()
	_, err := svc.OpenDirect(context.Background(), "t1", "u1", "u1")
	if err != ErrSelfChatForbidden {
		t.Fatalf("期望 ErrSelfChatForbidden, got %v", err)
	}
}

func TestOpenDirect_ParticipantInvalid(t *testing.T) {
	svc, _, _ := newService()
	// u9 不在租户 t1 成员映射中
	_, err := svc.OpenDirect(context.Background(), "t1", "u1", "u9")
	if err != ErrParticipantInvalid {
		t.Fatalf("期望 ErrParticipantInvalid, got %v", err)
	}
}

func TestOpenDirect_HappyPath(t *testing.T) {
	svc, repo, _ := newService()
	conv, err := svc.OpenDirect(context.Background(), "t1", "u1", "u2")
	if err != nil {
		t.Fatalf("期望成功, got %v", err)
	}
	if conv.PeerUserID != "u2" {
		t.Fatalf("对方 ID 应为 u2, got %q", conv.PeerUserID)
	}
	if conv.PeerName != "name-u2" {
		t.Fatalf("对方姓名未富化, got %q", conv.PeerName)
	}
	if repo.ensureKey != model.DirectKey("u1", "u2") {
		t.Fatalf("会话 key 规范化错误: %q", repo.ensureKey)
	}
}

func TestSendMessage_EmptyContent(t *testing.T) {
	svc, _, _ := newService()
	_, err := svc.SendMessage(context.Background(), "t1", "u1", "c1", "   ")
	if err != ErrEmptyContent {
		t.Fatalf("期望 ErrEmptyContent, got %v", err)
	}
}

func TestSendMessage_NonMember(t *testing.T) {
	svc, repo, _ := newService()
	repo.memberOK = false
	_, err := svc.SendMessage(context.Background(), "t1", "u1", "c1", "hi")
	if err != ErrConversationNotFound {
		t.Fatalf("非成员期望 ErrConversationNotFound, got %v", err)
	}
	if len(repo.sent) != 0 {
		t.Fatalf("非成员不应落库任何消息, got %d", len(repo.sent))
	}
}

func TestSendMessage_HappyPathNilHubNoPanic(t *testing.T) {
	svc, repo, _ := newService()
	repo.memberOK = true
	msg, err := svc.SendMessage(context.Background(), "t1", "u1", "c1", "hello")
	if err != nil {
		t.Fatalf("期望成功, got %v", err)
	}
	if msg.SenderName != "name-u1" {
		t.Fatalf("发送人姓名未富化, got %q", msg.SenderName)
	}
	if len(repo.sent) != 1 {
		t.Fatalf("应落库 1 条消息, got %d", len(repo.sent))
	}
}

func TestMarkRead_NonMember(t *testing.T) {
	svc, repo, _ := newService()
	repo.memberOK = false
	if err := svc.MarkRead(context.Background(), "t1", "u1", "c1"); err != ErrConversationNotFound {
		t.Fatalf("非成员期望 ErrConversationNotFound, got %v", err)
	}
	repo.memberOK = true
	if err := svc.MarkRead(context.Background(), "t1", "u1", "c1"); err != nil {
		t.Fatalf("成员标记已读应成功, got %v", err)
	}
	if len(repo.markRead) != 1 {
		t.Fatalf("应调用仓储标记 1 次, got %d", len(repo.markRead))
	}
}

// ---- 消息撤回 ----

// putMsg 向假仓储放入一条属于 c1、由 sender 在 offset 前发送的消息。
func putMsg(repo *fakeRepo, id, sender string, offset time.Duration) {
	repo.msgs[id] = &model.ChatMessage{
		ID: id, ConversationID: "c1", TenantID: "t1", SenderID: sender,
		Content: "hi", Type: model.MessageTypeText, CreatedAt: time.Now().Add(-offset),
	}
}

func TestRecallMessage_NonMember(t *testing.T) {
	svc, repo, _ := newService()
	repo.memberOK = false
	putMsg(repo, "m1", "u1", time.Minute)
	if err := svc.RecallMessage(context.Background(), "t1", "u1", "c1", "m1"); err != ErrConversationNotFound {
		t.Fatalf("非成员期望 ErrConversationNotFound, got %v", err)
	}
}

func TestRecallMessage_NotFound(t *testing.T) {
	svc, repo, _ := newService()
	repo.memberOK = true
	if err := svc.RecallMessage(context.Background(), "t1", "u1", "c1", "missing"); err != ErrMessageNotFound {
		t.Fatalf("期望 ErrMessageNotFound, got %v", err)
	}
}

func TestRecallMessage_OtherUsersMessage(t *testing.T) {
	svc, repo, _ := newService()
	repo.memberOK = true
	putMsg(repo, "m1", "u2", time.Minute) // 对方发的消息
	if err := svc.RecallMessage(context.Background(), "t1", "u1", "c1", "m1"); err != ErrNotMessageOwner {
		t.Fatalf("期望 ErrNotMessageOwner, got %v", err)
	}
	if len(repo.recalled) != 0 {
		t.Fatalf("撤回他人消息不应命中仓储, got %v", repo.recalled)
	}
}

func TestRecallMessage_WindowClosed(t *testing.T) {
	svc, repo, _ := newService()
	repo.memberOK = true
	putMsg(repo, "m1", "u1", RecallWindow+time.Minute) // 6 分钟前
	if err := svc.RecallMessage(context.Background(), "t1", "u1", "c1", "m1"); err != ErrRecallWindowClosed {
		t.Fatalf("期望 ErrRecallWindowClosed, got %v", err)
	}
}

func TestRecallMessage_AlreadyRecalled(t *testing.T) {
	svc, repo, _ := newService()
	repo.memberOK = true
	putMsg(repo, "m1", "u1", time.Minute)
	repo.msgs["m1"].Deleted = true
	if err := svc.RecallMessage(context.Background(), "t1", "u1", "c1", "m1"); err != ErrAlreadyRecalled {
		t.Fatalf("期望 ErrAlreadyRecalled, got %v", err)
	}
}

func TestRecallMessage_HappyPath(t *testing.T) {
	svc, repo, _ := newService()
	repo.memberOK = true
	putMsg(repo, "m1", "u1", time.Minute)
	if err := svc.RecallMessage(context.Background(), "t1", "u1", "c1", "m1"); err != nil {
		t.Fatalf("期望成功, got %v", err)
	}
	if len(repo.recalled) != 1 || repo.recalled[0] != "c1:m1" {
		t.Fatalf("应撤回 c1:m1 一次, got %v", repo.recalled)
	}
}

func TestToMessageResp_RecalledHidesContent(t *testing.T) {
	resp := ToMessageResp(&model.ChatMessage{
		ID: "m1", ConversationID: "c1", SenderID: "u1", Content: "secret",
		Type: model.MessageTypeText, Deleted: true, CreatedAt: time.Now(),
	})
	if !resp.Recalled {
		t.Fatal("期望 recalled=true")
	}
	if resp.Content != "" {
		t.Fatalf("已撤回消息不应外泄正文, got %q", resp.Content)
	}
}

// ---- 输入中 / 在线状态（hub 为 nil 不得 panic） ----

func TestNotifyTyping_NilHubNoPanic(t *testing.T) {
	svc, repo, _ := newService()
	repo.memberOK = true
	if err := svc.NotifyTyping(context.Background(), "t1", "u1", "c1", true); err != nil {
		t.Fatalf("成员上报输入状态应成功, got %v", err)
	}
	repo.memberOK = false
	if err := svc.NotifyTyping(context.Background(), "t1", "u1", "c1", true); err != ErrConversationNotFound {
		t.Fatalf("非成员期望 ErrConversationNotFound, got %v", err)
	}
}

func TestPresenceUsers_NilHubEmpty(t *testing.T) {
	svc, _, _ := newService()
	got := svc.PresenceUsers(context.Background())
	if len(got) != 0 {
		t.Fatalf("hub 未启用时应返回空列表, got %v", got)
	}
}

// ---- 群聊 ----

// bootstrapGroup 预设会话 c1（指定类型），调用者 u1 为成员且持有指定角色。
func bootstrapGroup(repo *fakeRepo, convType, role string) {
	repo.convs["c1"] = &model.Conversation{ID: "c1", TenantID: "t1", Type: convType, Name: "old-name"}
	repo.memberOK = true
	repo.role = role
	repo.memberIDs = []string{"u1", "u2"} // u1 调用者 / u2 可操作目标，均在群内
}

func TestCreateGroup_NameEmpty(t *testing.T) {
	svc, _, _ := newService()
	if _, err := svc.CreateGroup(context.Background(), "t1", "u1", "  ", []string{"u2"}); err != ErrGroupNameEmpty {
		t.Fatalf("期望 ErrGroupNameEmpty, got %v", err)
	}
}

func TestCreateGroup_MembersEmptyAfterDedup(t *testing.T) {
	svc, _, _ := newService()
	// 仅剩创建人自己与空串，去重剔除后无有效成员
	if _, err := svc.CreateGroup(context.Background(), "t1", "u1", "g", []string{"u1", ""}); err != ErrGroupMembersEmpty {
		t.Fatalf("期望 ErrGroupMembersEmpty, got %v", err)
	}
}

func TestCreateGroup_MemberLimit(t *testing.T) {
	svc, _, _ := newService()
	ids := make([]string, 0, model.GroupMaxMembers)
	for i := 0; i <= model.GroupMaxMembers; i++ { // 51 个有效成员 + 创建人 > 50
		ids = append(ids, string(rune('a'+i))+"u")
	}
	svc.members = nil // 跳过租户校验，与上限判定解耦
	if _, err := svc.CreateGroup(context.Background(), "t1", "u1", "g", ids); err != ErrGroupMemberLimit {
		t.Fatalf("期望 ErrGroupMemberLimit, got %v", err)
	}
}

func TestCreateGroup_MemberNotInTenant(t *testing.T) {
	svc, _, _ := newService()
	// fakeDir 仅 t1|u2 是成员，u9 不在租户内
	if _, err := svc.CreateGroup(context.Background(), "t1", "u1", "g", []string{"u2", "u9"}); err != ErrParticipantInvalid {
		t.Fatalf("期望 ErrParticipantInvalid, got %v", err)
	}
}

func TestCreateGroup_HappyPathStripsCreator(t *testing.T) {
	svc, repo, _ := newService()
	conv, err := svc.CreateGroup(context.Background(), "t1", "u1", " 研发群 ", []string{"u2", "u1", "u2"})
	if err != nil {
		t.Fatalf("期望成功, got %v", err)
	}
	if repo.created == nil || repo.created.Name != "研发群" {
		t.Fatalf("群名应去首尾空格后落库, got %+v", repo.created)
	}
	if conv.MyRole != model.MemberRoleOwner || conv.MemberCount != 2 || conv.PeerName != "研发群" {
		t.Fatalf("返回会话富化错误: role=%q count=%d name=%q", conv.MyRole, conv.MemberCount, conv.PeerName)
	}
}

func TestRenameGroup_NotGroup(t *testing.T) {
	svc, repo, _ := newService()
	bootstrapGroup(repo, model.ConversationTypeDirect, model.MemberRoleOwner)
	if err := svc.RenameGroup(context.Background(), "t1", "u1", "c1", "new"); err != ErrNotGroupConversation {
		t.Fatalf("期望 ErrNotGroupConversation, got %v", err)
	}
}

func TestRenameGroup_NonOwnerForbidden(t *testing.T) {
	svc, repo, _ := newService()
	bootstrapGroup(repo, model.ConversationTypeGroup, model.MemberRoleMember)
	if err := svc.RenameGroup(context.Background(), "t1", "u1", "c1", "new"); err != ErrOwnerOnly {
		t.Fatalf("期望 ErrOwnerOnly, got %v", err)
	}
	if len(repo.renames) != 0 {
		t.Fatalf("非群主改名不应命中仓储, got %v", repo.renames)
	}
}

func TestRenameGroup_HappyPath(t *testing.T) {
	svc, repo, _ := newService()
	bootstrapGroup(repo, model.ConversationTypeGroup, model.MemberRoleOwner)
	if err := svc.RenameGroup(context.Background(), "t1", "u1", "c1", "新群名"); err != nil {
		t.Fatalf("期望成功, got %v", err)
	}
	if repo.renames["c1"] != "新群名" {
		t.Fatalf("改名未落库, got %v", repo.renames)
	}
}

func TestAddMembers_NotGroup(t *testing.T) {
	svc, repo, _ := newService()
	bootstrapGroup(repo, model.ConversationTypeDirect, model.MemberRoleMember)
	if err := svc.AddMembers(context.Background(), "t1", "u1", "c1", []string{"u9"}); err != ErrNotGroupConversation {
		t.Fatalf("期望 ErrNotGroupConversation, got %v", err)
	}
}

func TestAddMembers_LimitExceeded(t *testing.T) {
	svc, repo, _ := newService()
	bootstrapGroup(repo, model.ConversationTypeGroup, model.MemberRoleMember)
	repo.count = model.GroupMaxMembers // 已满 50 人
	if err := svc.AddMembers(context.Background(), "t1", "u1", "c1", []string{"u2"}); err != ErrGroupMemberLimit {
		t.Fatalf("期望 ErrGroupMemberLimit, got %v", err)
	}
	if len(repo.addedIDs) != 0 {
		t.Fatalf("超限不应添加, got %v", repo.addedIDs)
	}
}

func TestAddMembers_DedupAndTenantCheck(t *testing.T) {
	svc, repo, _ := newService()
	bootstrapGroup(repo, model.ConversationTypeGroup, model.MemberRoleMember)
	if err := svc.AddMembers(context.Background(), "t1", "u1", "c1", []string{"u2", "u2", " "}); err != nil {
		t.Fatalf("期望成功, got %v", err)
	}
	// 去重剔除空串后仅提交 u2（重复邀请由仓储跳过）
	if len(repo.addedIDs) != 1 || repo.addedIDs[0] != "u2" {
		t.Fatalf("应仅提交去重后的 u2, got %v", repo.addedIDs)
	}
}

func TestRemoveMember_SelfForbidden(t *testing.T) {
	svc, repo, _ := newService()
	bootstrapGroup(repo, model.ConversationTypeGroup, model.MemberRoleOwner)
	if err := svc.RemoveMember(context.Background(), "t1", "u1", "c1", "u1"); err != ErrCannotRemoveSelf {
		t.Fatalf("期望 ErrCannotRemoveSelf, got %v", err)
	}
}

func TestRemoveMember_TargetNotInGroup(t *testing.T) {
	svc, repo, _ := newService()
	bootstrapGroup(repo, model.ConversationTypeGroup, model.MemberRoleOwner)
	repo.memberIDs = []string{"u1"} // 目标 uX 不在群内，调用者 u1 仍在
	if err := svc.RemoveMember(context.Background(), "t1", "u1", "c1", "uX"); err != ErrNotGroupMember {
		t.Fatalf("期望 ErrNotGroupMember, got %v", err)
	}
	if len(repo.removed) != 0 {
		t.Fatalf("目标不在群内不应命中移除, got %v", repo.removed)
	}
}

func TestRemoveMember_NonOwnerForbidden(t *testing.T) {
	svc, repo, _ := newService()
	bootstrapGroup(repo, model.ConversationTypeGroup, model.MemberRoleMember)
	if err := svc.RemoveMember(context.Background(), "t1", "u1", "c1", "u2"); err != ErrOwnerOnly {
		t.Fatalf("期望 ErrOwnerOnly, got %v", err)
	}
}

func TestRemoveMember_HappyPathKeepsConversation(t *testing.T) {
	svc, repo, _ := newService()
	bootstrapGroup(repo, model.ConversationTypeGroup, model.MemberRoleOwner)
	repo.count = 3
	if err := svc.RemoveMember(context.Background(), "t1", "u1", "c1", "u2"); err != nil {
		t.Fatalf("期望成功, got %v", err)
	}
	if len(repo.removed) != 1 || repo.removed[0] != "u2" {
		t.Fatalf("应移除 u2, got %v", repo.removed)
	}
	if len(repo.deleted) != 0 {
		t.Fatalf("群内尚有人不应删除会话, got %v", repo.deleted)
	}
}

func TestRemoveMember_LastMemberCleansUp(t *testing.T) {
	svc, repo, _ := newService()
	bootstrapGroup(repo, model.ConversationTypeGroup, model.MemberRoleOwner)
	repo.count = 0 // 移除后已无成员
	if err := svc.RemoveMember(context.Background(), "t1", "u1", "c1", "u2"); err != nil {
		t.Fatalf("期望成功, got %v", err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != "c1" {
		t.Fatalf("空群应软删会话, got %v", repo.deleted)
	}
}

func TestLeaveGroup_DirectForbidden(t *testing.T) {
	svc, repo, _ := newService()
	bootstrapGroup(repo, model.ConversationTypeDirect, model.MemberRoleMember)
	if err := svc.LeaveGroup(context.Background(), "t1", "u1", "c1"); err != ErrNotGroupConversation {
		t.Fatalf("期望 ErrNotGroupConversation, got %v", err)
	}
}

func TestLeaveGroup_HappyPath(t *testing.T) {
	svc, repo, _ := newService()
	bootstrapGroup(repo, model.ConversationTypeGroup, model.MemberRoleOwner)
	repo.count = 1
	if err := svc.LeaveGroup(context.Background(), "t1", "u1", "c1"); err != nil {
		t.Fatalf("期望成功, got %v", err)
	}
	if len(repo.removed) != 1 || repo.removed[0] != "u1" {
		t.Fatalf("应移除自己 u1, got %v", repo.removed)
	}
	if len(repo.deleted) != 0 {
		t.Fatalf("仍有成员不应删会话, got %v", repo.deleted)
	}
}

func TestListMembers_NonMember(t *testing.T) {
	svc, repo, _ := newService()
	repo.memberOK = false
	if _, err := svc.ListMembers(context.Background(), "t1", "u1", "c1"); err != ErrConversationNotFound {
		t.Fatalf("非成员期望 ErrConversationNotFound, got %v", err)
	}
}

func TestListMembers_EnrichesNames(t *testing.T) {
	svc, repo, _ := newService()
	repo.memberOK = true
	repo.memberList = []*model.ConversationMember{{UserID: "u1", Role: model.MemberRoleOwner}, {UserID: "u2", Role: model.MemberRoleMember}}
	members, err := svc.ListMembers(context.Background(), "t1", "u1", "c1")
	if err != nil {
		t.Fatalf("期望成功, got %v", err)
	}
	if len(members) != 2 || members[0].UserName != "name-u1" || members[1].UserName != "name-u2" {
		t.Fatalf("成员姓名未富化, got %+v", members)
	}
}

func TestToConversationResp_CarriesGroupFields(t *testing.T) {
	resp := ToConversationResp(&model.Conversation{
		ID: "c1", Type: model.ConversationTypeGroup, PeerName: "g", Name: "g",
		MemberCount: 5, MyRole: model.MemberRoleOwner, CreatedAt: time.Now(),
	})
	if resp.MemberCount != 5 || resp.MyRole != model.MemberRoleOwner || resp.PeerName != "g" {
		t.Fatalf("群字段未透出: %+v", resp)
	}
}
