package repository

import (
	"context"
	"strings"
	"time"

	"meteorx/internal/common/tenantctx"
	"meteorx/internal/modules/wiki/model"
	"meteorx/pkg/idgen"

	"gorm.io/gorm"
)

type WikiRepositoryExtended interface {
	WikiRepository

	CreateTag(ctx context.Context, tag *model.Tag) error
	ListTags(ctx context.Context, tenantID string) ([]*model.Tag, error)
	DeleteTag(ctx context.Context, id string) error

	AddDocumentTag(ctx context.Context, documentID, tagID string) error
	RemoveDocumentTag(ctx context.Context, documentID, tagID string) error
	ListDocumentTags(ctx context.Context, documentID string) ([]*model.DocumentTag, error)
	ListDocumentTagsWithDetails(ctx context.Context, documentID string) ([]*model.DocumentTag, error)

	CreateComment(ctx context.Context, comment *model.Comment) error
	ListComments(ctx context.Context, documentID string) ([]*model.Comment, error)
	ListCommentsByDocument(ctx context.Context, documentID string) ([]*model.Comment, error)
	ListRepliesByParentID(ctx context.Context, parentID string) ([]*model.Comment, error)
	UpdateComment(ctx context.Context, comment *model.Comment) error
	DeleteComment(ctx context.Context, id string) error
	GetComment(ctx context.Context, id string) (*model.Comment, error)

	CreateShareLink(ctx context.Context, link *model.ShareLink) error
	GetShareLinkByToken(ctx context.Context, token string) (*model.ShareLink, error)
	ListShareLinks(ctx context.Context, documentID string) ([]*model.ShareLink, error)
	UpdateShareLink(ctx context.Context, link *model.ShareLink) error
	DeleteShareLink(ctx context.Context, id string) error
	IncrementShareViewCount(ctx context.Context, id string) error

	CreateTemplate(ctx context.Context, template *model.DocumentTemplate) error
	ListTemplates(ctx context.Context, tenantID string, category string, isPublic bool) ([]*model.DocumentTemplate, error)
	GetTemplate(ctx context.Context, id string) (*model.DocumentTemplate, error)
	UpdateTemplate(ctx context.Context, template *model.DocumentTemplate) error
	DeleteTemplate(ctx context.Context, id string) error

	CreateAccessLog(ctx context.Context, log *model.DocumentAccessLog) error
	ListAccessLogs(ctx context.Context, documentID string, page, pageSize int) ([]*model.DocumentAccessLog, int64, error)
	GetDocumentStats(ctx context.Context, documentID string) (*model.DocumentStats, error)
	GetDocumentStatsExtended(ctx context.Context, documentID string) (*model.DocumentStats, error)

	CreateSubscription(ctx context.Context, sub *model.DocumentSubscription) error
	ListSubscriptions(ctx context.Context, documentID string) ([]*model.DocumentSubscription, error)
	DeleteSubscription(ctx context.Context, documentID, userID string) error
	GetSubscription(ctx context.Context, documentID, userID string) (*model.DocumentSubscription, error)

	CreateNotification(ctx context.Context, notification *model.Notification) error
	ListNotifications(ctx context.Context, userID string, page, pageSize int) ([]*model.Notification, int64, error)
	MarkNotificationAsRead(ctx context.Context, id string) error
	MarkAllNotificationsAsRead(ctx context.Context, userID string) error
	GetUnreadNotificationCount(ctx context.Context, userID string) (int64, error)

	AcquireEditLock(ctx context.Context, lock *model.EditLock) error
	ReleaseEditLock(ctx context.Context, documentID string) error
	GetEditLock(ctx context.Context, documentID string) (*model.EditLock, error)
	IsDocumentLocked(ctx context.Context, documentID string) (bool, error)
	RefreshEditLock(ctx context.Context, documentID string) error

	BatchDeleteNodes(ctx context.Context, nodeIDs []string) error
	BatchMoveNodes(ctx context.Context, nodeIDs []string, newParentID string) error

	CompareRevisions(ctx context.Context, documentID string, version1, version2 int) (string, string, error)
	
	GetDocumentByIDWithNode(ctx context.Context, id string) (*model.Document, *model.WikiNode, error)
	GetUserSubscriptions(ctx context.Context, userID string) ([]*model.DocumentSubscription, error)
	ListNodesByIDs(ctx context.Context, nodeIDs []string) ([]*model.WikiNode, error)
	GenerateDiff(ctx context.Context, oldContent, newContent string) string
}

type wikiRepositoryExtended struct {
	wikiRepository
}

func NewWikiRepositoryExtended(database *gorm.DB) WikiRepositoryExtended {
	return &wikiRepositoryExtended{
		wikiRepository: wikiRepository{db: database},
	}
}

func (r *wikiRepositoryExtended) getDB(ctx context.Context) *gorm.DB {
	return r.wikiRepository.getDB(ctx)
}

func (r *wikiRepositoryExtended) CreateTag(ctx context.Context, tag *model.Tag) error {
	tag.ID = idgen.NewULID()
	if tag.TenantID == "" {
		tag.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(tag).Error
}

func (r *wikiRepositoryExtended) ListTags(ctx context.Context, tenantID string) ([]*model.Tag, error) {
	var tags []*model.Tag
	query := tenantctx.FilterQuery(ctx, r.getDB(ctx), "tenant_id")
	err := query.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&tags).Error
	return tags, err
}

func (r *wikiRepositoryExtended) DeleteTag(ctx context.Context, id string) error {
	return r.getDB(ctx).Where("id = ?", id).Delete(&model.Tag{}).Error
}

func (r *wikiRepositoryExtended) AddDocumentTag(ctx context.Context, documentID, tagID string) error {
	dt := &model.DocumentTag{
		ID:         idgen.NewULID(),
		DocumentID: documentID,
		TagID:      tagID,
		TenantID:   tenantctx.TenantID(ctx),
	}
	return r.getDB(ctx).Create(dt).Error
}

func (r *wikiRepositoryExtended) RemoveDocumentTag(ctx context.Context, documentID, tagID string) error {
	return r.getDB(ctx).Where("document_id = ? AND tag_id = ?", documentID, tagID).Delete(&model.DocumentTag{}).Error
}

func (r *wikiRepositoryExtended) ListDocumentTags(ctx context.Context, documentID string) ([]*model.DocumentTag, error) {
	var tags []*model.DocumentTag
	err := r.getDB(ctx).Preload("Tag").Where("document_id = ?", documentID).Find(&tags).Error
	return tags, err
}

func (r *wikiRepositoryExtended) CreateComment(ctx context.Context, comment *model.Comment) error {
	comment.ID = idgen.NewULID()
	if comment.TenantID == "" {
		comment.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(comment).Error
}

func (r *wikiRepositoryExtended) ListComments(ctx context.Context, documentID string) ([]*model.Comment, error) {
	var comments []*model.Comment
	err := r.getDB(ctx).Where("document_id = ? AND status = ?", documentID, model.CommentStatusActive).
		Order("created_at ASC").Find(&comments).Error
	return comments, err
}

func (r *wikiRepositoryExtended) UpdateComment(ctx context.Context, comment *model.Comment) error {
	return r.getDB(ctx).Save(comment).Error
}

func (r *wikiRepositoryExtended) DeleteComment(ctx context.Context, id string) error {
	return r.getDB(ctx).Where("id = ?", id).Delete(&model.Comment{}).Error
}

func (r *wikiRepositoryExtended) GetComment(ctx context.Context, id string) (*model.Comment, error) {
	var comment model.Comment
	err := r.getDB(ctx).Where("id = ?", id).First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func (r *wikiRepositoryExtended) CreateShareLink(ctx context.Context, link *model.ShareLink) error {
	link.ID = idgen.NewULID()
	if link.TenantID == "" {
		link.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(link).Error
}

func (r *wikiRepositoryExtended) GetShareLinkByToken(ctx context.Context, token string) (*model.ShareLink, error) {
	var link model.ShareLink
	err := r.getDB(ctx).Where("token = ?", token).First(&link).Error
	if err != nil {
		return nil, err
	}
	return &link, nil
}

func (r *wikiRepositoryExtended) ListShareLinks(ctx context.Context, documentID string) ([]*model.ShareLink, error) {
	var links []*model.ShareLink
	err := r.getDB(ctx).Where("document_id = ?", documentID).Order("created_at DESC").Find(&links).Error
	return links, err
}

func (r *wikiRepositoryExtended) UpdateShareLink(ctx context.Context, link *model.ShareLink) error {
	return r.getDB(ctx).Save(link).Error
}

func (r *wikiRepositoryExtended) DeleteShareLink(ctx context.Context, id string) error {
	return r.getDB(ctx).Where("id = ?", id).Delete(&model.ShareLink{}).Error
}

func (r *wikiRepositoryExtended) IncrementShareViewCount(ctx context.Context, id string) error {
	return r.getDB(ctx).Model(&model.ShareLink{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + ?", 1)).Error
}

func (r *wikiRepositoryExtended) CreateTemplate(ctx context.Context, template *model.DocumentTemplate) error {
	template.ID = idgen.NewULID()
	if template.TenantID == "" {
		template.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(template).Error
}

func (r *wikiRepositoryExtended) ListTemplates(ctx context.Context, tenantID string, category string, isPublic bool) ([]*model.DocumentTemplate, error) {
	var templates []*model.DocumentTemplate
	query := r.getDB(ctx).Where("tenant_id = ? OR is_public = ?", tenantID, true)
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if !isPublic {
		query = query.Where("tenant_id = ?", tenantID)
	}
	err := query.Order("created_at DESC").Find(&templates).Error
	return templates, err
}

func (r *wikiRepositoryExtended) GetTemplate(ctx context.Context, id string) (*model.DocumentTemplate, error) {
	var template model.DocumentTemplate
	err := r.getDB(ctx).Where("id = ?", id).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *wikiRepositoryExtended) UpdateTemplate(ctx context.Context, template *model.DocumentTemplate) error {
	return r.getDB(ctx).Save(template).Error
}

func (r *wikiRepositoryExtended) DeleteTemplate(ctx context.Context, id string) error {
	return r.getDB(ctx).Where("id = ?", id).Delete(&model.DocumentTemplate{}).Error
}

func (r *wikiRepositoryExtended) CreateAccessLog(ctx context.Context, log *model.DocumentAccessLog) error {
	log.ID = idgen.NewULID()
	if log.TenantID == "" {
		log.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(log).Error
}

func (r *wikiRepositoryExtended) ListAccessLogs(ctx context.Context, documentID string, page, pageSize int) ([]*model.DocumentAccessLog, int64, error) {
	var logs []*model.DocumentAccessLog
	var total int64
	query := r.getDB(ctx).Model(&model.DocumentAccessLog{}).Where("document_id = ?", documentID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page > 0 && pageSize > 0 {
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}
	err := query.Order("created_at DESC").Find(&logs).Error
	return logs, total, err
}

func (r *wikiRepositoryExtended) GetDocumentStats(ctx context.Context, documentID string) (*model.DocumentStats, error) {
	stats := &model.DocumentStats{}
	query := r.getDB(ctx).Model(&model.DocumentAccessLog{}).Where("document_id = ?", documentID)

	query.Where("action = ?", model.ActionView).Count(&stats.TotalViews)
	query.Where("action = ?", model.ActionEdit).Count(&stats.TotalEdits)
	query.Where("action = ?", model.ActionDownload).Count(&stats.TotalDownloads)
	query.Where("action = ?", model.ActionShare).Count(&stats.TotalShares)

	r.getDB(ctx).Model(&model.DocumentAccessLog{}).
		Where("document_id = ? AND action = ?", documentID, model.ActionView).
		Distinct("user_id").Count(&stats.UniqueViewers)

	r.getDB(ctx).Model(&model.DocumentAccessLog{}).
		Where("document_id = ? AND action = ?", documentID, model.ActionView).
		Order("created_at DESC").First(&stats.LastViewedAt)

	return stats, nil
}

func (r *wikiRepositoryExtended) CreateSubscription(ctx context.Context, sub *model.DocumentSubscription) error {
	sub.ID = idgen.NewULID()
	if sub.TenantID == "" {
		sub.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(sub).Error
}

func (r *wikiRepositoryExtended) ListSubscriptions(ctx context.Context, documentID string) ([]*model.DocumentSubscription, error) {
	var subs []*model.DocumentSubscription
	err := r.getDB(ctx).Where("document_id = ?", documentID).Find(&subs).Error
	return subs, err
}

func (r *wikiRepositoryExtended) DeleteSubscription(ctx context.Context, documentID, userID string) error {
	return r.getDB(ctx).Where("document_id = ? AND user_id = ?", documentID, userID).
		Delete(&model.DocumentSubscription{}).Error
}

func (r *wikiRepositoryExtended) GetSubscription(ctx context.Context, documentID, userID string) (*model.DocumentSubscription, error) {
	var sub model.DocumentSubscription
	err := r.getDB(ctx).Where("document_id = ? AND user_id = ?", documentID, userID).First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *wikiRepositoryExtended) CreateNotification(ctx context.Context, notification *model.Notification) error {
	notification.ID = idgen.NewULID()
	if notification.TenantID == "" {
		notification.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(notification).Error
}

func (r *wikiRepositoryExtended) ListNotifications(ctx context.Context, userID string, page, pageSize int) ([]*model.Notification, int64, error) {
	var notifications []*model.Notification
	var total int64
	query := r.getDB(ctx).Model(&model.Notification{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page > 0 && pageSize > 0 {
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}
	err := query.Order("created_at DESC").Find(&notifications).Error
	return notifications, total, err
}

func (r *wikiRepositoryExtended) MarkNotificationAsRead(ctx context.Context, id string) error {
	return r.getDB(ctx).Model(&model.Notification{}).Where("id = ?", id).Update("is_read", true).Error
}

func (r *wikiRepositoryExtended) MarkAllNotificationsAsRead(ctx context.Context, userID string) error {
	return r.getDB(ctx).Model(&model.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}

func (r *wikiRepositoryExtended) GetUnreadNotificationCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.getDB(ctx).Model(&model.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&count).Error
	return count, err
}

func (r *wikiRepositoryExtended) AcquireEditLock(ctx context.Context, lock *model.EditLock) error {
	lock.ID = idgen.NewULID()
	if lock.TenantID == "" {
		lock.TenantID = tenantctx.TenantID(ctx)
	}
	if lock.ExpiresAt.IsZero() {
		lock.ExpiresAt = time.Now().Add(30 * time.Minute)
	}
	return r.getDB(ctx).Create(lock).Error
}

func (r *wikiRepositoryExtended) ReleaseEditLock(ctx context.Context, documentID string) error {
	return r.getDB(ctx).Where("document_id = ?", documentID).Delete(&model.EditLock{}).Error
}

func (r *wikiRepositoryExtended) GetEditLock(ctx context.Context, documentID string) (*model.EditLock, error) {
	var lock model.EditLock
	err := r.getDB(ctx).Where("document_id = ?", documentID).First(&lock).Error
	if err != nil {
		return nil, err
	}
	return &lock, nil
}

func (r *wikiRepositoryExtended) IsDocumentLocked(ctx context.Context, documentID string) (bool, error) {
	var lock model.EditLock
	err := r.getDB(ctx).Where("document_id = ? AND expires_at > ?", documentID, time.Now()).First(&lock).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *wikiRepositoryExtended) RefreshEditLock(ctx context.Context, documentID string) error {
	return r.getDB(ctx).Model(&model.EditLock{}).Where("document_id = ?", documentID).
		Update("expires_at", time.Now().Add(30*time.Minute)).Error
}

func (r *wikiRepositoryExtended) BatchDeleteNodes(ctx context.Context, nodeIDs []string) error {
	return r.getDB(ctx).Where("id IN ?", nodeIDs).Delete(&model.WikiNode{}).Error
}

func (r *wikiRepositoryExtended) BatchMoveNodes(ctx context.Context, nodeIDs []string, newParentID string) error {
	return r.getDB(ctx).Model(&model.WikiNode{}).Where("id IN ?", nodeIDs).
		Update("parent_id", newParentID).Error
}

func (r *wikiRepositoryExtended) CompareRevisions(ctx context.Context, documentID string, version1, version2 int) (string, string, error) {
	var rev1, rev2 model.DocumentRevision
	if err := r.getDB(ctx).Where("document_id = ? AND version = ?", documentID, version1).First(&rev1).Error; err != nil {
		return "", "", err
	}
	if err := r.getDB(ctx).Where("document_id = ? AND version = ?", documentID, version2).First(&rev2).Error; err != nil {
		return "", "", err
	}
	return rev1.Content, rev2.Content, nil
}

func (r *wikiRepositoryExtended) ListNodesByIDs(ctx context.Context, nodeIDs []string) ([]*model.WikiNode, error) {
	var nodes []*model.WikiNode
	err := r.getDB(ctx).Where("id IN ?", nodeIDs).Find(&nodes).Error
	return nodes, err
}

func (r *wikiRepositoryExtended) ListDocumentsByNodeIDs(ctx context.Context, nodeIDs []string) ([]*model.Document, error) {
	var docs []*model.Document
	err := r.getDB(ctx).Where("node_id IN ?", nodeIDs).Find(&docs).Error
	return docs, err
}

func (r *wikiRepositoryExtended) GetNodeByDocumentID(ctx context.Context, documentID string) (*model.WikiNode, error) {
	var doc model.Document
	if err := r.getDB(ctx).Where("id = ?", documentID).First(&doc).Error; err != nil {
		return nil, err
	}
	var node model.WikiNode
	if err := r.getDB(ctx).Where("id = ?", doc.NodeID).First(&node).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *wikiRepositoryExtended) ListDocumentTagsWithDetails(ctx context.Context, documentID string) ([]*model.DocumentTag, error) {
	var docTags []*model.DocumentTag
	err := r.getDB(ctx).Preload("Tag").Where("document_id = ?", documentID).Find(&docTags).Error
	return docTags, err
}

func (r *wikiRepositoryExtended) ListCommentsByDocument(ctx context.Context, documentID string) ([]*model.Comment, error) {
	var comments []*model.Comment
	err := r.getDB(ctx).Where("document_id = ? AND status = ?", documentID, model.CommentStatusActive).
		Order("created_at ASC").Find(&comments).Error
	return comments, err
}

func (r *wikiRepositoryExtended) ListRepliesByParentID(ctx context.Context, parentID string) ([]*model.Comment, error) {
	var comments []*model.Comment
	err := r.getDB(ctx).Where("parent_id = ? AND status = ?", parentID, model.CommentStatusActive).
		Order("created_at ASC").Find(&comments).Error
	return comments, err
}

func (r *wikiRepositoryExtended) CountDocumentsByTag(ctx context.Context, tagID string) (int64, error) {
	var count int64
	err := r.getDB(ctx).Model(&model.DocumentTag{}).Where("tag_id = ?", tagID).Count(&count).Error
	return count, err
}

func (r *wikiRepositoryExtended) SearchDocumentsByTags(ctx context.Context, tenantID string, tagIDs []string) ([]*model.Document, error) {
	var docs []*model.Document
	query := r.getDB(ctx).Model(&model.Document{}).
		Joins("JOIN wiki_document_tags ON wiki_document_tags.document_id = wiki_documents.id").
		Where("wiki_documents.tenant_id = ?", tenantID).
		Where("wiki_document_tags.tag_id IN ?", tagIDs)
	err := query.Find(&docs).Error
	return docs, err
}

func (r *wikiRepositoryExtended) ListTemplatesByCategory(ctx context.Context, tenantID string) (map[string][]*model.DocumentTemplate, error) {
	var templates []*model.DocumentTemplate
	query := r.getDB(ctx).Where("tenant_id = ? OR is_public = ?", tenantID, true)
	if err := query.Order("category, created_at DESC").Find(&templates).Error; err != nil {
		return nil, err
	}

	result := make(map[string][]*model.DocumentTemplate)
	for _, t := range templates {
		result[t.Category] = append(result[t.Category], t)
	}
	return result, nil
}

func (r *wikiRepositoryExtended) GetShareLinkStats(ctx context.Context, linkID string) (int, error) {
	var link model.ShareLink
	if err := r.getDB(ctx).Where("id = ?", linkID).First(&link).Error; err != nil {
		return 0, err
	}
	return link.ViewCount, nil
}

func (r *wikiRepositoryExtended) ListExpiredShareLinks(ctx context.Context) ([]*model.ShareLink, error) {
	var links []*model.ShareLink
	err := r.getDB(ctx).Where("expire_at IS NOT NULL AND expire_at < ?", time.Now()).Find(&links).Error
	return links, err
}

func (r *wikiRepositoryExtended) DeleteExpiredShareLinks(ctx context.Context) error {
	return r.getDB(ctx).Where("expire_at IS NOT NULL AND expire_at < ?", time.Now()).Delete(&model.ShareLink{}).Error
}

func (r *wikiRepositoryExtended) CleanExpiredEditLocks(ctx context.Context) error {
	return r.getDB(ctx).Where("expires_at < ?", time.Now()).Delete(&model.EditLock{}).Error
}

func (r *wikiRepositoryExtended) ListLockedDocuments(ctx context.Context) ([]*model.EditLock, error) {
	var locks []*model.EditLock
	err := r.getDB(ctx).Where("expires_at > ?", time.Now()).Find(&locks).Error
	return locks, err
}

func (r *wikiRepositoryExtended) GetUserSubscriptions(ctx context.Context, userID string) ([]*model.DocumentSubscription, error) {
	var subs []*model.DocumentSubscription
	err := r.getDB(ctx).Where("user_id = ?", userID).Find(&subs).Error
	return subs, err
}

func (r *wikiRepositoryExtended) ListUnreadNotifications(ctx context.Context, userID string, page, pageSize int) ([]*model.Notification, int64, error) {
	var notifications []*model.Notification
	var total int64
	query := r.getDB(ctx).Model(&model.Notification{}).Where("user_id = ? AND is_read = ?", userID, false)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page > 0 && pageSize > 0 {
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}
	err := query.Order("created_at DESC").Find(&notifications).Error
	return notifications, total, err
}

func (r *wikiRepositoryExtended) GetDocumentByIDWithNode(ctx context.Context, id string) (*model.Document, *model.WikiNode, error) {
	var doc model.Document
	if err := r.getDB(ctx).Where("id = ?", id).First(&doc).Error; err != nil {
		return nil, nil, err
	}
	var node model.WikiNode
	if err := r.getDB(ctx).Where("id = ?", doc.NodeID).First(&node).Error; err != nil {
		return &doc, nil, err
	}
	return &doc, &node, nil
}

func (r *wikiRepositoryExtended) GetDocumentStatsExtended(ctx context.Context, documentID string) (*model.DocumentStats, error) {
	stats := &model.DocumentStats{}

	r.getDB(ctx).Model(&model.DocumentAccessLog{}).
		Where("document_id = ? AND action = ?", documentID, model.ActionView).
		Count(&stats.TotalViews)

	r.getDB(ctx).Model(&model.DocumentAccessLog{}).
		Where("document_id = ? AND action = ?", documentID, model.ActionEdit).
		Count(&stats.TotalEdits)

	r.getDB(ctx).Model(&model.DocumentAccessLog{}).
		Where("document_id = ? AND action = ?", documentID, model.ActionDownload).
		Count(&stats.TotalDownloads)

	r.getDB(ctx).Model(&model.DocumentAccessLog{}).
		Where("document_id = ? AND action = ?", documentID, model.ActionShare).
		Count(&stats.TotalShares)

	r.getDB(ctx).Model(&model.DocumentAccessLog{}).
		Where("document_id = ? AND action = ?", documentID, model.ActionView).
		Distinct("user_id").Count(&stats.UniqueViewers)

	var lastLog model.DocumentAccessLog
	r.getDB(ctx).Where("document_id = ? AND action = ?", documentID, model.ActionView).
		Order("created_at DESC").First(&lastLog)
	stats.LastViewedAt = lastLog.CreatedAt

	return stats, nil
}

func (r *wikiRepositoryExtended) GenerateDiff(ctx context.Context, oldContent, newContent string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	maxLen := len(oldLines)
	if len(newLines) > maxLen {
		maxLen = len(newLines)
	}

	var diff strings.Builder
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
			diff.WriteString("  " + oldLine + "\n")
		} else {
			if oldLine != "" {
				diff.WriteString("- " + oldLine + "\n")
			}
			if newLine != "" {
				diff.WriteString("+ " + newLine + "\n")
			}
		}
	}

	return diff.String()
}