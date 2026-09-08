package service

import (
	"context"
	"errors"
	"testing"

	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"
	apperrors "meteorx/internal/pkg/apperrors"

	"github.com/stretchr/testify/assert"
)

func seedSpace(t *testing.T, repo *repository.MockWikiRepository, id string, name string, visibility int) {
	t.Helper()
	space := &model.WikiSpace{
		ID:         id,
		Name:       name,
		TenantID:   "tenant-1",
		Visibility: visibility,
		CreatedBy:  "user-1",
	}
	assert.NoError(t, repo.CreateSpace(context.Background(), space))
}

// TestGetSpace_ReturnsDetailWithRole 验证成员获取 Space 详情时返回成员数与自身角色。
func TestGetSpace_ReturnsDetailWithRole(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedSpace(t, repo, "space-1", "产品空间", model.VisibilityTenant)
	assert.NoError(t, repo.AddMember(context.Background(), &model.WikiSpaceMember{
		SpaceID: "space-1",
		UserID:  "user-1",
		Role:    model.SpaceRoleOwner,
	}))

	svc := NewWikiService(repo, txManager)
	resp, err := svc.GetSpace(serviceCtx("tenant-1"), "space-1", "user-1")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "space-1", resp.ID)
	assert.Equal(t, "产品空间", resp.Name)
	assert.Equal(t, int64(1), resp.MemberCount)
	assert.Equal(t, model.SpaceRoleOwner, resp.MyRole)
}

// TestGetSpace_NotFound 验证不存在的 Space 错误原样传播。
func TestGetSpace_NotFound(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	svc := NewWikiService(repo, txManager)

	_, err := svc.GetSpace(serviceCtx("tenant-1"), "missing", "user-1")

	assert.ErrorIs(t, err, repository.ErrWikiSpaceNotFound)
}

// TestGetSpace_PrivateSpaceNonMemberForbidden 验证非成员访问私有 Space 返回 403。
func TestGetSpace_PrivateSpaceNonMemberForbidden(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedSpace(t, repo, "space-1", "私有空间", model.VisibilityPrivate)
	repo.GetMemberFn = func(ctx context.Context, spaceID, userID string) (*model.WikiSpaceMember, error) {
		return nil, nil
	}
	svc := NewWikiService(repo, txManager)

	resp, err := svc.GetSpace(serviceCtx("tenant-1"), "space-1", "other-1")

	assert.Nil(t, resp)
	var appErr *apperrors.AppError
	assert.True(t, errors.As(err, &appErr), "error should be AppError type")
	if appErr != nil {
		assert.Equal(t, 403, appErr.StatusCode)
	}
}

// TestUpdateSpace_UpdatesFields 验证更新 Space 名称/描述成功。
func TestUpdateSpace_UpdatesFields(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedSpace(t, repo, "space-1", "旧名称", model.VisibilityTenant)
	svc := NewWikiService(repo, txManager)

	resp, err := svc.UpdateSpace(serviceCtx("tenant-1"), "space-1", "tenant-1", &dto.UpdateWikiSpaceReq{
		Name:        "新名称",
		Description: "新描述",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "新名称", resp.Name)
	assert.Equal(t, "新描述", resp.Description)
}

// TestListMembers_ReturnsMembers 验证成员列表返回。
func TestListMembers_ReturnsMembers(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedSpace(t, repo, "space-1", "空间", model.VisibilityTenant)
	for _, u := range []string{"user-1", "user-2"} {
		assert.NoError(t, repo.AddMember(context.Background(), &model.WikiSpaceMember{
			SpaceID: "space-1",
			UserID:  u,
			Role:    model.SpaceRoleEditor,
		}))
	}
	svc := NewWikiService(repo, txManager)

	members, err := svc.ListMembers(serviceCtx("tenant-1"), "space-1")

	assert.NoError(t, err)
	assert.Len(t, members, 2)
}

// TestDeleteSpace_CascadesSoftDeleteToNodesAndDocuments 验证删除 Space 时级联软删其下所有节点和文档。
func TestDeleteSpace_CascadesSoftDeleteToNodesAndDocuments(t *testing.T) {
	txManager, mock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedSpace(t, repo, "space-1", "待删除空间", model.VisibilityTenant)
	assert.NoError(t, repo.AddMember(context.Background(), &model.WikiSpaceMember{
		SpaceID: "space-1",
		UserID:  "user-1",
		Role:    model.SpaceRoleOwner,
	}))

	// 注入节点和文档数据
	repo.ListNodesBySpaceFn = func(ctx context.Context, spaceID string) ([]*model.WikiNode, error) {
		return []*model.WikiNode{
			{ID: "node-1", SpaceID: "space-1", Type: model.NodeTypeDocument, Title: "文档节点"},
			{ID: "node-2", SpaceID: "space-1", Type: model.NodeTypeFolder, Title: "文件夹节点"},
		}, nil
	}
	repo.ListChildNodesFn = func(ctx context.Context, parentID string) ([]*model.WikiNode, error) {
		return nil, nil
	}
	repo.GetDocumentByNodeIDFn = func(ctx context.Context, nodeID string) (*model.Document, error) {
		if nodeID == "node-1" {
			return &model.Document{ID: "doc-1", NodeID: "node-1"}, nil
		}
		return nil, repository.ErrDocumentNotFound
	}

	// 设置事务 mock 期望
	mock.ExpectBegin()
	mock.ExpectCommit()

	svc := NewWikiService(repo, txManager)
	err := svc.DeleteSpace(serviceCtx("tenant-1"), "space-1", "tenant-1")

	assert.NoError(t, err)
	assert.Equal(t, 2, repo.DeleteNodeCalls, "应调用 DeleteNode 两次（两个节点）")
	assert.Equal(t, 1, repo.DeleteDocumentCalls, "应调用 DeleteDocument 一次（仅文档节点有关联文档）")
	assert.GreaterOrEqual(t, repo.CreateTrashItemCalls, 3, "应创建至少 3 条回收站记录（空间+2节点+1文档）")
}

// TestDeleteSpace_RollbackOnError 验证级联删除中途失败时整体回滚。
func TestDeleteSpace_RollbackOnError(t *testing.T) {
	txManager, mock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	seedSpace(t, repo, "space-1", "空间", model.VisibilityTenant)
	assert.NoError(t, repo.AddMember(context.Background(), &model.WikiSpaceMember{
		SpaceID: "space-1",
		UserID:  "user-1",
		Role:    model.SpaceRoleOwner,
	}))

	repo.ListNodesBySpaceFn = func(ctx context.Context, spaceID string) ([]*model.WikiNode, error) {
		return []*model.WikiNode{
			{ID: "node-1", SpaceID: "space-1", Type: model.NodeTypeDocument, Title: "文档"},
		}, nil
	}
	repo.ListChildNodesFn = func(ctx context.Context, parentID string) ([]*model.WikiNode, error) {
		return nil, nil
	}
	repo.GetDocumentByNodeIDFn = func(ctx context.Context, nodeID string) (*model.Document, error) {
		return &model.Document{ID: "doc-1", NodeID: "node-1"}, nil
	}
	repo.DeleteDocumentErr = errors.New("数据库错误")

	// 设置事务 mock 期望：开始事务后回滚
	mock.ExpectBegin()
	mock.ExpectRollback()

	svc := NewWikiService(repo, txManager)
	err := svc.DeleteSpace(serviceCtx("tenant-1"), "space-1", "tenant-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "数据库错误")
}