// Package pagination 提供通用分页工具，封装分页参数解析、结果封装和 GORM 查询应用。
package pagination

import (
	"strings"

	"gorm.io/gorm"
)

const (
	// DefaultPage 默认页码
	DefaultPage = 1
	// DefaultPageSize 默认每页条数
	DefaultPageSize = 20
	// MaxPageSize 每页条数上限
	MaxPageSize = 100
)

// PageRequest 分页请求参数，包含页码、每页条数、排序与关键字。
type PageRequest struct {
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
	Keyword   string `json:"keyword,omitempty"`
}

// SortField 排序字段及其方向。
type SortField struct {
	Field string
	Order string
}

// NewPageRequest 创建并规范化分页请求（页码/每页条数越界时回退默认值）。
func NewPageRequest(page, pageSize int) *PageRequest {
	if page <= 0 {
		page = DefaultPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return &PageRequest{
		Page:     page,
		PageSize: pageSize,
	}
}

// Offset 计算当前页的起始偏移量。
func (p *PageRequest) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit 返回每页条数。
func (p *PageRequest) Limit() int {
	return p.PageSize
}

// PageResult 泛型分页结果，包含数据项、总数与分页信息。
type PageResult[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// NewPageResult 构造一个泛型分页结果。
func NewPageResult[T any](items []T, total int64, page, pageSize int) *PageResult[T] {
	return &PageResult[T]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}

// ApplySort 根据排序字段与方向为查询追加 ORDER BY（仅允许白名单字段）。
func ApplySort(query *gorm.DB, sortBy, sortOrder string, allowedFields map[string]string) *gorm.DB {
	if sortBy == "" {
		return query
	}
	column, ok := allowedFields[sortBy]
	if !ok {
		return query
	}
	order := "ASC"
	if strings.EqualFold(sortOrder, "desc") {
		order = "DESC"
	}
	return query.Order(column + " " + order)
}

// ApplyPagination 为查询追加 OFFSET/LIMIT。
func ApplyPagination(query *gorm.DB, page, pageSize int) *gorm.DB {
	if page <= 0 {
		page = DefaultPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return query.Offset((page - 1) * pageSize).Limit(pageSize)
}

// ApplyKeyword 为查询追加多字段 LIKE 关键字模糊匹配。
func ApplyKeyword(query *gorm.DB, keyword string, fields ...string) *gorm.DB {
	if keyword == "" || len(fields) == 0 {
		return query
	}
	kw := "%" + keyword + "%"
	expr := query.Where(fields[0]+" LIKE ?", kw)
	for _, f := range fields[1:] {
		expr = expr.Or(f+" LIKE ?", kw)
	}
	return query.Where(expr)
}

// Pagination 分页元信息。
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// PaginatedResult 带分页信息的数据包裹（响应体 {data,pagination}）。
type PaginatedResult struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// NewPagination 创建并规范化分页参数。
func NewPagination(page, pageSize int) *Pagination {
	if page <= 0 {
		page = DefaultPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return &Pagination{
		Page:     page,
		PageSize: pageSize,
	}
}

// Offset 计算当前页的起始偏移量。
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit 返回每页条数。
func (p *Pagination) Limit() int {
	return p.PageSize
}

// SetTotal 设置总条数。
func (p *Pagination) SetTotal(total int) {
	p.Total = total
}

// NewPaginatedResult 构造带分页包裹的结果对象。
func NewPaginatedResult(data interface{}, page, pageSize, total int) *PaginatedResult {
	pg := NewPagination(page, pageSize)
	pg.SetTotal(total)
	return &PaginatedResult{
		Data:       data,
		Pagination: *pg,
	}
}
