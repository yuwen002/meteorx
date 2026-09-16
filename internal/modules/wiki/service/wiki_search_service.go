package service

import (
	"context"
	"strings"

	"meteorx/internal/modules/wiki/dto"
	"meteorx/internal/search"
)

// Search 搜索 Wiki（标题 + 内容，自动过滤无权限内容，支持分页）
// 当搜索引擎可用时优先使用 MeiliSearch；不可用时降级为数据库 LIKE 搜索（向后兼容）。
func (s *wikiService) Search(ctx context.Context, tenantID string, userID string, query string, spaceID string, page, pageSize int) ([]*dto.SearchResultResp, int64, error) {
	if query == "" {
		return []*dto.SearchResultResp{}, 0, nil
	}

	// 优先使用全文搜索引擎
	if s.indexer != nil && s.engine != nil && s.engine.IsAvailable() {
		return s.searchWithEngine(ctx, tenantID, userID, query, spaceID, page, pageSize)
	}

	// 降级：数据库 LIKE 搜索
	return s.searchWithDB(ctx, tenantID, userID, query, spaceID, page, pageSize)
}

// searchWithEngine 使用搜索引擎搜索
func (s *wikiService) searchWithEngine(ctx context.Context, tenantID string, userID string, query string, spaceID string, page, pageSize int) ([]*dto.SearchResultResp, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	resp, err := s.engine.Search(ctx, &search.Query{
		Text:     query,
		TenantID: tenantID,
		SpaceID:  spaceID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		// 搜索引擎异常，静默降级到 DB 搜索
		return s.searchWithDB(ctx, tenantID, userID, query, spaceID, page, pageSize)
	}

	results := make([]*dto.SearchResultResp, 0, len(resp.Results))
	for _, r := range resp.Results {
		// 权限过滤
		if err := s.CheckNodePermission(ctx, r.NodeID, userID, "read"); err != nil {
			continue
		}

		results = append(results, &dto.SearchResultResp{
			ID:        r.ID,
			Type:      string(r.Type),
			Title:     r.Title,
			SpaceID:   r.SpaceID,
			NodeID:    r.NodeID,
			Snippet:   r.Snippet,
			Highlight: r.Highlight,
			UpdatedAt: r.UpdatedAt,
			Score:     r.Score,
		})
	}

	return results, resp.Total, nil
}

// searchWithDB 降级：使用数据库 LIKE 搜索
func (s *wikiService) searchWithDB(ctx context.Context, tenantID string, userID string, query string, spaceID string, page, pageSize int) ([]*dto.SearchResultResp, int64, error) {
	titleResults, err := s.repo.SearchNodesByTitle(ctx, tenantID, spaceID, query)
	if err != nil {
		return nil, 0, err
	}

	contentResults, err := s.repo.SearchDocumentsByContent(ctx, tenantID, spaceID, query)
	if err != nil {
		return nil, 0, err
	}

	var results []*dto.SearchResultResp

	for _, node := range titleResults {
		if err := s.CheckNodePermission(ctx, node.ID, userID, "read"); err != nil {
			continue
		}
		results = append(results, &dto.SearchResultResp{
			ID:        node.ID,
			Type:      "node",
			Title:     node.Title,
			SpaceID:   node.SpaceID,
			NodeID:    node.ID,
			Snippet:   "",
			Highlight: node.Title,
			UpdatedAt: node.UpdatedAt,
			Score:     1.0,
		})
	}

	for _, doc := range contentResults {
		node, err := s.repo.GetNodeByID(ctx, doc.NodeID)
		if err != nil {
			continue
		}
		if err := s.CheckNodePermission(ctx, doc.NodeID, userID, "read"); err != nil {
			continue
		}

		snippet := s.generateSnippet(doc.Content, query)
		results = append(results, &dto.SearchResultResp{
			ID:        doc.ID,
			Type:      "document",
			Title:     node.Title,
			SpaceID:   node.SpaceID,
			NodeID:    doc.NodeID,
			Snippet:   snippet,
			Highlight: query,
			UpdatedAt: doc.UpdatedAt,
			Score:     0.8,
		})
	}

	total := int64(len(results))

	start := (page - 1) * pageSize
	if start >= len(results) {
		return []*dto.SearchResultResp{}, total, nil
	}
	end := start + pageSize
	if end > len(results) {
		end = len(results)
	}

	return results[start:end], total, nil
}

// generateSnippet 生成搜索结果的摘要片段（围绕关键字位置）
func (s *wikiService) generateSnippet(content, query string) string {
	maxLength := 200
	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(query)

	idx := strings.Index(lowerContent, lowerQuery)
	if idx == -1 {
		if len(content) > maxLength {
			return content[:maxLength] + "..."
		}
		return content
	}

	start := idx - 50
	if start < 0 {
		start = 0
	}

	end := start + maxLength
	if end > len(content) {
		end = len(content)
	}

	snippet := content[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(content) {
		snippet = snippet + "..."
	}

	return snippet
}