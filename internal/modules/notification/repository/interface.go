// Package repository 提供通知公告数据访问层接口
package repository

import (
	"context"
	"meteorx/internal/modules/notification/model"
)

// AnnouncementRepository 公告数据访问接口
type AnnouncementRepository interface {
	// Create 创建公告
	Create(ctx context.Context, announcement *model.Announcement) error
	// Update 更新公告
	Update(ctx context.Context, announcement *model.Announcement) error
	// GetByID 根据ID获取公告
	GetByID(ctx context.Context, id string) (*model.Announcement, error)
	// Delete 删除公告（软删除）
	Delete(ctx context.Context, id string) error
	// List 分页查询公告（管理端）
	List(ctx context.Context, query *AnnouncementQuery) ([]*model.Announcement, int64, error)
	// ListForTenant 查询租户可见的已发布公告
	ListForTenant(ctx context.Context, tenantID string, page, pageSize int) ([]*model.Announcement, int64, error)
}

// AnnouncementQuery 公告查询条件
type AnnouncementQuery struct {
	Page     int
	PageSize int
	Keyword  string // 标题关键字
	Status   int    // 状态（0 全部 / 1 已发布 / 2 草稿 / 3 已下架）
	Scope    string // 范围（all / tenant）
}