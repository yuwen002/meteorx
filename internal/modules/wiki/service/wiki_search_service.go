package service

import (
	"context"
	"strings"

	"meteorx/internal/modules/wiki/dto"
)

// Search 搜索 Wiki（标题 + 内容，自动过滤无权限内容，支持分页）
func (s *wikiService) Search(ctx context.Context, tenantID string, userID string, query string, spaceID string, page, pageSize int) ([]*dto.SearchResultResp, int64, error) {
	if query == "" {
		return []*dto.SearchResultResp{}, 0, nil
	}

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
