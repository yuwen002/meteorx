package search

import (
	"context"
	"time"

	"meteorx/internal/modules/wiki/model"
	"meteorx/pkg/logger"
)

// WikiNodeProvider 提供给 Indexer 查询 Wiki 节点/文档详情的接口。
// Indexer 不直接依赖 wiki repository，通过此接口解耦。
type WikiNodeProvider interface {
	GetNodeByID(ctx context.Context, id string) (*model.WikiNode, error)
}

// WikiDocumentProvider Wiki 文档数据提供接口
type WikiDocumentProvider interface {
	GetDocumentByNodeID(ctx context.Context, nodeID string) (*model.Document, error)
}

// WikiIndexer Wiki 文档索引器，在文档创建/更新/删除时同步到搜索引擎。
type WikiIndexer struct {
	engine        Engine
	nodeProvider  WikiNodeProvider
	docProvider   WikiDocumentProvider
}

// NewWikiIndexer 创建 Wiki 索引器。
// engine 为 NoopEngine 时，Index/Delete 为空操作，Search 返回 ErrSearchEngineUnavailable。
func NewWikiIndexer(engine Engine, nodeProvider WikiNodeProvider, docProvider WikiDocumentProvider) *WikiIndexer {
	return &WikiIndexer{
		engine:       engine,
		nodeProvider: nodeProvider,
		docProvider:  docProvider,
	}
}

// IndexNode 将节点及关联文档索引到搜索引擎。
// 对于 document 类型的节点，会同时索引标题和文档内容。
func (idx *WikiIndexer) IndexNode(ctx context.Context, nodeID string) error {
	if !idx.engine.IsAvailable() {
		return nil // 搜索引擎不可用时静默跳过
	}

	node, err := idx.nodeProvider.GetNodeByID(ctx, nodeID)
	if err != nil {
		return err
	}

	doc := Document{
		ID:        node.ID,
		Type:      DocTypeWikiNode,
		Title:     node.Title,
		SpaceID:   node.SpaceID,
		NodeID:    node.ID,
		TenantID:  node.TenantID,
		UpdatedAt: node.UpdatedAt,
		CreatedAt: node.CreatedAt,
		OwnerID:   node.OwnerID,
	}

	// 如果是文档节点，同时索引文档内容
	if node.Type == model.NodeTypeDocument {
		document, err := idx.docProvider.GetDocumentByNodeID(ctx, node.ID)
		if err == nil && document != nil {
			doc.Content = document.Content
			doc.Type = DocTypeWikiDocument
			if document.UpdatedAt != nil {
				doc.UpdatedAt = *document.UpdatedAt
			}
		}
	}

	return idx.engine.Index(ctx, doc)
}

// DeleteNode 从搜索引擎中删除节点索引
func (idx *WikiIndexer) DeleteNode(ctx context.Context, nodeID string) error {
	if !idx.engine.IsAvailable() {
		return nil
	}
	return idx.engine.Delete(ctx, nodeID)
}

// BatchIndexAll 批量重建所有索引（启动时或数据迁移后调用）。
// 注意：数据量大时可能耗时较长，建议在后台 goroutine 中执行。
func (idx *WikiIndexer) BatchIndexAll(ctx context.Context, nodes []*model.WikiNode) error {
	if !idx.engine.IsAvailable() {
		logger.Warn("Search engine unavailable, skipping batch index")
		return nil
	}

	if len(nodes) == 0 {
		return nil
	}

	// 先清空再重建
	if err := idx.engine.ClearIndex(ctx); err != nil {
		return err
	}

	docs := make([]Document, 0, len(nodes))
	for _, node := range nodes {
		doc := Document{
			ID:        node.ID,
			Type:      DocTypeWikiNode,
			Title:     node.Title,
			SpaceID:   node.SpaceID,
			NodeID:    node.ID,
			TenantID:  node.TenantID,
			UpdatedAt: node.UpdatedAt,
			CreatedAt: node.CreatedAt,
			OwnerID:   node.OwnerID,
		}

		// 尝试获取文档内容
		if node.Type == model.NodeTypeDocument {
			document, err := idx.docProvider.GetDocumentByNodeID(ctx, node.ID)
			if err == nil && document != nil {
				doc.Content = document.Content
				doc.Type = DocTypeWikiDocument
				if document.UpdatedAt != nil {
					doc.UpdatedAt = *document.UpdatedAt
				}
			}
		}

		docs = append(docs, doc)

		// 分批写入，每批 100 条
		if len(docs) >= 100 {
			if err := idx.engine.Index(ctx, docs...); err != nil {
				return err
			}
			docs = docs[:0]
		}
	}

	// 写入剩余
	if len(docs) > 0 {
		if err := idx.engine.Index(ctx, docs...); err != nil {
			return err
		}
	}

	logger.Infof("Batch indexed %d wiki nodes to search engine", len(nodes))
	return nil
}

// WaitForPendingTasks 等待所有待处理索引任务完成（最大等待时间）。
// MeiliSearch 是异步索引的，此方法确保数据可被搜索到。
func (idx *WikiIndexer) WaitForPendingTasks(ctx context.Context, timeout time.Duration) error {
	if !idx.engine.IsAvailable() {
		return nil
	}

	// MeiliSearch 的 AddDocuments 默认会等待任务完成再返回
	// 此处留出接口方便未来扩展
	return nil
}