// Package dto 提供通知公告模块的数据传输对象
package dto

import (
	"meteorx/internal/modules/notification/model"
	"time"
)

// CreateAnnouncementReq 创建公告请求
type CreateAnnouncementReq struct {
	Title          string `json:"title" label:"公告标题" validate:"required,max=200"`
	Content        string `json:"content" label:"公告内容" validate:"required"`
	Scope          string `json:"scope" label:"公告范围" validate:"required,oneof=all tenant"`
	TargetTenantID string `json:"target_tenant_id" label:"目标租户ID"`
	Status         int    `json:"status" label:"状态" validate:"oneof=0 1 2"`
	PublishAt      string `json:"publish_at" label:"发布时间"`
	ExpireAt       string `json:"expire_at" label:"过期时间"`
}

// UpdateAnnouncementReq 更新公告请求
type UpdateAnnouncementReq struct {
	Title          string `json:"title" label:"公告标题" validate:"required,max=200"`
	Content        string `json:"content" label:"公告内容" validate:"required"`
	Scope          string `json:"scope" label:"公告范围" validate:"required,oneof=all tenant"`
	TargetTenantID string `json:"target_tenant_id" label:"目标租户ID"`
	Status         int    `json:"status" label:"状态" validate:"oneof=0 1 2"`
	PublishAt      string `json:"publish_at" label:"发布时间"`
	ExpireAt       string `json:"expire_at" label:"过期时间"`
}

// UpdateAnnouncementStatusReq 更新公告状态请求
type UpdateAnnouncementStatusReq struct {
	Status int `json:"status" label:"状态" validate:"required,oneof=1 2"`
}

// AnnouncementResp 公告响应
type AnnouncementResp struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	Scope          string `json:"scope"`
	TargetTenantID string `json:"target_tenant_id"`
	Status         int    `json:"status"`
	StatusText     string `json:"status_text"`
	PublisherID    string `json:"publisher_id"`
	PublishAt      string `json:"publish_at"`
	ExpireAt       string `json:"expire_at"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// ListAnnouncementsQuery 公告列表查询参数
type ListAnnouncementsQuery struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Keyword  string `json:"keyword" form:"keyword"`
	Status   int    `json:"status" form:"status"`
	Scope    string `json:"scope" form:"scope"`
}

// AnnouncementListResp 公告列表响应
type AnnouncementListResp struct {
	Items []*AnnouncementResp `json:"items"`
	Total int64               `json:"total"`
}

// ToAnnouncementResp 将领域模型转为响应 DTO
func ToAnnouncementResp(a *model.Announcement) *AnnouncementResp {
	resp := &AnnouncementResp{
		ID:             a.ID,
		Title:          a.Title,
		Content:        a.Content,
		Scope:          a.Scope,
		TargetTenantID: a.TargetTenantID,
		Status:         a.Status,
		StatusText:     statusText(a.Status),
		PublisherID:    a.PublisherID,
		CreatedAt:      a.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:      a.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if a.PublishAt != nil {
		resp.PublishAt = a.PublishAt.Format("2006-01-02 15:04:05")
	}
	if a.ExpireAt != nil {
		resp.ExpireAt = a.ExpireAt.Format("2006-01-02 15:04:05")
	}
	return resp
}

func statusText(status int) string {
	switch status {
	case model.AnnouncementStatusDraft:
		return "草稿"
	case model.AnnouncementStatusPublished:
		return "已发布"
	case model.AnnouncementStatusOffline:
		return "已下架"
	default:
		return "未知"
	}
}

// ParseTime 解析时间字符串（空返回 nil）
func ParseTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
	if err != nil {
		return nil
	}
	return &t
}
