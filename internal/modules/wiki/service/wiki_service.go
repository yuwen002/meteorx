package service

import (
	"context"
	"errors"
	"sort"
	"strings"
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

// WikiService Wiki 模块服务接口
type WikiService interface {
	// ========== WikiSpace ==========
	// CreateSpace 创建 Wiki Space，并将创建者自动添加为 Owner
	CreateSpace(ctx context.Context, tenantID string, userID string, req *dto.CreateWikiSpaceReq) (*dto.WikiSpaceResp, error)
	// GetSpace 获取 Space 详情
	GetSpace(ctx context.Context, id string, userID string) (*dto.WikiSpaceResp, error)
	// ListSpaces 列出用户可访问的 Spaces（分页）
	ListSpaces(ctx context.Context, tenantID string, userID string, page, pageSize int) ([]*dto.WikiSpaceResp, int64, error)
	// UpdateSpace 更新 Space 信息
	UpdateSpace(ctx context.Context, id string, tenantID string, req *dto.UpdateWikiSpaceReq) (*dto.WikiSpaceResp, error)
	// DeleteSpace 删除 Space（进入回收站）
	DeleteSpace(ctx context.Context, id string, tenantID string) error

	// ========== WikiNode ==========
	// CreateNode 创建节点（文件夹或文档）
	CreateNode(ctx context.Context, spaceID string, userID string, req *dto.CreateWikiNodeReq) (*dto.WikiNodeResp, error)
	// GetNode 获取节点详情
	GetNode(ctx context.Context, id string) (*dto.WikiNodeResp, error)
	// GetNodeTree 获取 Space 下的完整节点树（已排序）
	GetNodeTree(ctx context.Context, spaceID string) ([]*dto.WikiNodeTreeResp, error)
	// UpdateNode 更新节点（支持移动、重命名、排序）
	UpdateNode(ctx context.Context, id string, req *dto.UpdateWikiNodeReq) (*dto.WikiNodeResp, error)
	// DeleteNode 删除节点及其子树（进入回收站 + 级联删除）
	DeleteNode(ctx context.Context, id string) error
	// MoveNode 移动节点到新父节点（含安全校验）
	MoveNode(ctx context.Context, id string, newParentID string, userID string) error
	// SortNode 设置节点排序值
	SortNode(ctx context.Context, id string, sort int, userID string) error

	// ========== Document ==========
	// CreateDocument 创建文档（支持 Markdown 渲染）
	CreateDocument(ctx context.Context, nodeID string, userID string, req *dto.CreateDocumentReq) (*dto.DocumentResp, error)
	// GetDocument 获取文档内容
	GetDocument(ctx context.Context, nodeID string, userID string) (*dto.DocumentResp, error)
	// UpdateDocument 更新文档（事务保证 + Revision 自动创建）
	UpdateDocument(ctx context.Context, id string, userID string, req *dto.UpdateDocumentReq) (*dto.DocumentResp, error)
	// DeleteDocument 删除文档（进入回收站 + 级联删除附件）
	DeleteDocument(ctx context.Context, id string) error

	// ========== Document Revision ==========
	// ListRevisions 列出文档的所有历史版本
	ListRevisions(ctx context.Context, documentID string) ([]*dto.DocumentRevisionResp, error)
	// GetRevision 获取指定版本的历史内容
	GetRevision(ctx context.Context, documentID string, version int) (*dto.DocumentRevisionResp, error)
	// RestoreRevision 恢复到指定版本（自动保存当前版本为新 Revision）
	RestoreRevision(ctx context.Context, documentID string, version int, userID string) (*dto.DocumentResp, error)

	// ========== Space Members ==========
	// AddMember 添加 Space 成员
	AddMember(ctx context.Context, spaceID string, req *dto.WikiSpaceMemberReq) (*dto.WikiSpaceMemberResp, error)
	// RemoveMember 移除成员
	RemoveMember(ctx context.Context, spaceID, userID string) error
	// ListMembers 列出 Space 成员列表
	ListMembers(ctx context.Context, spaceID string) ([]*dto.WikiSpaceMemberResp, error)

	// ========== Node Permissions ==========
	// SetNodePermission 设置节点级别权限
	SetNodePermission(ctx context.Context, nodeID string, req *dto.SetNodePermissionReq) (*dto.NodePermissionResp, error)
	// GetNodePermissions 获取节点的所有权限配置
	GetNodePermissions(ctx context.Context, nodeID string) ([]*dto.NodePermissionResp, error)
	// RemoveNodePermission 移除节点级别权限
	RemoveNodePermission(ctx context.Context, nodeID, userID, permission string) error

	// ========== Permission Check ==========
	// CheckSpacePermission 检查 Space 级别权限
	CheckSpacePermission(ctx context.Context, spaceID, userID, action string) error
	// CheckNodePermission 检查 Node 级别权限（含继承 + Node 补充授权）
	CheckNodePermission(ctx context.Context, nodeID, userID, action string) error
	// CheckDocumentPermission 检查 Document 权限（基于 Node）
	CheckDocumentPermission(ctx context.Context, documentID, userID, action string) error
	// GetEffectivePermissions 获取用户在 Space 中的有效权限映射
	GetEffectivePermissions(ctx context.Context, spaceID, userID string) map[string]bool

	// ========== Trash ==========
	// ListTrashItems 列出回收站项目（分页）
	ListTrashItems(ctx context.Context, tenantID string, spaceID string, itemType string, page, pageSize int) ([]*dto.TrashItemResp, int64, error)
	// RestoreTrashItem 从回收站恢复项目
	RestoreTrashItem(ctx context.Context, id string, userID string) error
	// PermanentDeleteTrashItem 永久删除回收站项目
	PermanentDeleteTrashItem(ctx context.Context, id string, userID string) error

	// ========== Search ==========
	// Search 搜索 Wiki（标题 + 内容，自动过滤无权限内容）
	Search(ctx context.Context, tenantID string, userID string, query string, spaceID string, page, pageSize int) ([]*dto.SearchResultResp, int64, error)

	// ========== Attachment ==========
	// CreateAttachment 创建文档附件
	CreateAttachment(ctx context.Context, userID string, req *dto.CreateAttachmentReq) (*dto.AttachmentResp, error)
	// ListAttachments 列出文档的所有附件
	ListAttachments(ctx context.Context, documentID string, userID string) ([]*dto.AttachmentResp, error)
	// DeleteAttachment 删除附件
	DeleteAttachment(ctx context.Context, id string, userID string) error

	// ========== Stats ==========
	// GetStats 获取 Wiki 统计数据
	GetStats(ctx context.Context, tenantID string) (*dto.WikiStatsResp, error)
}

// wikiService Wiki 服务实现
type wikiService struct {
	repo        repository.WikiRepository
	tx          *db.TxManager
	markdownSvc MarkdownService
}

// NewWikiService 创建 Wiki Service 实例
func NewWikiService(repo repository.WikiRepository, tx *db.TxManager) WikiService {
	return &wikiService{
		repo:        repo,
		tx:          tx,
		markdownSvc: NewMarkdownService(),
	}
}

// CreateSpace 创建 Wiki Space，并将创建者自动添加为 Owner
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

// GetSpace 获取 Space 详情，需有 read 权限
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

// ListSpaces 列出用户可访问的所有 Spaces（分页）
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

// UpdateSpace 更新 Space 信息（名称、描述、可见性等）
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

// DeleteSpace 删除 Space，先创建回收站记录再物理删除
func (s *wikiService) DeleteSpace(ctx context.Context, id string, tenantID string) error {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckSpacePermission(ctx, id, userID, "space:delete"); err != nil {
		return err
	}

	space, err := s.repo.GetSpaceByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.MoveToTrash(ctx, model.TrashTypeSpace, id, id, space.Name, userID); err != nil {
		return err
	}

	return s.repo.DeleteSpace(ctx, id)
}

// buildSpaceResp 构建 Space 响应结构（包含成员数、节点数、当前用户角色）
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

// CreateNode 创建节点（文件夹或文档），文档类型会同时创建 Document 记录
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

// GetNode 获取节点详情，需有 read 权限
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

// GetNodeTree 获取 Space 下的完整节点树，按 Sort 排序
func (s *wikiService) GetNodeTree(ctx context.Context, spaceID string) ([]*dto.WikiNodeTreeResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckSpacePermission(ctx, spaceID, userID, "node:read"); err != nil {
		return nil, err
	}

	nodes, err := s.repo.ListNodesBySpace(ctx, spaceID)
	if err != nil {
		return nil, err
	}

	return s.buildTreeFromNodes(ctx, nodes), nil
}

// buildTreeFromNodes 将扁平节点列表构建为树形结构，按 Sort 排序
func (s *wikiService) buildTreeFromNodes(ctx context.Context, nodes []*model.WikiNode) []*dto.WikiNodeTreeResp {
	nodeMap := make(map[string]*dto.WikiNodeTreeResp)
	var roots []*dto.WikiNodeTreeResp

	for _, node := range nodes {
		resp, err := s.buildNodeResp(ctx, node)
		if err != nil {
			continue
		}
		treeNode := &dto.WikiNodeTreeResp{
			WikiNodeResp: *resp,
			Children:     []*dto.WikiNodeTreeResp{},
		}
		nodeMap[node.ID] = treeNode
	}

	for _, node := range nodes {
		treeNode, exists := nodeMap[node.ID]
		if !exists {
			continue
		}
		if node.ParentID == "" {
			roots = append(roots, treeNode)
		} else if parent, ok := nodeMap[node.ParentID]; ok {
			parent.Children = append(parent.Children, treeNode)
		} else {
			roots = append(roots, treeNode)
		}
	}

	sortTreeNodes(roots)
	return roots
}

// sortTreeNodes 递归排序节点树（按 Sort 字段）
func sortTreeNodes(nodes []*dto.WikiNodeTreeResp) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Sort < nodes[j].Sort
	})
	for _, node := range nodes {
		sortTreeNodes(node.Children)
	}
}

// buildNodeResp 构建节点响应结构（包含子节点数量）
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

// UpdateNode 更新节点信息（标题、图标、排序、状态、父节点等）
func (s *wikiService) UpdateNode(ctx context.Context, id string, req *dto.UpdateWikiNodeReq) (*dto.WikiNodeResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckNodePermission(ctx, id, userID, "update"); err != nil {
		return nil, err
	}

	node, err := s.repo.GetNodeByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.ParentID != "" {
		if err := s.validateMoveNode(ctx, node.ID, node.SpaceID, req.ParentID); err != nil {
			return nil, err
		}
		node.ParentID = req.ParentID
	}

	if req.Title != "" {
		node.Title = req.Title
	}
	node.Icon = req.Icon
	node.Sort = req.Sort
	if req.Status != 0 {
		node.Status = req.Status
	}

	if err := s.repo.UpdateNode(ctx, node); err != nil {
		return nil, err
	}
	return s.buildNodeResp(ctx, node)
}

// MoveNode 移动节点到新父节点，包含移动安全校验
func (s *wikiService) MoveNode(ctx context.Context, id string, newParentID string, userID string) error {
	if err := s.CheckNodePermission(ctx, id, userID, "update"); err != nil {
		return err
	}

	node, err := s.repo.GetNodeByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.validateMoveNode(ctx, id, node.SpaceID, newParentID); err != nil {
		return err
	}

	node.ParentID = newParentID
	return s.repo.UpdateNode(ctx, node)
}

// SortNode 设置节点排序值
func (s *wikiService) SortNode(ctx context.Context, id string, sort int, userID string) error {
	if err := s.CheckNodePermission(ctx, id, userID, "update"); err != nil {
		return err
	}

	node, err := s.repo.GetNodeByID(ctx, id)
	if err != nil {
		return err
	}

	node.Sort = sort
	return s.repo.UpdateNode(ctx, node)
}

// validateMoveNode 校验节点移动的安全性（防止循环、跨 Space、目标非文件夹）
func (s *wikiService) validateMoveNode(ctx context.Context, nodeID, nodeSpaceID, newParentID string) error {
	if newParentID == nodeID {
		return apperrors.ErrBadRequest("不能将节点移动到自身")
	}

	parent, err := s.repo.GetNodeByID(ctx, newParentID)
	if err != nil {
		return apperrors.ErrBadRequest("目标父节点不存在")
	}

	if parent.SpaceID != nodeSpaceID {
		return apperrors.ErrBadRequest("不能跨 Space 移动节点")
	}

	if parent.Type != model.NodeTypeFolder {
		return apperrors.ErrBadRequest("目标父节点必须是文件夹")
	}

	if s.isDescendant(ctx, nodeID, newParentID) {
		return apperrors.ErrBadRequest("不能将节点移动到其子节点下")
	}

	return nil
}

// isDescendant 检查 ancestorID 是否是 nodeID 的后代
func (s *wikiService) isDescendant(ctx context.Context, ancestorID, nodeID string) bool {
	children, err := s.repo.ListChildNodes(ctx, nodeID)
	if err != nil {
		return false
	}
	for _, child := range children {
		if child.ID == ancestorID {
			return true
		}
		if s.isDescendant(ctx, ancestorID, child.ID) {
			return true
		}
	}
	return false
}

// DeleteNode 删除节点及其子树，先创建回收站记录再级联物理删除
func (s *wikiService) DeleteNode(ctx context.Context, id string) error {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckNodePermission(ctx, id, userID, "delete"); err != nil {
		return err
	}

	node, err := s.repo.GetNodeByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.MoveToTrash(ctx, model.TrashTypeNode, id, node.SpaceID, node.Title, userID); err != nil {
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
			_ = s.repo.DeleteAttachmentsByDocument(ctx, doc.ID)
			_ = s.repo.DeleteDocument(ctx, doc.ID)
		}
	}

	return s.repo.DeleteNode(ctx, id)
}

// CreateDocument 在节点上创建文档，自动渲染 Markdown
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

	format := req.Format
	if format == "" {
		format = "markdown"
	}

	contentHTML := ""
	if format == "markdown" && req.Content != "" {
		contentHTML = s.markdownSvc.RenderAndSanitize(req.Content)
	}

	doc := &model.Document{
		NodeID:      nodeID,
		Content:     req.Content,
		ContentHTML: contentHTML,
		Format:      format,
	}
	if err := s.repo.CreateDocument(ctx, doc); err != nil {
		return nil, err
	}

	return s.buildDocumentResp(ctx, node, doc)
}

// GetDocument 获取文档内容
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

// UpdateDocument 更新文档内容，事务中自动创建 Revision 并使用乐观锁
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

		if req.Content != "" || req.Format != "" {
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

			if req.Content != "" {
				doc.Content = req.Content
			}
			if req.Format != "" {
				doc.Format = req.Format
			}

			if doc.Format == "markdown" && doc.Content != "" {
				doc.ContentHTML = s.markdownSvc.RenderAndSanitize(doc.Content)
			} else if doc.Format != "markdown" {
				doc.ContentHTML = ""
			}

			doc.CurrentVer++
			doc.LastEditedBy = userID
			now := time.Now()
			doc.LastEditedAt = &now
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

// DeleteDocument 删除文档，先创建回收站记录再级联删除附件
func (s *wikiService) DeleteDocument(ctx context.Context, id string) error {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckDocumentPermission(ctx, id, userID, "delete"); err != nil {
		return err
	}

	doc, err := s.repo.GetDocumentByID(ctx, id)
	if err != nil {
		return err
	}

	node, err := s.repo.GetNodeByID(ctx, doc.NodeID)
	if err != nil {
		return err
	}

	if err := s.MoveToTrash(ctx, model.TrashTypeDocument, id, node.SpaceID, node.Title, userID); err != nil {
		return err
	}

	return s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
		if err := s.repo.DeleteAttachmentsByDocument(txCtx, id); err != nil {
			return err
		}
		return s.repo.DeleteDocument(txCtx, id)
	})
}

// buildDocumentResp 构建文档响应结构（包含标题、内容、Revision 信息）
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

// ListRevisions 列出文档的所有历史版本
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

// GetRevision 获取指定版本的历史内容
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

// RestoreRevision 恢复到指定版本，自动保存当前状态为新 Revision
func (s *wikiService) RestoreRevision(ctx context.Context, documentID string, version int, userID string) (*dto.DocumentResp, error) {
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "update"); err != nil {
		return nil, err
	}

	var resp *dto.DocumentResp
	err := s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
		revision, err := s.repo.GetRevisionByVersion(txCtx, documentID, version)
		if err != nil {
			return err
		}

		doc, err := s.repo.GetDocumentByID(txCtx, documentID)
		if err != nil {
			return err
		}

		currentVer := doc.CurrentVer

		// 保存当前状态为新的 Revision
		currentRevision := &model.DocumentRevision{
			DocumentID:  doc.ID,
			Version:     currentVer,
			Content:     doc.Content,
			ContentHTML: doc.ContentHTML,
			Summary:     "Auto-saved before restore",
			EditedBy:    doc.LastEditedBy,
		}
		if err := s.repo.CreateRevision(txCtx, currentRevision); err != nil {
			return err
		}

		// 恢复到指定版本
		doc.Content = revision.Content
		doc.ContentHTML = revision.ContentHTML
		doc.CurrentVer = currentVer + 1
		doc.LastEditedBy = userID
		now := time.Now()
		doc.LastEditedAt = &now

		if err := s.repo.UpdateDocument(txCtx, doc, currentVer); err != nil {
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

// AddMember 添加 Space 成员
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

// RemoveMember 移除 Space 成员
func (s *wikiService) RemoveMember(ctx context.Context, spaceID, userID string) error {
	currentUserID := contextx.GetUserID(ctx)
	if err := s.CheckSpacePermission(ctx, spaceID, currentUserID, "member:manage"); err != nil {
		return err
	}

	return s.repo.RemoveMember(ctx, spaceID, userID)
}

// ListMembers 列出 Space 的成员列表
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

// SetNodePermission 设置节点级别权限（如已存在则更新）
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

// GetNodePermissions 获取节点的所有权限配置
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

// RemoveNodePermission 移除节点级别权限
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

// GetStats 获取 Wiki 统计数据（Space、Node、Document 数量及浏览数）
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

// ListTrashItems 列出回收站项目（分页，支持 Space 和类型过滤）
func (s *wikiService) ListTrashItems(ctx context.Context, tenantID string, spaceID string, itemType string, page, pageSize int) ([]*dto.TrashItemResp, int64, error) {
	userID := contextx.GetUserID(ctx)

	if spaceID != "" {
		if err := s.CheckSpacePermission(ctx, spaceID, userID, "space:read"); err != nil {
			return nil, 0, err
		}
	}

	items, total, err := s.repo.ListTrashItems(ctx, tenantID, spaceID, itemType, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	var resps []*dto.TrashItemResp
	for _, item := range items {
		resps = append(resps, &dto.TrashItemResp{
			ID:        item.ID,
			ItemType:  item.ItemType,
			ItemID:    item.ItemID,
			SpaceID:   item.SpaceID,
			Title:     item.Title,
			DeletedBy: item.DeletedBy,
			DeletedAt: item.DeletedAt,
			ExpiresAt: item.ExpiresAt,
		})
	}

	return resps, total, nil
}

// RestoreTrashItem 从回收站恢复项目（根据类型调用不同的恢复逻辑）
func (s *wikiService) RestoreTrashItem(ctx context.Context, id string, userID string) error {
	item, err := s.repo.GetTrashItem(ctx, id)
	if err != nil {
		return apperrors.ErrNotFound("回收站项目不存在")
	}

	if err := s.CheckSpacePermission(ctx, item.SpaceID, userID, "space:update"); err != nil {
		return err
	}

	return s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
		switch item.ItemType {
		case model.TrashTypeNode:
			return s.restoreNodeFromTrash(txCtx, item)
		case model.TrashTypeDocument:
			return s.restoreDocumentFromTrash(txCtx, item)
		case model.TrashTypeSpace:
			return s.restoreSpaceFromTrash(txCtx, item)
		default:
			return apperrors.ErrBadRequest("未知的项目类型")
		}
	})
}

// restoreNodeFromTrash 从回收站恢复节点
func (s *wikiService) restoreNodeFromTrash(ctx context.Context, item *model.TrashItem) error {
	return s.repo.DeleteTrashItem(ctx, item.ID)
}

// restoreDocumentFromTrash 从回收站恢复文档
func (s *wikiService) restoreDocumentFromTrash(ctx context.Context, item *model.TrashItem) error {
	return s.repo.DeleteTrashItem(ctx, item.ID)
}

// restoreSpaceFromTrash 从回收站恢复 Space
func (s *wikiService) restoreSpaceFromTrash(ctx context.Context, item *model.TrashItem) error {
	return s.repo.DeleteTrashItem(ctx, item.ID)
}

// PermanentDeleteTrashItem 永久删除回收站项目
func (s *wikiService) PermanentDeleteTrashItem(ctx context.Context, id string, userID string) error {
	item, err := s.repo.GetTrashItem(ctx, id)
	if err != nil {
		return apperrors.ErrNotFound("回收站项目不存在")
	}

	if err := s.CheckSpacePermission(ctx, item.SpaceID, userID, "space:delete"); err != nil {
		return err
	}

	return s.repo.DeleteTrashItem(ctx, id)
}

// MoveToTrash 将项目移动到回收站（30 天后自动过期）
func (s *wikiService) MoveToTrash(ctx context.Context, itemType string, itemID string, spaceID string, title string, userID string) error {
	expiresAt := time.Now().AddDate(0, 0, 30)
	item := &model.TrashItem{
		TenantID:  contextx.GetTenantID(ctx),
		ItemType:  itemType,
		ItemID:    itemID,
		SpaceID:   spaceID,
		Title:     title,
		DeletedBy: userID,
		ExpiresAt: expiresAt,
	}
	return s.repo.CreateTrashItem(ctx, item)
}

// Search 搜索 Wiki（标题 + 内容，自动过滤无权限内容，支持分页）
func (s *wikiService) Search(ctx context.Context, tenantID string, userID string, query string, spaceID string, page, pageSize int) ([]*dto.SearchResultResp, int64, error) {
	if query == "" {
		return []*dto.SearchResultResp{}, 0, nil
	}

	titleResults, err := s.repo.SearchNodesByTitle(ctx, tenantID, spaceID, query)
	if err != nil {
		return nil, 0, err
	}

	contentResults, err := s.repo.SearchDocumentsByContent(ctx, tenantID, spaceID, query)
	if err != nil {
		return nil, 0, err
	}

	var results []*dto.SearchResultResp

	for _, node := range titleResults {
		if err := s.CheckNodePermission(ctx, node.ID, userID, "read"); err != nil {
			continue
		}
		results = append(results, &dto.SearchResultResp{
			ID:        node.ID,
			Type:      "node",
			Title:     node.Title,
			SpaceID:   node.SpaceID,
			NodeID:    node.ID,
			Snippet:   "",
			Highlight: node.Title,
			UpdatedAt: node.UpdatedAt,
			Score:     1.0,
		})
	}

	for _, doc := range contentResults {
		node, err := s.repo.GetNodeByID(ctx, doc.NodeID)
		if err != nil {
			continue
		}
		if err := s.CheckNodePermission(ctx, doc.NodeID, userID, "read"); err != nil {
			continue
		}

		snippet := s.generateSnippet(doc.Content, query)
		results = append(results, &dto.SearchResultResp{
			ID:        doc.ID,
			Type:      "document",
			Title:     node.Title,
			SpaceID:   node.SpaceID,
			NodeID:    doc.NodeID,
			Snippet:   snippet,
			Highlight: query,
			UpdatedAt: doc.UpdatedAt,
			Score:     0.8,
		})
	}

	total := int64(len(results))

	start := (page - 1) * pageSize
	if start >= len(results) {
		return []*dto.SearchResultResp{}, total, nil
	}
	end := start + pageSize
	if end > len(results) {
		end = len(results)
	}

	return results[start:end], total, nil
}

// generateSnippet 生成搜索结果的摘要片段（围绕关键字位置）
func (s *wikiService) generateSnippet(content, query string) string {
	maxLength := 200
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)

	idx := strings.Index(lowerContent, lowerQuery)
	if idx == -1 {
		if len(content) > maxLength {
			return content[:maxLength] + "..."
		}
		return content
	}

	start := idx - 50
	if start < 0 {
		start = 0
	}

	end := start + maxLength
	if end > len(content) {
		end = len(content)
	}

	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}

	return snippet
}

// CreateAttachment 创建文档附件（需文档 update 权限）
func (s *wikiService) CreateAttachment(ctx context.Context, userID string, req *dto.CreateAttachmentReq) (*dto.AttachmentResp, error) {
	if _, err := s.repo.GetDocumentByID(ctx, req.DocumentID); err != nil {
		return nil, err
	}

	if err := s.CheckDocumentPermission(ctx, req.DocumentID, userID, "update"); err != nil {
		return nil, err
	}

	attachment := &model.Attachment{
		TenantID:   contextx.GetTenantID(ctx),
		DocumentID: req.DocumentID,
		FileName:   req.FileName,
		FileSize:   req.FileSize,
		MimeType:   req.MimeType,
		FileURL:    req.FileURL,
		UploadedBy: userID,
	}

	if err := s.repo.CreateAttachment(ctx, attachment); err != nil {
		return nil, err
	}

	return &dto.AttachmentResp{
		ID:         attachment.ID,
		DocumentID: attachment.DocumentID,
		FileName:   attachment.FileName,
		FileSize:   attachment.FileSize,
		MimeType:   attachment.MimeType,
		FileURL:    attachment.FileURL,
		UploadedBy: attachment.UploadedBy,
		CreatedAt:  attachment.CreatedAt,
	}, nil
}

// ListAttachments 列出文档的所有附件（需文档 read 权限）
func (s *wikiService) ListAttachments(ctx context.Context, documentID string, userID string) ([]*dto.AttachmentResp, error) {
	if err := s.CheckDocumentPermission(ctx, documentID, userID, "read"); err != nil {
		return nil, err
	}

	attachments, err := s.repo.ListAttachmentsByDocument(ctx, documentID)
	if err != nil {
		return nil, err
	}

	var resps []*dto.AttachmentResp
	for _, a := range attachments {
		resps = append(resps, &dto.AttachmentResp{
			ID:         a.ID,
			DocumentID: a.DocumentID,
			FileName:   a.FileName,
			FileSize:   a.FileSize,
			MimeType:   a.MimeType,
			FileURL:    a.FileURL,
			UploadedBy: a.UploadedBy,
			CreatedAt:  a.CreatedAt,
		})
	}

	return resps, nil
}

// DeleteAttachment 删除附件（需文档 update 权限）
func (s *wikiService) DeleteAttachment(ctx context.Context, id string, userID string) error {
	attachment, err := s.repo.GetAttachment(ctx, id)
	if err != nil {
		return apperrors.ErrNotFound("附件不存在")
	}

	if err := s.CheckDocumentPermission(ctx, attachment.DocumentID, userID, "update"); err != nil {
		return err
	}

	return s.repo.DeleteAttachment(ctx, id)
}

func (s *wikiService) generateID() string {
	return idgen.New()
}
