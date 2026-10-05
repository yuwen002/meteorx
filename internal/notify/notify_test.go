package notify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"meteorx/internal/ws"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeNotifier 可编排成功/失败与耗时的测试通知渠道，用于验证 Composite/Manager 路由逻辑。
type fakeNotifier struct {
	channel ChannelType
	name    string
	err     error
	calls   int32
	delay   time.Duration
	lastMsg *Message
}

func (f *fakeNotifier) Type() ChannelType { return f.channel }
func (f *fakeNotifier) Name() string      { return f.name }
func (f *fakeNotifier) Send(ctx context.Context, msg *Message) error {
	atomic.AddInt32(&f.calls, 1)
	f.lastMsg = msg
	if f.delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(f.delay):
		}
	}
	return f.err
}

// ---------------------------------------------------------------------------
// notifier.go 基础类型
// ---------------------------------------------------------------------------

func TestPriority_String(t *testing.T) {
	cases := map[Priority]string{
		PriorityLow: "low", PriorityNormal: "normal",
		PriorityHigh: "high", PriorityUrgent: "urgent",
		Priority(99): "unknown",
	}
	for p, want := range cases {
		assert.Equal(t, want, p.String(), "priority %d", p)
	}
}

func TestNotifyError_Wrap(t *testing.T) {
	inner := errors.New("smtp down")
	e := &NotifyError{Channel: ChannelEmail, MsgID: "m1", Err: inner}

	assert.Contains(t, e.Error(), "email")
	assert.Contains(t, e.Error(), "m1")
	assert.Contains(t, e.Error(), "smtp down")
	assert.ErrorIs(t, e, inner)
	assert.Equal(t, inner, e.Unwrap())
}

// ---------------------------------------------------------------------------
// composite_notifier.go 多渠道组合与路由
// ---------------------------------------------------------------------------

func TestCompositeNotifier_RegisterAndCount(t *testing.T) {
	c := NewCompositeNotifier()
	assert.Equal(t, 0, c.ChannelCount())

	c.AddChannel(&fakeNotifier{channel: ChannelEmail, name: "email"})
	c.AddChannel(&fakeNotifier{channel: ChannelWebSocket, name: "ws"})
	assert.Equal(t, 2, c.ChannelCount())
	assert.True(t, c.HasChannel(ChannelEmail))

	// nil 渠道应被忽略
	c.AddChannel(nil)
	assert.Equal(t, 2, c.ChannelCount())

	c.RemoveChannel(ChannelEmail)
	assert.False(t, c.HasChannel(ChannelEmail))
	assert.Equal(t, 1, c.ChannelCount())
}

func TestCompositeNotifier_SendToSpecificChannel(t *testing.T) {
	email := &fakeNotifier{channel: ChannelEmail, name: "email"}
	wsC := &fakeNotifier{channel: ChannelWebSocket, name: "ws"}
	c := NewCompositeNotifier()
	c.AddChannel(email)
	c.AddChannel(wsC)

	results := c.SendToChannel(context.Background(), ChannelEmail, &Message{ID: "1"})
	require.Len(t, results, 1)
	assert.Equal(t, ChannelEmail, results[0].Channel)
	assert.True(t, results[0].Success)
	assert.Equal(t, int32(1), email.calls)
	assert.Equal(t, int32(0), wsC.calls, "非目标渠道不应被调用")
}

func TestCompositeNotifier_SendToAll(t *testing.T) {
	email := &fakeNotifier{channel: ChannelEmail, name: "email"}
	wsC := &fakeNotifier{channel: ChannelWebSocket, name: "ws"}
	c := NewCompositeNotifier()
	c.AddChannel(email)
	c.AddChannel(wsC)

	results := c.SendToAll(context.Background(), &Message{ID: "1"})
	assert.Len(t, results, 2)
	assert.Equal(t, int32(1), email.calls)
	assert.Equal(t, int32(1), wsC.calls)
}

func TestCompositeNotifier_SendUnregisteredChannel(t *testing.T) {
	c := NewCompositeNotifier()
	results := c.SendToChannel(context.Background(), ChannelWebhook, &Message{ID: "x"})
	require.Len(t, results, 1)
	assert.False(t, results[0].Success)
	assert.Contains(t, results[0].Error.Error(), "channel not registered")
}

func TestCompositeNotifier_SendPartialFailure(t *testing.T) {
	boom := errors.New("boom")
	c := NewCompositeNotifier()
	c.AddChannel(&fakeNotifier{channel: ChannelEmail, name: "email", err: boom})
	c.AddChannel(&fakeNotifier{channel: ChannelWebSocket, name: "ws"})

	results := c.SendToAll(context.Background(), &Message{ID: "1"})
	require.Len(t, results, 2)

	var failed, ok int
	for _, r := range results {
		if r.Success {
			ok++
			assert.NoError(t, r.Error)
		} else {
			failed++
			assert.ErrorIs(t, r.Error, boom)
		}
	}
	assert.Equal(t, 1, ok)
	assert.Equal(t, 1, failed)
}

// ---------------------------------------------------------------------------
// manager.go 业务语义方法
// ---------------------------------------------------------------------------

func newTestManager(channels ...Notifier) *Manager {
	m := &Manager{composite: NewCompositeNotifier()}
	for _, ch := range channels {
		m.composite.AddChannel(ch)
	}
	return m
}

func TestManager_Send_FillsIDAndCreatedAt(t *testing.T) {
	email := &fakeNotifier{channel: ChannelEmail, name: "email"}
	m := newTestManager(email)

	msg := &Message{Title: "hi", Channel: ChannelEmail}
	results := m.Send(context.Background(), msg)
	require.Len(t, results, 1)

	assert.NotEmpty(t, msg.ID, "Send 应自动补全消息 ID")
	assert.False(t, msg.CreatedAt.IsZero(), "Send 应自动补全创建时间")
}

func TestManager_NotifyTaskReminder_SkipsWhenNoWS(t *testing.T) {
	m := newTestManager(&fakeNotifier{channel: ChannelEmail, name: "email"})
	// 未注册 WebSocket 渠道 -> 静默跳过，返回 nil
	assert.Nil(t, m.NotifyTaskReminder(context.Background(), "u1", "t", "c", "task_due", nil))
}

func TestManager_NotifyTaskReminder_SkipsEmptyRecipient(t *testing.T) {
	wsC := &fakeNotifier{channel: ChannelWebSocket, name: "ws"}
	m := newTestManager(wsC)
	assert.Nil(t, m.NotifyTaskReminder(context.Background(), "", "t", "c", "task_due", nil))
	assert.Equal(t, int32(0), wsC.calls)
}

func TestManager_NotifyTaskReminder_MergesExtra(t *testing.T) {
	wsC := &fakeNotifier{channel: ChannelWebSocket, name: "ws"}
	m := newTestManager(wsC)

	results := m.NotifyTaskReminder(context.Background(), "u1", "标题", "内容", "task_assigned",
		map[string]interface{}{"task_id": "T-1"})
	require.Len(t, results, 1)
	require.NotNil(t, wsC.lastMsg)

	assert.Equal(t, ChannelWebSocket, wsC.lastMsg.Channel)
	assert.Equal(t, []string{"u1"}, wsC.lastMsg.Recipients)
	assert.Equal(t, "task_assigned", wsC.lastMsg.Extra["type"], "reminderType 应写入 extra.type")
	assert.Equal(t, "T-1", wsC.lastMsg.Extra["task_id"], "调用方 extra 应被合并")
}

func TestManager_NotifyCancelRequestStatus_Rejected(t *testing.T) {
	email := &fakeNotifier{channel: ChannelEmail, name: "email"}
	wsC := &fakeNotifier{channel: ChannelWebSocket, name: "ws"}
	m := newTestManager(email, wsC)

	results := m.NotifyCancelRequestStatus(context.Background(), "ACME", "rejected", "不符合政策", "admin@x.com")
	// WebSocket + Email 两条
	assert.Len(t, results, 2)
	require.NotNil(t, email.lastMsg)
	assert.Contains(t, email.lastMsg.Content, "已被驳回")
	assert.Contains(t, email.lastMsg.Content, "不符合政策")
	assert.Equal(t, []string{"admin@x.com"}, email.lastMsg.Recipients)
}

func TestManager_NotifySubscriptionExpiry_OnlyEmailWhenNoWS(t *testing.T) {
	email := &fakeNotifier{channel: ChannelEmail, name: "email"}
	m := newTestManager(email)

	results := m.NotifySubscriptionExpiry(context.Background(), "ACME", 3, "admin@x.com")
	require.Len(t, results, 1)
	assert.Contains(t, email.lastMsg.Content, "3 天后到期")
}

func TestItoa(t *testing.T) {
	assert.Equal(t, "0", itoa(0))
	assert.Equal(t, "7", itoa(7))
	assert.Equal(t, "123", itoa(123))
}

func TestManager_GlobalAccessors(t *testing.T) {
	prev := GlobalManager
	t.Cleanup(func() { GlobalManager = prev })

	m := newTestManager()
	SetGlobalManager(m)
	assert.Same(t, m, GetGlobalManager())
}

// ---------------------------------------------------------------------------
// webhook_notifier.go payload 构建与 HTTP 发送
// ---------------------------------------------------------------------------

func TestWebhookNotifier_KindPayloads(t *testing.T) {
	msg := &Message{Title: "T", Content: "C", Priority: PriorityUrgent, CreatedAt: time.Unix(1700000000, 0)}

	// generic
	var generic genericWebhookPayload
	data := marshalPayload(t, NewWebhookNotifier(WebhookOption{Kind: WebhookGeneric}), msg)
	require.NoError(t, json.Unmarshal(data, &generic))
	assert.Equal(t, "T", generic.Title)
	assert.Equal(t, "urgent", generic.Priority)
	assert.Equal(t, int64(1700000000), generic.Time)

	// dingtalk
	var dt dingTalkPayload
	data = marshalPayload(t, NewWebhookNotifier(WebhookOption{Kind: WebhookDingTalk}), msg)
	require.NoError(t, json.Unmarshal(data, &dt))
	assert.Equal(t, "markdown", dt.MsgType)
	require.NotNil(t, dt.Markdown)
	assert.Contains(t, dt.Markdown.Text, "优先级: urgent")

	// wechat
	var wc weChatPayload
	data = marshalPayload(t, NewWebhookNotifier(WebhookOption{Kind: WebhookWeChat}), msg)
	require.NoError(t, json.Unmarshal(data, &wc))
	assert.Equal(t, "markdown", wc.MsgType)
	require.NotNil(t, wc.Markdown)

	// feishu
	var fs feishuPayload
	data = marshalPayload(t, NewWebhookNotifier(WebhookOption{Kind: WebhookFeishu}), msg)
	require.NoError(t, json.Unmarshal(data, &fs))
	assert.Equal(t, "post", fs.MsgType)
	require.NotNil(t, fs.Content)
	require.NotNil(t, fs.Content.Post)
	require.NotNil(t, fs.Content.Post.ZhCN)
	assert.Equal(t, "T", fs.Content.Post.ZhCN.Title)
}

// marshalPayload 调用 buildPayload 并序列化，返回 JSON 字节。
func marshalPayload(t *testing.T, n *WebhookNotifier, msg *Message) []byte {
	t.Helper()
	p, err := n.buildPayload(msg)
	require.NoError(t, err)
	b, err := json.Marshal(p)
	require.NoError(t, err)
	return b
}

func TestWebhookNotifier_NameAndConfigured(t *testing.T) {
	assert.Equal(t, "webhook-dingtalk", NewWebhookNotifier(WebhookOption{Kind: WebhookDingTalk}).Name())
	assert.Equal(t, ChannelWebhook, NewWebhookNotifier(WebhookOption{}).Type())
	assert.True(t, IsWebhookConfigured("https://example.com/hook"))
	assert.False(t, IsWebhookConfigured(""))
}

func TestWebhookNotifier_SendSuccess(t *testing.T) {
	var gotCT string
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewWebhookNotifier(WebhookOption{Kind: WebhookGeneric, URL: srv.URL})
	err := n.Send(context.Background(), &Message{Title: "T", Content: "C", CreatedAt: time.Now()})
	require.NoError(t, err)
	assert.Equal(t, "application/json", gotCT)
	assert.Contains(t, string(body), "\"title\":\"T\"")
}

func TestWebhookNotifier_SendErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer srv.Close()

	n := NewWebhookNotifier(WebhookOption{URL: srv.URL})
	err := n.Send(context.Background(), &Message{CreatedAt: time.Now()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "502")
}

func TestWebhookNotifier_SendContextCanceled(t *testing.T) {
	n := NewWebhookNotifier(WebhookOption{URL: "https://example.com"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := n.Send(ctx, &Message{})
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// email_notifier.go 纯逻辑（不触发 SMTP）
// ---------------------------------------------------------------------------

func TestNewEmailNotifier_NilEmailer(t *testing.T) {
	// emailer 为 nil 时渠道不可用
	// 注意：返回值类型是 *EmailNotifier，需以接口变量判空才能捕获 typed-nil 之外的语义
	assert.True(t, NewEmailNotifier(nil, "from", "name") == nil)
}

func TestEscapeHTML(t *testing.T) {
	in := `<a href="x">&'quote'</a>`
	out := escapeHTML(in)
	assert.NotContains(t, out, "<")
	assert.Contains(t, out, "&lt;")
	assert.Contains(t, out, "&amp;")
	assert.Contains(t, out, "&quot;")
	assert.Contains(t, out, "&#39;")
}

func TestEmailNotifier_BuildHTMLBody(t *testing.T) {
	n := &EmailNotifier{from: "a@b.com", fromName: "MeteorX"}
	body := n.buildHTMLBody(&Message{Title: "标题", Content: "<script>x</script>", Priority: PriorityUrgent})

	assert.Contains(t, body, "标题")
	assert.Contains(t, body, "🔴 紧急")
	// 内容应被转义，避免 HTML 注入
	assert.NotContains(t, body, "<script>")
	assert.Contains(t, body, "&lt;script&gt;")
}

func TestEmailNotifier_Send_NoRecipients(t *testing.T) {
	// emailer 为 nil 也不会走到发送逻辑：先返回 no recipients
	n := &EmailNotifier{}
	err := n.Send(context.Background(), &Message{Title: "t"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no recipients")
}

func TestEmailNotifier_Send_ContextCanceled(t *testing.T) {
	n := &EmailNotifier{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := n.Send(ctx, &Message{Recipients: []string{"a@b.com"}})
	assert.ErrorIs(t, err, context.Canceled)
}

// ---------------------------------------------------------------------------
// ws_notifier.go 站内信推送
// ---------------------------------------------------------------------------

func TestNewWSNotifier_NilHub(t *testing.T) {
	assert.True(t, NewWSNotifier(nil) == nil)
}

func TestWSNotifier_SendBroadcast(t *testing.T) {
	hub := ws.NewHub() // 不启动 Run，SendToAll 仅入缓冲通道
	n := NewWSNotifier(hub)

	err := n.Send(context.Background(), &Message{Title: "T", Content: "C", Priority: PriorityNormal, CreatedAt: time.Now()})
	require.NoError(t, err)
}

func TestWSNotifier_SendToUser_NoPanic(t *testing.T) {
	hub := ws.NewHub()
	n := NewWSNotifier(hub)

	err := n.Send(context.Background(), &Message{
		Title: "T", Content: "C",
		Recipients: []string{"u1", "", "u2"},
		CreatedAt:  time.Now(),
		Extra:      map[string]interface{}{"type": "task_due"},
	})
	require.NoError(t, err)
}

func TestWSNotifier_SendContextCanceled(t *testing.T) {
	n := NewWSNotifier(ws.NewHub())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := n.Send(ctx, &Message{})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestJoinStrings(t *testing.T) {
	assert.Equal(t, "", joinStrings(nil, "; "))
	assert.Equal(t, "a", joinStrings([]string{"a"}, "; "))
	assert.Equal(t, "a; b; c", joinStrings([]string{"a", "b", "c"}, "; "))
}
