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
