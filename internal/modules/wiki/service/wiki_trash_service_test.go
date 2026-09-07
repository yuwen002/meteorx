package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"
	apperrors "meteorx/internal/pkg/apperrors"

	"github.com/stretchr/testify/assert"
)

func newTrashSvc(t *testing.T) (*repository.MockWikiRepository, *wikiService) {
	t.Helper()
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	svc := NewWikiService(repo, txManager)
	return repo, svc.(*wikiService)
}

func seedTrash(repo *repository.MockWikiRepository, id, itemType, itemID, spaceID string) {
	repo.SeedTrashItem(&model.TrashItem{
		ID:        id,
		TenantID:  "tenant-1",
		ItemType:  itemType,
		ItemID:    itemID,
		SpaceID:   spaceID,
		Title:     "deleted " + itemType,
		DeletedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
}

// TestMoveToTrash_SetsExpiryAndTenant 验证入回收站记录：租户/类型/30 天过期
func TestMoveToTrash_SetsExpiryAndTenant(t *testing.T) {
	repo, svc := newTrashSvc(t)
	ctx := serviceCtx("tenant-1")

	err := svc.MoveToTrash(ctx, model.TrashTypeDocument, "doc-1", "space-1", "文档标题", "user-1")

	assert.NoError(t, err)
	assert.NotNil(t, repo.LastCreatedTrash)
	item := repo.LastCreatedTrash
	if item == nil {
		return
	}
	assert.Equal(t, model.TrashTypeDocument, item.ItemType)
	assert.Equal(t, "doc-1", item.ItemID)
	assert.Equal(t, "space-1", item.SpaceID)
	assert.Equal(t, "文档标题", item.Title)
	assert.Equal(t, "tenant-1", item.TenantID)
	assert.Equal(t, "user-1", item.DeletedBy)
	// 默认保留 30 天
	now := time.Now()
	assert.True(t, item.ExpiresAt.After(now.AddDate(0, 0, 29)))
	assert.True(t, item.ExpiresAt.Before(now.AddDate(0, 0, 31)))
}

// TestRestoreTrashItem_DocumentSuccess 验证恢复文档：真实还原 + 清理回收站记录 + 提交
func TestRestoreTrashItem_DocumentSuccess(t *testing.T) {
	txManager, sqlMock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedTrash(repo, "trash-1", model.TrashTypeDocument, "doc-9", "space-1")
	sqlMock.ExpectBegin()
	sqlMock.ExpectCommit()
	svc := NewWikiService(repo, txManager)

	err := svc.RestoreTrashItem(serviceCtx("tenant-1"), "trash-1", "user-1")

	assert.NoError(t, err)
	assert.Equal(t, 1, repo.DeleteTrashItemCalls)
}

// TestRestoreTrashItem_NodeAndSpaceDispatch 验证节点/空间类型的恢复分发
func TestRestoreTrashItem_NodeAndSpaceDispatch(t *testing.T) {
	for _, tt := range []struct {
		name     string
		itemType string
	}{
		{"node", model.TrashTypeNode},
		{"space", model.TrashTypeSpace},
	} {
		t.Run(tt.name, func(t *testing.T) {
			txManager, sqlMock := newTxManager(t)
			repo := repository.NewMockWikiRepository()
			seedTrash(repo, "trash-x", tt.itemType, "entity-1", "space-1")
			sqlMock.ExpectBegin()
			sqlMock.ExpectCommit()
			svc := NewWikiService(repo, txManager)

			err := svc.RestoreTrashItem(serviceCtx("tenant-1"), "trash-x", "user-1")

			assert.NoError(t, err)
			assert.Equal(t, 1, repo.DeleteTrashItemCalls)
		})
	}
}

// TestRestoreTrashItem_EntityMissing 实体已被彻底删除时（软删记录缺失）按“清理记录”处理，不报错
func TestRestoreTrashItem_EntityMissing(t *testing.T) {
	txManager, sqlMock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedTrash(repo, "trash-1", model.TrashTypeDocument, "doc-gone", "space-1")
	repo.RestoreErr = repository.ErrDocumentNotFound
	sqlMock.ExpectBegin()
	sqlMock.ExpectCommit()
	svc := NewWikiService(repo, txManager)

	err := svc.RestoreTrashItem(serviceCtx("tenant-1"), "trash-1", "user-1")

	assert.NoError(t, err)
	assert.Equal(t, 1, repo.DeleteTrashItemCalls)
}

// TestRestoreTrashItem_RestoreRealErrorRollsBack 还原遇到真实数据库错误时回滚且不清理记录
func TestRestoreTrashItem_RestoreRealErrorRollsBack(t *testing.T) {
	txManager, sqlMock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedTrash(repo, "trash-1", model.TrashTypeSpace, "space-9", "space-9")
	repo.RestoreErr = errors.New("restore db down")
	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()
	svc := NewWikiService(repo, txManager)

	err := svc.RestoreTrashItem(serviceCtx("tenant-1"), "trash-1", "user-1")

	assert.EqualError(t, err, "restore db down")
	assert.Equal(t, 0, repo.DeleteTrashItemCalls)
}

// TestRestoreTrashItem_UnknownType 未知项目类型返回 400
func TestRestoreTrashItem_UnknownType(t *testing.T) {
	txManager, sqlMock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedTrash(repo, "trash-1", "widget", "x-1", "space-1")
	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()
	svc := NewWikiService(repo, txManager)

	err := svc.RestoreTrashItem(serviceCtx("tenant-1"), "trash-1", "user-1")

	var appErr *apperrors.AppError
	assert.True(t, errors.As(err, &appErr))
	if appErr != nil {
		assert.Equal(t, 400, appErr.StatusCode)
	}
	assert.Equal(t, 0, repo.DeleteTrashItemCalls)
}

// TestRestoreTrashItem_ViewerForbidden viewer 无 space:update 权限，恢复被拒绝
func TestRestoreTrashItem_ViewerForbidden(t *testing.T) {
	repo, svc := newTrashSvc(t)
	seedTrash(repo, "trash-1", model.TrashTypeDocument, "doc-9", "space-1")
	repo.GetMemberFn = func(ctx context.Context, spaceID, userID string) (*model.WikiSpaceMember, error) {
		return &model.WikiSpaceMember{SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleViewer}, nil
	}

	err := svc.RestoreTrashItem(serviceCtx("tenant-1"), "trash-1", "user-1")

	var appErr *apperrors.AppError
	assert.True(t, errors.As(err, &appErr))
	if appErr != nil {
		assert.Equal(t, 403, appErr.StatusCode)
	}
	assert.Equal(t, 0, repo.DeleteTrashItemCalls)
}

// TestPermanentDeleteTrashItem_Success 永久删除：同一事务内物理清除 + 清理回收站记录
func TestPermanentDeleteTrashItem_Success(t *testing.T) {
	txManager, sqlMock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedTrash(repo, "trash-1", model.TrashTypeDocument, "doc-9", "space-1")
	sqlMock.ExpectBegin()
	sqlMock.ExpectCommit()
	svc := NewWikiService(repo, txManager)

	err := svc.PermanentDeleteTrashItem(serviceCtx("tenant-1"), "trash-1", "user-1")

	assert.NoError(t, err)
	assert.Equal(t, 1, repo.DeleteTrashItemCalls)
}

// TestPermanentDeleteTrashItem_EntityMissing 实体已被清空时静默容忍并清理记录（同事务提交）
func TestPermanentDeleteTrashItem_EntityMissing(t *testing.T) {
	txManager, sqlMock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedTrash(repo, "trash-1", model.TrashTypeNode, "node-gone", "space-1")
	repo.PurgeErr = repository.ErrWikiNodeNotFound
	sqlMock.ExpectBegin()
	sqlMock.ExpectCommit()
	svc := NewWikiService(repo, txManager)

	err := svc.PermanentDeleteTrashItem(serviceCtx("tenant-1"), "trash-1", "user-1")

	assert.NoError(t, err)
	assert.Equal(t, 1, repo.DeleteTrashItemCalls)
}

// TestPermanentDeleteTrashItem_RealError 物理删除真实错误原样返回、事务回滚且不清理记录
func TestPermanentDeleteTrashItem_RealError(t *testing.T) {
	txManager, sqlMock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedTrash(repo, "trash-1", model.TrashTypeSpace, "space-9", "space-9")
	repo.PurgeErr = errors.New("purge db down")
	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()
	svc := NewWikiService(repo, txManager)

	err := svc.PermanentDeleteTrashItem(serviceCtx("tenant-1"), "trash-1", "user-1")

	assert.EqualError(t, err, "purge db down")
	assert.Equal(t, 0, repo.DeleteTrashItemCalls)
}

// TestListTrashItems_TypeFilter 列表按类型过滤并透传字段
func TestListTrashItems_TypeFilter(t *testing.T) {
	repo, svc := newTrashSvc(t)
	seedTrash(repo, "trash-doc", model.TrashTypeDocument, "doc-1", "space-1")
	seedTrash(repo, "trash-node", model.TrashTypeNode, "node-1", "space-1")
	seedTrash(repo, "trash-other-space", model.TrashTypeDocument, "doc-2", "space-2")

	items, total, err := svc.ListTrashItems(serviceCtx("tenant-1"), "tenant-1", "space-1", model.TrashTypeDocument, 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	if assert.Len(t, items, 1) {
		assert.Equal(t, "trash-doc", items[0].ID)
		assert.Equal(t, "doc-1", items[0].ItemID)
	}
}

// TestListTrashItems_Forbidden 指定空间但无 space:read 权限（非成员且空间私有）时拒绝
func TestListTrashItems_Forbidden(t *testing.T) {
	repo, svc := newTrashSvc(t)
	seedTrash(repo, "trash-doc", model.TrashTypeDocument, "doc-1", "space-1")
	// 非成员：GetMember 返回空成员 + 私有空间
	repo.GetMemberFn = func(_ context.Context, spaceID, userID string) (*model.WikiSpaceMember, error) {
		return nil, nil
	}
	if err := repo.CreateSpace(serviceCtx("tenant-1"), &model.WikiSpace{ID: "space-1", Visibility: model.VisibilityPrivate}); err != nil {
		t.Fatalf("seed space: %v", err)
	}

	items, total, err := svc.ListTrashItems(serviceCtx("tenant-1"), "tenant-1", "space-1", "", 1, 20)

	assert.Nil(t, items)
	assert.Equal(t, int64(0), total)
	var appErr *apperrors.AppError
	assert.True(t, errors.As(err, &appErr))
	if appErr != nil {
		assert.Equal(t, 403, appErr.StatusCode)
	}
}
