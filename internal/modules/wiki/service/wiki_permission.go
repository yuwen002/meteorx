package service

import (
	"context"

	"meteorx/internal/modules/wiki/model"
	apperrors "meteorx/internal/pkg/apperrors"
)

// Space 角色权限矩阵
var spaceRolePermissions = map[string]map[string]bool{
	model.SpaceRoleOwner: {
		"space:read":     true,
		"space:update":   true,
		"space:delete":   true,
		"member:manage":  true,
		"node:create":    true,
		"node:read":      true,
		"node:update":    true,
		"node:delete":    true,
		"document:create": true,
		"document:read":  true,
		"document:update": true,
		"document:delete": true,
		"revision:read":  true,
		"revision:restore": true,
	},
	model.SpaceRoleAdmin: {
		"space:read":     true,
		"space:update":   true,
		"member:manage":  true,
		"node:create":    true,
		"node:read":      true,
		"node:update":    true,
		"node:delete":    true,
		"document:create": true,
		"document:read":  true,
		"document:update": true,
		"document:delete": true,
		"revision:read":  true,
		"revision:restore": true,
	},
	model.SpaceRoleEditor: {
		"node:create":    true,
		"node:read":      true,
		"node:update":    true,
		"document:create": true,
		"document:read":  true,
		"document:update": true,
		"revision:read":  true,
		"revision:restore": true,
	},
	model.SpaceRoleViewer: {
		"node:read":     true,
		"document:read": true,
		"revision:read": true,
	},
}

// 节点权限到动作的映射
var nodePermToAction = map[string]string{
	model.NodePermView:   "read",
	model.NodePermEdit:   "update",
	model.NodePermDelete: "delete",
}

// CheckSpacePermission 检查用户在 Space 中的权限
func (s *wikiService) CheckSpacePermission(ctx context.Context, spaceID, userID string, action string) error {
	member, err := s.repo.GetMember(ctx, spaceID, userID)
	if err != nil {
		return apperrors.ErrForbidden("获取成员信息失败")
	}

	// 非成员检查 Space 可见性
	if member == nil {
		space, err := s.repo.GetSpaceByID(ctx, spaceID)
		if err != nil {
			return err
		}
		if space.Visibility != model.VisibilityTenant && space.Visibility != model.VisibilityPublic {
			return apperrors.ErrForbidden("您不是该空间的成员")
		}
		// 对非私有空间，只允许 read 操作
		if action != "read" {
			return apperrors.ErrForbidden("您没有执行此操作的权限")
		}
		return nil
	}

	// 根据角色检查权限
	perms, exists := spaceRolePermissions[member.Role]
	if !exists {
		return apperrors.ErrForbidden("无效的成员角色")
	}

	if !perms[action] {
		return apperrors.ErrForbidden("您没有执行此操作的权限")
	}

	return nil
}

// CheckNodePermission 检查用户对 Node 的权限（包含权限继承）
func (s *wikiService) CheckNodePermission(ctx context.Context, nodeID, userID string, action string) error {
	node, err := s.repo.GetNodeByID(ctx, nodeID)
	if err != nil {
		return err
	}

	// 1. 检查 Space 级别权限
	spaceID := node.SpaceID
	spaceAction := mapActionToSpaceLevel(action)
	if err := s.CheckSpacePermission(ctx, spaceID, userID, spaceAction); err != nil {
		return err
	}

	// 2. 检查 Node 级别权限（覆盖 Space 权限）
	if err := s.checkNodeLevelPermission(ctx, nodeID, userID, action); err != nil {
		return err
	}

	return nil
}

// checkNodeLevelPermission 检查节点级别的权限设置
func (s *wikiService) checkNodeLevelPermission(ctx context.Context, nodeID, userID string, action string) error {
	// 获取用户在该节点的所有权限
	perms, err := s.repo.GetUserNodePermissions(ctx, nodeID, userID)
	if err != nil {
		return err
	}

	// 如果没有节点级别权限，使用默认的 Space 权限（已在 CheckSpacePermission 中检查）
	if len(perms) == 0 {
		return nil
	}

	// 检查节点级别的拒绝/允许
	for _, perm := range perms {
		if perm.Permission == model.NodePermEdit && (action == "update" || action == "delete") {
			return nil
		}
		if perm.Permission == model.NodePermDelete && action == "delete" {
			return nil
		}
	}

	// 有节点权限但没有匹配的，返回权限不足
	return apperrors.ErrForbidden("您没有执行此操作的节点权限")
}

// mapActionToSpaceLevel 将操作映射到 Space 级别的权限检查
func mapActionToSpaceLevel(action string) string {
	switch action {
	case "read":
		return "node:read"
	case "create":
		return "node:create"
	case "update":
		return "node:update"
	case "delete":
		return "node:delete"
	default:
		return action
	}
}

// CheckDocumentPermission 检查用户对 Document 的权限
func (s *wikiService) CheckDocumentPermission(ctx context.Context, documentID, userID string, action string) error {
	doc, err := s.repo.GetDocumentByID(ctx, documentID)
	if err != nil {
		return err
	}

	return s.CheckNodePermission(ctx, doc.NodeID, userID, action)
}

// GetEffectivePermissions 获取用户在 Space 中的有效权限列表
func (s *wikiService) GetEffectivePermissions(ctx context.Context, spaceID, userID string) map[string]bool {
	result := make(map[string]bool)

	member, err := s.repo.GetMember(ctx, spaceID, userID)
	if err != nil || member == nil {
		return result
	}

	perms, exists := spaceRolePermissions[member.Role]
	if !exists {
		return result
	}

	for k, v := range perms {
		result[k] = v
	}

	return result
}