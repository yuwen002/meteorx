package notify

import (
	"context"
	"fmt"
	"meteorx/internal/ws"
)

// WSNotifier WebSocket 站内信通知渠道
// 将消息通过 WebSocket Hub 推送给在线用户
type WSNotifier struct {
	hub *ws.Hub
}

// NewWSNotifier 创建 WebSocket 通知渠道
// hub 为 nil 时返回 nil，表示渠道不可用
func NewWSNotifier(hub *ws.Hub) *WSNotifier {
	if hub == nil {
		return nil
	}
	return &WSNotifier{hub: hub}
}

func (n *WSNotifier) Type() ChannelType { return ChannelWebSocket }

func (n *WSNotifier) Name() string { return "websocket" }

// Send 发送 WebSocket 通知
// msg.Recipients 为 userID 列表，为空时广播给所有在线用户
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

	if len(msg.Recipients) == 0 {
		// 广播给所有在线用户
		n.hub.SendToAll(wrapped)
		return nil
	}

	// 发送给指定用户
	var errs []string
	for _, userID := range msg.Recipients {
		if userID == "" {
			continue
		}
		n.hub.SendToUser(userID, wrapped)
	}

	if len(errs) > 0 {
		return fmt.Errorf("websocket send partial failures: %s", joinStrings(errs, "; "))
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