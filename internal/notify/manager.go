package notify

import (
	"context"
	"time"

	"meteorx/internal/pkg/emailer"
	"meteorx/internal/ws"
	"meteorx/pkg/idgen"
)

// Manager 通知管理器 - 将所有渠道整合为统一入口
// 提供业务语义化的通知方法，各模块直接调用
type Manager struct {
	composite *CompositeNotifier
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
// 通知所有在线用户（WebSocket）+ 目标用户的邮件
func (m *Manager) NotifyAnnouncement(ctx context.Context, title, content string, targetUserIDs []string) []SendResult {
	msg := &Message{
		Title:      title,
		Content:    content,
		Priority:   PriorityLow,
		Recipients: targetUserIDs,
		Extra: map[string]interface{}{
			"type": "announcement",
		},
	}
	return m.composite.SendToAll(ctx, msg)
}

// NotifyCancelRequestStatus 取消申请状态变更通知
// 通知租户管理员
func (m *Manager) NotifyCancelRequestStatus(ctx context.Context, tenantName string, status string, reason string, adminEmail string) []SendResult {
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

	msg := &Message{
		Title:      title,
		Content:    content,
		Priority:   PriorityHigh,
		Recipients: []string{adminEmail},
		Extra: map[string]interface{}{
			"type":   "cancel_request",
			"status": status,
		},
	}

	// WebSocket 推送给租户管理员，邮件发送
	var results []SendResult
	if m.composite.HasChannel(ChannelWebSocket) {
		msg.Channel = ChannelWebSocket
		results = append(results, m.composite.Send(ctx, msg)...)
	}
	if adminEmail != "" && m.composite.HasChannel(ChannelEmail) {
		msg.Channel = ChannelEmail
		msg.Recipients = []string{adminEmail}
		results = append(results, m.composite.Send(ctx, msg)...)
	}

	return results
}

// NotifyAlert 告警通知
// 发送给告警规则配置的通知目标
func (m *Manager) NotifyAlert(ctx context.Context, ruleName, alertContent string, priority Priority, targets []string) []SendResult {
	msg := &Message{
		Title:      "[告警] " + ruleName,
		Content:    alertContent,
		Priority:   priority,
		Recipients: targets,
		Extra: map[string]interface{}{
			"type": "alert",
		},
	}

	// 告警通知发送到所有可用渠道
	return m.composite.SendToAll(ctx, msg)
}

// NotifySubscriptionExpiry 订阅到期通知
func (m *Manager) NotifySubscriptionExpiry(ctx context.Context, tenantName string, daysLeft int, adminEmail string) []SendResult {
	title := "订阅即将到期"
	content := "租户 \"" + tenantName + "\" 的订阅将在 " + itoa(daysLeft) + " 天后到期，请及时续费。"

	msg := &Message{
		Title:      title,
		Content:    content,
		Priority:   PriorityNormal,
		Recipients: []string{adminEmail},
		Extra: map[string]interface{}{
			"type":      "subscription_expiry",
			"days_left": daysLeft,
		},
	}

	var results []SendResult
	if adminEmail != "" && m.composite.HasChannel(ChannelEmail) {
		msg.Channel = ChannelEmail
		results = append(results, m.composite.Send(ctx, msg)...)
	}
	if m.composite.HasChannel(ChannelWebSocket) {
		msg.Channel = ChannelWebSocket
		results = append(results, m.composite.Send(ctx, msg)...)
	}

	return results
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