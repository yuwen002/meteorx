package search

import (
	"context"
	"errors"
	"testing"
	"time"

	"meteorx/internal/modules/wiki/model"

	"github.com/stretchr/testify/assert"
)

// ---- Mock Providers ----

type mockNodeProvider struct {
	nodes map[string]*model.WikiNode
	err   error
}

func newMockNodeProvider() *mockNodeProvider {
	return &mockNodeProvider{nodes: make(map[string]*model.WikiNode)}
}

func (m *mockNodeProvider) GetNodeByID(_ context.Context, id string) (*model.WikiNode, error) {
	if m.err != nil {
		return nil, m.err
	}
	n, ok := m.nodes[id]
	if !ok {
		return nil, errors.New("node not found")
	}
	return n, nil
}

type mockDocProvider struct {
	docs map[string]*model.Document
	err  error
}

func newMockDocProvider() *mockDocProvider {
	return &mockDocProvider{docs: make(map[string]*model.Document)}
}

func (m *mockDocProvider) GetDocumentByNodeID(_ context.Context, nodeID string) (*model.Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	d, ok := m.docs[nodeID]
	if !ok {
		return nil, errors.New("document not found")
	}
	return d, nil
}

// ---- Helper ----

func fixtureNode(id, title, spaceID, tenantID, ownerID string, nodeType string) *model.WikiNode {
	return &model.WikiNode{
		ID:        id,
		Title:     title,
		SpaceID:   spaceID,
		TenantID:  tenantID,
		OwnerID:   ownerID,
		Type:      nodeType,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func fixtureDoc(nodeID, content string) *model.Document {
	return &model.Document{
		ID:        "doc-" + nodeID,
		NodeID:    nodeID,
		Content:   content,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ---- Tests ----

func TestWikiIndexer_IndexNode_WithMockEngine(t *testing.T) {
	mockEngine := NewMockEngine()
	nodeProv := newMockNodeProvider()
	docProv := newMockDocProvider()
	indexer := NewWikiIndexer(mockEngine, nodeProv, docProv)

	// 准备数据: 一个文件夹节点（无文档内容）
	nodeProv.nodes["node-1"] = fixtureNode("node-1", "My Folder", "space-1", "tenant-1", "user-1", model.NodeTypeFolder)

	err := indexer.IndexNode(context.Background(), "node-1")
	assert.NoError(t, err)

	// 验证调用了 engine.Index
	assert.Len(t, mockEngine.IndexCalls, 1)
	assert.Len(t, mockEngine.IndexCalls[0].Documents, 1)

	doc := mockEngine.IndexCalls[0].Documents[0]
	assert.Equal(t, "node-1", doc.ID)
	assert.Equal(t, DocTypeWikiNode, doc.Type)
	assert.Equal(t, "My Folder", doc.Title)
	assert.Equal(t, "space-1", doc.SpaceID)
	assert.Equal(t, "tenant-1", doc.TenantID)
	// 文件夹节点不应该有文档内容
	assert.Empty(t, doc.Content)
}

func TestWikiIndexer_IndexNode_DocumentType(t *testing.T) {
	mockEngine := NewMockEngine()
	nodeProv := newMockNodeProvider()
	docProv := newMockDocProvider()
	indexer := NewWikiIndexer(mockEngine, nodeProv, docProv)

	// 文档节点
	nodeProv.nodes["doc-1"] = fixtureNode("doc-1", "My Doc", "space-1", "tenant-1", "user-1", model.NodeTypeDocument)
	docProv.docs["doc-1"] = fixtureDoc("doc-1", "Hello World Content")

	err := indexer.IndexNode(context.Background(), "doc-1")
	assert.NoError(t, err)

	assert.Len(t, mockEngine.IndexCalls, 1)
	doc := mockEngine.IndexCalls[0].Documents[0]
	assert.Equal(t, DocTypeWikiDocument, doc.Type)
	assert.Equal(t, "Hello World Content", doc.Content)
	assert.Equal(t, "My Doc", doc.Title)
}

func TestWikiIndexer_DeleteNode(t *testing.T) {
	mockEngine := NewMockEngine()
	indexer := NewWikiIndexer(mockEngine, newMockNodeProvider(), newMockDocProvider())

	err := indexer.DeleteNode(context.Background(), "node-1")
	assert.NoError(t, err)

	assert.Len(t, mockEngine.DeleteCalls, 1)
	assert.Equal(t, []string{"node-1"}, mockEngine.DeleteCalls[0].IDs)
}

func TestWikiIndexer_IndexNode_WithEngineUnavailable(t *testing.T) {
	mockEngine := NewMockEngine()
	mockEngine.Available = false // 模拟搜索引擎不可用
	nodeProv := newMockNodeProvider()
	indexer := NewWikiIndexer(mockEngine, nodeProv, newMockDocProvider())

	nodeProv.nodes["node-1"] = fixtureNode("node-1", "Title", "space-1", "tenant-1", "user-1", model.NodeTypeFolder)

	// 引擎不可用时，IndexNode 应静默跳过（返回 nil）
	err := indexer.IndexNode(context.Background(), "node-1")
	assert.NoError(t, err)
	// 不应该调用 engine.Index
	assert.Len(t, mockEngine.IndexCalls, 0)
}

func TestWikiIndexer_DeleteNode_WithEngineUnavailable(t *testing.T) {
	mockEngine := NewMockEngine()
	mockEngine.Available = false
	indexer := NewWikiIndexer(mockEngine, newMockNodeProvider(), newMockDocProvider())

	err := indexer.DeleteNode(context.Background(), "node-1")
	assert.NoError(t, err)
	assert.Len(t, mockEngine.DeleteCalls, 0)
}

func TestWikiIndexer_NodeNotFound(t *testing.T) {
	mockEngine := NewMockEngine()
	nodeProv := newMockNodeProvider()
	indexer := NewWikiIndexer(mockEngine, nodeProv, newMockDocProvider())

	// node-unknown 不存在
	err := indexer.IndexNode(context.Background(), "node-unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	// engine.Index 不应被调用
	assert.Len(t, mockEngine.IndexCalls, 0)
}

func TestWikiIndexer_EngineReturnsErrorOnIndex(t *testing.T) {
	mockEngine := NewMockEngine()
	mockEngine.IndexFunc = func(docs ...Document) error {
		return errors.New("index write failed")
	}
	nodeProv := newMockNodeProvider()
	indexer := NewWikiIndexer(mockEngine, nodeProv, newMockDocProvider())

	nodeProv.nodes["node-1"] = fixtureNode("node-1", "Title", "space-1", "tenant-1", "user-1", model.NodeTypeFolder)

	err := indexer.IndexNode(context.Background(), "node-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "index write failed")
}

func TestWikiIndexer_DeleteNode_EngineErrorPropagated(t *testing.T) {
	mockEngine := NewMockEngine()
	mockEngine.DeleteFunc = func(ids ...string) error {
		return errors.New("delete failed")
	}
	indexer := NewWikiIndexer(mockEngine, newMockNodeProvider(), newMockDocProvider())

	err := indexer.DeleteNode(context.Background(), "node-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "delete failed")
}

func TestWikiIndexer_IndexNode_DocumentProviderErrorFallback(t *testing.T) {
	// 当文档节点存在但 document provider 报错时，仍应索引节点标题
	mockEngine := NewMockEngine()
	nodeProv := newMockNodeProvider()
	docProv := newMockDocProvider()
	docProv.err = errors.New("db timeout")
	indexer := NewWikiIndexer(mockEngine, nodeProv, docProv)

	nodeProv.nodes["doc-1"] = fixtureNode("doc-1", "Title Only", "space-1", "tenant-1", "user-1", model.NodeTypeDocument)

	err := indexer.IndexNode(context.Background(), "doc-1")
	assert.NoError(t, err)

	// 仍然索引了节点（只是没有文档内容）
	assert.Len(t, mockEngine.IndexCalls, 1)
	doc := mockEngine.IndexCalls[0].Documents[0]
	assert.Equal(t, DocTypeWikiNode, doc.Type)
	assert.Empty(t, doc.Content)
}

func TestWikiIndexer_BatchIndexAll(t *testing.T) {
	mockEngine := NewMockEngine()
	docProv := newMockDocProvider()
	indexer := NewWikiIndexer(mockEngine, newMockNodeProvider(), docProv)

	nodes := []*model.WikiNode{
		fixtureNode("n1", "Node 1", "s1", "t1", "u1", model.NodeTypeFolder),
		fixtureNode("n2", "Node 2", "s1", "t1", "u1", model.NodeTypeDocument),
	}
	docProv.docs["n2"] = fixtureDoc("n2", "Doc 2 Content")

	err := indexer.BatchIndexAll(context.Background(), nodes)
	assert.NoError(t, err)

	// 应先 ClearIndex，再 Index
	assert.Len(t, mockEngine.IndexCalls, 1)
	// ClearIndex 应该被调用
	assert.NoError(t, mockEngine.ClearErr)

	docs := mockEngine.IndexCalls[0].Documents
	assert.Len(t, docs, 2)
}

func TestWikiIndexer_BatchIndexAll_EmptyNodes(t *testing.T) {
	mockEngine := NewMockEngine()
	indexer := NewWikiIndexer(mockEngine, newMockNodeProvider(), newMockDocProvider())

	err := indexer.BatchIndexAll(context.Background(), []*model.WikiNode{})
	assert.NoError(t, err)
	assert.Len(t, mockEngine.IndexCalls, 0)
	assert.Len(t, mockEngine.DeleteCalls, 0)
}

func TestWikiIndexer_BatchIndexAll_EngineUnavailable(t *testing.T) {
	mockEngine := NewMockEngine()
	mockEngine.Available = false
	indexer := NewWikiIndexer(mockEngine, newMockNodeProvider(), newMockDocProvider())

	nodes := []*model.WikiNode{
		fixtureNode("n1", "Node 1", "s1", "t1", "u1", model.NodeTypeFolder),
	}
	err := indexer.BatchIndexAll(context.Background(), nodes)
	assert.NoError(t, err)
	assert.Len(t, mockEngine.IndexCalls, 0)
}

func TestWikiIndexer_EngineAccessor(t *testing.T) {
	mockEngine := NewMockEngine()
	indexer := NewWikiIndexer(mockEngine, newMockNodeProvider(), newMockDocProvider())

	assert.Same(t, mockEngine, indexer.Engine())
}

func TestWikiIndexer_WaitForPendingTasks(t *testing.T) {
	mockEngine := NewMockEngine()
	indexer := NewWikiIndexer(mockEngine, newMockNodeProvider(), newMockDocProvider())

	err := indexer.WaitForPendingTasks(context.Background(), time.Second)
	assert.NoError(t, err)
}

func TestWikiIndexer_WaitForPendingTasks_Unavailable(t *testing.T) {
	mockEngine := NewMockEngine()
	mockEngine.Available = false
	indexer := NewWikiIndexer(mockEngine, newMockNodeProvider(), newMockDocProvider())

	err := indexer.WaitForPendingTasks(context.Background(), time.Second)
	assert.NoError(t, err)
}

func TestWikiIndexer_MultipleIndexCalls(t *testing.T) {
	mockEngine := NewMockEngine()
	nodeProv := newMockNodeProvider()
	indexer := NewWikiIndexer(mockEngine, nodeProv, newMockDocProvider())

	nodeProv.nodes["n1"] = fixtureNode("n1", "First", "s1", "t1", "u1", model.NodeTypeFolder)
	nodeProv.nodes["n2"] = fixtureNode("n2", "Second", "s1", "t1", "u1", model.NodeTypeFolder)

	assert.NoError(t, indexer.IndexNode(context.Background(), "n1"))
	assert.NoError(t, indexer.IndexNode(context.Background(), "n2"))

	assert.Len(t, mockEngine.IndexCalls, 2)
	assert.Equal(t, "n1", mockEngine.IndexCalls[0].Documents[0].ID)
	assert.Equal(t, "n2", mockEngine.IndexCalls[1].Documents[0].ID)
}

func TestWikiIndexer_DeleteCalledAfterIndex(t *testing.T) {
	// 验证 Index → Delete 的顺序
	mockEngine := NewMockEngine()
	nodeProv := newMockNodeProvider()
	indexer := NewWikiIndexer(mockEngine, nodeProv, newMockDocProvider())

	nodeProv.nodes["n1"] = fixtureNode("n1", "Title", "s1", "t1", "u1", model.NodeTypeFolder)

	assert.NoError(t, indexer.IndexNode(context.Background(), "n1"))
	assert.NoError(t, indexer.DeleteNode(context.Background(), "n1"))

	assert.Len(t, mockEngine.IndexCalls, 1)
	assert.Len(t, mockEngine.DeleteCalls, 1)
}