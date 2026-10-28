// Package dto 定义协作即时消息（Chat）模块的请求/响应数据传输对象。
package dto

// CreateDirectReq 发起/获取 1:1 私聊会话请求。
type CreateDirectReq struct {
	ParticipantID string `json:"participant_id" validate:"required,max=26" label:"对方用户ID"` // 私聊对方用户 ID
}

// SendMessageReq 发送消息请求。
type SendMessageReq struct {
	Content string `json:"content" validate:"required,max=2000" label:"消息内容"` // 消息正文
}

// ConversationResp 会话响应。
type ConversationResp struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	PeerUserID  string `json:"peer_user_id"`
	PeerName    string `json:"peer_name"`
	LastMessage string `json:"last_message"`
	LastAt      string `json:"last_at,omitempty"`
	UnreadCount int64  `json:"unread_count"`
	CreatedAt   string `json:"created_at"`
}

// ChatMessageResp 聊天消息响应。
type ChatMessageResp struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	SenderName     string `json:"sender_name,omitempty"`
	Content        string `json:"content"`
	Type           string `json:"type"`
	CreatedAt      string `json:"created_at"`
}
