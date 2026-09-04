package service

import (
	"context"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"
	apperrors "meteorx/internal/pkg/apperrors"

	"gorm.io/gorm"
)

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
