package service

import (
	"context"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"

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
