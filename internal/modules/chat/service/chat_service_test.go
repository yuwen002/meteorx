package service

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/modules/chat/model"
	"meteorx/internal/modules/chat/repository"
)

// ---- 假仓储：仅记录调用，满足接口，便于聚焦服务层逻辑 ----

type fakeRepo struct {
	convs     map[string]*model.Conversation
	memberOK  bool // IsMember 返回值
	sent      []*model.ChatMessage
	markRead  []string
	ensureKey string // EnsureDirectConversation 生成/复用会话记录的 key
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{convs: map[string]*model.Conversation{}}
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
