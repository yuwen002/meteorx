// Package model 定义协作即时消息（Chat）模块的领域模型。
// 首版仅支持同租户内的 1:1 私聊（direct 会话），会话由参与者唯一确定，
// 消息按会话线性存储，成员各自维护已读水位以计算未读数。
package model

import "time"

// 会话类型常量
const (
	// ConversationTypeDirect 1:1 私聊会话（首版唯一支持的类型）
	ConversationTypeDirect = "direct"
)

// 消息类型常量
const (
	// MessageTypeText 纯文本消息（首版唯一支持的类型）
	MessageTypeText = "text"
)

// Conversation 会话领域模型。
// 首版为 1:1 私聊：同一租户内两个用户由 DeterministicKey 唯一确定一个会话，
// 重复发起私聊时按 key 复用既有会话，不产生新行。
type Conversation struct {
	ID        string    // 会话唯一标识（ULID）
	TenantID  string    // 所属租户 ID（多租户隔离，私聊双方必须同租户）
	Type      string    // 会话类型：direct
	Key       string    // 会话去重键（direct 类型下由参与者 ID 规范化生成，租户内唯一）
	CreatedAt time.Time // 创建时间
	UpdatedAt time.Time // 更新时间

	// 以下为读取响应时由服务层富化的非持久化字段
	PeerUserID  string // 对方用户 ID（direct 会话中除自己外的成员）
	PeerName    string // 对方展示名（昵称优先，回退用户名）
	LastMessage string // 会话最后一条消息摘要（无消息时为空）
	LastAt      *time.Time
	UnreadCount int64 // 当前用户在该会话的未读数
}

// ConversationMember 会话成员领域模型。
// 记录用户与会话的归属关系及各自的已读水位（LastReadAt），用于计算未读数。
type ConversationMember struct {
	ID             string     // 成员记录唯一标识（ULID）
	ConversationID string     // 所属会话 ID
	TenantID       string     // 所属租户 ID（冗余，便于隔离查询）
	UserID         string     // 成员用户 ID
	LastReadAt     *time.Time // 该成员最后已读时间，用于计算未读；为空表示从未读
	JoinedAt       time.Time  // 加入时间
	UserName       string     // 展示名（富化字段，不持久化）
}

// ChatMessage 聊天消息领域模型。
type ChatMessage struct {
	ID             string    // 消息唯一标识（ULID）
	ConversationID string    // 所属会话 ID
	TenantID       string    // 所属租户 ID（多租户隔离）
	SenderID       string    // 发送人用户 ID
	Content        string    // 消息正文
	Type           string    // 消息类型：text
	CreatedAt      time.Time // 发送时间

	SenderName string // 发送人展示名（富化字段，不持久化）
}

// DirectKey 依据两个参与者 ID 生成 1:1 私聊会话的去重键。
// 对参与者做确定性排序，保证 (a,b) 与 (b,a) 生成同一 key，从而复用会话。
func DirectKey(userA, userB string) string {
	if userA > userB {
		userA, userB = userB, userA
	}
	return "direct:" + userA + ":" + userB
}
