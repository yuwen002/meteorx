package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"meteorx/internal/common/jwt"
	"meteorx/internal/config"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.done)
	assert.Equal(t, 0, hub.Count())
}

func TestHubRunAndStop(t *testing.T) {
	hub := NewHub()

	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	hub.Stop()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("hub.Run did not exit after Stop")
	}
}

func TestHubCount(t *testing.T) {
	hub := NewHub()
	assert.Equal(t, 0, hub.Count())

	hub.mu.Lock()
	hub.clients["user1"] = map[*Client]bool{{UserID: "user1"}: true}
	hub.mu.Unlock()
	assert.Equal(t, 1, hub.Count())

	hub.mu.Lock()
	hub.clients["user2"] = map[*Client]bool{{UserID: "user2"}: true, {UserID: "user2"}: true}
	hub.mu.Unlock()
	assert.Equal(t, 3, hub.Count())
}

func TestHubCountByUser(t *testing.T) {
	hub := NewHub()

	assert.Equal(t, 0, hub.CountByUser("user1"))

	hub.mu.Lock()
	hub.clients["user1"] = map[*Client]bool{{UserID: "user1"}: true, {UserID: "user1"}: true}
	hub.mu.Unlock()
	assert.Equal(t, 2, hub.CountByUser("user1"))
	assert.Equal(t, 0, hub.CountByUser("user2"))
}

func TestHubSendToAll(t *testing.T) {
	hub := NewHub()

	msg := &Message{
		Type:    MsgTypeAnnouncement,
		Payload: "hello",
		Time:    time.Now().Unix(),
	}

	hub.SendToAll(msg)
}

// TestHubSendToTenant_OnlyMatchingTenant 锁住租户定向推送：只发给目标租户的连接，
// 其它租户的连接一条都不得收到（定向公告不得变相全量广播）。
func TestHubSendToTenant_OnlyMatchingTenant(t *testing.T) {
	hub := NewHub()
	c1 := &Client{UserID: "u1", TenantID: "t1", send: make(chan *Message, 4), done: make(chan struct{})}
	c2 := &Client{UserID: "u2", TenantID: "t2", send: make(chan *Message, 4), done: make(chan struct{})}
	c3 := &Client{UserID: "u3", TenantID: "t1", send: make(chan *Message, 4), done: make(chan struct{})}

	hub.mu.Lock()
	hub.clients["u1"] = map[*Client]bool{c1: true}
	hub.clients["u2"] = map[*Client]bool{c2: true}
	hub.clients["u3"] = map[*Client]bool{c3: true}
	hub.mu.Unlock()

	hub.SendToTenant("t1", &Message{Type: MsgTypeAnnouncement, Payload: "scoped", Time: time.Now().Unix()})

	select {
	case m := <-c1.send:
		assert.Equal(t, "scoped", m.Payload)
	default:
		t.Fatal("t1 的在线连接未收到定向消息")
	}
	select {
	case <-c3.send:
	default:
		t.Fatal("t1 的第二个连接未收到定向消息")
	}
	select {
	case <-c2.send:
		t.Fatal("t2 的连接收到了不属于自己租户的消息（跨租户泄露）")
	default:
	}
}

// TestServeWS_CarriesTenantIDFromClaims 定向公告依赖连接上记录的租户：
// handler 已解析出 claims.TenantID，若不随连接下传到 Client，
// Hub.SendToTenant 永远找不到任何连接，定向公告会静默空转。
func TestServeWS_CarriesTenantIDFromClaims(t *testing.T) {
	helper := jwt.NewTokenHelper(config.JWTConfig{Secret: "ws-test-secret", Expiration: "1h", Issuer: "ws-test"})
	token, err := helper.GenerateToken("u1", "t-1", nil)
	require.NoError(t, err)

	hub := NewHub()
	go hub.Run()
	defer hub.Stop()

	srv := httptest.NewServer(http.HandlerFunc(NewHandler(hub, helper).ServeWS))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	require.Eventually(t, func() bool {
		hub.mu.RLock()
		defer hub.mu.RUnlock()
		for _, clients := range hub.clients {
			for c := range clients {
				if c.UserID == "u1" && c.TenantID == "t-1" {
					return true
				}
			}
		}
		return false
	}, 2*time.Second, 10*time.Millisecond, "连接必须携带 claims 中的租户")
}

func TestHubSendToUser_NoClients(t *testing.T) {
	hub := NewHub()

	msg := &Message{
		Type:    MsgTypeAlert,
		Payload: "alert!",
		Time:    time.Now().Unix(),
	}

	hub.SendToUser("user1", msg)
	assert.Equal(t, 0, hub.CountByUser("user1"))
}

func TestMessageTypes(t *testing.T) {
	assert.Equal(t, MessageType("alert"), MsgTypeAlert)
	assert.Equal(t, MessageType("announcement"), MsgTypeAnnouncement)
	assert.Equal(t, MessageType("ping"), MsgTypePing)
	assert.Equal(t, MessageType("pong"), MsgTypePong)
	assert.Equal(t, MessageType("unread_count"), MsgTypeUnreadCount)
}

func TestMessageStruct(t *testing.T) {
	msg := Message{
		Type:    MsgTypeAlert,
		Payload: "test payload",
		Time:    1234567890,
	}
	assert.Equal(t, MsgTypeAlert, msg.Type)
	assert.Equal(t, "test payload", msg.Payload)
	assert.Equal(t, int64(1234567890), msg.Time)
}

func TestHubStop(t *testing.T) {
	hub := NewHub()

	go hub.Run()
	time.Sleep(10 * time.Millisecond)
	hub.Stop()
}

func TestHubMultipleStop(t *testing.T) {
	hub := NewHub()

	go hub.Run()
	time.Sleep(10 * time.Millisecond)
	hub.Stop()

	assert.Panics(t, func() {
		hub.Stop()
	})
}

func TestHubCleanup(t *testing.T) {
	hub := NewHub()

	client1 := &Client{
		UserID: "user1",
		done:   make(chan struct{}),
		send:   make(chan *Message, 64),
	}
	client2 := &Client{
		UserID: "user1",
		done:   make(chan struct{}),
		send:   make(chan *Message, 64),
	}

	hub.mu.Lock()
	hub.clients["user1"] = map[*Client]bool{client1: true, client2: true}
	hub.mu.Unlock()

	assert.Equal(t, 2, hub.Count())

	close(client1.done)
	hub.cleanup()

	assert.Equal(t, 1, hub.Count())
	assert.Equal(t, 1, hub.CountByUser("user1"))
}