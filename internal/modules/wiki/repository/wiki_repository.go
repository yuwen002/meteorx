package repository

import (
	"context"
	"errors"
	"time"

	"meteorx/internal/common/tenantctx"
	"meteorx/internal/modules/wiki/model"
	db "meteorx/internal/pkg/db"
	"meteorx/pkg/idgen"

	"gorm.io/gorm"
)

var (
	ErrWikiSpaceNotFound       = errors.New("wiki space not found")
	ErrWikiNodeNotFound        = errors.New("wiki node not found")
	ErrDocumentNotFound        = errors.New("document not found")
	ErrRevisionNotFound        = errors.New("document revision not found")
	ErrWikiSpaceExist          = errors.New("wiki space already exists")
	ErrWikiNodeTypeInvalid     = errors.New("wiki node type invalid")
	ErrDocumentVersionConflict = errors.New("document version conflict")
)

type WikiRepository interface {
	CreateSpace(ctx context.Context, space *model.WikiSpace) error
	GetSpaceByID(ctx context.Context, id string) (*model.WikiSpace, error)
	ListSpaces(ctx context.Context, tenantID string, userID string, page, pageSize int) ([]*model.WikiSpace, int64, error)
	UpdateSpace(ctx context.Context, space *model.WikiSpace) error
	DeleteSpace(ctx context.Context, id string) error

	CreateNode(ctx context.Context, node *model.WikiNode) error
	GetNodeByID(ctx context.Context, id string) (*model.WikiNode, error)
	ListNodesBySpace(ctx context.Context, spaceID string) ([]*model.WikiNode, error)
	ListChildNodes(ctx context.Context, parentID string) ([]*model.WikiNode, error)
	UpdateNode(ctx context.Context, node *model.WikiNode) error
	DeleteNode(ctx context.Context, id string) error
	CountChildNodes(ctx context.Context, parentID string) (int64, error)

	CreateDocument(ctx context.Context, doc *model.Document) error
	GetDocumentByNodeID(ctx context.Context, nodeID string) (*model.Document, error)
	GetDocumentByID(ctx context.Context, id string) (*model.Document, error)
	// UpdateDocument 更新文档内容。expectedVer 为乐观锁期望的当前版本号，
	// 若数据库中 current_ver 与 expectedVer 不一致则返回 ErrDocumentVersionConflict。
	UpdateDocument(ctx context.Context, doc *model.Document, expectedVer int) error
	DeleteDocument(ctx context.Context, id string) error
	IncrementViewCount(ctx context.Context, id string) error

	CreateRevision(ctx context.Context, revision *model.DocumentRevision) error
	ListRevisions(ctx context.Context, documentID string) ([]*model.DocumentRevision, error)
	GetRevisionByVersion(ctx context.Context, documentID string, version int) (*model.DocumentRevision, error)

	AddMember(ctx context.Context, member *model.WikiSpaceMember) error
	RemoveMember(ctx context.Context, spaceID, userID string) error
	ListMembers(ctx context.Context, spaceID string) ([]*model.WikiSpaceMember, error)
	GetMember(ctx context.Context, spaceID, userID string) (*model.WikiSpaceMember, error)

	// Node Permission
	CreateNodePermission(ctx context.Context, perm *model.WikiNodePermission) error
	GetNodePermissions(ctx context.Context, nodeID string) ([]*model.WikiNodePermission, error)
	GetNodePermission(ctx context.Context, nodeID, userID, permission string) (*model.WikiNodePermission, error)
	UpdateNodePermission(ctx context.Context, perm *model.WikiNodePermission) error
	DeleteNodePermission(ctx context.Context, id string) error
	DeleteNodePermissionsByNode(ctx context.Context, nodeID string) error
	GetUserNodePermissions(ctx context.Context, nodeID, userID string) ([]*model.WikiNodePermission, error)

	GetWikiStats(ctx context.Context, tenantID string) (*model.WikiStats, error)
}

type wikiRepository struct {
	db *gorm.DB
}

func NewWikiRepository(database *gorm.DB) WikiRepository {
	return &wikiRepository{db: database}
}

// getDB 返回当前上下文对应的数据库连接。
// 若调用方处于事务中（通过 db.GetDB 从 ctx 取到 tx），则返回事务连接，
// 使所有查询自动加入事务；否则返回默认连接。
func (r *wikiRepository) getDB(ctx context.Context) *gorm.DB {
	return db.GetDB(ctx, r.db).WithContext(ctx)
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.WikiSpace{},
		&model.WikiNode{},
		&model.Document{},
		&model.DocumentRevision{},
		&model.WikiSpaceMember{},
		&model.WikiNodePermission{},
	)
}

func (r *wikiRepository) CreateSpace(ctx context.Context, space *model.WikiSpace) error {
	if space.ID == "" {
		space.ID = idgen.New()
	}
	if space.TenantID == "" {
		space.TenantID = tenantctx.TenantID(ctx)
	}
	space.CreatedAt = time.Now()
	space.UpdatedAt = time.Now()
	return r.getDB(ctx).Create(space).Error
}

func (r *wikiRepository) GetSpaceByID(ctx context.Context, id string) (*model.WikiSpace, error) {
	var space model.WikiSpace
	query := tenantctx.FilterQuery(ctx, r.getDB(ctx), "tenant_id")
	err := query.Where("id = ?", id).First(&space).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWikiSpaceNotFound
	}
	return &space, err
}

func (r *wikiRepository) ListSpaces(ctx context.Context, tenantID string, userID string, page, pageSize int) ([]*model.WikiSpace, int64, error) {
	var spaces []*model.WikiSpace
	var total int64

	query := r.getDB(ctx).Model(&model.WikiSpace{}).Where("tenant_id = ?", tenantID)

	if userID != "" {
		query = query.Joins("LEFT JOIN wiki_space_members wsm ON wsm.space_id = wiki_spaces.id AND wsm.user_id = ?", userID).
			Where("wiki_spaces.visibility = ? OR wsm.user_id IS NOT NULL", model.VisibilityTenant)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&spaces).Error; err != nil {
		return nil, 0, err
	}

	return spaces, total, nil
}

func (r *wikiRepository) UpdateSpace(ctx context.Context, space *model.WikiSpace) error {
	space.UpdatedAt = time.Now()
	result := r.getDB(ctx).Model(&model.WikiSpace{}).Where("id = ?", space.ID).Updates(map[string]interface{}{
		"name":        space.Name,
		"description": space.Description,
		"icon":        space.Icon,
		"visibility":  space.Visibility,
		"updated_at":  space.UpdatedAt,
	})
	if result.RowsAffected == 0 {
		return ErrWikiSpaceNotFound
	}
	return result.Error
}

func (r *wikiRepository) DeleteSpace(ctx context.Context, id string) error {
	result := r.getDB(ctx).Delete(&model.WikiSpace{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return ErrWikiSpaceNotFound
	}
	return result.Error
}

func (r *wikiRepository) CreateNode(ctx context.Context, node *model.WikiNode) error {
	if node.ID == "" {
		node.ID = idgen.New()
	}
	if node.TenantID == "" {
		node.TenantID = tenantctx.TenantID(ctx)
	}
	node.CreatedAt = time.Now()
	node.UpdatedAt = time.Now()
	return r.getDB(ctx).Create(node).Error
}

func (r *wikiRepository) GetNodeByID(ctx context.Context, id string) (*model.WikiNode, error) {
	var node model.WikiNode
	query := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").
		Model(&model.WikiNode{}).
		Where("id = ?", id)
	err := query.First(&node).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWikiNodeNotFound
	}
	return &node, err
}

func (r *wikiRepository) ListNodesBySpace(ctx context.Context, spaceID string) ([]*model.WikiNode, error) {
	var nodes []*model.WikiNode
	err := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").
		Where("space_id = ? AND parent_id = ''", spaceID).
		Order("sort ASC, created_at ASC").Find(&nodes).Error
	return nodes, err
}

func (r *wikiRepository) ListChildNodes(ctx context.Context, parentID string) ([]*model.WikiNode, error) {
	var nodes []*model.WikiNode
	err := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").
		Where("parent_id = ?", parentID).
		Order("sort ASC, created_at ASC").Find(&nodes).Error
	return nodes, err
}

func (r *wikiRepository) UpdateNode(ctx context.Context, node *model.WikiNode) error {
	node.UpdatedAt = time.Now()
	query := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").Model(&model.WikiNode{}).Where("id = ?", node.ID)
	result := query.Updates(map[string]interface{}{
		"title":      node.Title,
		"icon":       node.Icon,
		"parent_id":  node.ParentID,
		"sort":       node.Sort,
		"status":     node.Status,
		"updated_at": node.UpdatedAt,
	})
	if result.RowsAffected == 0 {
		return ErrWikiNodeNotFound
	}
	return result.Error
}

func (r *wikiRepository) DeleteNode(ctx context.Context, id string) error {
	query := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id")
	result := query.Delete(&model.WikiNode{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return ErrWikiNodeNotFound
	}
	return result.Error
}

func (r *wikiRepository) CountChildNodes(ctx context.Context, parentID string) (int64, error) {
	var count int64
	err := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").
		Model(&model.WikiNode{}).Where("parent_id = ?", parentID).Count(&count).Error
	return count, err
}

func (r *wikiRepository) CreateDocument(ctx context.Context, doc *model.Document) error {
	if doc.ID == "" {
		doc.ID = idgen.New()
	}
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()
	return r.getDB(ctx).Create(doc).Error
}

func (r *wikiRepository) GetDocumentByNodeID(ctx context.Context, nodeID string) (*model.Document, error) {
	var doc model.Document
	query := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").
		Model(&model.Document{}).
		Where("node_id = ?", nodeID)
	err := query.First(&doc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrDocumentNotFound
	}
	return &doc, err
}

func (r *wikiRepository) GetDocumentByID(ctx context.Context, id string) (*model.Document, error) {
	var doc model.Document
	query := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").
		Model(&model.Document{}).
		Where("id = ?", id)
	err := query.First(&doc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrDocumentNotFound
	}
	return &doc, err
}

func (r *wikiRepository) UpdateDocument(ctx context.Context, doc *model.Document, expectedVer int) error {
	doc.UpdatedAt = time.Now()
	now := time.Now()
	query := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").Model(&model.Document{}).
		Where("id = ? AND current_ver = ?", doc.ID, expectedVer)
	result := query.Updates(map[string]interface{}{
		"content":        doc.Content,
		"content_html":   doc.ContentHTML,
		"format":         doc.Format,
		"current_ver":    doc.CurrentVer,
		"last_edited_by": doc.LastEditedBy,
		"last_edited_at": &now,
		"updated_at":     doc.UpdatedAt,
	})
	if result.Error != nil {
		return result.Error
	}
	// RowsAffected == 0 可能是文档不存在，也可能是版本冲突
	if result.RowsAffected == 0 {
		var count int64
		exists := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").Model(&model.Document{}).
			Where("id = ?", doc.ID).Count(&count).Error
		if exists != nil || count == 0 {
			return ErrDocumentNotFound
		}
		return ErrDocumentVersionConflict
	}
	return nil
}

func (r *wikiRepository) DeleteDocument(ctx context.Context, id string) error {
	query := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id")
	result := query.Delete(&model.Document{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return ErrDocumentNotFound
	}
	return result.Error
}

func (r *wikiRepository) IncrementViewCount(ctx context.Context, id string) error {
	return tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").
		Model(&model.Document{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

func (r *wikiRepository) CreateRevision(ctx context.Context, revision *model.DocumentRevision) error {
	if revision.ID == "" {
		revision.ID = idgen.New()
	}
	if revision.TenantID == "" {
		revision.TenantID = tenantctx.TenantID(ctx)
	}
	revision.CreatedAt = time.Now()
	return r.getDB(ctx).Create(revision).Error
}

func (r *wikiRepository) ListRevisions(ctx context.Context, documentID string) ([]*model.DocumentRevision, error) {
	var revisions []*model.DocumentRevision
	err := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").
		Where("document_id = ?", documentID).Order("version DESC").Find(&revisions).Error
	return revisions, err
}

func (r *wikiRepository) GetRevisionByVersion(ctx context.Context, documentID string, version int) (*model.DocumentRevision, error) {
	var revision model.DocumentRevision
	err := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").
		Where("document_id = ? AND version = ?", documentID, version).First(&revision).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRevisionNotFound
	}
	return &revision, err
}

func (r *wikiRepository) AddMember(ctx context.Context, member *model.WikiSpaceMember) error {
	if member.ID == "" {
		member.ID = idgen.New()
	}
	if member.TenantID == "" {
		member.TenantID = tenantctx.TenantID(ctx)
	}
	member.CreatedAt = time.Now()
	member.UpdatedAt = time.Now()
	return r.getDB(ctx).Create(member).Error
}

func (r *wikiRepository) RemoveMember(ctx context.Context, spaceID, userID string) error {
	return tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").
		Where("space_id = ? AND user_id = ?", spaceID, userID).Delete(&model.WikiSpaceMember{}).Error
}

func (r *wikiRepository) ListMembers(ctx context.Context, spaceID string) ([]*model.WikiSpaceMember, error) {
	var members []*model.WikiSpaceMember
	err := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").
		Where("space_id = ?", spaceID).Order("created_at ASC").Find(&members).Error
	return members, err
}

func (r *wikiRepository) GetMember(ctx context.Context, spaceID, userID string) (*model.WikiSpaceMember, error) {
	var member model.WikiSpaceMember
	err := tenantctx.Scope(ctx, r.getDB(ctx), "tenant_id").
		Where("space_id = ? AND user_id = ?", spaceID, userID).First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &member, err
}

func (r *wikiRepository) GetWikiStats(ctx context.Context, tenantID string) (*model.WikiStats, error) {
	var stats model.WikiStats

	r.getDB(ctx).Model(&model.WikiSpace{}).Where("tenant_id = ?", tenantID).Count(&stats.TotalSpaces)

	var nodeCount int64
	r.getDB(ctx).Model(&model.WikiNode{}).Where("tenant_id = ?", tenantID).Count(&nodeCount)
	stats.TotalNodes = nodeCount

	var docCount int64
	r.getDB(ctx).Model(&model.Document{}).Where("tenant_id = ?", tenantID).Count(&docCount)
	stats.TotalDocuments = docCount

	var viewCount int64
	_ = r.getDB(ctx).Model(&model.Document{}).
		Where("tenant_id = ?", tenantID).
		Select("COALESCE(SUM(view_count), 0)").
		Scan(&viewCount).Error
	stats.TotalViews = viewCount

	return &stats, nil
}

// CreateNodePermission 创建节点权限
func (r *wikiRepository) CreateNodePermission(ctx context.Context, perm *model.WikiNodePermission) error {
	if perm.ID == "" {
		perm.ID = idgen.New()
	}
	perm.CreatedAt = time.Now()
	perm.UpdatedAt = time.Now()
	return r.getDB(ctx).Create(perm).Error
}

// GetNodePermissions 获取节点的所有权限
func (r *wikiRepository) GetNodePermissions(ctx context.Context, nodeID string) ([]*model.WikiNodePermission, error) {
	var perms []*model.WikiNodePermission
	err := r.getDB(ctx).Where("node_id = ?", nodeID).Find(&perms).Error
	return perms, err
}

// GetNodePermission 获取节点特定权限
func (r *wikiRepository) GetNodePermission(ctx context.Context, nodeID, userID, permission string) (*model.WikiNodePermission, error) {
	var perm model.WikiNodePermission
	err := r.getDB(ctx).Where("node_id = ? AND user_id = ? AND permission = ?", nodeID, userID, permission).First(&perm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &perm, err
}

// UpdateNodePermission 更新节点权限
func (r *wikiRepository) UpdateNodePermission(ctx context.Context, perm *model.WikiNodePermission) error {
	perm.UpdatedAt = time.Now()
	result := r.getDB(ctx).Model(&model.WikiNodePermission{}).Where("id = ?", perm.ID).Updates(map[string]interface{}{
		"permission": perm.Permission,
		"updated_at": perm.UpdatedAt,
	})
	if result.RowsAffected == 0 {
		return ErrWikiNodeNotFound
	}
	return result.Error
}

// DeleteNodePermission 删除单个节点权限
func (r *wikiRepository) DeleteNodePermission(ctx context.Context, id string) error {
	return r.getDB(ctx).Delete(&model.WikiNodePermission{}, "id = ?", id).Error
}

// DeleteNodePermissionsByNode 删除节点的所有权限
func (r *wikiRepository) DeleteNodePermissionsByNode(ctx context.Context, nodeID string) error {
	return r.getDB(ctx).Where("node_id = ?", nodeID).Delete(&model.WikiNodePermission{}).Error
}

// GetUserNodePermissions 获取用户在节点上的所有权限
func (r *wikiRepository) GetUserNodePermissions(ctx context.Context, nodeID, userID string) ([]*model.WikiNodePermission, error) {
	var perms []*model.WikiNodePermission
	err := r.getDB(ctx).Where("node_id = ? AND user_id = ?", nodeID, userID).Find(&perms).Error
	return perms, err
}