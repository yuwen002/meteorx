package service

import (
	"context"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"
	apperrors "meteorx/internal/pkg/apperrors"

	"gorm.io/gorm"
)

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
func (s *wikiService) ListSpaces(ctx context.Context, tenantID string, userID string, keyword string, page, pageSize int) ([]*dto.WikiSpaceResp, int64, error) {
	spaces, total, err := s.repo.ListSpaces(ctx, tenantID, userID, keyword, page, pageSize)
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
// UpdateSpace 更新 Space 信息（userID 用于权限校验，语义上非 tenantID）
func (s *wikiService) UpdateSpace(ctx context.Context, id string, userID string, req *dto.UpdateWikiSpaceReq) (*dto.WikiSpaceResp, error) {
	if err := s.CheckSpacePermission(ctx, id, userID, "space:update"); err != nil {
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

// DeleteSpace 删除 Space，级联软删空间下所有节点和文档，再创建回收站记录并软删空间
func (s *wikiService) DeleteSpace(ctx context.Context, id string, tenantID string) error {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckSpacePermission(ctx, id, userID, "space:delete"); err != nil {
		return err
	}

	space, err := s.repo.GetSpaceByID(ctx, id)
	if err != nil {
		return err
	}

	return s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
		nodes, err := s.repo.ListNodesBySpace(txCtx, id)
		if err != nil {
			return err
		}

		for _, node := range nodes {
			if err := s.softDeleteNodeAndDescendants(txCtx, node, userID); err != nil {
				return err
			}
		}

		if err := s.MoveToTrash(txCtx, model.TrashTypeSpace, id, id, space.Name, userID); err != nil {
			return err
		}

		return s.repo.DeleteSpace(txCtx, id)
	})
}

// softDeleteNodeAndDescendants 级联软删节点及其所有后代（含关联文档）
func (s *wikiService) softDeleteNodeAndDescendants(ctx context.Context, node *model.WikiNode, userID string) error {
	children, err := s.repo.ListChildNodes(ctx, node.ID)
	if err != nil {
		return err
	}

	for _, child := range children {
		if err := s.softDeleteNodeAndDescendants(ctx, child, userID); err != nil {
			return err
		}
	}

	if node.Type == model.NodeTypeDocument {
		doc, err := s.repo.GetDocumentByNodeID(ctx, node.ID)
		if err != nil && err != repository.ErrDocumentNotFound {
			return err
		}
		if doc != nil {
			if err := s.MoveToTrash(ctx, model.TrashTypeDocument, doc.ID, node.SpaceID, node.Title, userID); err != nil {
				return err
			}
			if err := s.repo.DeleteDocument(ctx, doc.ID); err != nil {
				return err
			}
		}
	}

	if err := s.MoveToTrash(ctx, model.TrashTypeNode, node.ID, node.SpaceID, node.Title, userID); err != nil {
		return err
	}

	return s.repo.DeleteNode(ctx, node.ID)
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

// AddMember 添加或更新 Space 成员（幂等：已存在则更新角色，避免重复成员记录）
func (s *wikiService) AddMember(ctx context.Context, spaceID string, req *dto.WikiSpaceMemberReq) (*dto.WikiSpaceMemberResp, error) {
	userID := contextx.GetUserID(ctx)
	if err := s.CheckSpacePermission(ctx, spaceID, userID, "member:manage"); err != nil {
		return nil, err
	}

	// 已存在：仅更新角色，保证操作幂等
	existing, err := s.repo.GetMember(ctx, spaceID, req.UserID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// 唯一 owner 不可被降级（避免知识库失去所有者）
		if existing.Role == model.SpaceRoleOwner && req.Role != model.SpaceRoleOwner {
			members, err := s.repo.ListMembers(ctx, spaceID)
			if err != nil {
				return nil, err
			}
			ownerCount := 0
			for _, m := range members {
				if m.Role == model.SpaceRoleOwner {
					ownerCount++
				}
			}
			if ownerCount <= 1 {
				return nil, apperrors.ErrBadRequest("知识库至少需要保留一名所有者")
			}
		}
		existing.Role = req.Role
		if err := s.repo.UpdateMemberRole(ctx, existing); err != nil {
			return nil, err
		}
		return &dto.WikiSpaceMemberResp{
			ID:      existing.ID,
			SpaceID: existing.SpaceID,
			UserID:  existing.UserID,
			Role:    existing.Role,
		}, nil
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

// RemoveMember 移除 Space 成员（知识库至少保留一名 owner）
func (s *wikiService) RemoveMember(ctx context.Context, spaceID, userID string) error {
	currentUserID := contextx.GetUserID(ctx)
	if err := s.CheckSpacePermission(ctx, spaceID, currentUserID, "member:manage"); err != nil {
		return err
	}

	target, err := s.repo.GetMember(ctx, spaceID, userID)
	if err != nil {
		return err
	}
	if target == nil {
		return apperrors.ErrNotFound("成员不存在")
	}
	if target.Role == model.SpaceRoleOwner {
		members, err := s.repo.ListMembers(ctx, spaceID)
		if err != nil {
			return err
		}
		ownerCount := 0
		for _, m := range members {
			if m.Role == model.SpaceRoleOwner {
				ownerCount++
			}
		}
		if ownerCount <= 1 {
			return apperrors.ErrBadRequest("知识库至少需要保留一名所有者")
		}
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