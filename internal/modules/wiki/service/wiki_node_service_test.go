package service

import (
	"errors"
	"testing"

	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"
	apperrors "meteorx/internal/pkg/apperrors"

	"github.com/stretchr/testify/assert"
)

// TestMoveNode_SelfMoveRejected 验证节点不能移动到自身。
func TestMoveNode_SelfMoveRejected(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	svc := NewWikiService(repo, txManager)

	err := svc.MoveNode(serviceCtx("tenant-1"), "node-1", "node-1", "user-1")

	var appErr *apperrors.AppError
	assert.True(t, errors.As(err, &appErr), "error should be AppError type")
	if appErr != nil {
		assert.Equal(t, 400, appErr.StatusCode)
		assert.Equal(t, "不能将节点移动到自身", appErr.Message)
	}
}

// TestMoveNode_TargetNotFolderRejected 验证节点只能移动到文件夹下。
func TestMoveNode_TargetNotFolderRejected(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	svc := NewWikiService(repo, txManager)

	err := svc.MoveNode(serviceCtx("tenant-1"), "node-1", "doc-node", "user-1")

	var appErr *apperrors.AppError
	assert.True(t, errors.As(err, &appErr), "error should be AppError type")
	if appErr != nil {
		assert.Equal(t, 400, appErr.StatusCode)
		assert.Equal(t, "目标父节点必须是文件夹", appErr.Message)
	}
}

// TestCreateNode_CreatesFolder 验证创建文件夹类型节点。
func TestCreateNode_CreatesFolder(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	svc := NewWikiService(repo, txManager)

	resp, err := svc.CreateNode(serviceCtx("tenant-1"), "space-1", "user-1", &dto.CreateWikiNodeReq{
		Type:  model.NodeTypeFolder,
		Title: "需求文档",
		Sort:  1,
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, model.NodeTypeFolder, resp.Type)
	assert.Equal(t, "需求文档", resp.Title)
	assert.Equal(t, int64(0), resp.ChildCount)
}

// TestDeleteNode_CommitsWithTrash 验证删除节点时事务正常提交（回收站记录 + 级联删除）。
func TestDeleteNode_CommitsWithTrash(t *testing.T) {
	txManager, mock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	svc := NewWikiService(repo, txManager)

	mock.ExpectBegin()
	mock.ExpectCommit()

	err := svc.DeleteNode(serviceCtx("tenant-1"), "node-1")

	assert.NoError(t, err)
}

// TestSortNode_UpdatesSort 验证节点排序更新无错误。
func TestSortNode_UpdatesSort(t *testing.T) {
	txManager, _ := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	svc := NewWikiService(repo, txManager)

	err := svc.SortNode(serviceCtx("tenant-1"), "node-1", 5, "user-1")

	assert.NoError(t, err)
}
