package service

import (
	"context"
	"strings"

	"meteorx/internal/modules/wiki/model"
	apperrors "meteorx/internal/pkg/apperrors"
)

// Space 角色权限矩阵
var spaceRolePermissions = map[string]map[string]bool{
	model.SpaceRoleOwner: {
		"space:read":       true,
		"space:update":     true,
		"space:delete":     true,
		"member:manage":    true,
		"node:create":      true,
		"node:read":        true,
		"node:update":      true,
		"node:delete":      true,
		"document:create":  true,
		"document:read":    true,
		"document:update":  true,
		"document:delete":  true,
		"revision:read":    true,
		"revision:restore": true,
	},
	model.SpaceRoleAdmin: {
		"space:read":       true,
		"space:update":     true,
		"member:manage":    true,
		"node:create":      true,
		"node:read":        true,
		"node:update":      true,
		"node:delete":      true,
		"document:create":  true,
		"document:read":    true,
		"document:update":  true,
		"document:delete":  true,
		"revision:read":    true,
		"revision:restore": true,
	},
	model.SpaceRoleEditor: {
		"node:create":      true,
		"node:read":        true,
		"node:update":      true,
		"document:create":  true,
		"document:read":    true,
		"document:update":  true,
		"revision:read":    true,
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
		// 对非私有空间，只允许只读类操作（space:read / node:read / document:read / revision:read 或裸 read）
		if !isReadAction(action) {
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
// Node 权限是 Space 权限的补充：可以为特定用户在特定节点授予额外权限
func (s *wikiService) CheckNodePermission(ctx context.Context, nodeID, userID string, action string) error {
	node, err := s.repo.GetNodeByID(ctx, nodeID)
	if err != nil {
		return err
	}

	// 1. 首先检查 Space 级别权限（基础权限）
	spaceID := node.SpaceID
	spaceAction := mapActionToSpaceLevel(action)
	spaceErr := s.CheckSpacePermission(ctx, spaceID, userID, spaceAction)

	// 2. 检查 Node 级别权限（可以覆盖/补充 Space 权限）
	nodeLevelErr := s.checkNodeLevelPermission(ctx, nodeID, userID, action)

	// 如果 Space 权限通过，直接允许
	if spaceErr == nil {
		return nil
	}

	// 如果 Space 权限不通过，但 Node 权限通过，则允许（Node 权限可以授予额外权限）
	if nodeLevelErr == nil {
		return nil
	}

	// 两个都不通过，返回 Space 权限错误
	return spaceErr
}

// checkNodeLevelPermission 检查节点级别的权限设置
// Node 权限可以为用户在特定节点授予额外权限（即使 Space 级别没有权限）
func (s *wikiService) checkNodeLevelPermission(ctx context.Context, nodeID, userID string, action string) error {
	perms, err := s.repo.GetUserNodePermissions(ctx, nodeID, userID)
	if err != nil {
		return err
	}

	// 如果没有节点级别权限，直接返回错误（让调用方决定使用 Space 权限）
	if len(perms) == 0 {
		return apperrors.ErrForbidden("无节点级别权限")
	}

	// 检查节点级别的权限
	for _, perm := range perms {
		switch perm.Permission {
		case model.NodePermView:
			if action == "read" {
				return nil
			}
		case model.NodePermEdit:
			if action == "read" || action == "update" {
				return nil
			}
		case model.NodePermDelete:
			if action == "read" || action == "update" || action == "delete" {
				return nil
			}
		}
	}

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

// isReadAction 判断 action 是否为只读类权限动作（space:read / node:read / document:read / revision:read 等）
func isReadAction(action string) bool {
	return action == "read" || strings.HasSuffix(action, ":read")
}
