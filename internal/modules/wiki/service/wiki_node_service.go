package service

import (
	"context"
	"errors"
	"sort"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"
	apperrors "meteorx/internal/pkg/apperrors"

	"gorm.io/gorm"
)

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
