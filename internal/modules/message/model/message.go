// Package model 定义站内消息（收件箱）领域模型。
// 站内消息是「持久化的站内信」：实时推送（WebSocket）只解决在线可达，
// 落库后才支持离线补推（重新登录拉未读）与已读状态。
package model

import "time"

// 消息分类常量。
// 分类决定前端图标与筛选页签，取值与业务来源一一对应。
const (
	// CategorySystem 系统消息（注销审批结果等）
	CategorySystem = "system"
	// CategoryAnnouncement 公告
	CategoryAnnouncement = "announcement"
	// CategoryTask 任务（指派 / 到期提醒）
	CategoryTask = "task"
	// CategoryPlan 套餐与订阅（到期提醒）
	CategoryPlan = "plan"
	// CategoryAudit 安全告警
	CategoryAudit = "audit"
)

// IsValidCategory 判断给定字符串是否为合法的消息分类。
func IsValidCategory(c string) bool {
	switch c {
	case CategorySystem, CategoryAnnouncement, CategoryTask, CategoryPlan, CategoryAudit:
		return true
	}
	return false
}

// AllCategories 返回全部分类，顺序即前端偏好设置的展示顺序。
// 新增分类时必须同时登记到这里与 IsValidCategory：偏好接口按本函数逐项返回，
// 漏登记会让用户在界面上看不到该分类的开关，等于永久默认接收。
func AllCategories() []string {
	return []string{CategorySystem, CategoryAnnouncement, CategoryTask, CategoryPlan, CategoryAudit}
}

// IsMandatoryCategory 判断分类是否强制接收（不允许关闭）。
// 安全告警是审计视野的唯一留痕：允许关闭等于允许管理员关掉自己的告警，
// 事后无从追溯，因此该分类的偏好只能「接收」。
func IsMandatoryCategory(c string) bool {
	return c == CategoryAudit
}

// CategoryPreference 单个分类的接收偏好。
// 没有记录即默认接收：新分类上线时无需给存量用户补行。
type CategoryPreference struct {
	Category string
	Muted    bool
}

// Message 单条站内消息。
// 消息归属于「收件人」而非租户：同一事件会给多个成员各写一行，
// 因此已读状态天然按人隔离，无需额外的已读关联表。
type Message struct {
	ID       string // 消息唯一标识（ULID）
	TenantID string // 所属租户 ID（多租户隔离）
	UserID   string // 收件人用户 ID
	Category string // 分类：system / announcement / task / plan / audit
	Title    string // 标题
	Content  string // 正文（纯文本，前端按纯文本渲染避免 XSS）
	// Link 前端跳转路由（如 /task），为空表示不可跳转
	Link     string
	Priority int // 优先级：0 low / 1 normal / 2 high / 3 urgent
	// SourceType/SourceID 记录消息来源，便于按业务追溯与后续去重
	SourceType string
	SourceID   string
	// DedupeKey 去重键：同一收件人 + 同一键只落一条（重试 / 多副本 / 重启补偿）。
	// 为空表示不参与去重，每次投递都写新行。
	DedupeKey string
	ReadAt    *time.Time // 已读时间，为空表示未读
	CreatedAt time.Time  // 创建时间
}

// IsUnread 判断消息是否未读。
func (m *Message) IsUnread() bool { return m.ReadAt == nil }
