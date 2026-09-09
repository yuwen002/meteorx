package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"meteorx/pkg/logger"
)

const (
	// 写入超时
	writeWait = 10 * time.Second
	// 心跳间隔（必须小于对端超时时间）
	pingPeriod = 25 * time.Second
	// 最大消息大小
	maxMessageSize = 4096
	// 发送缓冲区大小
	sendBufferSize = 64
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许所有来源（生产环境需限制）
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client WebSocket 客户端连接
type Client struct {
	Hub      *Hub
	UserID   string
	conn     *websocket.Conn
	send     chan *Message
	done     chan struct{}
	mu       sync.Mutex
	closed   bool
}

// NewClient 创建客户端连接
func NewClient(hub *Hub, userID string, conn *websocket.Conn) *Client {
	return &Client{
		Hub:    hub,
		UserID: userID,
		conn:   conn,
		send:   make(chan *Message, sendBufferSize),
		done:   make(chan struct{}),
	}
}

// ReadPump 读取消息循环（从 WebSocket 读取数据）
func (c *Client) ReadPump() {
	defer func() {
		c.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.Debugf("[WS] Read error for user=%s: %v", c.UserID, err)
			}
			break
		}

		// 处理客户端消息（心跳等）
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case MsgTypePing:
			// 响应心跳
			pong := &Message{
				Type:    MsgTypePong,
				Payload: map[string]interface{}{"time": time.Now().UnixMilli()},
				Time:    time.Now().UnixMilli(),
			}
			select {
			case c.send <- pong:
			default:
			}
		}
	}
}

// WritePump 写入消息循环（从 send 通道读取并写入 WebSocket）
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// 通道已关闭
				c.writeCloseMessage()
				return
			}

			if err := c.writeJSON(message); err != nil {
				logger.Debugf("[WS] Write error for user=%s: %v", c.UserID, err)
				return
			}

		case <-ticker.C:
			// 发送心跳
			if err := c.writePingMessage(); err != nil {
				return
			}

		case <-c.done:
			return
		}
	}
}

// Send 发送消息
func (c *Client) Send(msg *Message) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	select {
	case c.send <- msg:
	default:
		logger.Warnf("[WS] Send buffer full for user=%s", c.UserID)
	}
}

// Close 关闭连接
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	c.closed = true
	close(c.done)
	c.conn.Close()
}

func (c *Client) writeJSON(msg *Message) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteJSON(msg)
}

func (c *Client) writePingMessage() error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteMessage(websocket.PingMessage, nil)
}

func (c *Client) writeCloseMessage() {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	c.conn.WriteMessage(websocket.CloseMessage, []byte{})
}

// ServeWS 处理 WebSocket 升级请求
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, userID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Errorf("[WS] Upgrade failed for user=%s: %v", userID, err)
		return
	}

	client := NewClient(hub, userID, conn)
	hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}

// NewMessage 创建消息
func NewMessage(msgType MessageType, payload interface{}) *Message {
	return &Message{
		Type:    msgType,
		Payload: payload,
		Time:    time.Now().UnixMilli(),
	}
}