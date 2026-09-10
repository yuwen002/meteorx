// Package notify 提供多渠道消息通知能力。
// 支持的通知渠道：
//   - email:      SMTP 邮件
//   - webhook:    Webhook（支持钉钉/企业微信/飞书/通用 Webhook）
//   - websocket:  站内信（基于 ws.Hub）
//
// 使用方式：
//
//	manager := notify.NewManager(emailer, wsHub)
//	manager.AddChannel("webhook", webhookNotifier)
//	manager.Send(ctx, &notify.Message{...})
package notify

import (
	"context"
	"time"
)

// Priority 通知优先级
type Priority int

const (
	PriorityLow    Priority = iota // 低优先级（如公告推送）
	PriorityNormal                 // 正常优先级
	PriorityHigh                   // 高优先级（如审批提醒）
	PriorityUrgent                 // 紧急（如安全告警）
)

func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityUrgent:
		return "urgent"
	default:
		return "unknown"
	}
}

// ChannelType 通知渠道类型
type ChannelType string

const (
	ChannelEmail    ChannelType = "email"
	ChannelWebhook  ChannelType = "webhook"
	ChannelWebSocket ChannelType = "websocket"
)

// Message 统一通知消息结构
type Message struct {
	ID          string                 // 消息唯一ID
	Title       string                 // 消息标题
	Content     string                 // 消息正文（纯文本或HTML）
	Channel     ChannelType            // 目标渠道
	Priority    Priority               // 优先级
	Recipients  []string               // 收件人列表（邮箱/用户ID/Webhook URL）
	TemplateID  string                 // 模板ID（可选）
	Extra       map[string]interface{} // 扩展参数（渠道特定参数）
	CreatedAt   time.Time              // 创建时间
}

// Notifier 通知渠道接口
// 每个渠道（email/webhook/websocket）都需要实现此接口
type Notifier interface {
	// Type 返回渠道类型
	Type() ChannelType

	// Send 发送单条通知
	// 返回 error 表示发送失败，调用方负责重试或记录失败
	Send(ctx context.Context, msg *Message) error

	// Name 返回渠道名称（用于日志和监控）
	Name() string
}

// SendResult 发送结果
type SendResult struct {
	Channel ChannelType // 渠道类型
	Success bool       // 是否成功
	Error   error      // 错误信息
	Latency time.Duration // 发送耗时
}

// NotifyError 通知错误
type NotifyError struct {
	Channel ChannelType
	MsgID   string
	Err     error
}

func (e *NotifyError) Error() string {
	return "[" + string(e.Channel) + "] send " + e.MsgID + " failed: " + e.Err.Error()
}

func (e *NotifyError) Unwrap() error {
	return e.Err
}