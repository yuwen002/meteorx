package notify

import "context"

// InboxMessage 站内消息落库请求。
// 与 Message 的区别：Message 面向「渠道投递」（一次一个渠道、收件人是邮箱或用户 ID），
// InboxMessage 面向「收件箱持久化」——它只认收件人用户 ID，
// 由消息模块负责加分类、写库并推送未读角标。
type InboxMessage struct {
	TenantID string   // 目标租户：UserIDs 为空时据此展开租户成员
	UserIDs  []string // 收件人用户 ID（不传邮箱）
	// Type 业务类型（如 task_due / announcement），落库实现据此映射分类
	Type     string
	Category string // 可选：调用方直接指定分类时优先使用
	Title    string
	Content  string
	Link     string // 前端跳转路由，可为空
	Priority Priority
	// SourceType/SourceID 来源溯源，便于按业务排查
	SourceType string
	SourceID   string
	// DedupeKey 去重键：同一收件人 + 同一键只落一条。
	// 为空时落库实现可回退为「SourceType:SourceID」；两者都定位不到事件则不去重。
	// 调用方应按「业务事件 + 时间粒度」构造，避免把跨天的重复提醒误合并。
	DedupeKey string
}

// InboxWriter 站内消息落库能力，由消息模块实现。
// 声明为接口而非直接依赖，避免 notify ⇄ message 循环依赖，
// 未装配时（如集成测试只注册桩渠道）站内信落库自动跳过。
type InboxWriter interface {
	Deliver(ctx context.Context, msg *InboxMessage) error
}
