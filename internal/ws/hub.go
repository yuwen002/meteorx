package ws

import (
	"encoding/json"
	"sync"
	"time"

	"meteorx/pkg/logger"
)

// MessageType 消息类型
type MessageType string

const (
	MsgTypeAlert       MessageType = "alert"        // 告警通知
	MsgTypeAnnouncement MessageType = "announcement"  // 公告通知
	MsgTypePing        MessageType = "ping"          // 心跳
	MsgTypePong        MessageType = "pong"          // 心跳响应
	MsgTypeUnreadCount MessageType = "unread_count"  // 未读数量更新
)

// Message WebSocket 消息结构
type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
	Time    int64       `json:"time"`
}

// Hub 连接管理器
type Hub struct {
	mu       sync.RWMutex
	clients  map[string]map[*Client]bool // userID -> clients
	broadcast chan *Message              // 广播通道
	register  chan *Client               // 注册通道
	unregister chan *Client              // 注销通道
	done      chan struct{}             // 关闭信号
}

// NewHub 创建 Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		done:       make(chan struct{}),
	}
}

// Run 启动 Hub 事件循环
func (h *Hub) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.UserID] == nil {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true
			h.mu.Unlock()
			logger.Debugf("[WS] Client registered: user=%s, total=%d", client.UserID, h.Count())

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.UserID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.UserID)
					}
				}
			}
			h.mu.Unlock()
			logger.Debugf("[WS] Client unregistered: user=%s, total=%d", client.UserID, h.Count())

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, clients := range h.clients {
				for client := range clients {
					select {
					case client.send <- message:
					default:
						// 发送缓冲区已满，丢弃消息
						logger.Warnf("[WS] Send buffer full for user=%s, dropping message", client.UserID)
					}
				}
			}
			h.mu.RUnlock()

		case <-ticker.C:
			// 定期清理过期连接
			h.cleanup()

		case <-h.done:
			return
		}
	}
}

// SendToUser 向指定用户发送消息
func (h *Hub) SendToUser(userID string, msg *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[userID]; ok {
		payload, _ := json.Marshal(msg)
		for client := range clients {
			select {
			case client.send <- msg:
			default:
				logger.Warnf("[WS] Send buffer full for user=%s, dropping message", userID)
			}
		}
		logger.Debugf("[WS] Sent to user=%s: %s", userID, string(payload))
	}
}

// SendToAll 向所有用户广播消息
func (h *Hub) SendToAll(msg *Message) {
	select {
	case h.broadcast <- msg:
	default:
		logger.Warnf("[WS] Broadcast channel full, dropping message")
	}
}

// Count 返回当前连接数
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, clients := range h.clients {
		total += len(clients)
	}
	return total
}

// CountByUser 返回指定用户的连接数
func (h *Hub) CountByUser(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.clients[userID])
}

// cleanup 清理已断开的连接
func (h *Hub) cleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for userID, clients := range h.clients {
		for client := range clients {
			select {
			case <-client.done:
				delete(clients, client)
				close(client.send)
			default:
			}
		}
		if len(clients) == 0 {
			delete(h.clients, userID)
		}
	}
}

// Stop 停止 Hub
func (h *Hub) Stop() {
	close(h.done)
}