package ws

import "sync"

var (
	globalHub *Hub
	once      sync.Once
)

// InitHub 初始化全局 WebSocket Hub
func InitHub() *Hub {
	once.Do(func() {
		globalHub = NewHub()
		go globalHub.Run()
	})
	return globalHub
}

// GetHub 获取全局 WebSocket Hub 实例
func GetHub() *Hub {
	if globalHub == nil {
		return InitHub()
	}
	return globalHub
}

// StopHub 停止全局 WebSocket Hub
func StopHub() {
	if globalHub != nil {
		globalHub.Stop()
	}
}

// SendAlert 发送告警通知给指定用户
func SendAlert(userID string, alert interface{}) {
	hub := GetHub()
	msg := NewMessage(MsgTypeAlert, alert)
	hub.SendToUser(userID, msg)
}

// SendAlertToAll 广播告警通知给所有用户
func SendAlertToAll(alert interface{}) {
	hub := GetHub()
	msg := NewMessage(MsgTypeAlert, alert)
	hub.SendToAll(msg)
}

// SendAnnouncement 发送公告通知给所有用户
func SendAnnouncement(announcement interface{}) {
	hub := GetHub()
	msg := NewMessage(MsgTypeAnnouncement, announcement)
	hub.SendToAll(msg)
}

// SendUnreadCount 更新指定用户的未读数量
func SendUnreadCount(userID string, count int64) {
	hub := GetHub()
	msg := NewMessage(MsgTypeUnreadCount, map[string]int64{"count": count})
	hub.SendToUser(userID, msg)
}