package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"
	db "meteorx/internal/pkg/db"
	"meteorx/pkg/idgen"

	"gorm.io/gorm"
)

type WikiServiceExtended interface {
	WikiService

	CreateTag(ctx context.Context, req *dto.CreateTagReq) (*dto.TagResp, error)
	ListTags(ctx context.Context) ([]*dto.TagResp, error)
	DeleteTag(ctx context.Context, id string) error

	AddDocumentTag(ctx context.Context, documentID, tagID string) error
	RemoveDocumentTag(ctx context.Context, documentID, tagID string) error
	ListDocumentTags(ctx context.Context, documentID string) ([]*dto.DocumentTagResp, error)

	CreateComment(ctx context.Context, req *dto.CreateCommentReq) (*dto.CommentResp, error)
	ListComments(ctx context.Context, documentID string) ([]*dto.CommentResp, error)
	UpdateComment(ctx context.Context, id string, req *dto.UpdateCommentReq) error
	DeleteComment(ctx context.Context, id string) error

	CreateShareLink(ctx context.Context, req *dto.ShareLinkReq) (*dto.ShareLinkResp, error)
	GetShareLink(ctx context.Context, token, password string) (*dto.ShareLinkResp, error)
	ListShareLinks(ctx context.Context, documentID string) ([]*dto.ShareLinkResp, error)
	DeleteShareLink(ctx context.Context, id string) error

	CreateTemplate(ctx context.Context, req *dto.CreateTemplateReq) (*dto.DocumentTemplateResp, error)
	ListTemplates(ctx context.Context, category string) ([]*dto.DocumentTemplateResp, error)
	GetTemplate(ctx context.Context, id string) (*dto.DocumentTemplateResp, error)
	UpdateTemplate(ctx context.Context, id string, req *dto.UpdateTemplateReq) (*dto.DocumentTemplateResp, error)
	DeleteTemplate(ctx context.Context, id string) error

	GetDocumentStats(ctx context.Context, documentID string) (*dto.DocumentStatsResp, error)
	ListAccessLogs(ctx context.Context, documentID string, page, pageSize int) ([]*dto.DocumentAccessLogResp, int64, error)

	SubscribeDocument(ctx context.Context, documentID, notifyType string) (*dto.SubscriptionResp, error)
	UnsubscribeDocument(ctx context.Context, documentID string) error
	ListUserSubscriptions(ctx context.Context) ([]*dto.SubscriptionResp, error)

	ListNotifications(ctx context.Context, page, pageSize int) ([]*dto.NotificationResp, int64, error)
	MarkNotificationAsRead(ctx context.Context, id string) error
	MarkAllNotificationsAsRead(ctx context.Context) error
	GetUnreadNotificationCount(ctx context.Context) (int64, error)

	AcquireEditLock(ctx context.Context, documentID string) (*dto.EditLockResp, error)
	ReleaseEditLock(ctx context.Context, documentID string) error
	RefreshEditLock(ctx context.Context, documentID string) error
	GetEditLock(ctx context.Context, documentID string) (*dto.EditLockResp, error)

	BatchDeleteNodes(ctx context.Context, nodeIDs []string) error
	BatchMoveNodes(ctx context.Context, nodeIDs []string, newParentID string) error

	CompareRevisions(ctx context.Context, documentID string, version1, version2 int) (*dto.DiffResult, error)

	ExportDocument(ctx context.Context, documentID, format string) ([]byte, string, error)
}

type wikiServiceExtended struct {
	wikiService
}

func NewWikiServiceExtended(repo repository.WikiRepository, tx *db.TxManager) WikiServiceExtended {
	extRepo, ok := repo.(repository.WikiRepositoryExtended)
	if !ok {
		extRepo = repository.NewWikiRepositoryExtended(repo.(*repository.wikiRepository).db)
	}
	
	svc := &wikiService{
		repo:        extRepo,
		tx:          tx,
		markdownSvc: NewMarkdownService(),
	}
	
	return &wikiServiceExtended{
		wikiService: *svc,
	}
}

func (s *wikiServiceExtended) getExtendedRepo() repository.WikiRepositoryExtended {
	return s.wikiService.repo.(repository.WikiRepositoryExtended)
}

func (s *wikiServiceExtended) CreateTag(ctx context.Context, req *dto.CreateTagReq) (*dto.TagResp, error) {
	userID := contextx.GetUserID(ctx)
	tag := &model.Tag{
		Name:      req.Name,
		Color:     req.Color,
		CreatedBy: userID,
	}

	if err := s.getExtendedRepo().CreateTag(ctx, tag); err != nil {
		return nil, err
	}

	return &dto.TagResp{
		ID:        tag.ID,
		Name:      tag.Name,
		Color:     tag.Color,
		CreatedBy: tag.CreatedBy,
		CreatedAt: tag.CreatedAt,
	}, nil
}

func (s *wikiServiceExtended) ListTags(ctx context.Context) ([]*dto.TagResp, error) {
	tenantID := contextx.GetTenantID(ctx)
	tags, err := s.getExtendedRepo().ListTags(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var resps []*dto.TagResp
	for _, tag := range tags {
		resps = append(resps, &dto.TagResp{
			ID:        tag.ID,
			Name:      tag.Name,
			Color:     tag.Color,
			CreatedBy: tag.CreatedBy,
			CreatedAt: tag.CreatedAt,
		})
	}
	return resps, nil
}

func (s *wikiServiceExtended) DeleteTag(ctx context.Context, id string) error {
	return s.getExtendedRepo().DeleteTag(ctx, id)
}

func (s *wikiServiceExtended) AddDocumentTag(ctx context.Context, documentID, tagID string) error {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "update"); err != nil {
		return err
	}
	return s.getExtendedRepo().AddDocumentTag(ctx, documentID, tagID)
}

func (s *wikiServiceExtended) RemoveDocumentTag(ctx context.Context, documentID, tagID string) error {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "update"); err != nil {
		return err
	}
	return s.getExtendedRepo().RemoveDocumentTag(ctx, documentID, tagID)
}

func (s *wikiServiceExtended) ListDocumentTags(ctx context.Context, documentID string) ([]*dto.DocumentTagResp, error) {
	tags, err := s.getExtendedRepo().ListDocumentTagsWithDetails(ctx, documentID)
	if err != nil {
		return nil, err
	}

	var resps []*dto.DocumentTagResp
	for _, dt := range tags {
		resps = append(resps, &dto.DocumentTagResp{
			ID:         dt.ID,
			DocumentID: dt.DocumentID,
			Tag: dto.TagResp{
				ID:        dt.Tag.ID,
				Name:      dt.Tag.Name,
				Color:     dt.Tag.Color,
				CreatedBy: dt.Tag.CreatedBy,
				CreatedAt: dt.Tag.CreatedAt,
			},
			CreatedAt: dt.CreatedAt,
		})
	}
	return resps, nil
}

func (s *wikiServiceExtended) CreateComment(ctx context.Context, req *dto.CreateCommentReq) (*dto.CommentResp, error) {
	userID := contextx.GetUserID(ctx)
	
	if err := s.CheckNodePermission(ctx, req.NodeID, userID, "read"); err != nil {
		return nil, err
	}

	comment := &model.Comment{
		DocumentID: req.DocumentID,
		NodeID:     req.NodeID,
		ParentID:   req.ParentID,
		Content:    req.Content,
		CreatedBy:  userID,
		MentionIDs: req.MentionIDs,
		Status:     model.CommentStatusActive,
	}

	if err := s.getExtendedRepo().CreateComment(ctx, comment); err != nil {
		return nil, err
	}

	if len(req.MentionIDs) > 0 {
		mentionedIDs := strings.Split(req.MentionIDs, ",")
		for _, mentionedID := range mentionedIDs {
			mentionedID = strings.TrimSpace(mentionedID)
			if mentionedID == "" {
				continue
			}
			_ = s.getExtendedRepo().CreateNotification(ctx, &model.Notification{
				UserID:      mentionedID,
				Type:        model.NotificationTypeMention,
				Title:       "有人在评论中提到了你",
				Content:     req.Content,
				RelatedID:   comment.ID,
				RelatedType: "comment",
			})
		}
	}

	return s.buildCommentResp(ctx, comment)
}

func (s *wikiServiceExtended) ListComments(ctx context.Context, documentID string) ([]*dto.CommentResp, error) {
	userID := contextx.GetUserID(ctx)
	
	doc, node, err := s.getExtendedRepo().GetDocumentByIDWithNode(ctx, documentID)
	if err != nil {
		return nil, err
	}
	
	if err := s.CheckNodePermission(ctx, node.ID, userID, "read"); err != nil {
		return nil, err
	}

	comments, err := s.getExtendedRepo().ListCommentsByDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}

	var resps []*dto.CommentResp
	for _, c := range comments {
		if c.ParentID == "" {
			resp, _ := s.buildCommentResp(ctx, c)
			replies, _ := s.getExtendedRepo().ListRepliesByParentID(ctx, c.ID)
			for _, r := range replies {
				replyResp, _ := s.buildCommentResp(ctx, r)
				resp.Replies = append(resp.Replies, *replyResp)
			}
			resps = append(resps, resp)
		}
	}

	_ = doc
	return resps, nil
}

func (s *wikiServiceExtended) UpdateComment(ctx context.Context, id string, req *dto.UpdateCommentReq) error {
	userID := contextx.GetUserID(ctx)
	
	comment, err := s.getExtendedRepo().GetComment(ctx, id)
	if err != nil {
		return err
	}

	if comment.CreatedBy != userID {
		return fmt.Errorf("permission denied")
	}

	comment.Content = req.Content
	return s.getExtendedRepo().UpdateComment(ctx, comment)
}

func (s *wikiServiceExtended) DeleteComment(ctx context.Context, id string) error {
	userID := contextx.GetUserID(ctx)
	
	comment, err := s.getExtendedRepo().GetComment(ctx, id)
	if err != nil {
		return err
	}

	if comment.CreatedBy != userID {
		if err := s.CheckNodePermission(ctx, comment.NodeID, userID, "delete"); err != nil {
			return err
		}
	}

	return s.getExtendedRepo().DeleteComment(ctx, id)
}

func (s *wikiServiceExtended) buildCommentResp(ctx context.Context, comment *model.Comment) (*dto.CommentResp, error) {
	return &dto.CommentResp{
		ID:         comment.ID,
		DocumentID: comment.DocumentID,
		NodeID:     comment.NodeID,
		ParentID:   comment.ParentID,
		Content:    comment.Content,
		CreatedBy:  comment.CreatedBy,
		MentionIDs: comment.MentionIDs,
		Status:     comment.Status,
		CreatedAt:  comment.CreatedAt,
		UpdatedAt:  comment.UpdatedAt,
	}, nil
}

func (s *wikiServiceExtended) CreateShareLink(ctx context.Context, req *dto.ShareLinkReq) (*dto.ShareLinkResp, error) {
	userID := contextx.GetUserID(ctx)
	
	if err := s.CheckDocumentPermission(ctx, req.DocumentID, userID, "read"); err != nil {
		return nil, err
	}

	doc, node, err := s.getExtendedRepo().GetDocumentByIDWithNode(ctx, req.DocumentID)
	if err != nil {
		return nil, err
	}

	token := s.generateToken()
	link := &model.ShareLink{
		DocumentID:    req.DocumentID,
		NodeID:        node.ID,
		Token:         token,
		Password:      req.Password,
		ExpireAt:      req.ExpireAt,
		MaxViews:      req.MaxViews,
		AllowDownload: req.AllowDownload,
		CreatedBy:     userID,
	}

	if err := s.getExtendedRepo().CreateShareLink(ctx, link); err != nil {
		return nil, err
	}

	_ = s.getExtendedRepo().CreateAccessLog(ctx, &model.DocumentAccessLog{
		DocumentID: req.DocumentID,
		NodeID:     node.ID,
		UserID:     userID,
		Action:     model.ActionShare,
	})

	return s.buildShareLinkResp(ctx, link), nil
}

func (s *wikiServiceExtended) GetShareLink(ctx context.Context, token, password string) (*dto.ShareLinkResp, error) {
	link, err := s.getExtendedRepo().GetShareLinkByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if link.ExpireAt != nil && link.ExpireAt.Before(time.Now()) {
		return nil, fmt.Errorf("分享链接已过期")
	}

	if link.MaxViews > 0 && link.ViewCount >= link.MaxViews {
		return nil, fmt.Errorf("分享链接已达到最大访问次数")
	}

	if link.Password != "" && link.Password != password {
		return nil, fmt.Errorf("密码错误")
	}

	_ = s.getExtendedRepo().IncrementShareViewCount(ctx, link.ID)

	return s.buildShareLinkResp(ctx, link), nil
}

func (s *wikiServiceExtended) ListShareLinks(ctx context.Context, documentID string) ([]*dto.ShareLinkResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "read"); err != nil {
		return nil, err
	}

	links, err := s.getExtendedRepo().ListShareLinks(ctx, documentID)
	if err != nil {
		return nil, err
	}

	var resps []*dto.ShareLinkResp
	for _, link := range links {
		resps = append(resps, s.buildShareLinkResp(ctx, link))
	}
	return resps, nil
}

func (s *wikiServiceExtended) DeleteShareLink(ctx context.Context, id string) error {
	userID := contextx.GetUserID(ctx)
	
	link, err := s.getExtendedRepo().GetShareLinkByToken(ctx, id)
	if err != nil {
		return err
	}

	if err := s.CheckDocumentPermission(ctx, link.DocumentID, userID, "update"); err != nil {
		return err
	}

	return s.getExtendedRepo().DeleteShareLink(ctx, id)
}

func (s *wikiServiceExtended) buildShareLinkResp(ctx context.Context, link *model.ShareLink) *dto.ShareLinkResp {
	shareURL := fmt.Sprintf("/wiki/share/%s", link.Token)
	return &dto.ShareLinkResp{
		ID:            link.ID,
		DocumentID:    link.DocumentID,
		NodeID:        link.NodeID,
		Token:         link.Token,
		ExpireAt:      link.ExpireAt,
		MaxViews:      link.MaxViews,
		ViewCount:     link.ViewCount,
		AllowDownload: link.AllowDownload,
		CreatedBy:     link.CreatedBy,
		CreatedAt:     link.CreatedAt,
		ShareURL:      shareURL,
	}
}

func (s *wikiServiceExtended) generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *wikiServiceExtended) CreateTemplate(ctx context.Context, req *dto.CreateTemplateReq) (*dto.DocumentTemplateResp, error) {
	userID := contextx.GetUserID(ctx)
	tenantID := contextx.GetTenantID(ctx)

	template := &model.DocumentTemplate{
		Name:        req.Name,
		Description: req.Description,
		Content:     req.Content,
		Format:      req.Format,
		Category:    req.Category,
		IsPublic:    req.IsPublic,
		CreatedBy:   userID,
		TenantID:    tenantID,
	}

	if err := s.getExtendedRepo().CreateTemplate(ctx, template); err != nil {
		return nil, err
	}

	return &dto.DocumentTemplateResp{
		ID:          template.ID,
		Name:        template.Name,
		Description: template.Description,
		Content:     template.Content,
		Format:      template.Format,
		Category:    template.Category,
		IsPublic:    template.IsPublic,
		CreatedBy:   template.CreatedBy,
		CreatedAt:   template.CreatedAt,
		UpdatedAt:   template.UpdatedAt,
	}, nil
}

func (s *wikiServiceExtended) ListTemplates(ctx context.Context, category string) ([]*dto.DocumentTemplateResp, error) {
	tenantID := contextx.GetTenantID(ctx)
	templates, err := s.getExtendedRepo().ListTemplates(ctx, tenantID, category, false)
	if err != nil {
		return nil, err
	}

	var resps []*dto.DocumentTemplateResp
	for _, t := range templates {
		resps = append(resps, &dto.DocumentTemplateResp{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Content:     t.Content,
			Format:      t.Format,
			Category:    t.Category,
			IsPublic:    t.IsPublic,
			CreatedBy:   t.CreatedBy,
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
		})
	}
	return resps, nil
}

func (s *wikiServiceExtended) GetTemplate(ctx context.Context, id string) (*dto.DocumentTemplateResp, error) {
	template, err := s.getExtendedRepo().GetTemplate(ctx, id)
	if err != nil {
		return nil, err
	}

	return &dto.DocumentTemplateResp{
		ID:          template.ID,
		Name:        template.Name,
		Description: template.Description,
		Content:     template.Content,
		Format:      template.Format,
		Category:    template.Category,
		IsPublic:    template.IsPublic,
		CreatedBy:   template.CreatedBy,
		CreatedAt:   template.CreatedAt,
		UpdatedAt:   template.UpdatedAt,
	}, nil
}

func (s *wikiServiceExtended) UpdateTemplate(ctx context.Context, id string, req *dto.UpdateTemplateReq) (*dto.DocumentTemplateResp, error) {
	userID := contextx.GetUserID(ctx)
	
	template, err := s.getExtendedRepo().GetTemplate(ctx, id)
	if err != nil {
		return nil, err
	}

	if template.CreatedBy != userID {
		return nil, fmt.Errorf("permission denied")
	}

	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Description != "" {
		template.Description = req.Description
	}
	if req.Content != "" {
		template.Content = req.Content
	}
	if req.Category != "" {
		template.Category = req.Category
	}
	template.IsPublic = req.IsPublic

	if err := s.getExtendedRepo().UpdateTemplate(ctx, template); err != nil {
		return nil, err
	}

	return &dto.DocumentTemplateResp{
		ID:          template.ID,
		Name:        template.Name,
		Description: template.Description,
		Content:     template.Content,
		Format:      template.Format,
		Category:    template.Category,
		IsPublic:    template.IsPublic,
		CreatedBy:   template.CreatedBy,
		CreatedAt:   template.CreatedAt,
		UpdatedAt:   template.UpdatedAt,
	}, nil
}

func (s *wikiServiceExtended) DeleteTemplate(ctx context.Context, id string) error {
	userID := contextx.GetUserID(ctx)
	
	template, err := s.getExtendedRepo().GetTemplate(ctx, id)
	if err != nil {
		return err
	}

	if template.CreatedBy != userID {
		return fmt.Errorf("permission denied")
	}

	return s.getExtendedRepo().DeleteTemplate(ctx, id)
}

func (s *wikiServiceExtended) GetDocumentStats(ctx context.Context, documentID string) (*dto.DocumentStatsResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "read"); err != nil {
		return nil, err
	}

	stats, err := s.getExtendedRepo().GetDocumentStatsExtended(ctx, documentID)
	if err != nil {
		return nil, err
	}

	return &dto.DocumentStatsResp{
		TotalViews:     stats.TotalViews,
		TotalEdits:     stats.TotalEdits,
		TotalDownloads: stats.TotalDownloads,
		TotalShares:    stats.TotalShares,
		UniqueViewers:  stats.UniqueViewers,
		LastViewedAt:   stats.LastViewedAt,
	}, nil
}

func (s *wikiServiceExtended) ListAccessLogs(ctx context.Context, documentID string, page, pageSize int) ([]*dto.DocumentAccessLogResp, int64, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "read"); err != nil {
		return nil, 0, err
	}

	logs, total, err := s.getExtendedRepo().ListAccessLogs(ctx, documentID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	var resps []*dto.DocumentAccessLogResp
	for _, log := range logs {
		resps = append(resps, &dto.DocumentAccessLogResp{
			ID:         log.ID,
			DocumentID: log.DocumentID,
			UserID:     log.UserID,
			Action:     log.Action,
			IPAddress:  log.IPAddress,
			CreatedAt:  log.CreatedAt,
		})
	}
	return resps, total, nil
}

func (s *wikiServiceExtended) SubscribeDocument(ctx context.Context, documentID, notifyType string) (*dto.SubscriptionResp, error) {
	userID := contextx.GetUserID(ctx)
	
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "read"); err != nil {
		return nil, err
	}

	doc, node, err := s.getExtendedRepo().GetDocumentByIDWithNode(ctx, documentID)
	if err != nil {
		return nil, err
	}

	existing, _ := s.getExtendedRepo().GetSubscription(ctx, documentID, userID)
	if existing != nil {
		return &dto.SubscriptionResp{
			ID:         existing.ID,
			DocumentID: existing.DocumentID,
			NodeID:     existing.NodeID,
			UserID:     existing.UserID,
			NotifyType: existing.NotifyType,
			CreatedAt:  existing.CreatedAt,
		}, nil
	}

	sub := &model.DocumentSubscription{
		DocumentID: documentID,
		NodeID:     node.ID,
		UserID:     userID,
		NotifyType: notifyType,
	}

	if err := s.getExtendedRepo().CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	_ = doc
	return &dto.SubscriptionResp{
		ID:         sub.ID,
		DocumentID: sub.DocumentID,
		NodeID:     sub.NodeID,
		UserID:     sub.UserID,
		NotifyType: sub.NotifyType,
		CreatedAt:  sub.CreatedAt,
	}, nil
}

func (s *wikiServiceExtended) UnsubscribeDocument(ctx context.Context, documentID string) error {
	userID := contextx.GetUserID(ctx)
	return s.getExtendedRepo().DeleteSubscription(ctx, documentID, userID)
}

func (s *wikiServiceExtended) ListUserSubscriptions(ctx context.Context) ([]*dto.SubscriptionResp, error) {
	userID := contextx.GetUserID(ctx)
	subs, err := s.getExtendedRepo().GetUserSubscriptions(ctx, userID)
	if err != nil {
		return nil, err
	}

	var resps []*dto.SubscriptionResp
	for _, sub := range subs {
		resps = append(resps, &dto.SubscriptionResp{
			ID:         sub.ID,
			DocumentID: sub.DocumentID,
			NodeID:     sub.NodeID,
			UserID:     sub.UserID,
			NotifyType: sub.NotifyType,
			CreatedAt:  sub.CreatedAt,
		})
	}
	return resps, nil
}

func (s *wikiServiceExtended) ListNotifications(ctx context.Context, page, pageSize int) ([]*dto.NotificationResp, int64, error) {
	userID := contextx.GetUserID(ctx)
	notifications, total, err := s.getExtendedRepo().ListNotifications(ctx, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	var resps []*dto.NotificationResp
	for _, n := range notifications {
		resps = append(resps, &dto.NotificationResp{
			ID:          n.ID,
			Type:        n.Type,
			Title:       n.Title,
			Content:     n.Content,
			RelatedID:   n.RelatedID,
			RelatedType: n.RelatedType,
			IsRead:      n.IsRead,
			CreatedAt:   n.CreatedAt,
		})
	}
	return resps, total, nil
}

func (s *wikiServiceExtended) MarkNotificationAsRead(ctx context.Context, id string) error {
	return s.getExtendedRepo().MarkNotificationAsRead(ctx, id)
}

func (s *wikiServiceExtended) MarkAllNotificationsAsRead(ctx context.Context) error {
	userID := contextx.GetUserID(ctx)
	return s.getExtendedRepo().MarkAllNotificationsAsRead(ctx, userID)
}

func (s *wikiServiceExtended) GetUnreadNotificationCount(ctx context.Context) (int64, error) {
	userID := contextx.GetUserID(ctx)
	return s.getExtendedRepo().GetUnreadNotificationCount(ctx, userID)
}

func (s *wikiServiceExtended) AcquireEditLock(ctx context.Context, documentID string) (*dto.EditLockResp, error) {
	userID := contextx.GetUserID(ctx)
	
	locked, err := s.getExtendedRepo().IsDocumentLocked(ctx, documentID)
	if err != nil {
		return nil, err
	}

	if locked {
		existingLock, err := s.getExtendedRepo().GetEditLock(ctx, documentID)
		if err != nil {
			return nil, err
		}

		if existingLock.UserID == userID {
			_ = s.getExtendedRepo().RefreshEditLock(ctx, documentID)
			return &dto.EditLockResp{
				DocumentID: documentID,
				UserID:     existingLock.UserID,
				LockedAt:   existingLock.LockedAt,
				ExpiresAt:  existingLock.ExpiresAt,
				CanEdit:    true,
			}, nil
		}

		return &dto.EditLockResp{
			DocumentID: documentID,
			UserID:     existingLock.UserID,
			LockedAt:   existingLock.LockedAt,
			ExpiresAt:  existingLock.ExpiresAt,
			CanEdit:    false,
		}, nil
	}

	lock := &model.EditLock{
		DocumentID: documentID,
		UserID:     userID,
		LockedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(30 * time.Minute),
	}

	if err := s.getExtendedRepo().AcquireEditLock(ctx, lock); err != nil {
		return nil, err
	}

	return &dto.EditLockResp{
		DocumentID: documentID,
		UserID:     userID,
		LockedAt:   lock.LockedAt,
		ExpiresAt:  lock.ExpiresAt,
		CanEdit:    true,
	}, nil
}

func (s *wikiServiceExtended) ReleaseEditLock(ctx context.Context, documentID string) error {
	userID := contextx.GetUserID(ctx)
	
	lock, err := s.getExtendedRepo().GetEditLock(ctx, documentID)
	if err != nil {
		return err
	}

	if lock.UserID != userID {
		return fmt.Errorf("permission denied")
	}

	return s.getExtendedRepo().ReleaseEditLock(ctx, documentID)
}

func (s *wikiServiceExtended) RefreshEditLock(ctx context.Context, documentID string) error {
	return s.getExtendedRepo().RefreshEditLock(ctx, documentID)
}

func (s *wikiServiceExtended) GetEditLock(ctx context.Context, documentID string) (*dto.EditLockResp, error) {
	lock, err := s.getExtendedRepo().GetEditLock(ctx, documentID)
	if err != nil {
		return nil, err
	}

	return &dto.EditLockResp{
		DocumentID: documentID,
		UserID:     lock.UserID,
		LockedAt:   lock.LockedAt,
		ExpiresAt:  lock.ExpiresAt,
		CanEdit:    true,
	}, nil
}

func (s *wikiServiceExtended) BatchDeleteNodes(ctx context.Context, nodeIDs []string) error {
	userID := contextx.GetUserID(ctx)
	
	return s.wikiService.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
		for _, nodeID := range nodeIDs {
			if err := s.wikiService.DeleteNode(txCtx, nodeID); err != nil {
				return err
			}
		}
		return nil
	})

	_ = userID
}

func (s *wikiServiceExtended) BatchMoveNodes(ctx context.Context, nodeIDs []string, newParentID string) error {
	userID := contextx.GetUserID(ctx)
	
	nodes, err := s.getExtendedRepo().ListNodesByIDs(ctx, nodeIDs)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		if err := s.wikiService.CheckNodePermission(ctx, node.ID, userID, "update"); err != nil {
			return err
		}
	}

	return s.getExtendedRepo().BatchMoveNodes(ctx, nodeIDs, newParentID)
}

func (s *wikiServiceExtended) CompareRevisions(ctx context.Context, documentID string, version1, version2 int) (*dto.DiffResult, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "read"); err != nil {
		return nil, err
	}

	oldContent, newContent, err := s.getExtendedRepo().CompareRevisions(ctx, documentID, version1, version2)
	if err != nil {
		return nil, err
	}

	diffStr := s.getExtendedRepo().GenerateDiff(ctx, oldContent, newContent)
	
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")
	
	var diffs []dto.DiffLine
	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	lineNum := 0
	for i := 0; i < maxLen; i++ {
		oldLine := ""
		newLine := ""
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine == newLine {
			lineNum++
			diffs = append(diffs, dto.DiffLine{
				Type:    "unchanged",
				LineNum: lineNum,
				Content: oldLine,
				OldLine: i + 1,
				NewLine: i + 1,
			})
		} else {
			if oldLine != "" {
				lineNum++
				diffs = append(diffs, dto.DiffLine{
					Type:    "removed",
					LineNum: lineNum,
					Content: oldLine,
					OldLine: i + 1,
				})
			}
			if newLine != "" {
				lineNum++
				diffs = append(diffs, dto.DiffLine{
					Type:    "added",
					LineNum: lineNum,
					Content: newLine,
					NewLine: i + 1,
				})
			}
		}
	}

	_ = diffStr
	return &dto.DiffResult{
		OldVersion: version1,
		NewVersion: version2,
		Diffs:      diffs,
	}, nil
}

func (s *wikiServiceExtended) ExportDocument(ctx context.Context, documentID, format string) ([]byte, string, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "read"); err != nil {
		return nil, "", err
	}

	doc, _, err := s.getExtendedRepo().GetDocumentByIDWithNode(ctx, documentID)
	if err != nil {
		return nil, "", err
	}

	var content []byte
	var filename string

	switch format {
	case "markdown":
		content = []byte(doc.Content)
		filename = "document.md"
	case "html":
		content = []byte(doc.ContentHTML)
		filename = "document.html"
	case "pdf":
		content = []byte(doc.ContentHTML)
		filename = "document.pdf"
	default:
		return nil, "", fmt.Errorf("unsupported format: %s", format)
	}

	_ = s.getExtendedRepo().CreateAccessLog(ctx, &model.DocumentAccessLog{
		DocumentID: documentID,
		NodeID:     doc.NodeID,
		UserID:     userID,
		Action:     model.ActionDownload,
	})

	return content, filename, nil
}