// Package dto 定义协作即时消息（Chat）模块的请求/响应数据传输对象。
package dto

// CreateDirectReq 发起/获取 1:1 私聊会话请求。
type CreateDirectReq struct {
	ParticipantID string `json:"participant_id" validate:"required,max=26" label:"对方用户ID"` // 私聊对方用户 ID
}

// CreateGroupReq 创建群聊会话请求。
type CreateGroupReq struct {
	Name      string   `json:"name" validate:"required,min=1,max=50" label:"群名称"`                      // 群名称
	MemberIDs []string `json:"member_ids" validate:"required,min=1,max=49,dive,max=26" label:"成员ID列表"` // 初始成员（不含创建人）
}

// RenameGroupReq 群改名请求。
type RenameGroupReq struct {
	Name string `json:"name" validate:"required,min=1,max=50" label:"群名称"` // 新群名称
}

// AddMembersReq 向会话添加成员请求。
type AddMembersReq struct {
	UserIDs []string `json:"user_ids" validate:"required,min=1,max=49,dive,max=26" label:"用户ID列表"` // 待加入的用户 ID
}

// SendMessageReq 发送消息请求。
type SendMessageReq struct {
	Content string `json:"content" validate:"required,max=2000" label:"消息内容"` // 消息正文
}

// TypingReq 输入中提示请求（瞬态事件，不落库）。
type TypingReq struct {
	Typing bool `json:"typing" label:"是否正在输入"` // true 开始输入，false 停止
}

// RecallPush 消息撤回的 WS 推送载荷。
type RecallPush struct {
	ConversationID string `json:"conversation_id"` // 所属会话 ID
	MessageID      string `json:"message_id"`      // 被撤回的消息 ID
}

// TypingPush 输入中提示的 WS 推送载荷。
type TypingPush struct {
	ConversationID string `json:"conversation_id"` // 所属会话 ID
	UserID         string `json:"user_id"`         // 正在输入的用户 ID
	Typing         bool   `json:"typing"`          // 输入状态
}

// PresenceResp 在线状态响应：当前在线（持有 WS 连接）的用户 ID 集合。
type PresenceResp struct {
	OnlineUserIDs []string `json:"online_user_ids"` // 在线用户 ID（无序，前端自行判定）
}

// ConversationResp 会话响应。
type ConversationResp struct {
	ID          string `json:"id"`
	Type        string `json:"type"`         // direct / group
	PeerUserID  string `json:"peer_user_id"` // 仅 direct 会话返回对方 ID
	PeerName    string `json:"peer_name"`    // 展示名：direct 为对方昵称，group 为群名称
	LastMessage string `json:"last_message"`
	LastAt      string `json:"last_at,omitempty"`
	UnreadCount int64  `json:"unread_count"`
	MemberCount int64  `json:"member_count"` // 成员数（仅 group 有意义）
	MyRole      string `json:"my_role"`      // 当前用户在该会话的角色：owner / member
	CreatedAt   string `json:"created_at"`
}

// MemberResp 会话成员响应。
type MemberResp struct {
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Role     string `json:"role"` // owner / member
	JoinedAt string `json:"joined_at"`
}

// ChatMessageResp 聊天消息响应。
type ChatMessageResp struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	SenderName     string `json:"sender_name,omitempty"`
	Content        string `json:"content"`
	Type           string `json:"type"`
	Recalled       bool   `json:"recalled"` // 是否已被发送人撤回（软撤回，正文为空）
	CreatedAt      string `json:"created_at"`
}
