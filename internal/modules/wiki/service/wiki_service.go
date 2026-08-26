package service

import (
	"context"
	"errors"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"
	apperrors "meteorx/internal/pkg/apperrors"
	db "meteorx/internal/pkg/db"
	"meteorx/pkg/idgen"

	"gorm.io/gorm"
)

type WikiService interface {
	// WikiSpace
	CreateSpace(ctx context.Context, tenantID string, userID string, req *dto.CreateWikiSpaceReq) (*dto.WikiSpaceResp, error)
	GetSpace(ctx context.Context, id string, userID string) (*dto.WikiSpaceResp, error)
	ListSpaces(ctx context.Context, tenantID string, userID string, page, pageSize int) ([]*dto.WikiSpaceResp, int64, error)
	UpdateSpace(ctx context.Context, id string, tenantID string, req *dto.UpdateWikiSpaceReq) (*dto.WikiSpaceResp, error)
	DeleteSpace(ctx context.Context, id string, tenantID string) error

	// WikiNode
	CreateNode(ctx context.Context, spaceID string, userID string, req *dto.CreateWikiNodeReq) (*dto.WikiNodeResp, error)
	GetNode(ctx context.Context, id string) (*dto.WikiNodeResp, error)
	GetNodeTree(ctx context.Context, spaceID string) ([]*dto.WikiNodeTreeResp, error)
	UpdateNode(ctx context.Context, id string, req *dto.UpdateWikiNodeReq) (*dto.WikiNodeResp, error)
	DeleteNode(ctx context.Context, id string) error

	// Document
	CreateDocument(ctx context.Context, nodeID string, userID string, req *dto.CreateDocumentReq) (*dto.DocumentResp, error)
	GetDocument(ctx context.Context, nodeID string, userID string) (*dto.DocumentResp, error)
	UpdateDocument(ctx context.Context, id string, userID string, req *dto.UpdateDocumentReq) (*dto.DocumentResp, error)
	DeleteDocument(ctx context.Context, id string) error

	// Document Revision
	ListRevisions(ctx context.Context, documentID string) ([]*dto.DocumentRevisionResp, error)
	GetRevision(ctx context.Context, documentID string, version int) (*dto.DocumentRevisionResp, error)
	RestoreRevision(ctx context.Context, documentID string, version int, userID string) (*dto.DocumentResp, error)

	// Space Members
	AddMember(ctx context.Context, spaceID string, req *dto.WikiSpaceMemberReq) (*dto.WikiSpaceMemberResp, error)
	RemoveMember(ctx context.Context, spaceID, userID string) error
	ListMembers(ctx context.Context, spaceID string) ([]*dto.WikiSpaceMemberResp, error)

	// Node Permissions
	SetNodePermission(ctx context.Context, nodeID string, req *dto.SetNodePermissionReq) (*dto.NodePermissionResp, error)
	GetNodePermissions(ctx context.Context, nodeID string) ([]*dto.NodePermissionResp, error)
	RemoveNodePermission(ctx context.Context, nodeID, userID, permission string) error

	// Permission Check
	CheckSpacePermission(ctx context.Context, spaceID, userID, action string) error
	CheckNodePermission(ctx context.Context, nodeID, userID, action string) error
	CheckDocumentPermission(ctx context.Context, documentID, userID, action string) error
	GetEffectivePermissions(ctx context.Context, spaceID, userID string) map[string]bool

	// Stats
	GetStats(ctx context.Context, tenantID string) (*dto.WikiStatsResp, error)
}

type wikiService struct {
	repo repository.WikiRepository
	tx   *db.TxManager
}

func NewWikiService(repo repository.WikiRepository, tx *db.TxManager) WikiService {
	return &wikiService{repo: repo, tx: tx}
}

func (s *wikiService) CreateSpace(ctx context.Context, tenantID string, userID string, req *dto.CreateWikiSpaceReq) (*dto.WikiSpaceResp, error) {
	var resp *dto.WikiSpaceResp
	err := s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
		space := &model.WikiSpace{
			Name:        req.Name,
			Description: req.Description,
			Icon:        req.Icon,
			Visibility:  req.Visibility,
			TenantID:    tenantID,
			CreatedBy:   userID,
		}

		if err := s.repo.CreateSpace(txCtx, space); err != nil {
			return err
		}

		member := &model.WikiSpaceMember{
			SpaceID: space.ID,
			UserID:  userID,
			Role:    model.SpaceRoleOwner,
		}
		if err := s.repo.AddMember(txCtx, member); err != nil {
			return err
		}

		buildResp, err := s.buildSpaceResp(txCtx, space, userID)
		if err != nil {
			return err
		}
		resp = buildResp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *wikiService) GetSpace(ctx context.Context, id string, userID string) (*dto.WikiSpaceResp, error) {
	if err := s.CheckSpacePermission(ctx, id, userID, "space:read"); err != nil {
		return nil, err
	}

	space, err := s.repo.GetSpaceByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.buildSpaceResp(ctx, space, userID)
}

func (s *wikiService) ListSpaces(ctx context.Context, tenantID string, userID string, page, pageSize int) ([]*dto.WikiSpaceResp, int64, error) {
	spaces, total, err := s.repo.ListSpaces(ctx, tenantID, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	var resps []*dto.WikiSpaceResp
	for _, space := range spaces {
		resp, err := s.buildSpaceResp(ctx, space, userID)
		if err != nil {
			continue
		}
		resps = append(resps, resp)
	}
	return resps, total, nil
}

func (s *wikiService) UpdateSpace(ctx context.Context, id string, tenantID string, req *dto.UpdateWikiSpaceReq) (*dto.WikiSpaceResp, error) {
	if err := s.CheckSpacePermission(ctx, id, tenantID, "space:update"); err != nil {
		return nil, err
	}

	space, err := s.repo.GetSpaceByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		space.Name = req.Name
	}
	space.Description = req.Description
	space.Icon = req.Icon
	if req.Visibility != 0 {
		space.Visibility = req.Visibility
	}

	if err := s.repo.UpdateSpace(ctx, space); err != nil {
		return nil, err
	}

	return s.buildSpaceResp(ctx, space, "")
}

func (s *wikiService) DeleteSpace(ctx context.Context, id string, tenantID string) error {
	if err := s.CheckSpacePermission(ctx, id, tenantID, "space:delete"); err != nil {
		return err
	}

	return s.repo.DeleteSpace(ctx, id)
}

func (s *wikiService) buildSpaceResp(ctx context.Context, space *model.WikiSpace, userID string) (*dto.WikiSpaceResp, error) {
	resp := &dto.WikiSpaceResp{
		ID:          space.ID,
		TenantID:    space.TenantID,
		Name:        space.Name,
		Description: space.Description,
		Icon:        space.Icon,
		Visibility:  space.Visibility,
		CreatedBy:   space.CreatedBy,
		CreatedAt:   space.CreatedAt,
		UpdatedAt:   space.UpdatedAt,
	}

	members, err := s.repo.ListMembers(ctx, space.ID)
	if err == nil {
		resp.MemberCount = int64(len(members))
		for _, m := range members {
			if m.UserID == userID {
				resp.MyRole = m.Role
				break
			}
		}
	}

	nodes, _ := s.repo.ListNodesBySpace(ctx, space.ID)
	resp.NodeCount = int64(len(nodes))

	return resp, nil
}

func (s *wikiService) CreateNode(ctx context.Context, spaceID string, userID string, req *dto.CreateWikiNodeReq) (*dto.WikiNodeResp, error) {
	if err := s.CheckSpacePermission(ctx, spaceID, userID, "node:create"); err != nil {
		return nil, err
	}

	if req.Type == model.NodeTypeDocument {
		var resp *dto.WikiNodeResp
		err := s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
			var parentID string
			if req.ParentID != "" {
				parentID = req.ParentID
				parent, err := s.repo.GetNodeByID(txCtx, parentID)
				if err != nil {
					return err
				}
				if parent.Type != model.NodeTypeFolder {
					return errors.New("document parent must be a folder")
				}
			}

			node := &model.WikiNode{
				SpaceID:  spaceID,
				ParentID: parentID,
				Type:     model.NodeTypeDocument,
				Title:    req.Title,
				Icon:     req.Icon,
				Sort:     req.Sort,
				OwnerID:  userID,
				Status:   model.NodeStatusActive,
			}
			if err := s.repo.CreateNode(txCtx, node); err != nil {
				return err
			}

			doc := &model.Document{
				NodeID:  node.ID,
				Content: req.Content,
				Format:  "markdown",
			}
			if req.Content == "" {
				doc.Content = "# " + req.Title + "\n\n"
			}
			if err := s.repo.CreateDocument(txCtx, doc); err != nil {
				return err
			}

			buildResp, err := s.buildNodeResp(txCtx, node)
			if err != nil {
				return err
			}
			resp = buildResp
			return nil
		})
		if err != nil {
			return nil, err
		}
		return resp, nil
	}

	var parentID string
	if req.ParentID != "" {
		parentID = req.ParentID
		parent, err := s.repo.GetNodeByID(ctx, parentID)
		if err != nil {
			return nil, err
		}
		if parent.Type != model.NodeTypeFolder {
			return nil, errors.New("folder parent must be a folder")
		}
	}

	node := &model.WikiNode{
		SpaceID:  spaceID,
		ParentID: parentID,
		Type:     model.NodeTypeFolder,
		Title:    req.Title,
		Icon:     req.Icon,
		Sort:     req.Sort,
		OwnerID:  userID,
		Status:   model.NodeStatusActive,
	}
	if err := s.repo.CreateNode(ctx, node); err != nil {
		return nil, err
	}

	return s.buildNodeResp(ctx, node)
}

func (s *wikiService) GetNode(ctx context.Context, id string) (*dto.WikiNodeResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckNodePermission(ctx, id, userID, "read"); err != nil {
		return nil, err
	}

	node, err := s.repo.GetNodeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.buildNodeResp(ctx, node)
}

func (s *wikiService) GetNodeTree(ctx context.Context, spaceID string) ([]*dto.WikiNodeTreeResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckSpacePermission(ctx, spaceID, userID, "node:read"); err != nil {
		return nil, err
	}

	nodes, err := s.repo.ListNodesBySpace(ctx, spaceID)
	if err != nil {
		return nil, err
	}

	var roots []*dto.WikiNodeTreeResp
	for _, node := range nodes {
		root := &dto.WikiNodeTreeResp{}
		resp, err := s.buildNodeResp(ctx, node)
		if err != nil {
			continue
		}
		root.WikiNodeResp = *resp
		root.Children = s.buildNodeChildren(ctx, node.ID)
		roots = append(roots, root)
	}
	return roots, nil
}

func (s *wikiService) buildNodeChildren(ctx context.Context, parentID string) []*dto.WikiNodeTreeResp {
	nodes, err := s.repo.ListChildNodes(ctx, parentID)
	if err != nil {
		return nil
	}
	var children []*dto.WikiNodeTreeResp
	for _, node := range nodes {
		child := &dto.WikiNodeTreeResp{}
		resp, err := s.buildNodeResp(ctx, node)
		if err != nil {
			continue
		}
		child.WikiNodeResp = *resp
		child.Children = s.buildNodeChildren(ctx, node.ID)
		children = append(children, child)
	}
	return children
}

func (s *wikiService) buildNodeResp(ctx context.Context, node *model.WikiNode) (*dto.WikiNodeResp, error) {
	resp := &dto.WikiNodeResp{
		ID:        node.ID,
		SpaceID:   node.SpaceID,
		ParentID:  node.ParentID,
		Type:      node.Type,
		Title:     node.Title,
		Icon:      node.Icon,
		Sort:      node.Sort,
		Status:    node.Status,
		OwnerID:   node.OwnerID,
		CreatedAt: node.CreatedAt,
		UpdatedAt: node.UpdatedAt,
	}

	count, _ := s.repo.CountChildNodes(ctx, node.ID)
	resp.ChildCount = count
	return resp, nil
}

func (s *wikiService) UpdateNode(ctx context.Context, id string, req *dto.UpdateWikiNodeReq) (*dto.WikiNodeResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckNodePermission(ctx, id, userID, "update"); err != nil {
		return nil, err
	}

	node, err := s.repo.GetNodeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != "" {
		node.Title = req.Title
	}
	node.Icon = req.Icon
	if req.ParentID != "" {
		node.ParentID = req.ParentID
	}
	node.Sort = req.Sort
	if req.Status != 0 {
		node.Status = req.Status
	}

	if err := s.repo.UpdateNode(ctx, node); err != nil {
		return nil, err
	}
	return s.buildNodeResp(ctx, node)
}

func (s *wikiService) DeleteNode(ctx context.Context, id string) error {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckNodePermission(ctx, id, userID, "delete"); err != nil {
		return err
	}

	return s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
		return s.deleteNodeRecursive(txCtx, id)
	})
}

// deleteNodeRecursive 递归删除节点及其所有后代。
// 仅在被 WithTx 包裹的最外层调用，避免递归产生嵌套事务。
func (s *wikiService) deleteNodeRecursive(ctx context.Context, id string) error {
	children, _ := s.repo.ListChildNodes(ctx, id)
	for _, child := range children {
		if err := s.deleteNodeRecursive(ctx, child.ID); err != nil {
			return err
		}
	}

	node, err := s.repo.GetNodeByID(ctx, id)
	if err != nil {
		return err
	}

	if node.Type == model.NodeTypeDocument {
		doc, err := s.repo.GetDocumentByNodeID(ctx, id)
		if err == nil && doc != nil {
			_ = s.repo.DeleteDocument(ctx, doc.ID)
		}
	}

	return s.repo.DeleteNode(ctx, id)
}

func (s *wikiService) CreateDocument(ctx context.Context, nodeID string, userID string, req *dto.CreateDocumentReq) (*dto.DocumentResp, error) {
	if err := s.CheckNodePermission(ctx, nodeID, userID, "create"); err != nil {
		return nil, err
	}

	node, err := s.repo.GetNodeByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if node.Type != model.NodeTypeDocument {
		return nil, errors.New("node is not a document")
	}

	doc := &model.Document{
		NodeID:  nodeID,
		Content: req.Content,
		Format:  req.Format,
	}
	if doc.Format == "" {
		doc.Format = "markdown"
	}
	if err := s.repo.CreateDocument(ctx, doc); err != nil {
		return nil, err
	}

	return s.buildDocumentResp(ctx, node, doc)
}

func (s *wikiService) GetDocument(ctx context.Context, nodeID string, userID string) (*dto.DocumentResp, error) {
	if err := s.CheckNodePermission(ctx, nodeID, userID, "read"); err != nil {
		return nil, err
	}

	node, err := s.repo.GetNodeByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	doc, err := s.repo.GetDocumentByNodeID(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	_ = s.repo.IncrementViewCount(ctx, doc.ID)

	return s.buildDocumentResp(ctx, node, doc)
}

func (s *wikiService) UpdateDocument(ctx context.Context, id string, userID string, req *dto.UpdateDocumentReq) (*dto.DocumentResp, error) {
	if err := s.CheckDocumentPermission(ctx, id, userID, "update"); err != nil {
		return nil, err
	}

	var resp *dto.DocumentResp
	err := s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
		doc, err := s.repo.GetDocumentByID(txCtx, id)
		if err != nil {
			return err
		}
		expectedVer := doc.CurrentVer

		if req.Content != "" {
			revision := &model.DocumentRevision{
				DocumentID:  doc.ID,
				Version:     doc.CurrentVer,
				Content:     doc.Content,
				ContentHTML: doc.ContentHTML,
				Summary:     req.Summary,
				EditedBy:    doc.LastEditedBy,
			}
			if err := s.repo.CreateRevision(txCtx, revision); err != nil {
				return err
			}

			doc.Content = req.Content
			doc.CurrentVer++
			doc.LastEditedBy = userID
			now := time.Now()
			doc.LastEditedAt = &now
		}

		if req.Format != "" {
			doc.Format = req.Format
		}

		if err := s.repo.UpdateDocument(txCtx, doc, expectedVer); err != nil {
			if errors.Is(err, repository.ErrDocumentVersionConflict) {
				return apperrors.NewConflict("文档已被其他用户修改，请刷新后重试")
			}
			return err
		}

		node, err := s.repo.GetNodeByID(txCtx, doc.NodeID)
		if err != nil {
			return err
		}
		buildResp, err := s.buildDocumentResp(txCtx, node, doc)
		if err != nil {
			return err
		}
		resp = buildResp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *wikiService) DeleteDocument(ctx context.Context, id string) error {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, id, userID, "delete"); err != nil {
		return err
	}

	return s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
		doc, err := s.repo.GetDocumentByID(txCtx, id)
		if err != nil {
			return err
		}
		if err := s.deleteNodeRecursive(txCtx, doc.NodeID); err != nil {
			return err
		}
		return s.repo.DeleteDocument(txCtx, id)
	})
}

func (s *wikiService) buildDocumentResp(ctx context.Context, node *model.WikiNode, doc *model.Document) (*dto.DocumentResp, error) {
	title := ""
	if node != nil {
		title = node.Title
	}

	return &dto.DocumentResp{
		ID:           doc.ID,
		NodeID:       doc.NodeID,
		Title:        title,
		Content:      doc.Content,
		ContentHTML:  doc.ContentHTML,
		Format:       doc.Format,
		CurrentVer:   doc.CurrentVer,
		ViewCount:    doc.ViewCount,
		LastEditedBy: doc.LastEditedBy,
		LastEditedAt: doc.LastEditedAt,
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
	}, nil
}

func (s *wikiService) ListRevisions(ctx context.Context, documentID string) ([]*dto.DocumentRevisionResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "read"); err != nil {
		return nil, err
	}

	revisions, err := s.repo.ListRevisions(ctx, documentID)
	if err != nil {
		return nil, err
	}

	var resps []*dto.DocumentRevisionResp
	for _, r := range revisions {
		resps = append(resps, &dto.DocumentRevisionResp{
			ID:          r.ID,
			DocumentID:  r.DocumentID,
			Version:     r.Version,
			Content:     r.Content,
			ContentHTML: r.ContentHTML,
			Summary:     r.Summary,
			EditedBy:    r.EditedBy,
			CreatedAt:   r.CreatedAt,
		})
	}
	return resps, nil
}

func (s *wikiService) GetRevision(ctx context.Context, documentID string, version int) (*dto.DocumentRevisionResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "read"); err != nil {
		return nil, err
	}

	revision, err := s.repo.GetRevisionByVersion(ctx, documentID, version)
	if err != nil {
		return nil, err
	}
	return &dto.DocumentRevisionResp{
		ID:          revision.ID,
		DocumentID:  revision.DocumentID,
		Version:     revision.Version,
		Content:     revision.Content,
		ContentHTML: revision.ContentHTML,
		Summary:     revision.Summary,
		EditedBy:    revision.EditedBy,
		CreatedAt:   revision.CreatedAt,
	}, nil
}

func (s *wikiService) RestoreRevision(ctx context.Context, documentID string, version int, userID string) (*dto.DocumentResp, error) {
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "update"); err != nil {
		return nil, err
	}

	revision, err := s.repo.GetRevisionByVersion(ctx, documentID, version)
	if err != nil {
		return nil, err
	}

	doc, err := s.repo.GetDocumentByID(ctx, documentID)
	if err != nil {
		return nil, err
	}

	currentVer := doc.CurrentVer
	doc.Content = revision.Content
	doc.ContentHTML = revision.ContentHTML
	doc.CurrentVer = currentVer + 1
	doc.LastEditedBy = userID
	now := time.Now()
	doc.LastEditedAt = &now

	if err := s.repo.UpdateDocument(ctx, doc, currentVer); err != nil {
		if errors.Is(err, repository.ErrDocumentVersionConflict) {
			return nil, apperrors.NewConflict("文档已被其他用户修改，请刷新后重试")
		}
		return nil, err
	}

	node, _ := s.repo.GetNodeByID(ctx, doc.NodeID)
	return s.buildDocumentResp(ctx, node, doc)
}

func (s *wikiService) AddMember(ctx context.Context, spaceID string, req *dto.WikiSpaceMemberReq) (*dto.WikiSpaceMemberResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckSpacePermission(ctx, spaceID, userID, "member:manage"); err != nil {
		return nil, err
	}

	member := &model.WikiSpaceMember{
		SpaceID: spaceID,
		UserID:  req.UserID,
		Role:    req.Role,
	}
	if err := s.repo.AddMember(ctx, member); err != nil {
		return nil, err
	}

	return &dto.WikiSpaceMemberResp{
		ID:      member.ID,
		SpaceID: member.SpaceID,
		UserID:  member.UserID,
		Role:    member.Role,
	}, nil
}

func (s *wikiService) RemoveMember(ctx context.Context, spaceID, userID string) error {
	currentUserID := contextx.GetUserID(ctx)
	if err := s.CheckSpacePermission(ctx, spaceID, currentUserID, "member:manage"); err != nil {
		return err
	}

	return s.repo.RemoveMember(ctx, spaceID, userID)
}

func (s *wikiService) ListMembers(ctx context.Context, spaceID string) ([]*dto.WikiSpaceMemberResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckSpacePermission(ctx, spaceID, userID, "space:read"); err != nil {
		return nil, err
	}

	members, err := s.repo.ListMembers(ctx, spaceID)
	if err != nil {
		return nil, err
	}

	var resps []*dto.WikiSpaceMemberResp
	for _, m := range members {
		resps = append(resps, &dto.WikiSpaceMemberResp{
			ID:      m.ID,
			SpaceID: m.SpaceID,
			UserID:  m.UserID,
			Role:    m.Role,
		})
	}
	return resps, nil
}

func (s *wikiService) SetNodePermission(ctx context.Context, nodeID string, req *dto.SetNodePermissionReq) (*dto.NodePermissionResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckNodePermission(ctx, nodeID, userID, "update"); err != nil {
		return nil, err
	}

	existing, err := s.repo.GetNodePermission(ctx, nodeID, req.UserID, req.Permission)
	if err != nil {
		return nil, err
	}

	var perm *model.WikiNodePermission
	if existing != nil {
		existing.Permission = req.Permission
		if err := s.repo.UpdateNodePermission(ctx, existing); err != nil {
			return nil, err
		}
		perm = existing
	} else {
		perm = &model.WikiNodePermission{
			NodeID:     nodeID,
			UserID:     req.UserID,
			Permission: req.Permission,
		}
		if err := s.repo.CreateNodePermission(ctx, perm); err != nil {
			return nil, err
		}
	}

	return &dto.NodePermissionResp{
		ID:         perm.ID,
		NodeID:     perm.NodeID,
		UserID:     perm.UserID,
		Permission: perm.Permission,
		CreatedAt:  perm.CreatedAt,
		UpdatedAt:  perm.UpdatedAt,
	}, nil
}

func (s *wikiService) GetNodePermissions(ctx context.Context, nodeID string) ([]*dto.NodePermissionResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckNodePermission(ctx, nodeID, userID, "read"); err != nil {
		return nil, err
	}

	perms, err := s.repo.GetNodePermissions(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	var resps []*dto.NodePermissionResp
	for _, p := range perms {
		resps = append(resps, &dto.NodePermissionResp{
			ID:         p.ID,
			NodeID:     p.NodeID,
			UserID:     p.UserID,
			Permission: p.Permission,
			CreatedAt:  p.CreatedAt,
			UpdatedAt:  p.UpdatedAt,
		})
	}
	return resps, nil
}

func (s *wikiService) RemoveNodePermission(ctx context.Context, nodeID, userID, permission string) error {
	currentUserID := contextx.GetUserID(ctx)
	if err := s.CheckNodePermission(ctx, nodeID, currentUserID, "update"); err != nil {
		return err
	}

	perm, err := s.repo.GetNodePermission(ctx, nodeID, userID, permission)
	if err != nil {
		return err
	}
	if perm == nil {
		return apperrors.ErrNotFound("权限记录不存在")
	}

	return s.repo.DeleteNodePermission(ctx, perm.ID)
}

func (s *wikiService) GetStats(ctx context.Context, tenantID string) (*dto.WikiStatsResp, error) {
	stats, err := s.repo.GetWikiStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &dto.WikiStatsResp{
		TotalSpaces:    stats.TotalSpaces,
		TotalNodes:     stats.TotalNodes,
		TotalDocuments: stats.TotalDocuments,
		TotalViews:     stats.TotalViews,
	}, nil
}

func (s *wikiService) generateID() string {
	return idgen.New()
}