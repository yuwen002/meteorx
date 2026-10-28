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
	convs     map[string]*model.Conversation
	memberOK  bool // IsMember 返回值
	sent      []*model.ChatMessage
	markRead  []string
	ensureKey string // EnsureDirectConversation 生成/复用会话记录的 key
	msgs      map[string]*model.ChatMessage // messageID -> 消息，供撤回用例定位
	recalled  []string                      // 已调用 RecallMessage 的 "convID:msgID"
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{convs: map[string]*model.Conversation{}, msgs: map[string]*model.ChatMessage{}}
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
	return f.memberOK, nil
}

func (f *fakeRepo) ListMembers(ctx context.Context, conversationID string) ([]*model.ConversationMember, error) {
	return []*model.ConversationMember{{UserID: "u1"}, {UserID: "u2"}}, nil
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
