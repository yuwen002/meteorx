package service

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"
	apperrors "meteorx/internal/pkg/apperrors"
	"meteorx/pkg/idgen"

	"gorm.io/gorm"
)

// expireBatchSize 过期清理单轮扫描上限：避免积压场景下一次载入并串行清除全部条目
// 导致 HTTP 手动触发入口超时（前端 axios 15s）后在级联中途被 context 取消。
const expireBatchSize = 200

// expireTrashMu 进程内互斥：同一进程内定时任务与手动运维端点同时触发时，
// 后到者直接跳过，避免重复扫描与交叉清除。
var expireTrashMu sync.Mutex

// 跨副本分布式锁：多副本部署时各副本的定时任务/运维端点通过 Redis SetNX 抢占，
// 未抢到时直接跳过；TTL 需覆盖单轮清理的最长预期耗时，实例崩溃后由 TTL 自动释放。
const (
	expireLockKey = "wiki:trash:expire:lock"
	expireLockTTL = 30 * time.Minute
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
// ExpireTrashItems 清理已过期的回收站项目（定时任务/运维端点入口，跨租户全局扫描）。
// 分批直至排空过期积压（每批 expireBatchSize 条，仓储内逐条事务）；
// 已有清理在执行时直接跳过（TryLock），不依赖调用者租户上下文，返回清理数量。
func (s *wikiService) ExpireTrashItems(ctx context.Context) (int, error) {
	if !expireTrashMu.TryLock() {
		log.Printf("[WikiTrash] 过期清理已有实例在执行，本次跳过")
		return 0, nil
	}
	defer expireTrashMu.Unlock()

	// 分布式锁仅在 Redis 可用时生效；获取失败（网络/命令错误）时降级为仅进程内互斥，
	// 重复执行的安全性由条目级事务与幂等 purge 兜底。
	if s.expireCache != nil && s.expireCache.IsAvailable() {
		token := idgen.New()
		acquired, err := s.expireCache.SetNX(ctx, expireLockKey, token, expireLockTTL)
		if err != nil {
			log.Printf("[WikiTrash] 分布式锁获取失败，降级为进程内互斥: %v", err)
		} else if !acquired {
			log.Printf("[WikiTrash] 其他实例持有过期清理分布式锁，本次跳过")
			return 0, nil
		} else {
			defer s.releaseExpireLock(expireLockKey, token)
		}
	}

	total := 0
	for {
		n, err := s.repo.ExpireTrashItems(ctx, expireBatchSize)
		total += n
		if err != nil {
			// 单条失败不阻断已完成的计数，上抛供调用方记录告警；剩余条目留待下轮
			return total, err
		}
		if n < expireBatchSize || ctx.Err() != nil {
			// 本轮未触及上限即已排空；上下文被取消时同样停止，避免无意义循环
			return total, nil
		}
	}
}

// releaseExpireLock 释放分布式锁（仅当持有者令牌匹配时删除）。
// Get+Delete 非原子，但正常路径下远早于 TTL 到期，误删窗口可忽略；
// 释放失败仅记日志，留待 TTL 自然过期兜底。使用独立 context，避免请求 ctx 已取消导致无法释放。
func (s *wikiService) releaseExpireLock(key, token string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cur, err := s.expireCache.Get(ctx, key)
	if err != nil || cur != token {
		return
	}
	if err := s.expireCache.Delete(ctx, key); err != nil {
		log.Printf("[WikiTrash] 分布式锁释放失败，等待 TTL 过期兜底: %v", err)
	}
}

func (s *wikiService) PermanentDeleteTrashItem(ctx context.Context, id string, userID string) error {
	item, err := s.repo.GetTrashItem(ctx, id)
	if err != nil {
		return apperrors.ErrNotFound("回收站项目不存在")
	}

	if err := s.CheckSpacePermission(ctx, item.SpaceID, userID, "space:delete"); err != nil {
		return err
	}

	// 物理清除（多张关联表）与回收站记录兜底清理必须同一事务，
	// 避免中途失败留下半删数据。实体已不存在（软删记录缺失）时仍容忍并清理回收站行。
	return s.tx.WithTx(ctx, func(txCtx context.Context, _ *gorm.DB) error {
		var purgeErr error
		switch item.ItemType {
		case model.TrashTypeDocument:
			purgeErr = s.repo.PurgeDocument(txCtx, item.ItemID)
		case model.TrashTypeNode:
			purgeErr = s.repo.PurgeNodeTree(txCtx, item.ItemID)
		case model.TrashTypeSpace:
			purgeErr = s.repo.PurgeSpaceTree(txCtx, item.ItemID)
		default:
			return apperrors.ErrBadRequest("未知的项目类型")
		}
		if purgeErr != nil && !isTrashEntityMissing(purgeErr) {
			return purgeErr
		}
		// Purge 内部已清理关联 trash 记录，此处兜底确保列表不再残留该行
		return s.repo.DeleteTrashItem(txCtx, item.ID)
	})
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
