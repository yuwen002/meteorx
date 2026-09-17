package ws

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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