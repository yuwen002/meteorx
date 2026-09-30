// Package repository 定义 Wiki 模块扩展数据访问接口（标签/评论/分享/模板/统计/订阅/通知/编辑锁/评审）及 GORM 实现。
package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"meteorx/internal/common/tenantctx"
	"meteorx/internal/modules/wiki/model"
	"meteorx/pkg/idgen"

	"gorm.io/gorm"
)

// WikiRepositoryExtended Wiki 扩展仓库接口，包含标签、评论、分享、模板、订阅、通知、编辑锁等
type WikiRepositoryExtended interface {
	WikiRepository

	CreateTag(ctx context.Context, tag *model.Tag) error
	ListTags(ctx context.Context, tenantID string) ([]*model.Tag, error)
	GetTagByID(ctx context.Context, id string) (*model.Tag, error)
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
	GetShareLinkByID(ctx context.Context, id string) (*model.ShareLink, error)
	ListShareLinks(ctx context.Context, documentID string) ([]*model.ShareLink, error)
	UpdateShareLink(ctx context.Context, link *model.ShareLink) error
	DeleteShareLink(ctx context.Context, id string) error
	IncrementShareViewCount(ctx context.Context, id string, maxViews int) (bool, error)
	// GetSharedDocument 按文档 ID+租户读取（供免登录分享场景显式传租户，不依赖请求上下文）
	GetSharedDocument(ctx context.Context, documentID, tenantID string) (*model.Document, error)
	// GetSharedNode 按节点 ID+租户读取（分享文档标题来自节点）
	GetSharedNode(ctx context.Context, nodeID, tenantID string) (*model.WikiNode, error)

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
	MarkNotificationAsRead(ctx context.Context, id, userID string) error
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

	UpdateDocumentPublishStatus(ctx context.Context, documentID string, status string, reviewedBy string, reviewedAt *time.Time, publishedAt *time.Time) error
	CreateReviewComment(ctx context.Context, comment *model.ReviewComment) error
	ListReviewComments(ctx context.Context, documentID string) ([]*model.ReviewComment, error)
	ListPendingReviewDocuments(ctx context.Context, tenantID string, page, pageSize int) ([]*model.Document, int64, error)
}

// wikiRepositoryExtended Wiki 扩展仓库实现
type wikiRepositoryExtended struct {
	wikiRepository
}

// NewWikiRepositoryExtended 创建 Wiki 扩展仓库实例
func NewWikiRepositoryExtended(database *gorm.DB) WikiRepositoryExtended {
	return &wikiRepositoryExtended{
		wikiRepository: wikiRepository{db: database},
	}
}

// getDB 获取带租户过滤的数据库连接
func (r *wikiRepositoryExtended) getDB(ctx context.Context) *gorm.DB {
	return r.wikiRepository.getDB(ctx)
}

// CreateTag 创建标签
func (r *wikiRepositoryExtended) CreateTag(ctx context.Context, tag *model.Tag) error {
	tag.ID = idgen.NewULID()
	if tag.TenantID == "" {
		tag.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(tag).Error
}

// ListTags 列出指定租户的所有标签
func (r *wikiRepositoryExtended) ListTags(ctx context.Context, tenantID string) ([]*model.Tag, error) {
	var tags []*model.Tag
	query := tenantctx.FilterQuery(ctx, r.getDB(ctx), "tenant_id")
	err := query.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&tags).Error
	return tags, err
}

// GetTagByID 根据ID获取标签
func (r *wikiRepositoryExtended) GetTagByID(ctx context.Context, id string) (*model.Tag, error) {
	var tag model.Tag
	query := tenantctx.FilterQuery(ctx, r.getDB(ctx), "tenant_id")
	err := query.Where("id = ?", id).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

// DeleteTag 删除标签
func (r *wikiRepositoryExtended) DeleteTag(ctx context.Context, id string) error {
	return r.getDB(ctx).Where("id = ?", id).Delete(&model.Tag{}).Error
}

// AddDocumentTag 为文档添加标签关联
func (r *wikiRepositoryExtended) AddDocumentTag(ctx context.Context, documentID, tagID string) error {
	dt := &model.DocumentTag{
		ID:         idgen.NewULID(),
		DocumentID: documentID,
		TagID:      tagID,
		TenantID:   tenantctx.TenantID(ctx),
	}
	return r.getDB(ctx).Create(dt).Error
}

// RemoveDocumentTag 移除文档标签关联
func (r *wikiRepositoryExtended) RemoveDocumentTag(ctx context.Context, documentID, tagID string) error {
	return r.getDB(ctx).Where("document_id = ? AND tag_id = ?", documentID, tagID).Delete(&model.DocumentTag{}).Error
}

// ListDocumentTags 列出文档的所有标签关联（含标签详情）
func (r *wikiRepositoryExtended) ListDocumentTags(ctx context.Context, documentID string) ([]*model.DocumentTag, error) {
	var tags []*model.DocumentTag
	err := r.getDB(ctx).Preload("Tag").Where("document_id = ?", documentID).Find(&tags).Error
	return tags, err
}

// CreateComment 创建评论
func (r *wikiRepositoryExtended) CreateComment(ctx context.Context, comment *model.Comment) error {
	comment.ID = idgen.NewULID()
	if comment.TenantID == "" {
		comment.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(comment).Error
}

// ListComments 列出文档的活跃评论
func (r *wikiRepositoryExtended) ListComments(ctx context.Context, documentID string) ([]*model.Comment, error) {
	var comments []*model.Comment
	err := r.getDB(ctx).Where("document_id = ? AND status = ?", documentID, model.CommentStatusActive).
		Order("created_at ASC").Find(&comments).Error
	return comments, err
}

// UpdateComment 更新评论
func (r *wikiRepositoryExtended) UpdateComment(ctx context.Context, comment *model.Comment) error {
	return r.getDB(ctx).Save(comment).Error
}

// DeleteComment 软删除评论（保留数据，避免父评论删除后子回复成为孤儿）
func (r *wikiRepositoryExtended) DeleteComment(ctx context.Context, id string) error {
	now := time.Now()
	return r.getDB(ctx).Model(&model.Comment{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     model.CommentStatusDeleted,
			"deleted_at": now,
		}).Error
}

// GetComment 根据ID获取评论
func (r *wikiRepositoryExtended) GetComment(ctx context.Context, id string) (*model.Comment, error) {
	var comment model.Comment
	err := r.getDB(ctx).Where("id = ?", id).First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// CreateShareLink 创建分享链接
func (r *wikiRepositoryExtended) CreateShareLink(ctx context.Context, link *model.ShareLink) error {
	link.ID = idgen.NewULID()
	if link.TenantID == "" {
		link.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(link).Error
}

// GetShareLinkByToken 根据分享令牌获取分享链接
func (r *wikiRepositoryExtended) GetShareLinkByToken(ctx context.Context, token string) (*model.ShareLink, error) {
	var link model.ShareLink
	err := r.getDB(ctx).Where("token = ?", token).First(&link).Error
	if err != nil {
		return nil, err
	}
	return &link, nil
}

// GetShareLinkByID 按主键查询分享链接。
func (r *wikiRepositoryExtended) GetShareLinkByID(ctx context.Context, id string) (*model.ShareLink, error) {
	var link model.ShareLink
	err := r.getDB(ctx).Where("id = ?", id).First(&link).Error
	if err != nil {
		return nil, err
	}
	return &link, nil
}

// ListShareLinks 列出指定文档的全部分享链接（按创建时间倒序）。
func (r *wikiRepositoryExtended) ListShareLinks(ctx context.Context, documentID string) ([]*model.ShareLink, error) {
	var links []*model.ShareLink
	err := r.getDB(ctx).Where("document_id = ?", documentID).Order("created_at DESC").Find(&links).Error
	return links, err
}

// UpdateShareLink 全量更新分享链接。
func (r *wikiRepositoryExtended) UpdateShareLink(ctx context.Context, link *model.ShareLink) error {
	return r.getDB(ctx).Save(link).Error
}

// DeleteShareLink 删除指定分享链接。
func (r *wikiRepositoryExtended) DeleteShareLink(ctx context.Context, id string) error {
	return r.getDB(ctx).Where("id = ?", id).Delete(&model.ShareLink{}).Error
}

// GetSharedDocument 按文档 ID+租户读取文档（免登录分享场景显式传租户，不依赖请求上下文）。
// 文档软删后查询自然返回未命中，分享随即失效。
func (r *wikiRepositoryExtended) GetSharedDocument(ctx context.Context, documentID, tenantID string) (*model.Document, error) {
	var doc model.Document
	if err := r.getDB(ctx).Where("id = ? AND tenant_id = ?", documentID, tenantID).First(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

// GetSharedNode 按节点 ID+租户读取节点（分享标题来源；节点软删后分享标题回退为空）
func (r *wikiRepositoryExtended) GetSharedNode(ctx context.Context, nodeID, tenantID string) (*model.WikiNode, error) {
	var node model.WikiNode
	if err := r.getDB(ctx).Where("id = ? AND tenant_id = ?", nodeID, tenantID).First(&node).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

// IncrementShareViewCount 原子条件自增：仅当未超过 max_views（max_views=0 表示不限次数）时 +1，
// 通过单条 UPDATE + RowsAffected 避免并发请求绕过次数上限
func (r *wikiRepositoryExtended) IncrementShareViewCount(ctx context.Context, id string, maxViews int) (bool, error) {
	res := r.getDB(ctx).Model(&model.ShareLink{}).
		Where("id = ? AND (max_views = 0 OR view_count < ?)", id, maxViews).
		UpdateColumn("view_count", gorm.Expr("view_count + 1"))
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// CreateTemplate 新建文档模板，自动补齐 ID 与租户ID。
func (r *wikiRepositoryExtended) CreateTemplate(ctx context.Context, template *model.DocumentTemplate) error {
	template.ID = idgen.NewULID()
	if template.TenantID == "" {
		template.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(template).Error
}

// ListTemplates 列出租户自有及公共模板，可按分类过滤；isPublic=false 时仅返回本租户模板。
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

// GetTemplate 按主键查询文档模板。
func (r *wikiRepositoryExtended) GetTemplate(ctx context.Context, id string) (*model.DocumentTemplate, error) {
	var template model.DocumentTemplate
	err := r.getDB(ctx).Where("id = ?", id).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// UpdateTemplate 全量更新文档模板。
func (r *wikiRepositoryExtended) UpdateTemplate(ctx context.Context, template *model.DocumentTemplate) error {
	return r.getDB(ctx).Save(template).Error
}

// DeleteTemplate 删除指定文档模板。
func (r *wikiRepositoryExtended) DeleteTemplate(ctx context.Context, id string) error {
	return r.getDB(ctx).Where("id = ?", id).Delete(&model.DocumentTemplate{}).Error
}

// CreateAccessLog 记录一条文档访问日志，自动补齐 ID 与租户ID。
func (r *wikiRepositoryExtended) CreateAccessLog(ctx context.Context, log *model.DocumentAccessLog) error {
	log.ID = idgen.NewULID()
	if log.TenantID == "" {
		log.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(log).Error
}

// ListAccessLogs 分页列出指定文档的访问日志及总数。
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

// GetDocumentStats 汇总文档的浏览/编辑/下载/分享次数及独立浏览者、最近浏览时间。
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

// CreateSubscription 新建文档订阅，自动补齐 ID 与租户ID。
func (r *wikiRepositoryExtended) CreateSubscription(ctx context.Context, sub *model.DocumentSubscription) error {
	sub.ID = idgen.NewULID()
	if sub.TenantID == "" {
		sub.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(sub).Error
}

// ListSubscriptions 列出指定文档的全部订阅者。
func (r *wikiRepositoryExtended) ListSubscriptions(ctx context.Context, documentID string) ([]*model.DocumentSubscription, error) {
	var subs []*model.DocumentSubscription
	err := r.getDB(ctx).Where("document_id = ?", documentID).Find(&subs).Error
	return subs, err
}

// DeleteSubscription 取消指定用户对文档的订阅。
func (r *wikiRepositoryExtended) DeleteSubscription(ctx context.Context, documentID, userID string) error {
	return r.getDB(ctx).Where("document_id = ? AND user_id = ?", documentID, userID).
		Delete(&model.DocumentSubscription{}).Error
}

// GetSubscription 查询指定用户对文档的订阅记录。
func (r *wikiRepositoryExtended) GetSubscription(ctx context.Context, documentID, userID string) (*model.DocumentSubscription, error) {
	var sub model.DocumentSubscription
	err := r.getDB(ctx).Where("document_id = ? AND user_id = ?", documentID, userID).First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// CreateNotification 创建站内通知，自动补齐 ID 与租户ID。
func (r *wikiRepositoryExtended) CreateNotification(ctx context.Context, notification *model.Notification) error {
	notification.ID = idgen.NewULID()
	if notification.TenantID == "" {
		notification.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(notification).Error
}

// ListNotifications 分页列出用户的全部通知及总数。
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

// MarkNotificationAsRead 将指定通知标记为已读（同时校验归属人，防越权）。
func (r *wikiRepositoryExtended) MarkNotificationAsRead(ctx context.Context, id, userID string) error {
	// 同时匹配 user_id，防止越权把他人通知标记为已读
	return r.getDB(ctx).Model(&model.Notification{}).Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true).Error
}

// MarkAllNotificationsAsRead 将用户全部未读通知标记为已读。
func (r *wikiRepositoryExtended) MarkAllNotificationsAsRead(ctx context.Context, userID string) error {
	return r.getDB(ctx).Model(&model.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}

// GetUnreadNotificationCount 统计用户未读通知数量。
func (r *wikiRepositoryExtended) GetUnreadNotificationCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.getDB(ctx).Model(&model.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&count).Error
	return count, err
}

// AcquireEditLock 获取文档编辑锁，默认 30 分钟过期，自动补齐 ID 与租户ID。
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

// ReleaseEditLock 释放并删除指定文档的编辑锁。
func (r *wikiRepositoryExtended) ReleaseEditLock(ctx context.Context, documentID string) error {
	return r.getDB(ctx).Where("document_id = ?", documentID).Delete(&model.EditLock{}).Error
}

// GetEditLock 无锁时返回 (nil, nil)，调用方将“无锁”视为可编辑
func (r *wikiRepositoryExtended) GetEditLock(ctx context.Context, documentID string) (*model.EditLock, error) {
	var lock model.EditLock
	err := r.getDB(ctx).Where("document_id = ?", documentID).First(&lock).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &lock, nil
}

// IsDocumentLocked 判断文档是否处于未过期的锁定状态。
func (r *wikiRepositoryExtended) IsDocumentLocked(ctx context.Context, documentID string) (bool, error) {
	var lock model.EditLock
	err := r.getDB(ctx).Where("document_id = ? AND expires_at > ?", documentID, time.Now()).First(&lock).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// RefreshEditLock 刷新文档编辑锁的过期时间（续期 30 分钟）。
func (r *wikiRepositoryExtended) RefreshEditLock(ctx context.Context, documentID string) error {
	return r.getDB(ctx).Model(&model.EditLock{}).Where("document_id = ?", documentID).
		Update("expires_at", time.Now().Add(30*time.Minute)).Error
}

// BatchDeleteNodes 批量软删除指定节点。
func (r *wikiRepositoryExtended) BatchDeleteNodes(ctx context.Context, nodeIDs []string) error {
	return r.getDB(ctx).Where("id IN ?", nodeIDs).Delete(&model.WikiNode{}).Error
}

// BatchMoveNodes 批量将指定节点移动到新的父节点下。
func (r *wikiRepositoryExtended) BatchMoveNodes(ctx context.Context, nodeIDs []string, newParentID string) error {
	return r.getDB(ctx).Model(&model.WikiNode{}).Where("id IN ?", nodeIDs).
		Update("parent_id", newParentID).Error
}

// CompareRevisions 取回同一文档两个版本的原始内容，供上层 diff 对比。
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

// ListNodesByIDs 批量按 ID 查询节点。
func (r *wikiRepositoryExtended) ListNodesByIDs(ctx context.Context, nodeIDs []string) ([]*model.WikiNode, error) {
	var nodes []*model.WikiNode
	err := r.getDB(ctx).Where("id IN ?", nodeIDs).Find(&nodes).Error
	return nodes, err
}

// ListDocumentsByNodeIDs 批量查询多个节点下的文档。
func (r *wikiRepositoryExtended) ListDocumentsByNodeIDs(ctx context.Context, nodeIDs []string) ([]*model.Document, error) {
	var docs []*model.Document
	err := r.getDB(ctx).Where("node_id IN ?", nodeIDs).Find(&docs).Error
	return docs, err
}

// GetNodeByDocumentID 根据文档 ID 反查其所属节点。
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

// ListDocumentTagsWithDetails 列出文档的标签关联并预加载标签详情。
func (r *wikiRepositoryExtended) ListDocumentTagsWithDetails(ctx context.Context, documentID string) ([]*model.DocumentTag, error) {
	var docTags []*model.DocumentTag
	err := r.getDB(ctx).Preload("Tag").Where("document_id = ?", documentID).Find(&docTags).Error
	return docTags, err
}

// ListCommentsByDocument 列出指定文档的活跃评论（按创建时间正序）。
func (r *wikiRepositoryExtended) ListCommentsByDocument(ctx context.Context, documentID string) ([]*model.Comment, error) {
	var comments []*model.Comment
	err := r.getDB(ctx).Where("document_id = ? AND status = ?", documentID, model.CommentStatusActive).
		Order("created_at ASC").Find(&comments).Error
	return comments, err
}

// ListRepliesByParentID 列出指定父评论下的活跃回复。
func (r *wikiRepositoryExtended) ListRepliesByParentID(ctx context.Context, parentID string) ([]*model.Comment, error) {
	var comments []*model.Comment
	err := r.getDB(ctx).Where("parent_id = ? AND status = ?", parentID, model.CommentStatusActive).
		Order("created_at ASC").Find(&comments).Error
	return comments, err
}

// CountDocumentsByTag 统计使用指定标签的文档数量。
func (r *wikiRepositoryExtended) CountDocumentsByTag(ctx context.Context, tagID string) (int64, error) {
	var count int64
	err := r.getDB(ctx).Model(&model.DocumentTag{}).Where("tag_id = ?", tagID).Count(&count).Error
	return count, err
}

// SearchDocumentsByTags 按标签集合检索租户下的文档（join 标签关联表）。
func (r *wikiRepositoryExtended) SearchDocumentsByTags(ctx context.Context, tenantID string, tagIDs []string) ([]*model.Document, error) {
	var docs []*model.Document
	query := r.getDB(ctx).Model(&model.Document{}).
		Joins("JOIN wiki_document_tags ON wiki_document_tags.document_id = wiki_documents.id").
		Where("wiki_documents.tenant_id = ?", tenantID).
		Where("wiki_document_tags.tag_id IN ?", tagIDs)
	err := query.Find(&docs).Error
	return docs, err
}

// ListTemplatesByCategory 按分类归组返回租户自有及公共模板。
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

// GetShareLinkStats 获取分享链接的当前浏览量。
func (r *wikiRepositoryExtended) GetShareLinkStats(ctx context.Context, linkID string) (int, error) {
	var link model.ShareLink
	if err := r.getDB(ctx).Where("id = ?", linkID).First(&link).Error; err != nil {
		return 0, err
	}
	return link.ViewCount, nil
}

// ListExpiredShareLinks 列出已过期的分享链接。
func (r *wikiRepositoryExtended) ListExpiredShareLinks(ctx context.Context) ([]*model.ShareLink, error) {
	var links []*model.ShareLink
	err := r.getDB(ctx).Where("expire_at IS NOT NULL AND expire_at < ?", time.Now()).Find(&links).Error
	return links, err
}

// DeleteExpiredShareLinks 物理删除已过期的分享链接。
func (r *wikiRepositoryExtended) DeleteExpiredShareLinks(ctx context.Context) error {
	return r.getDB(ctx).Where("expire_at IS NOT NULL AND expire_at < ?", time.Now()).Delete(&model.ShareLink{}).Error
}

// CleanExpiredEditLocks 清理已过期的编辑锁。
func (r *wikiRepositoryExtended) CleanExpiredEditLocks(ctx context.Context) error {
	return r.getDB(ctx).Where("expires_at < ?", time.Now()).Delete(&model.EditLock{}).Error
}

// ListLockedDocuments 列出当前未过期、处于锁定状态的编辑锁。
func (r *wikiRepositoryExtended) ListLockedDocuments(ctx context.Context) ([]*model.EditLock, error) {
	var locks []*model.EditLock
	err := r.getDB(ctx).Where("expires_at > ?", time.Now()).Find(&locks).Error
	return locks, err
}

// GetUserSubscriptions 列出用户订阅的全部文档。
func (r *wikiRepositoryExtended) GetUserSubscriptions(ctx context.Context, userID string) ([]*model.DocumentSubscription, error) {
	var subs []*model.DocumentSubscription
	err := r.getDB(ctx).Where("user_id = ?", userID).Find(&subs).Error
	return subs, err
}

// ListUnreadNotifications 分页列出用户未读通知及总数。
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

// GetDocumentByIDWithNode 按 ID 查询文档并连带返回其所属节点。
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

// GetDocumentStatsExtended 扩展统计文档的浏览/编辑/下载/分享及独立浏览者、最近浏览时间。
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

// GenerateDiff 逐行对比新旧内容，生成 unified 风格的差异文本。
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

// UpdateDocumentPublishStatus 更新文档发布状态，按需写入评审人/评审时间/发布时间，回退草稿时清空评审信息。
func (r *wikiRepositoryExtended) UpdateDocumentPublishStatus(ctx context.Context, documentID string, status string, reviewedBy string, reviewedAt *time.Time, publishedAt *time.Time) error {
	updates := map[string]interface{}{
		"publish_status": status,
	}
	if reviewedBy != "" {
		updates["reviewed_by"] = reviewedBy
	}
	if reviewedAt != nil {
		updates["reviewed_at"] = reviewedAt
	}
	if publishedAt != nil {
		updates["published_at"] = publishedAt
	}
	if status == model.PublishStatusDraft {
		updates["reviewed_by"] = ""
		updates["reviewed_at"] = nil
		updates["published_at"] = nil
	}
	return r.getDB(ctx).Model(&model.Document{}).Where("id = ?", documentID).Updates(updates).Error
}

// CreateReviewComment 创建评审评论，自动补齐 ID 与租户ID。
func (r *wikiRepositoryExtended) CreateReviewComment(ctx context.Context, comment *model.ReviewComment) error {
	comment.ID = idgen.NewULID()
	if comment.TenantID == "" {
		comment.TenantID = tenantctx.TenantID(ctx)
	}
	return r.getDB(ctx).Create(comment).Error
}

// ListReviewComments 列出指定文档的评审评论（按创建时间倒序）。
func (r *wikiRepositoryExtended) ListReviewComments(ctx context.Context, documentID string) ([]*model.ReviewComment, error) {
	var comments []*model.ReviewComment
	err := r.getDB(ctx).Where("document_id = ?", documentID).Order("created_at DESC").Find(&comments).Error
	return comments, err
}

// ListPendingReviewDocuments 分页查询租户下待评审/已驳回的文档及总数。
func (r *wikiRepositoryExtended) ListPendingReviewDocuments(ctx context.Context, tenantID string, page, pageSize int) ([]*model.Document, int64, error) {
	var docs []*model.Document
	var total int64

	query := r.getDB(ctx).Model(&model.Document{}).Where("tenant_id = ? AND publish_status IN ? AND deleted_at IS NULL", tenantID, []string{model.PublishStatusPendingReview, model.PublishStatusRejected})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&docs).Error; err != nil {
		return nil, 0, err
	}

	return docs, total, nil
}
