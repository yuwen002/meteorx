// Package search 提供全文检索引擎抽象，支持 MeiliSearch 与数据库降级。
//
// 使用方式：
//
//	// 在启动时根据配置创建引擎
//	engine := search.NewEngine(cfg.Search)
//
//	// Wiki 模块初始化时注入 Indexer
//	indexer := search.NewWikiIndexer(engine, wikiRepo)
//
//	// 文档变更时同步索引
//	indexer.IndexDocument(ctx, doc)
//	indexer.DeleteDocument(ctx, docID)
//
//	// 搜索
//	results, err := engine.Search(ctx, &search.Query{...})
package search

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// DocType 文档类型
type DocType string

const (
	DocTypeWikiNode     DocType = "wiki_node"
	DocTypeWikiDocument DocType = "wiki_document"
)

// Document 搜索引擎中的文档
type Document struct {
	ID        string    `json:"id"`
	Type      DocType   `json:"type"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	SpaceID   string    `json:"space_id"`
	NodeID    string    `json:"node_id"`
	TenantID  string    `json:"tenant_id"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
	OwnerID   string    `json:"owner_id"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Document
	Snippet   string  `json:"snippet"`
	Highlight string  `json:"highlight"`
	Score     float64 `json:"score"`
}

// Query 搜索查询
type Query struct {
	Text     string `json:"q"`
	TenantID string `json:"tenant_id"`
	SpaceID  string `json:"space_id,omitempty"`
	// Types 限定文档类型，为空时不限
	Types    []DocType `json:"types,omitempty"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
}

// SearchResponse 搜索结果响应
type SearchResponse struct {
	Results  []SearchResult `json:"results"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// Engine 搜索引擎接口
type Engine interface {
	// Index 创建或更新文档索引
	Index(ctx context.Context, docs ...Document) error
	// Delete 从索引中删除文档
	Delete(ctx context.Context, ids ...string) error
	// Search 执行搜索
	Search(ctx context.Context, q *Query) (*SearchResponse, error)
	// ClearIndex 清空索引（用于重建）
	ClearIndex(ctx context.Context) error
	// Close 关闭搜索引擎连接
	Close() error
	// IsAvailable 搜索引擎是否可用
	IsAvailable() bool
}

var (
	ErrSearchEngineUnavailable = errors.New("search engine is not available")
	ErrIndexNotFound           = errors.New("index not found")
)

// Provider 搜索引擎提供商类型
type Provider string

const (
	ProviderMeiliSearch Provider = "meilisearch"
	ProviderNone        Provider = "none"
)

// Config 搜索引擎配置
type Config struct {
	Provider    string `json:"provider"`
	Host        string `json:"host"`
	APIKey      string `json:"api_key"`
	IndexPrefix string `json:"index_prefix"`
}

// NewEngine 根据配置创建搜索引擎实例。
// provider 为 "none" 或空时返回 NoopEngine（降级为数据库搜索，不报错）。
func NewEngine(cfg Config) Engine {
	prefix := cfg.IndexPrefix
	if prefix == "" {
		prefix = "meteorx_"
	}

	switch Provider(cfg.Provider) {
	case ProviderMeiliSearch:
		if cfg.Host == "" {
			// 有 provider 但没配 host，警告后降级
			return newNoopEngine(fmt.Errorf("meilisearch host is empty, falling back to database search"))
		}
		e, err := newMeiliSearchEngine(cfg.Host, cfg.APIKey, prefix)
		if err != nil {
			return newNoopEngine(fmt.Errorf("failed to create meilisearch engine: %w", err))
		}
		return e
	case ProviderNone:
		return newNoopEngine(nil)
	default:
		return newNoopEngine(nil)
	}
}