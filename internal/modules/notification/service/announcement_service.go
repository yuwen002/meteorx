// Package service 提供通知公告业务逻辑
package service

import (
	"context"
	"errors"
	"meteorx/internal/modules/notification/dto"
	"meteorx/internal/modules/notification/model"
	"meteorx/internal/modules/notification/repository"
	"meteorx/internal/notify"
	"meteorx/pkg/idgen"
	"time"
)

// ErrTenantTargetRequired 公告范围指定租户但未给出目标租户，属客户端参数错误
var ErrTenantTargetRequired = errors.New("公告范围为目标租户时必须指定 target_tenant_id")

// AnnouncementPublishCallback 公告发布回调（多渠道通知）
type AnnouncementPublishCallback func(ctx context.Context, title, content string, scope string, targetTenantID string)

// AnnouncementService 公告业务服务
type AnnouncementService struct {
	repo       repository.AnnouncementRepository
	onPublish  AnnouncementPublishCallback
}

// NewAnnouncementService 创建公告服务
func NewAnnouncementService(repo repository.AnnouncementRepository) *AnnouncementService {
	return &AnnouncementService{
		repo: repo,
		onPublish: defaultPublishCallback,
	}
}

// SetPublishCallback 设置公告发布回调
func (s *AnnouncementService) SetPublishCallback(cb AnnouncementPublishCallback) {
	if cb != nil {
		s.onPublish = cb
	}
}

// defaultPublishCallback 默认发布回调：通过全局通知管理器发送通知
func defaultPublishCallback(ctx context.Context, title, content string, scope string, targetTenantID string) {
	mgr := notify.GetGlobalManager()
	if mgr == nil {
		return
	}

	// 仅租户定向公告带目标租户；全平台公告一律清空，避免携带残留目标造成误定向
	if scope != model.AnnouncementScopeTenant {
		targetTenantID = ""
	}
	_ = mgr.NotifyAnnouncement(ctx, title, content, targetTenantID)

	// 如果 Webhook 配置了 announcement 事件，也会自动推送
}

// Create 创建公告
func (s *AnnouncementService) Create(ctx context.Context, publisherID string, req dto.CreateAnnouncementReq) (*dto.AnnouncementResp, error) {
	if req.Scope == model.AnnouncementScopeTenant && req.TargetTenantID == "" {
		return nil, ErrTenantTargetRequired
	}

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

	// 发布时触发多渠道通知
	if a.Status == model.AnnouncementStatusPublished && s.onPublish != nil {
		s.onPublish(ctx, a.Title, a.Content, a.Scope, a.TargetTenantID)
	}

	return dto.ToAnnouncementResp(a), nil
}

// Update 更新公告
func (s *AnnouncementService) Update(ctx context.Context, id string, req dto.UpdateAnnouncementReq) (*dto.AnnouncementResp, error) {
	if req.Scope == model.AnnouncementScopeTenant && req.TargetTenantID == "" {
		return nil, ErrTenantTargetRequired
	}

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

	// 发布公告时触发多渠道通知
	if a.Status == model.AnnouncementStatusPublished && s.onPublish != nil {
		s.onPublish(ctx, a.Title, a.Content, a.Scope, a.TargetTenantID)
	}

	return dto.ToAnnouncementResp(a), nil
}

// UpdateStatus 更新公告状态（发布/下架）
func (s *AnnouncementService) UpdateStatus(ctx context.Context, id string, status int) (*dto.AnnouncementResp, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	prevStatus := a.Status
	a.Status = status
	a.UpdatedAt = time.Now()
	if status == model.AnnouncementStatusPublished && a.PublishAt == nil {
		now := time.Now()
		a.PublishAt = &now
	}

	if err := s.repo.Update(ctx, a); err != nil {
		return nil, err
	}

	// 后台行内「发布」按钮走本方法：只有 非已发布→已发布 的翻转才发通知，
	// 下架与重复发布都不得重复推送
	if status == model.AnnouncementStatusPublished && prevStatus != model.AnnouncementStatusPublished && s.onPublish != nil {
		s.onPublish(ctx, a.Title, a.Content, a.Scope, a.TargetTenantID)
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