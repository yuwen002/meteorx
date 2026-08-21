package repository

import (
	"context"
	"errors"
	"time"

	"meteorx/internal/common/tenantctx"
	"meteorx/internal/modules/wiki/model"
	"meteorx/pkg/ulid"

	"gorm.io/gorm"
)

var (
	ErrWikiSpaceNotFound   = errors.New("wiki space not found")
	ErrWikiNodeNotFound    = errors.New("wiki node not found")
	ErrDocumentNotFound    = errors.New("document not found")
	ErrRevisionNotFound    = errors.New("document revision not found")
	ErrWikiSpaceExist      = errors.New("wiki space already exists")
	ErrWikiNodeTypeInvalid = errors.New("wiki node type invalid")
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
	UpdateDocument(ctx context.Context, doc *model.Document) error
	DeleteDocument(ctx context.Context, id string) error
	IncrementViewCount(ctx context.Context, id string) error

	CreateRevision(ctx context.Context, revision *model.DocumentRevision) error
	ListRevisions(ctx context.Context, documentID string) ([]*model.DocumentRevision, error)
	GetRevisionByVersion(ctx context.Context, documentID string, version int) (*model.DocumentRevision, error)

	AddMember(ctx context.Context, member *model.WikiSpaceMember) error
	RemoveMember(ctx context.Context, spaceID, userID string) error
	ListMembers(ctx context.Context, spaceID string) ([]*model.WikiSpaceMember, error)
	GetMember(ctx context.Context, spaceID, userID string) (*model.WikiSpaceMember, error)

	GetWikiStats(ctx context.Context, tenantID string) (*model.WikiStats, error)
}

type wikiRepository struct {
	db *gorm.DB
}

func NewWikiRepository(db *gorm.DB) WikiRepository {
	return &wikiRepository{db: db}
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
		space.ID = ulid.Generate()
	}
	space.CreatedAt = time.Now()
	space.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(space).Error
}

func (r *wikiRepository) GetSpaceByID(ctx context.Context, id string) (*model.WikiSpace, error) {
	var space model.WikiSpace
	query := tenantctx.FilterQuery(ctx, r.db.WithContext(ctx), "tenant_id")
	err := query.Where("id = ?", id).First(&space).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWikiSpaceNotFound
	}
	return &space, err
}

func (r *wikiRepository) ListSpaces(ctx context.Context, tenantID string, userID string, page, pageSize int) ([]*model.WikiSpace, int64, error) {
	var spaces []*model.WikiSpace
	var total int64

	query := r.db.WithContext(ctx).Model(&model.WikiSpace{}).Where("tenant_id = ?", tenantID)

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
	result := r.db.WithContext(ctx).Model(&model.WikiSpace{}).Where("id = ?", space.ID).Updates(map[string]interface{}{
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
	result := r.db.WithContext(ctx).Delete(&model.WikiSpace{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return ErrWikiSpaceNotFound
	}
	return result.Error
}

func (r *wikiRepository) CreateNode(ctx context.Context, node *model.WikiNode) error {
	if node.ID == "" {
		node.ID = ulid.Generate()
	}
	node.CreatedAt = time.Now()
	node.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(node).Error
}

func (r *wikiRepository) GetNodeByID(ctx context.Context, id string) (*model.WikiNode, error) {
	var node model.WikiNode
	query := r.db.WithContext(ctx).Model(&model.WikiNode{}).
		Joins("JOIN wiki_spaces ws ON ws.id = wiki_nodes.space_id").
		Where("wiki_nodes.id = ?", id)
	query = tenantctx.FilterQuery(ctx, query, "ws.tenant_id")
	err := query.First(&node).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWikiNodeNotFound
	}
	return &node, err
}

func (r *wikiRepository) ListNodesBySpace(ctx context.Context, spaceID string) ([]*model.WikiNode, error) {
	var nodes []*model.WikiNode
	err := r.db.WithContext(ctx).Where("space_id = ? AND parent_id = ''", spaceID).Order("sort ASC, created_at ASC").Find(&nodes).Error
	return nodes, err
}

func (r *wikiRepository) ListChildNodes(ctx context.Context, parentID string) ([]*model.WikiNode, error) {
	var nodes []*model.WikiNode
	err := r.db.WithContext(ctx).Where("parent_id = ?", parentID).Order("sort ASC, created_at ASC").Find(&nodes).Error
	return nodes, err
}

func (r *wikiRepository) UpdateNode(ctx context.Context, node *model.WikiNode) error {
	node.UpdatedAt = time.Now()
	result := r.db.WithContext(ctx).Model(&model.WikiNode{}).Where("id = ?", node.ID).Updates(map[string]interface{}{
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
	result := r.db.WithContext(ctx).Delete(&model.WikiNode{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return ErrWikiNodeNotFound
	}
	return result.Error
}

func (r *wikiRepository) CountChildNodes(ctx context.Context, parentID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.WikiNode{}).Where("parent_id = ?", parentID).Count(&count).Error
	return count, err
}

func (r *wikiRepository) CreateDocument(ctx context.Context, doc *model.Document) error {
	if doc.ID == "" {
		doc.ID = ulid.Generate()
	}
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(doc).Error
}

func (r *wikiRepository) GetDocumentByNodeID(ctx context.Context, nodeID string) (*model.Document, error) {
	var doc model.Document
	query := r.db.WithContext(ctx).Model(&model.Document{}).
		Joins("JOIN wiki_nodes wn ON wn.id = wiki_documents.node_id").
		Joins("JOIN wiki_spaces ws ON ws.id = wn.space_id").
		Where("wiki_documents.node_id = ?", nodeID)
	query = tenantctx.FilterQuery(ctx, query, "ws.tenant_id")
	err := query.First(&doc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrDocumentNotFound
	}
	return &doc, err
}

func (r *wikiRepository) GetDocumentByID(ctx context.Context, id string) (*model.Document, error) {
	var doc model.Document
	query := r.db.WithContext(ctx).Model(&model.Document{}).
		Joins("JOIN wiki_nodes wn ON wn.id = wiki_documents.node_id").
		Joins("JOIN wiki_spaces ws ON ws.id = wn.space_id").
		Where("wiki_documents.id = ?", id)
	query = tenantctx.FilterQuery(ctx, query, "ws.tenant_id")
	err := query.First(&doc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrDocumentNotFound
	}
	return &doc, err
}

func (r *wikiRepository) UpdateDocument(ctx context.Context, doc *model.Document) error {
	doc.UpdatedAt = time.Now()
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&model.Document{}).Where("id = ?", doc.ID).Updates(map[string]interface{}{
		"content":        doc.Content,
		"content_html":   doc.ContentHTML,
		"format":         doc.Format,
		"current_ver":    doc.CurrentVer,
		"last_edited_by": doc.LastEditedBy,
		"last_edited_at": &now,
		"updated_at":     doc.UpdatedAt,
	})
	if result.RowsAffected == 0 {
		return ErrDocumentNotFound
	}
	return result.Error
}

func (r *wikiRepository) DeleteDocument(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&model.Document{}, "id = ?", id)
	if result.RowsAffected == 0 {
		return ErrDocumentNotFound
	}
	return result.Error
}

func (r *wikiRepository) IncrementViewCount(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&model.Document{}).Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

func (r *wikiRepository) CreateRevision(ctx context.Context, revision *model.DocumentRevision) error {
	if revision.ID == "" {
		revision.ID = ulid.Generate()
	}
	revision.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(revision).Error
}

func (r *wikiRepository) ListRevisions(ctx context.Context, documentID string) ([]*model.DocumentRevision, error) {
	var revisions []*model.DocumentRevision
	err := r.db.WithContext(ctx).Where("document_id = ?", documentID).Order("version DESC").Find(&revisions).Error
	return revisions, err
}

func (r *wikiRepository) GetRevisionByVersion(ctx context.Context, documentID string, version int) (*model.DocumentRevision, error) {
	var revision model.DocumentRevision
	err := r.db.WithContext(ctx).Where("document_id = ? AND version = ?", documentID, version).First(&revision).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRevisionNotFound
	}
	return &revision, err
}

func (r *wikiRepository) AddMember(ctx context.Context, member *model.WikiSpaceMember) error {
	if member.ID == "" {
		member.ID = ulid.Generate()
	}
	member.CreatedAt = time.Now()
	member.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(member).Error
}

func (r *wikiRepository) RemoveMember(ctx context.Context, spaceID, userID string) error {
	return r.db.WithContext(ctx).Where("space_id = ? AND user_id = ?", spaceID, userID).Delete(&model.WikiSpaceMember{}).Error
}

func (r *wikiRepository) ListMembers(ctx context.Context, spaceID string) ([]*model.WikiSpaceMember, error) {
	var members []*model.WikiSpaceMember
	err := r.db.WithContext(ctx).Where("space_id = ?", spaceID).Order("created_at ASC").Find(&members).Error
	return members, err
}

func (r *wikiRepository) GetMember(ctx context.Context, spaceID, userID string) (*model.WikiSpaceMember, error) {
	var member model.WikiSpaceMember
	err := r.db.WithContext(ctx).Where("space_id = ? AND user_id = ?", spaceID, userID).First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &member, err
}

func (r *wikiRepository) GetWikiStats(ctx context.Context, tenantID string) (*model.WikiStats, error) {
	var stats model.WikiStats

	r.db.WithContext(ctx).Model(&model.WikiSpace{}).Where("tenant_id = ?", tenantID).Count(&stats.TotalSpaces)

	var nodeCount int64
	r.db.WithContext(ctx).Model(&model.WikiNode{}).
		Joins("JOIN wiki_spaces ws ON ws.id = wiki_nodes.space_id").
		Where("ws.tenant_id = ?", tenantID).
		Count(&nodeCount)
	stats.TotalNodes = nodeCount

	var docCount int64
	r.db.WithContext(ctx).Model(&model.Document{}).
		Joins("JOIN wiki_nodes wn ON wn.id = wiki_documents.node_id").
		Joins("JOIN wiki_spaces ws ON ws.id = wn.space_id").
		Where("ws.tenant_id = ?", tenantID).
		Count(&docCount)
	stats.TotalDocuments = docCount

	var viewCount int64
	_ = r.db.WithContext(ctx).Model(&model.Document{}).
		Joins("JOIN wiki_nodes wn ON wn.id = wiki_documents.node_id").
		Joins("JOIN wiki_spaces ws ON ws.id = wn.space_id").
		Where("ws.tenant_id = ?", tenantID).
		Select("COALESCE(SUM(view_count), 0)").
		Scan(&viewCount).Error
	stats.TotalViews = viewCount

	return &stats, nil
}