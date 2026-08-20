// Package model 提供通知公告相关的领域模型
package model

import "time"

// 公告状态常量
const (
	AnnouncementStatusDraft     = 0 // 草稿
	AnnouncementStatusPublished = 1 // 已发布
	AnnouncementStatusOffline   = 2 // 已下架
)

// 公告范围常量
const (
	AnnouncementScopeAll    = "all"    // 全平台
	AnnouncementScopeTenant = "tenant" // 指定租户
)

// Announcement 平台通知公告
type Announcement struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Content        string     `json:"content"`
	Scope          string     `json:"scope"`            // all / tenant
	TargetTenantID string     `json:"target_tenant_id"` // scope=tenant 时指定租户
	Status         int        `json:"status"`           // 0 草稿 / 1 已发布 / 2 已下架
	PublisherID    string     `json:"publisher_id"`     // 发布人（平台管理员）
	PublishAt      *time.Time `json:"publish_at"`
	ExpireAt       *time.Time `json:"expire_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
