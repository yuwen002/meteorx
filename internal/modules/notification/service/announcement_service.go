// Package service 提供通知公告业务逻辑
package service

import (
	"context"
	"meteorx/internal/modules/notification/dto"
	"meteorx/internal/modules/notification/model"
	"meteorx/internal/modules/notification/repository"
	"meteorx/pkg/idgen"
	"time"
)

// AnnouncementService 公告业务服务
type AnnouncementService struct {
	repo repository.AnnouncementRepository
}

// NewAnnouncementService 创建公告服务
func NewAnnouncementService(repo repository.AnnouncementRepository) *AnnouncementService {
	return &AnnouncementService{repo: repo}
}

// Create 创建公告
func (s *AnnouncementService) Create(ctx context.Context, publisherID string, req dto.CreateAnnouncementReq) (*dto.AnnouncementResp, error) {
	a := &model.Announcement{
		ID:             idgen.New(),
		Title:          req.Title,
		Content:        req.Content,
		Scope:          req.Scope,
		TargetTenantID: req.TargetTenantID,
		Status:         req.Status,
		PublisherID:    publisherID,
		PublishAt:      dto.ParseTime(req.PublishAt),
		ExpireAt:       dto.ParseTime(req.ExpireAt),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	// 新建即发布时记录发布时间
	if a.Status == model.AnnouncementStatusPublished && a.PublishAt == nil {
		now := time.Now()
		a.PublishAt = &now
	}

	if err := s.repo.Create(ctx, a); err != nil {
		return nil, err
	}
	return dto.ToAnnouncementResp(a), nil
}

// Update 更新公告
func (s *AnnouncementService) Update(ctx context.Context, id string, req dto.UpdateAnnouncementReq) (*dto.AnnouncementResp, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	a.Title = req.Title
	a.Content = req.Content
	a.Scope = req.Scope
	a.TargetTenantID = req.TargetTenantID
	a.Status = req.Status
	a.PublishAt = dto.ParseTime(req.PublishAt)
	a.ExpireAt = dto.ParseTime(req.ExpireAt)
	a.UpdatedAt = time.Now()
	// 更新为已发布时补记发布时间
	if a.Status == model.AnnouncementStatusPublished && a.PublishAt == nil {
		now := time.Now()
		a.PublishAt = &now
	}

	if err := s.repo.Update(ctx, a); err != nil {
		return nil, err
	}
	return dto.ToAnnouncementResp(a), nil
}

// UpdateStatus 更新公告状态（发布/下架）
func (s *AnnouncementService) UpdateStatus(ctx context.Context, id string, status int) (*dto.AnnouncementResp, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	a.Status = status
	a.UpdatedAt = time.Now()
	if status == model.AnnouncementStatusPublished && a.PublishAt == nil {
		now := time.Now()
		a.PublishAt = &now
	}

	if err := s.repo.Update(ctx, a); err != nil {
		return nil, err
	}
	return dto.ToAnnouncementResp(a), nil
}

// GetByID 获取公告详情
func (s *AnnouncementService) GetByID(ctx context.Context, id string) (*dto.AnnouncementResp, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.ToAnnouncementResp(a), nil
}

// Delete 删除公告
func (s *AnnouncementService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// List 分页查询公告
func (s *AnnouncementService) List(ctx context.Context, query *dto.ListAnnouncementsQuery) (*dto.AnnouncementListResp, error) {
	repoQuery := &repository.AnnouncementQuery{
		Page:     query.Page,
		PageSize: query.PageSize,
		Keyword:  query.Keyword,
		Status:   query.Status,
		Scope:    query.Scope,
	}

	items, total, err := s.repo.List(ctx, repoQuery)
	if err != nil {
		return nil, err
	}

	respItems := make([]*dto.AnnouncementResp, len(items))
	for i, a := range items {
		respItems[i] = dto.ToAnnouncementResp(a)
	}
	return &dto.AnnouncementListResp{Items: respItems, Total: total}, nil
}

// ListForTenant 查询租户可见的公告列表
func (s *AnnouncementService) ListForTenant(ctx context.Context, tenantID string, page, pageSize int) (*dto.AnnouncementListResp, error) {
	items, total, err := s.repo.ListForTenant(ctx, tenantID, page, pageSize)
	if err != nil {
		return nil, err
	}

	respItems := make([]*dto.AnnouncementResp, len(items))
	for i, a := range items {
		respItems[i] = dto.ToAnnouncementResp(a)
	}
	return &dto.AnnouncementListResp{Items: respItems, Total: total}, nil
}