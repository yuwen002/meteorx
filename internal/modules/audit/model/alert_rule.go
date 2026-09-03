package model

import "time"

// AlertRule 告警规则模型
type AlertRule struct {
	ID              string    // 规则ID
	Name            string    // 规则名称
	Description     string    // 规则描述
	Enabled         bool      // 是否启用
	TriggerType     string    // 触发类型（risk_level/action/user）
	TriggerValue    string    // 触发值（如 high/critical/delete）
	NotifyChannels  string    // 通知渠道（JSON数组：["email","dingtalk","wechat"]）
	NotifyTargets   string    // 通知目标（JSON数组：邮箱/手机号/webhook）
	NotifyTemplate  string    // 通知模板
	CooldownMinutes int       // 冷却时间（分钟），避免重复告警
	CreatedAt       time.Time // 创建时间
	UpdatedAt       time.Time // 更新时间
}

// AuditAlert 审计告警记录
type AuditAlert struct {
	ID          string    // 告警ID
	RuleID      string    // 触发规则ID
	RuleName    string    // 规则名称
	AuditLogID  string    // 关联审计日志ID
	UserID      string    // 触发用户ID
	Username    string    // 触发用户名
	RiskLevel   string    // 风险等级
	Action      string    // 操作类型
	Message     string    // 告警消息
	Notified    bool      // 是否已通知
	NotifyTime  time.Time // 通知时间
	CreatedAt   time.Time // 创建时间
}

// TriggerType 触发类型
const (
	TriggerTypeRiskLevel = "risk_level" // 按风险等级触发
	TriggerTypeAction    = "action"     // 按操作类型触发
	TriggerTypeUser      = "user"       // 按特定用户触发
)

// NotifyChannel 通知渠道
const (
	NotifyChannelEmail    = "email"    // 邮件
	NotifyChannelDingTalk = "dingtalk" // 钉钉
	NotifyChannelWeChat   = "wechat"   // 企业微信
	NotifyChannelWebhook  = "webhook"  // 自定义Webhook
)

// AlertStats 告警统计
type AlertStats struct {
	TotalAlerts    int64            `json:"total_alerts"`
	TodayAlerts    int64            `json:"today_alerts"`
	NotifiedCount  int64            `json:"notified_count"`
	PendingCount   int64            `json:"pending_count"`
	RiskLevelStats map[string]int64 `json:"risk_level_stats"`
	RuleStats      map[string]int64 `json:"rule_stats"`
	Trend          []TrendPoint     `json:"trend"`
	TopRules       []RuleCount      `json:"top_rules"`
}

// TrendPoint 趋势点
type TrendPoint struct {
	Date    string `json:"date"`
	Count   int64  `json:"count"`
	Success int64  `json:"success"`
	Failure int64  `json:"failure"`
}

// RuleCount 规则计数
type RuleCount struct {
	RuleID   string `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Count    int64  `json:"count"`
}