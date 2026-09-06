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

// RestoreTrashItem 从回收站恢复项目：先真实还原软删实体（含整棵节点子树），再移除回收站记录。
// 此前该逻辑只删 trash 记录而不还原数据，导致“恢复”后的资源必然不存在。
func (s *wikiService) RestoreTrashItem(ctx context.Context, id string, userID string) error {
	item, err := s.repo.GetTrashItem(ctx, id)
	if err != nil {
		return apperrors.ErrNotFound("回收站项目不存在")
	}

	if err := s.CheckSpacePermission(ctx, item.SpaceID, userID, "space:update"); err != nil {
		return err
	}

	return s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
		var restoreErr error
		switch item.ItemType {
		case model.TrashTypeNode:
			restoreErr = s.repo.RestoreNodeTree(txCtx, item.ItemID)
		case model.TrashTypeDocument:
			restoreErr = s.repo.RestoreDocument(txCtx, item.ItemID)
		case model.TrashTypeSpace:
			restoreErr = s.repo.RestoreSpace(txCtx, item.ItemID)
		default:
			return apperrors.ErrBadRequest("未知的项目类型")
		}
		if restoreErr != nil && !isTrashEntityMissing(restoreErr) {
			return restoreErr
		}
		// 实体已不存在（如先被彻底删除）时按“记录清理”处理
		return s.repo.DeleteTrashItem(txCtx, item.ID)
	})
}

// PermanentDeleteTrashItem 永久删除回收站项目：物理清除实体及其关联数据，避免软删数据永久残留。
func (s *wikiService) PermanentDeleteTrashItem(ctx context.Context, id string, userID string) error {
	item, err := s.repo.GetTrashItem(ctx, id)
	if err != nil {
		return apperrors.ErrNotFound("回收站项目不存在")
	}

	if err := s.CheckSpacePermission(ctx, item.SpaceID, userID, "space:delete"); err != nil {
		return err
	}

	var purgeErr error
	switch item.ItemType {
	case model.TrashTypeDocument:
		purgeErr = s.repo.PurgeDocument(ctx, item.ItemID)
	case model.TrashTypeNode:
		purgeErr = s.repo.PurgeNodeTree(ctx, item.ItemID)
	case model.TrashTypeSpace:
		purgeErr = s.repo.PurgeSpaceTree(ctx, item.ItemID)
	default:
		return apperrors.ErrBadRequest("未知的项目类型")
	}
	if purgeErr != nil && !isTrashEntityMissing(purgeErr) {
		return purgeErr
	}
	// Purge 内部已清理关联 trash 记录，此处兜底确保列表不再残留该行
	return s.repo.DeleteTrashItem(ctx, item.ID)
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

// isTrashEntityMissing 判断错误是否源于实体记录已不存在
func isTrashEntityMissing(err error) bool {
	return errors.Is(err, repository.ErrWikiNodeNotFound) ||
		errors.Is(err, repository.ErrDocumentNotFound) ||
		errors.Is(err, repository.ErrWikiSpaceNotFound) ||
		errors.Is(err, gorm.ErrRecordNotFound)
}
