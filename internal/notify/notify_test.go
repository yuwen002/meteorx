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

	results := m.NotifyCancelRequestStatus(context.Background(), "ACME", "rejected", "不符合政策", "admin@x.com", []string{"u-1"})
	// WebSocket + Email 两条
	assert.Len(t, results, 2)
	require.NotNil(t, email.lastMsg)
	assert.Contains(t, email.lastMsg.Content, "已被驳回")
	assert.Contains(t, email.lastMsg.Content, "不符合政策")
	assert.Equal(t, []string{"admin@x.com"}, email.lastMsg.Recipients)
}

// TestManager_NotifyCancelRequestStatus_WSTargetsUserIDsNotEmail 与到期提醒同族：
// 旧实现把邮箱当 WS 收件人，而 hub.clients 以 claims.UserID 为键，永远 miss 且不报错。
func TestManager_NotifyCancelRequestStatus_WSTargetsUserIDsNotEmail(t *testing.T) {
	email := &fakeNotifier{channel: ChannelEmail, name: "email"}
	wsC := &fakeNotifier{channel: ChannelWebSocket, name: "ws"}
	m := newTestManager(email, wsC)

	results := m.NotifyCancelRequestStatus(context.Background(), "ACME", "approved", "", "admin@x.com", []string{"u-1", "u-2"})
	require.Len(t, results, 2)

	require.NotNil(t, wsC.lastMsg)
	assert.Equal(t, []string{"u-1", "u-2"}, wsC.lastMsg.Recipients, "WS 收件人必须是用户 ID")
	assert.NotContains(t, wsC.lastMsg.Recipients, "admin@x.com", "邮箱不得混入 WS 收件人")
	assert.Equal(t, ChannelWebSocket, wsC.lastMsg.Channel)

	require.NotNil(t, email.lastMsg)
	assert.Equal(t, []string{"admin@x.com"}, email.lastMsg.Recipients, "邮箱仅用于邮件渠道")
	assert.Equal(t, ChannelEmail, email.lastMsg.Channel)
}

// TestManager_NotifyCancelRequestStatus_ChannelMessagesIndependent 两个渠道不得复用同一个
// *Message：渠道可能持有对象引用，复用后一份消息的收件人与渠道会被后续渠道覆盖。
// 上面那条 WS 断言 Channel == ChannelWebSocket 在本用例里同样关键（旧写法两渠道拿到的是同一个指针）。
func TestManager_NotifyCancelRequestStatus_ChannelMessagesIndependent(t *testing.T) {
	email := &fakeNotifier{channel: ChannelEmail, name: "email"}
	wsC := &fakeNotifier{channel: ChannelWebSocket, name: "ws"}
	m := newTestManager(email, wsC)

	m.NotifyCancelRequestStatus(context.Background(), "ACME", "approved", "", "admin@x.com", []string{"u-1"})
	require.NotNil(t, email.lastMsg)
	require.NotNil(t, wsC.lastMsg)
	assert.NotSame(t, email.lastMsg, wsC.lastMsg, "两渠道的消息对象必须各自独立")
}

// TestManager_NotifyCancelRequestStatus_SkipsWSWithoutMembers 无成员 ID 时不应产生空转的 WS 结果。
func TestManager_NotifyCancelRequestStatus_SkipsWSWithoutMembers(t *testing.T) {
	wsC := &fakeNotifier{channel: ChannelWebSocket, name: "ws"}
	m := newTestManager(wsC)

	results := m.NotifyCancelRequestStatus(context.Background(), "ACME", "approved", "", "admin@x.com", nil)
	assert.Empty(t, results)
	assert.Equal(t, int32(0), wsC.calls)
}

func TestManager_NotifySubscriptionExpiry_OnlyEmailWhenNoWS(t *testing.T) {
	email := &fakeNotifier{channel: ChannelEmail, name: "email"}
	m := newTestManager(email)

	results := m.NotifySubscriptionExpiry(context.Background(), "ACME", 3, "admin@x.com", nil)
	require.Len(t, results, 1)
	assert.Contains(t, email.lastMsg.Content, "3 天后到期")
}

// TestManager_NotifySubscriptionExpiry_WSTargetsUserIDsNotEmail 锁住 WS 收件人的语义。
// 旧实现把 tenant.ContactEmail 直接当 userID 传进 WS 渠道，而 hub.clients 以 claims.UserID 为键，
// 邮箱永远 miss，SendToUser 对 miss 静默丢弃且不报错——默认部署（email.enabled=false、ws.enabled=true）
// 下到期提醒 100% 空转，还被当成发送成功记入已提醒。
func TestManager_NotifySubscriptionExpiry_WSTargetsUserIDsNotEmail(t *testing.T) {
	email := &fakeNotifier{channel: ChannelEmail, name: "email"}
	wsC := &fakeNotifier{channel: ChannelWebSocket, name: "ws"}
	m := newTestManager(email, wsC)

	results := m.NotifySubscriptionExpiry(context.Background(), "ACME", 3, "admin@x.com", []string{"u-1", "u-2"})
	require.Len(t, results, 2)

	require.NotNil(t, wsC.lastMsg)
	assert.Equal(t, []string{"u-1", "u-2"}, wsC.lastMsg.Recipients, "WS 收件人必须是用户 ID")
	assert.NotContains(t, wsC.lastMsg.Recipients, "admin@x.com", "邮箱不得混入 WS 收件人")

	require.NotNil(t, email.lastMsg)
	assert.Equal(t, []string{"admin@x.com"}, email.lastMsg.Recipients, "邮箱仅用于邮件渠道")
}

// TestManager_NotifySubscriptionExpiry_SkipsWSWithoutMembers 无成员 ID 时不得产生 WS 结果：
// 否则调用方会把「空转发来的 Success=true」当成已提醒，跨重启也不会重试。
func TestManager_NotifySubscriptionExpiry_SkipsWSWithoutMembers(t *testing.T) {
	wsC := &fakeNotifier{channel: ChannelWebSocket, name: "ws"}
	m := newTestManager(wsC)

	results := m.NotifySubscriptionExpiry(context.Background(), "ACME", 3, "admin@x.com", nil)
	assert.Empty(t, results)
	assert.Equal(t, int32(0), wsC.calls)
}

// TestManager_NotifySubscriptionExpiry_NoChannelReturnsNoResults 渠道全不可用时返回空结果，
// 让调用方能区分「真的没送出去」与「已送达」。
func TestManager_NotifySubscriptionExpiry_NoChannelReturnsNoResults(t *testing.T) {
	m := newTestManager()

	results := m.NotifySubscriptionExpiry(context.Background(), "ACME", 3, "admin@x.com", []string{"u-1"})
	assert.Empty(t, results)
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

// stubHub wsHub 的测试桩：只登记在线连接数，并记录实际投递了哪些人。
type stubHub struct {
	online map[string]int
	sent   []string
	all    int
}

func (s *stubHub) SendToUser(userID string, _ *ws.Message) { s.sent = append(s.sent, userID) }
func (s *stubHub) SendToAll(_ *ws.Message)                 { s.all++ }
func (s *stubHub) CountByUser(userID string) int           { return s.online[userID] }

func TestWSNotifier_SendBroadcast(t *testing.T) {
	hub := ws.NewHub() // 不启动 Run，SendToAll 仅入缓冲通道
	n := NewWSNotifier(hub)

	err := n.Send(context.Background(), &Message{Title: "T", Content: "C", Priority: PriorityNormal, CreatedAt: time.Now()})
	require.NoError(t, err)
}

// TestWSNotifier_SendToUser_AllOfflineReportsFailure 锁住空转不再被当成成功。
// 旧写法 `var errs []string` 从未 append，无论收件人在不在都返回 nil，
// CompositeNotifier 按 err == nil 汇报 Success=true，调用方无法区分
// 「确实推进了在线连接」与「一个人都没找到」。
func TestWSNotifier_SendToUser_AllOfflineReportsFailure(t *testing.T) {
	hub := &stubHub{online: map[string]int{}}
	n := &WSNotifier{hub: hub}

	err := n.Send(context.Background(), &Message{
		Title: "T", Content: "C",
		Recipients: []string{"u1", "u2"},
		CreatedAt:  time.Now(),
	})
	require.Error(t, err, "一个收件人都不可达时不得报成功")
	assert.Contains(t, err.Error(), "u1")
	assert.Empty(t, hub.sent, "离线用户不应被投递")
}

// TestWSNotifier_SendToUser_DeliversOnlyOnline 部分在线仍算送达（否则在线用户
// 拿到的推送会被同批离线用户抹掉），但只有在线的那一个被投递。
func TestWSNotifier_SendToUser_DeliversOnlyOnline(t *testing.T) {
	hub := &stubHub{online: map[string]int{"u1": 2}}
	n := &WSNotifier{hub: hub}

	err := n.Send(context.Background(), &Message{
		Title: "T", Content: "C",
		Recipients: []string{"u1", "u2"},
		CreatedAt:  time.Now(),
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"u1"}, hub.sent)
}

// TestWSNotifier_Send_BlankRecipientsBroadcast 收件人全为空串时走广播，
// 不能因「过滤后一个不剩」而被误判为「无人在线」。
func TestWSNotifier_Send_BlankRecipientsBroadcast(t *testing.T) {
	hub := &stubHub{}
	n := &WSNotifier{hub: hub}

	require.NoError(t, n.Send(context.Background(), &Message{
		Title: "T", Content: "C", Recipients: []string{"", ""}, CreatedAt: time.Now(),
	}))
	assert.Equal(t, 1, hub.all)
	assert.Empty(t, hub.sent)
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

// ---------------------------------------------------------------------------
// manager.go NotifyAlert 按渠道类型路由
// ---------------------------------------------------------------------------

// TestManager_NotifyAlert_EmailsOnlyReachEmailChannel 告警通知按渠道类型分发：
// emails 只进邮件渠道，webhook 用渠道绑定的全局 URL、不收收件人，WS 不得被牵进——
// 旧签名把规则里的混合 targets（邮箱 + http URL）直喂 SendToAll：邮件渠道对 URL
// 地址发信失败、WS 拿邮箱永远 miss，而 targets 为空时（wiki 运维告警）空收件人
// 还会把含用户行为明细的告警广播给全体在线用户。
func TestManager_NotifyAlert_EmailsOnlyReachEmailChannel(t *testing.T) {
	email := &fakeNotifier{channel: ChannelEmail, name: "email"}
	wsC := &fakeNotifier{channel: ChannelWebSocket, name: "ws"}
	hook := &fakeNotifier{channel: ChannelWebhook, name: "webhook"}
	m := newTestManager(email, wsC, hook)

	results := m.NotifyAlert(context.Background(), "高风险操作", "用户 alice 删除了资源",
		PriorityUrgent, []string{"a@x.com", "b@x.com"})
	require.Len(t, results, 2, "只有 email 与 webhook 两条结果，WS 不得被牵进")
	assert.Equal(t, int32(0), wsC.calls, "告警不得进 WS：收件人类型不对且广播会泄露用户行为明细")

	require.NotNil(t, email.lastMsg)
	assert.Equal(t, ChannelEmail, email.lastMsg.Channel)
	assert.Equal(t, []string{"a@x.com", "b@x.com"}, email.lastMsg.Recipients)
	assert.Equal(t, "[告警] 高风险操作", email.lastMsg.Title)

	require.NotNil(t, hook.lastMsg)
	assert.Equal(t, ChannelWebhook, hook.lastMsg.Channel)
	assert.Empty(t, hook.lastMsg.Recipients, "webhook 收件人由渠道绑定，不得塞入邮箱")
}

// TestManager_NotifyAlert_SkipsEmailWithoutAddresses 无邮箱收件人时不得空跑邮件渠道：
// EmailNotifier.Send 对空收件人返回 "no recipients" 错误，产生一条永远失败的噪声结果，
// 会干扰调用方的送达判定。
func TestManager_NotifyAlert_SkipsEmailWithoutAddresses(t *testing.T) {
	email := &fakeNotifier{channel: ChannelEmail, name: "email"}
	hook := &fakeNotifier{channel: ChannelWebhook, name: "webhook"}
	m := newTestManager(email, hook)

	results := m.NotifyAlert(context.Background(), "清理失败", "定时任务执行失败", PriorityHigh, nil)
	require.Len(t, results, 1, "只剩 webhook 一条结果")
	assert.Equal(t, int32(0), email.calls)
	assert.Equal(t, int32(1), hook.calls)
}

// TestManager_NotifyAlert_NoChannelsNoResults 未注册任何渠道时返回空结果（no-op），
// 与 wiki 运维告警调用点的既有契约一致。
func TestManager_NotifyAlert_NoChannelsNoResults(t *testing.T) {
	m := newTestManager()
	assert.Empty(t, m.NotifyAlert(context.Background(), "r", "c", PriorityHigh, []string{"a@x.com"}))
}

// TestManager_NotifyAlert_ChannelMessagesIndependent 两渠道各构造一份 Message：
// 复用同一对象时渠道持有的引用会被后续渠道覆盖（与 NotifySubscriptionExpiry 同口径）。
func TestManager_NotifyAlert_ChannelMessagesIndependent(t *testing.T) {
	email := &fakeNotifier{channel: ChannelEmail, name: "email"}
	hook := &fakeNotifier{channel: ChannelWebhook, name: "webhook"}
	m := newTestManager(email, hook)

	m.NotifyAlert(context.Background(), "r", "c", PriorityUrgent, []string{"a@x.com"})
	require.NotNil(t, email.lastMsg)
	require.NotNil(t, hook.lastMsg)
	assert.Equal(t, ChannelEmail, email.lastMsg.Channel, "两渠道消息对象不得相互覆盖")
	assert.Equal(t, ChannelWebhook, hook.lastMsg.Channel)
	assert.Equal(t, []string{"a@x.com"}, email.lastMsg.Recipients, "邮件收件人不得被后发渠道清空")
}
