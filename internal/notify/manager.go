package notify

import (
	"context"
	"net/url"
	"strings"
	"time"

	"meteorx/internal/pkg/emailer"
	"meteorx/internal/ws"
	"meteorx/pkg/idgen"
	"meteorx/pkg/logger"
)

// Manager 通知管理器 - 将所有渠道整合为统一入口
// 提供业务语义化的通知方法，各模块直接调用
type Manager struct {
	composite *CompositeNotifier
	// inbox 站内信落库能力（可选）：实时推送只解决在线可达，
	// 落库后离线用户重新登录才能看到漏掉的消息
	inbox InboxWriter
}

// NewManager 创建通知管理器
// 自动注册可用的通知渠道：
//   - email: 如果 emailer 已配置
//   - websocket: 如果 ws.Hub 已初始化
func NewManager(e *emailer.Emailer, from, fromName string, hub *ws.Hub) *Manager {
	m := &Manager{
		composite: NewCompositeNotifier(),
	}

	// 注册 WebSocket 站内信渠道
	if hub != nil {
		m.composite.AddChannel(NewWSNotifier(hub))
	}

	// 注册邮件渠道
	if e != nil {
		m.composite.AddChannel(NewEmailNotifier(e, from, fromName))
	}

	return m
}

// AddWebhook 动态注册 Webhook 渠道
func (m *Manager) AddWebhook(name string, opt WebhookOption) {
	m.composite.AddChannel(NewWebhookNotifier(opt))
}

// SetInbox 装配站内信落库能力（由 bootstrap 在消息模块创建后调用）。
// 未装配时站内信落库整体跳过，其余渠道行为不变。
func (m *Manager) SetInbox(w InboxWriter) {
	m.inbox = w
}

// deliverInbox 落库站内信，失败仅告警。
// 刻意不并入 SendResult 参与「送达销账」判定：站内信是补推兜底，
// 把它算作送达会让邮件/WS 的失败被一条本地写库掩盖。
func (m *Manager) deliverInbox(ctx context.Context, msg *InboxMessage) {
	if m.inbox == nil {
		return
	}
	if err := m.inbox.Deliver(ctx, msg); err != nil {
		logger.Warnf("[Notify] 站内信落库失败 type=%s tenant=%s recipients=%d: %v",
			msg.Type, msg.TenantID, len(msg.UserIDs), err)
	}
}

// AddChannel 注册自定义渠道
func (m *Manager) AddChannel(n Notifier) {
	m.composite.AddChannel(n)
}

// Send 发送消息到指定渠道
func (m *Manager) Send(ctx context.Context, msg *Message) []SendResult {
	if msg.ID == "" {
		msg.ID = idgen.New()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	return m.composite.Send(ctx, msg)
}

// NotifyAnnouncement 发布公告通知
// 公告只走 WebSocket 站内信与平台 Webhook，不进邮件渠道：公告没有「邮件收件人」
// 概念，空收件人只会让 EmailNotifier 返回 "no recipients" 噪声。targetTenantID
// 非空表示租户定向公告，WS 渠道据此只推给该租户的在线连接（不得广播）。
func (m *Manager) NotifyAnnouncement(ctx context.Context, title, content, targetTenantID string) []SendResult {
	extra := map[string]interface{}{"type": "announcement"}
	if targetTenantID != "" {
		extra["tenant_id"] = targetTenantID
	}

	var results []SendResult
	if m.composite.HasChannel(ChannelWebSocket) {
		results = append(results, m.composite.Send(ctx, &Message{
			Title: title, Content: content, Priority: PriorityLow,
			Channel: ChannelWebSocket, TenantID: targetTenantID, Extra: extra,
		})...)
	}
	if m.composite.HasChannel(ChannelWebhook) {
		results = append(results, m.composite.Send(ctx, &Message{
			Title: title, Content: content, Priority: PriorityLow,
			Channel: ChannelWebhook, TenantID: targetTenantID, Extra: extra,
		})...)
	}
	// 只有租户定向公告才落收件箱：平台级公告需要向全平台用户逐人写行，
	// 开销随用户量线性增长，而公告本身在「公告列表」常驻可查。
	// 不设 DedupeKey：每次发布都是独立事件，同标题公告也应各自可见。
	if targetTenantID != "" {
		m.deliverInbox(ctx, &InboxMessage{
			TenantID: targetTenantID, Type: "announcement",
			Title: title, Content: content, Priority: PriorityLow,
			SourceType: "announcement",
		})
	}
	return results
}

// NotifyCancelRequestStatus 取消申请状态变更通知
// adminEmail 为邮件收件人（租户联系人），memberUserIDs 为 WebSocket 收件人（租户成员用户 ID）。
// 与 NotifySubscriptionExpiry 同一口径：两个收件人类型不可互换，hub.clients 以
// claims.UserID 为键，传邮箱只会静默 miss；每渠道各构造一份 Message 以免相互覆盖。
func (m *Manager) NotifyCancelRequestStatus(ctx context.Context, tenantName string, status string, reason string,
	adminEmail string, memberUserIDs []string) []SendResult {
	title := "租户注销申请状态更新"
	content := "您的租户 \"" + tenantName + "\" 的注销申请"
	switch status {
	case "approved":
		content += "已通过审批"
		if reason != "" {
			content += "。备注：" + reason
		}
	case "rejected":
		content += "已被驳回"
		if reason != "" {
			content += "。原因：" + reason
		}
	default:
		content += "状态变更为: " + status
	}

	extra := map[string]interface{}{
		"type":   "cancel_request",
		"status": status,
	}

	var results []SendResult
	if adminEmail != "" && m.composite.HasChannel(ChannelEmail) {
		results = append(results, m.composite.Send(ctx, &Message{
			Title: title, Content: content, Priority: PriorityHigh,
			Channel: ChannelEmail, Recipients: []string{adminEmail}, Extra: extra,
		})...)
	}
	// 成员为空时不产生 WS 结果：否则空转发回的 Success=true 会被当成已通知
	if len(memberUserIDs) > 0 && m.composite.HasChannel(ChannelWebSocket) {
		results = append(results, m.composite.Send(ctx, &Message{
			Title: title, Content: content, Priority: PriorityHigh,
			Channel: ChannelWebSocket, Recipients: memberUserIDs, Extra: extra,
		})...)
	}
	if len(memberUserIDs) > 0 {
		m.deliverInbox(ctx, &InboxMessage{
			UserIDs: memberUserIDs, Type: "cancel_request",
			Title: title, Content: content, Priority: PriorityHigh,
			SourceType: "tenant_cancel_request", SourceID: status,
			// 同一租户同一天的同一种审批结果只落一条（审批重试 / 多副本安全）
			DedupeKey: dailyDedupeKey("cancel_request", tenantName, status),
		})
	}

	return results
}

// AlertNotice 告警通知参数。
// 收件人按渠道类型分开：Emails 只装邮箱（邮件渠道），UserIDs 只装用户 ID（收件箱）——
// 收件箱只认用户 ID，传邮箱只会静默 miss（与到期提醒 / 注销通知同一口径）。
type AlertNotice struct {
	// ID 告警记录 ID：收件箱据此去重，同一条告警重复投递只落一条；
	// 为空（如定时任务自检告警）时不参与去重
	ID       string
	RuleName string
	Content  string
	Priority Priority
	Emails   []string
	// UserIDs 收件箱收件人，通常由规则配置的邮箱反查得到。
	// 告警含用户行为明细，调用方不得在此传入租户展开或全平台广播的结果
	UserIDs []string
}

// NotifyAlert 告警通知。
// Emails 只进邮件渠道（已由调用方按类型拆分），为空时不跑邮件渠道。
// Webhook 渠道的收件人由渠道自身绑定、不收 msg.Recipients；WS 不参与——
// 旧写法把规则里的混合 targets（邮箱 + http URL）直喂 SendToAll：邮件对 URL 发信失败、
// WS 拿邮箱永远 miss，而 targets 为空时（wiki 运维告警）空收件人还会把含用户行为
// 明细的告警广播给全体在线用户。
// UserIDs 非空时额外落一条收件箱消息：邮箱只在能收信时有意义，收件箱让同一批
// 收件人重新登录后仍能看到同一条告警；落库失败不影响渠道结果与送达销账。
func (m *Manager) NotifyAlert(ctx context.Context, n AlertNotice) []SendResult {
	title := "[告警] " + n.RuleName
	extra := map[string]interface{}{"type": "alert"}

	var results []SendResult
	// 每渠道各构造一份 Message：渠道可能持有对象引用，复用会被后续渠道覆盖
	if len(n.Emails) > 0 && m.composite.HasChannel(ChannelEmail) {
		results = append(results, m.composite.Send(ctx, &Message{
			Title: title, Content: n.Content, Priority: n.Priority,
			Channel: ChannelEmail, Recipients: n.Emails, Extra: extra,
		})...)
	}
	if m.composite.HasChannel(ChannelWebhook) {
		results = append(results, m.composite.Send(ctx, &Message{
			Title: title, Content: n.Content, Priority: n.Priority,
			Channel: ChannelWebhook, Extra: extra,
		})...)
	}
	if len(n.UserIDs) > 0 {
		// 分类显式指定为 audit（与 message.CategoryForType("alert") 一致）：
		// 告警属安全审计类，不依赖类型映射的默认分支，避免未来改动后
		// 被并入「系统消息」而漏看。
		dedupeKey := ""
		// 链接带上告警记录 ID：点开直接定位到触发这条消息的那条告警。
		// 只落列表首页的话，同类告警堆到几十上百条时用户无从对应到具体事件。
		link := "/system/audit/alert"
		if n.ID != "" {
			// 一条告警是一次独立事件：按 ID 永久去重，渠道重试 / 多副本只落一条。
			// 不用自然日粒度——重复的告警本身没有「第二天再提醒一次」的语义。
			dedupeKey = "alert:" + n.ID
			link += "?alert_id=" + url.QueryEscape(n.ID)
		}
		m.deliverInbox(ctx, &InboxMessage{
			UserIDs: n.UserIDs, Type: "alert", Category: "audit",
			Title: title, Content: n.Content, Link: link,
			Priority: n.Priority, SourceType: "audit_alert", SourceID: n.ID,
			DedupeKey: dedupeKey,
		})
	}
	return results
}

// NotifySubscriptionExpiry 订阅到期通知
// adminEmail 为邮件收件人（租户联系人），memberUserIDs 为 WebSocket 收件人（租户成员用户 ID）。
// 两个收件人类型不可互换：hub.clients 以 claims.UserID 为键，传邮箱会静默 miss。
// tenantID 只用于收件箱链接定位（租户管理页的套餐配置弹窗），不参与 WS 路由：
// 收件人已按租户筛过，再带 TenantID 会让定向推送退化成整租户广播。
// 无任何可用渠道时返回空结果，供调用方判定「未送达」而不销账。
func (m *Manager) NotifySubscriptionExpiry(ctx context.Context, tenantID, tenantName string, daysLeft int, adminEmail string, memberUserIDs []string) []SendResult {
	title := "订阅即将到期"
	content := "租户 \"" + tenantName + "\" 的订阅将在 " + itoa(daysLeft) + " 天后到期，请及时续费。"

	extra := map[string]interface{}{
		"type":      "subscription_expiry",
		"days_left": daysLeft,
	}

	var results []SendResult
	// 每渠道各构造一份 Message：渠道可能持有传入对象的引用（推送队列、测试桩），
	// 复用同一对象再改 Recipients/Channel 会让先前那份消息的收件人被后续渠道覆盖
	if adminEmail != "" && m.composite.HasChannel(ChannelEmail) {
		results = append(results, m.composite.Send(ctx, &Message{
			Title: title, Content: content, Priority: PriorityNormal,
			Channel: ChannelEmail, Recipients: []string{adminEmail}, Extra: extra,
		})...)
	}
	// 成员为空时不产生 WS 结果：否则空转发回的 Success=true 会被当成已提醒
	if len(memberUserIDs) > 0 && m.composite.HasChannel(ChannelWebSocket) {
		results = append(results, m.composite.Send(ctx, &Message{
			Title: title, Content: content, Priority: PriorityNormal,
			Channel: ChannelWebSocket, Recipients: memberUserIDs, Extra: extra,
		})...)
	}
	if len(memberUserIDs) > 0 {
		// 链接只带租户 ID：到期天数由套餐详情接口实时返回（effective_days），
		// 写进 query 只会让用户在消息停留几天后看到与服务端不一致的快照。
		link := "/system/tenant"
		if tenantID != "" {
			link = "/system/tenant?tenant_id=" + url.QueryEscape(tenantID)
		}
		m.deliverInbox(ctx, &InboxMessage{
			UserIDs: memberUserIDs, Type: "subscription_expiry",
			Title: title, Content: content, Link: link, Priority: PriorityNormal,
			SourceType: "subscription", SourceID: itoa(daysLeft),
			// 同一租户同一剩余天数一天只落一条；到期日后移会得到新键
			DedupeKey: dailyDedupeKey("subscription", tenantName, itoa(daysLeft)),
		})
	}

	return results
}

// NotifyTaskReminder 任务提醒站内信
// 通过 WebSocket 定向推送给指定用户（recipientID 为用户 ID），
// reminderType 取值如 task_assigned / task_due，供前端区分展示。
// 未注册 WebSocket 渠道或接收人为空时静默跳过。
func (m *Manager) NotifyTaskReminder(ctx context.Context, recipientID, title, content, reminderType string, extra map[string]interface{}) []SendResult {
	if recipientID == "" {
		return nil
	}

	// 站内信落库不依赖 WS 渠道：用户当前离线时依然要能在收件箱里看到这条提醒
	sourceID := ""
	if extra != nil {
		if v, ok := extra["task_id"].(string); ok {
			sourceID = v
		}
	}
	// 同一任务的同类提醒一天只落一条：渠道重试 / 多副本 / 重启补偿不得重复刷屏；
	// 跨天仍会各自落库，与逾期提醒的 24h 冷却语义一致。
	dedupeKey := ""
	// 链接带上任务 ID：点开直接定位到这条任务，与告警站内信带 alert_id 同一套口径。
	// 只落任务列表首页的话，用户得自己在一页页里翻出被提醒的是哪一条。
	link := "/task"
	if sourceID != "" {
		dedupeKey = dailyDedupeKey("task", sourceID, reminderType)
		link = "/task?task_id=" + url.QueryEscape(sourceID)
	}
	m.deliverInbox(ctx, &InboxMessage{
		UserIDs: []string{recipientID}, Type: reminderType,
		Title: title, Content: content, Link: link, Priority: PriorityNormal,
		SourceType: "task", SourceID: sourceID, DedupeKey: dedupeKey,
	})

	if !m.composite.HasChannel(ChannelWebSocket) {
		return nil
	}

	payloadExtra := map[string]interface{}{"type": reminderType}
	for k, v := range extra {
		payloadExtra[k] = v
	}

	msg := &Message{
		Title:      title,
		Content:    content,
		Channel:    ChannelWebSocket,
		Priority:   PriorityNormal,
		Recipients: []string{recipientID},
		Extra:      payloadExtra,
	}
	return m.composite.Send(ctx, msg)
}

// dailyDedupeKey 构造收件箱去重键：业务定位键 + 自然日。
// 时间粒度取「天」而非「次」：同一天的重复投递（渠道重试、多副本、重启后的补偿）
// 只应留下一条，而跨天的重复提醒（逾期任务每日提醒、订阅每日临期提醒）仍各自落库。
// 长度上限与 user_messages.dedupe_key 列宽一致，超长按字符截断。
func dailyDedupeKey(parts ...string) string {
	segs := make([]string, 0, len(parts)+1)
	segs = append(segs, parts...)
	segs = append(segs, time.Now().Format("20060102"))
	key := strings.Join(segs, ":")
	const maxLen = 191
	if r := []rune(key); len(r) > maxLen {
		key = string(r[:maxLen])
	}
	return key
}

// itoa 简单的整数转字符串
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// ChannelCount 返回已注册渠道数量
func (m *Manager) ChannelCount() int {
	return m.composite.ChannelCount()
}

// GlobalManager 全局通知管理器（由 bootstrap 初始化）
var GlobalManager *Manager

// SetGlobalManager 设置全局通知管理器
func SetGlobalManager(m *Manager) {
	GlobalManager = m
}

// GetGlobalManager 获取全局通知管理器
func GetGlobalManager() *Manager {
	return GlobalManager
}
