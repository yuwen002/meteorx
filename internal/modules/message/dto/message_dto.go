// Package dto 定义站内消息接口的请求/响应结构。
package dto

import (
	"time"

	"meteorx/internal/modules/message/model"
)

// MessageResp 单条消息响应。
type MessageResp struct {
	ID         string     `json:"id"`
	Category   string     `json:"category"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Link       string     `json:"link"`
	Priority   int        `json:"priority"`
	SourceType string     `json:"source_type"`
	SourceID   string     `json:"source_id"`
	ReadAt     *time.Time `json:"read_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ToResp 将领域模型转换为响应结构。
func ToResp(m *model.Message) *MessageResp {
	return &MessageResp{
		ID:         m.ID,
		Category:   m.Category,
		Title:      m.Title,
		Content:    m.Content,
		Link:       m.Link,
		Priority:   m.Priority,
		SourceType: m.SourceType,
		SourceID:   m.SourceID,
		ReadAt:     m.ReadAt,
		CreatedAt:  m.CreatedAt,
	}
}

// UnreadCountResp 未读数响应。
// total 为全局未读数（铃铛角标），by_category 供各页签角标使用。
type UnreadCountResp struct {
	Total      int64            `json:"total"`
	ByCategory map[string]int64 `json:"by_category"`
}

// MarkAllReadReq 全部已读请求体。
type MarkAllReadReq struct {
	// Category 为空表示全部标记已读，否则只处理该分类
	Category string `json:"category"`
}

// PreferenceItem 单个分类的消息偏好。
type PreferenceItem struct {
	Category string `json:"category"`
	// Muted 为 true 表示关闭接收（该类消息不再落库）
	Muted bool `json:"muted"`
	// Mandatory 为 true 表示强制接收，前端应禁用开关
	Mandatory bool `json:"mandatory"`
}

// PreferencesResp 消息偏好响应。
// 按固定顺序返回全部分类：未设置过的分类也在列表里，前端无需自己补默认值。
type PreferencesResp struct {
	Items []PreferenceItem `json:"items"`
}

// UpdatePreferenceReq 单条偏好变更请求体。
// 一次只改一个分类：开关是即时生效的，不存在「半提交」问题，
// 也就不需要列表 + 事务去保证一批开关的原子性。
type UpdatePreferenceReq struct {
	Category string `json:"category" validate:"required" label:"分类"`
	Muted    bool   `json:"muted"`
}
