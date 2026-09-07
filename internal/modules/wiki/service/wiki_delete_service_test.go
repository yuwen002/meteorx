package service

import (
	"context"
	"errors"
	"testing"

	"meteorx/internal/modules/wiki/model"
	"meteorx/internal/modules/wiki/repository"

	"github.com/stretchr/testify/assert"
)

// TestDeleteDocument_PreservesAttachmentsForRestore 文档软删必须保留附件记录，
// 保证从回收站恢复后附件仍可访问（旧实现物理删除附件导致无法恢复）
func TestDeleteDocument_PreservesAttachmentsForRestore(t *testing.T) {
	txManager, sqlMock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	repo.SeedDocument(&model.Document{ID: "doc-1", NodeID: "node-1"})
	repo.GetNodeFn = func(ctx context.Context, id string) (*model.WikiNode, error) {
		return &model.WikiNode{ID: id, SpaceID: "space-1", Title: "标题", Type: model.NodeTypeDocument}, nil
	}
	sqlMock.ExpectBegin()
	sqlMock.ExpectCommit()
	svc := NewWikiService(repo, txManager)

	err := svc.DeleteDocument(serviceCtx("tenant-1"), "doc-1")

	assert.NoError(t, err)
	assert.Equal(t, 1, repo.DeleteDocumentCalls)
	// 关键断言：不再物理删除附件
	assert.Equal(t, 0, repo.DeleteAttachmentsByDocumentCalls)
	// 回收站记录已创建
	if assert.NotNil(t, repo.LastCreatedTrash) {
		assert.Equal(t, model.TrashTypeDocument, repo.LastCreatedTrash.ItemType)
		assert.Equal(t, "doc-1", repo.LastCreatedTrash.ItemID)
		assert.Equal(t, "space-1", repo.LastCreatedTrash.SpaceID)
	}
}

// TestDeleteDocument_DatabaseErrorRollsBack 软删遇到数据库错误时事务回滚
func TestDeleteDocument_DatabaseErrorRollsBack(t *testing.T) {
	txManager, sqlMock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	repo.SeedDocument(&model.Document{ID: "doc-1", NodeID: "node-1"})
	repo.GetNodeFn = func(ctx context.Context, id string) (*model.WikiNode, error) {
		return &model.WikiNode{ID: id, SpaceID: "space-1", Title: "标题", Type: model.NodeTypeDocument}, nil
	}
	repo.DeleteDocumentErr = errors.New("delete db down")
	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()
	svc := NewWikiService(repo, txManager)

	err := svc.DeleteDocument(serviceCtx("tenant-1"), "doc-1")

	assert.EqualError(t, err, "delete db down")
	assert.Equal(t, 1, repo.DeleteDocumentCalls)
}

// TestDeleteNode_CascadeKeepsDocumentAttachments 删除文档型节点：
// 级联软删文档但不物理清除附件，附件在文档恢复后保持完整
func TestDeleteNode_CascadeKeepsDocumentAttachments(t *testing.T) {
	txManager, sqlMock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	repo.SeedDocument(&model.Document{ID: "doc-1", NodeID: "node-1"})
	repo.GetNodeFn = func(ctx context.Context, id string) (*model.WikiNode, error) {
		return &model.WikiNode{ID: id, SpaceID: "space-1", Title: "标题", Type: model.NodeTypeDocument}, nil
	}
	sqlMock.ExpectBegin()
	sqlMock.ExpectCommit()
	svc := NewWikiService(repo, txManager)

	err := svc.DeleteNode(serviceCtx("tenant-1"), "node-1")

	assert.NoError(t, err)
	assert.Equal(t, 1, repo.DeleteNodeCalls)
	assert.Equal(t, 1, repo.DeleteDocumentCalls)
	assert.Equal(t, 0, repo.DeleteAttachmentsByDocumentCalls)
	if assert.NotNil(t, repo.LastCreatedTrash) {
		assert.Equal(t, model.TrashTypeNode, repo.LastCreatedTrash.ItemType)
		assert.Equal(t, "node-1", repo.LastCreatedTrash.ItemID)
	}
}

// TestDeleteNode_DocumentDeleteErrorAbortsAndRollsBack 级联中文档软删失败：
// 错误向外传播且整个节点删除事务回滚（不再静默吞错）
func TestDeleteNode_DocumentDeleteErrorAbortsAndRollsBack(t *testing.T) {
	txManager, sqlMock := newTxManager(t)
	repo := repository.NewMockWikiRepository()
	repo.SeedDocument(&model.Document{ID: "doc-1", NodeID: "node-1"})
	repo.GetNodeFn = func(ctx context.Context, id string) (*model.WikiNode, error) {
		return &model.WikiNode{ID: id, SpaceID: "space-1", Title: "标题", Type: model.NodeTypeDocument}, nil
	}
	repo.DeleteDocumentErr = errors.New("delete db down")
	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()
	svc := NewWikiService(repo, txManager)

	err := svc.DeleteNode(serviceCtx("tenant-1"), "node-1")

	assert.EqualError(t, err, "delete db down")
	assert.Equal(t, 1, repo.DeleteDocumentCalls)
	// 文档软删失败时不允许继续删除节点
	assert.Equal(t, 0, repo.DeleteNodeCalls)
}
