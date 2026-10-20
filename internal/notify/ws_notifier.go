package notify

import (
	"context"
	"fmt"
	"meteorx/internal/ws"
	"meteorx/pkg/logger"
)

// wsHub 本渠道对 Hub 的最小依赖面（*ws.Hub 天然满足）。
// 声明为接口是为了让可达性判定可测：真实 Hub 的连接表私有，
// 测试里无法造出一个「在线用户」，只能桩掉。
type wsHub interface {
	SendToUser(userID string, msg *ws.Message)
	SendToAll(msg *ws.Message)
	CountByUser(userID string) int
}

// WSNotifier WebSocket 站内信通知渠道
// 将消息通过 WebSocket Hub 推送给在线用户
type WSNotifier struct {
	hub wsHub
}

// NewWSNotifier 创建 WebSocket 通知渠道
// hub 为 nil 时返回 nil，表示渠道不可用
func NewWSNotifier(hub *ws.Hub) *WSNotifier {
	if hub == nil {
		return nil
	}
	return &WSNotifier{hub: hub}
}

// Type 返回 WebSocket 渠道类型。
func (n *WSNotifier) Type() ChannelType { return ChannelWebSocket }

// Name 返回渠道名称。
func (n *WSNotifier) Name() string { return "websocket" }

// Send 发送 WebSocket 通知
// msg.Recipients 为 userID 列表，为空时广播给所有在线用户。
// 无人可达时返回错误：Hub 对 miss 的连接静默丢弃（hub.clients 以 claims.UserID
// 为键），旧写法又从不往 errs 里 append，于是「一个人都没找到」也返回 nil，
// CompositeNotifier 按 err == nil 汇报 Success=true，调用方无从分辨空转与送达。
// 部分在线仍返回 nil：推给了人就算送达，离线名单只记日志。
func (n *WSNotifier) Send(ctx context.Context, msg *Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	payload := map[string]interface{}{
		"title":    msg.Title,
		"content":  msg.Content,
		"priority": msg.Priority.String(),
		"time":     msg.CreatedAt.Unix(),
	}
	if msg.Extra != nil {
		for k, v := range msg.Extra {
			payload[k] = v
		}
	}

	wrapped := &ws.Message{
		Type:    ws.MsgTypeAnnouncement,
		Payload: payload,
		Time:    msg.CreatedAt.UnixMilli(),
	}

	// 先滤掉空串收件人：全为空串时语义等同未指定，走广播而非「无人在线」
	recipients := make([]string, 0, len(msg.Recipients))
	for _, userID := range msg.Recipients {
		if userID != "" {
			recipients = append(recipients, userID)
		}
	}
	if len(recipients) == 0 {
		// 广播给所有在线用户
		n.hub.SendToAll(wrapped)
		return nil
	}

	var delivered int
	var offline []string
	for _, userID := range recipients {
		if n.hub.CountByUser(userID) == 0 {
			offline = append(offline, userID)
			continue
		}
		n.hub.SendToUser(userID, wrapped)
		delivered++
	}

	if delivered == 0 {
		return fmt.Errorf("websocket no recipient online: %s", joinStrings(offline, "; "))
	}
	if len(offline) > 0 {
		logger.Warnf("[WS] %d/%d 收件人不在线，推送未送达: %s", len(offline), len(recipients), joinStrings(offline, "; "))
	}
	return nil
}

// joinStrings 拼接字符串切片
func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
