// Package model 定义协作即时消息（Chat）模块的领域模型。
// 当前支持同租户内的 1:1 私聊（direct 会话），消息按会话线性存储，
// 成员各自维护已读水位以计算未读数；发送人可在时限内软撤回自己的消息。
// Conversation.Type 与 Message.Type 均为扩展位，后续群聊/富媒体消息只需新增取值。
package model

import "time"

// 会话类型常量
const (
	// ConversationTypeDirect 1:1 私聊会话，会话由参与者唯一确定
	ConversationTypeDirect = "direct"
	// ConversationTypeGroup 群聊会话：多人参与、可增减成员，名称由成员维护
	ConversationTypeGroup = "group"
)

// 会话成员角色常量（仅对 group 会话有义，direct 成员均记为 member）
const (
	// MemberRoleOwner 创建人：可改名、移除成员
	MemberRoleOwner = "owner"
	// MemberRoleMember 普通成员：可发消息、邀请成员、退群
	MemberRoleMember = "member"
)

// 群规模限制
const (
	// GroupMinMembers 初始成员下限（不含创建人）
	GroupMinMembers = 1
	// GroupMaxMembers 群成员上限（含创建人）
	GroupMaxMembers = 50
)

// 消息类型常量
const (
	// MessageTypeText 纯文本消息
	MessageTypeText = "text"
	// MessageTypeImage 图片消息：Content 存媒体元信息 JSON，访问 URL 读取时临时签发
	MessageTypeImage = "image"
	// MessageTypeFile 文件消息：同图片，正文为 {file,name,size} JSON
	MessageTypeFile = "file"
)

// Conversation 会话领域模型。
// direct：同一租户内两个用户由 DirectKey 唯一确定一个会话，重复发起时按 key 复用既有会话；
// group：去重键取新生成的 ULID（天然唯一），名称存于 Name，创建人在成员表中标记为 owner。
type Conversation struct {
	ID        string    // 会话唯一标识（ULID）
	TenantID  string    // 所属租户 ID（多租户隔离，成员必须同租户）
	Type      string    // 会话类型：direct / group
	Key       string    // 会话去重键（direct 由参与者 ID 规范化生成；group 为生成的 ULID）
	Name      string    // 会话名称（仅 group 使用，direct 为空）
	CreatedAt time.Time // 创建时间
	UpdatedAt time.Time // 更新时间

	// 以下为读取响应时由服务层富化的非持久化字段
	PeerUserID      string // 对方用户 ID（仅 direct 会话填充；group 为空）
	PeerName        string // 展示名：direct 为对方昵称，group 为群名称
	LastMessage     string // 会话最后一条消息摘要（无消息时为空；媒体消息为元信息 JSON，前端按类型渲染）
	LastMessageType string // 最后一条消息类型（text/image/file），供前端格式化摘要
	LastAt          *time.Time
	UnreadCount     int64  // 当前用户在该会话的未读数
	MemberCount     int64  // 成员数（仅 group 富化）
	MyRole          string // 当前用户在该会话中的角色：owner / member（列表富化）
}

// ConversationMember 会话成员领域模型。
// 记录用户与会话的归属关系、角色及各自的已读水位（LastReadAt），用于计算未读数与权限判定。
type ConversationMember struct {
	ID             string     // 成员记录唯一标识（ULID）
	ConversationID string     // 所属会话 ID
	TenantID       string     // 所属租户 ID（冗余，便于隔离查询）
	UserID         string     // 成员用户 ID
	Role           string     // 成员角色：owner / member
	LastReadAt     *time.Time // 该成员最后已读时间，用于计算未读；为空表示从未读
	JoinedAt       time.Time  // 加入时间
	UserName       string     // 展示名（富化字段，不持久化）
}

// ChatMessage 聊天消息领域模型。
type ChatMessage struct {
	ID             string     // 消息唯一标识（ULID）
	ConversationID string     // 所属会话 ID
	TenantID       string     // 所属租户 ID（多租户隔离）
	SenderID       string     // 发送人用户 ID
	Content        string     // 消息正文（撤回后置空）
	Type           string     // 消息类型：text
	MentionIDs     string     // @提及的用户 ID 列表（逗号分隔，仅保留会话内有效成员）
	Deleted        bool       // 是否已撤回（软撤回标记，保留行以维持审计可追溯）
	DeletedAt      *time.Time // 撤回时间
	CreatedAt      time.Time  // 发送时间

	SenderName string // 发送人展示名（富化字段，不持久化）
	FileURL    string // 媒体消息的短时效访问 URL（富化字段，不持久化；未启用签名时为空）
}

// DirectKey 依据两个参与者 ID 生成 1:1 私聊会话的去重键。
// 对参与者做确定性排序，保证 (a,b) 与 (b,a) 生成同一 key，从而复用会话。
func DirectKey(userA, userB string) string {
	if userA > userB {
		userA, userB = userB, userA
	}
	return "direct:" + userA + ":" + userB
}
