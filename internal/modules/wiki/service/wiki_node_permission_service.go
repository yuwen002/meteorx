package service

import (
	"context"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"
	apperrors "meteorx/internal/pkg/apperrors"
)

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
