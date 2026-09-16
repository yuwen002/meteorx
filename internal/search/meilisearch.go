package search

import (
	"context"
	"fmt"
	"time"

	"meteorx/pkg/logger"

	"github.com/meilisearch/meilisearch-go"
)

const (
	// wikiIndex 文档索引名称（前缀 + "wiki"）
	wikiIndex = "wiki"

	// defaultSearchLimit 默认搜索限制
	defaultSearchLimit = 20

	// maxSearchLimit 最大搜索限制
	maxSearchLimit = 100
)

type meiliSearchEngine struct {
	client meilisearch.ServiceManager
	prefix string
	avail  bool
}

func newMeiliSearchEngine(host, apiKey, prefix string) (*meiliSearchEngine, error) {
	client := meilisearch.New(host, meilisearch.WithAPIKey(apiKey))

	// 健康检查
	health, err := client.Health()
	if err != nil {
		return nil, fmt.Errorf("meilisearch health check failed: %w", err)
	}

	e := &meiliSearchEngine{
		client: client,
		prefix: prefix,
		avail:  true,
	}

	logger.Infof("MeiliSearch connected: status=%s", health.Status)

	// 确保索引存在（索引在首次添加文档时自动创建，但提前创建方便配置）
	if err := e.ensureIndex(wikiIndex); err != nil {
		logger.Warnf("Failed to ensure search index, will auto-create on first write: %v", err)
	}

	return e, nil
}

func (e *meiliSearchEngine) ensureIndex(indexName string) error {
	fullName := e.prefix + indexName
	_, err := e.client.Index(fullName).UpdateFilterableAttributes(
		&[]interface{}{"tenant_id", "space_id", "type", "node_id"},
	)
	if err != nil {
		return err
	}

	// 配置可排序字段
	_, err = e.client.Index(fullName).UpdateSortableAttributes(
		&[]string{"updated_at", "created_at"},
	)
	if err != nil {
		logger.Warnf("Failed to configure sortable attributes: %v", err)
	}

	// 配置搜索able属性（权重）
	_, err = e.client.Index(fullName).UpdateRankingRules(
		&[]string{
			"words",
			"typo",
			"proximity",
			"attribute",
			"sort",
			"exactness",
		},
	)
	if err != nil {
		logger.Warnf("Failed to configure ranking rules: %v", err)
	}

	return nil
}

func (e *meiliSearchEngine) fullIndexName() string {
	return e.prefix + wikiIndex
}

func (e *meiliSearchEngine) Index(ctx context.Context, docs ...Document) error {
	if !e.avail {
		return ErrSearchEngineUnavailable
	}

	if len(docs) == 0 {
		return nil
	}

	// 将内部 Document 转换为 map 方便发送
	docMaps := make([]interface{}, len(docs))
	for i, doc := range docs {
		docMaps[i] = map[string]interface{}{
			"id":         doc.ID,
			"type":       string(doc.Type),
			"title":      doc.Title,
			"content":    doc.Content,
			"space_id":   doc.SpaceID,
			"node_id":    doc.NodeID,
			"tenant_id":  doc.TenantID,
			"updated_at": doc.UpdatedAt.Unix(),
			"created_at": doc.CreatedAt.Unix(),
			"owner_id":   doc.OwnerID,
		}
	}

	_, err := e.client.Index(e.fullIndexName()).AddDocuments(docMaps, nil)
	if err != nil {
		return fmt.Errorf("meilisearch index failed: %w", err)
	}

	logger.Debugf("Indexed %d document(s) to MeiliSearch", len(docs))
	return nil
}

func (e *meiliSearchEngine) Delete(ctx context.Context, ids ...string) error {
	if !e.avail {
		return ErrSearchEngineUnavailable
	}

	if len(ids) == 0 {
		return nil
	}

	_, err := e.client.Index(e.fullIndexName()).DeleteDocuments(ids, nil)
	if err != nil {
		return fmt.Errorf("meilisearch delete failed: %w", err)
	}

	logger.Debugf("Deleted %d document(s) from MeiliSearch", len(ids))
	return nil
}

func (e *meiliSearchEngine) Search(ctx context.Context, q *Query) (*SearchResponse, error) {
	if !e.avail {
		return nil, ErrSearchEngineUnavailable
	}

	limit := q.PageSize
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	offset := (q.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	// 构建搜索请求
	searchReq := &meilisearch.SearchRequest{
		Offset:                int64(offset),
		Limit:                 int64(limit),
		Filter:                e.buildFilter(q),
		Sort:                  []string{"updated_at:desc"},
		AttributesToHighlight: []string{"title", "content"},
		HighlightPreTag:       "<em>",
		HighlightPostTag:      "</em>",
		ShowMatchesPosition:   false,
	}

	resp, err := e.client.Index(e.fullIndexName()).Search(q.Text, searchReq)
	if err != nil {
		return nil, fmt.Errorf("meilisearch search failed: %w", err)
	}

	// 解析结果 — Hit 类型为 map[string]json.RawMessage，使用 DecodeInto 解析
	type hitDoc struct {
		ID        string  `json:"id"`
		Type      string  `json:"type"`
		Title     string  `json:"title"`
		Content   string  `json:"content"`
		SpaceID   string  `json:"space_id"`
		NodeID    string  `json:"node_id"`
		TenantID  string  `json:"tenant_id"`
		OwnerID   string  `json:"owner_id"`
		UpdatedAt float64 `json:"updated_at"`
		CreatedAt float64 `json:"created_at"`
		Score     float64 `json:"_score"`
		Formatted *struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"_formatted"`
	}

	results := make([]SearchResult, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		var doc hitDoc
		if err := hit.DecodeInto(&doc); err != nil {
			continue
		}
		result := SearchResult{
			Document: Document{
				ID:        doc.ID,
				Type:      DocType(doc.Type),
				Title:     doc.Title,
				Content:   doc.Content,
				SpaceID:   doc.SpaceID,
				NodeID:    doc.NodeID,
				TenantID:  doc.TenantID,
				OwnerID:   doc.OwnerID,
				UpdatedAt: time.Unix(int64(doc.UpdatedAt), 0),
				CreatedAt: time.Unix(int64(doc.CreatedAt), 0),
			},
			Score: doc.Score,
		}
		if doc.Formatted != nil {
			result.Highlight = doc.Formatted.Title
			result.Snippet = truncateContent(doc.Formatted.Content, 200)
		}
		results = append(results, result)
	}

	return &SearchResponse{
		Results:  results,
		Total:    resp.EstimatedTotalHits,
		Page:     q.Page,
		PageSize: limit,
	}, nil
}

func (e *meiliSearchEngine) buildFilter(q *Query) string {
	var filter string

	// 租户隔离是硬性过滤
	if q.TenantID != "" {
		filter = fmt.Sprintf("tenant_id = %s", q.TenantID)
	}

	// 空间过滤
	if q.SpaceID != "" {
		if filter != "" {
			filter += " AND "
		}
		filter += fmt.Sprintf("space_id = %s", q.SpaceID)
	}

	// 文档类型过滤
	if len(q.Types) > 0 {
		if filter != "" {
			filter += " AND "
		}
		typeFilter := ""
		for i, t := range q.Types {
			if i > 0 {
				typeFilter += " OR "
			}
			typeFilter += fmt.Sprintf("type = %s", string(t))
		}
		filter += "(" + typeFilter + ")"
	}

	return filter
}

func truncateContent(content string, maxLen int) string {
	runes := []rune(content)
	if len(runes) <= maxLen {
		return content
	}
	return string(runes[:maxLen]) + "..."
}

func (e *meiliSearchEngine) ClearIndex(ctx context.Context) error {
	if !e.avail {
		return ErrSearchEngineUnavailable
	}

	_, err := e.client.Index(e.fullIndexName()).DeleteAllDocuments(nil)
	if err != nil {
		return fmt.Errorf("meilisearch clear index failed: %w", err)
	}

	logger.Info("MeiliSearch index cleared")
	return nil
}

func (e *meiliSearchEngine) Close() error {
	e.avail = false
	logger.Info("MeiliSearch engine closed")
	return nil
}

func (e *meiliSearchEngine) IsAvailable() bool {
	return e.avail
}
