package search

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEngine_Noop_WhenProviderNone(t *testing.T) {
	engine := NewEngine(Config{Provider: "none"})
	assert.NotNil(t, engine)
	assert.False(t, engine.IsAvailable())
	assert.NoError(t, engine.Close())
}

func TestNewEngine_Noop_WhenProviderEmpty(t *testing.T) {
	engine := NewEngine(Config{Provider: ""})
	assert.NotNil(t, engine)
	assert.False(t, engine.IsAvailable())
}

func TestNewEngine_Noop_WhenProviderMeilisearchButNoHost(t *testing.T) {
	// 有 provider 但没配 host → 降级为 Noop（不 panic）
	engine := NewEngine(Config{Provider: "meilisearch", Host: ""})
	assert.NotNil(t, engine)
	assert.False(t, engine.IsAvailable())
}

func TestNewEngine_Noop_WhenUnknownProvider(t *testing.T) {
	engine := NewEngine(Config{Provider: "elasticsearch"})
	assert.NotNil(t, engine)
	assert.False(t, engine.IsAvailable())
}

func TestNoopEngine_Index_ReturnsNil(t *testing.T) {
	engine := NewEngine(Config{Provider: "none"})
	err := engine.Index(context.Background(), Document{ID: "1", Title: "test"})
	assert.NoError(t, err)
}

func TestNoopEngine_Delete_ReturnsNil(t *testing.T) {
	engine := NewEngine(Config{Provider: "none"})
	err := engine.Delete(context.Background(), "1", "2")
	assert.NoError(t, err)
}

func TestNoopEngine_Search_ReturnsUnavailableError(t *testing.T) {
	engine := NewEngine(Config{Provider: "none"})
	resp, err := engine.Search(context.Background(), &Query{Text: "hello"})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, ErrSearchEngineUnavailable))
}

func TestNoopEngine_ClearIndex_ReturnsNil(t *testing.T) {
	engine := NewEngine(Config{Provider: "none"})
	err := engine.ClearIndex(context.Background())
	assert.NoError(t, err)
}

func TestNoopEngine_Close_ReturnsNil(t *testing.T) {
	engine := NewEngine(Config{Provider: "none"})
	err := engine.Close()
	assert.NoError(t, err)
}

func TestNewEngine_DefaultIndexPrefix(t *testing.T) {
	// 即使 provider=none，index_prefix 配置也应被正确解析（Noop 不依赖 prefix，
	// 但确保调用不 panic）
	engine := NewEngine(Config{Provider: "none", IndexPrefix: ""})
	assert.NotNil(t, engine)
}

func TestDocument_Fields(t *testing.T) {
	// 验证 Document 结构体字段类型正确（编译期保证）
	doc := Document{
		ID:       "doc-1",
		Type:     DocTypeWikiNode,
		Title:    "Test Title",
		Content:  "Test Content",
		SpaceID:  "space-1",
		NodeID:   "node-1",
		TenantID: "tenant-1",
	}
	assert.Equal(t, "doc-1", doc.ID)
	assert.Equal(t, DocTypeWikiNode, doc.Type)
	assert.Equal(t, "Test Title", doc.Title)
}

func TestQuery_Defaults(t *testing.T) {
	q := &Query{Text: "search term"}
	assert.Equal(t, "search term", q.Text)
	assert.Equal(t, 0, q.Page)
	assert.Equal(t, 0, q.PageSize)
	assert.Empty(t, q.Types)
}

func TestDocTypeConstants(t *testing.T) {
	assert.Equal(t, DocType("wiki_node"), DocTypeWikiNode)
	assert.Equal(t, DocType("wiki_document"), DocTypeWikiDocument)
}

func TestErrorVariables(t *testing.T) {
	assert.Error(t, ErrSearchEngineUnavailable)
	assert.Error(t, ErrIndexNotFound)
	assert.Contains(t, ErrSearchEngineUnavailable.Error(), "not available")
}