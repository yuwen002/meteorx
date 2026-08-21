package pagination

import (
	"strings"

	"gorm.io/gorm"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type PageRequest struct {
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	SortBy    string `json:"sort_by,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
	Keyword   string `json:"keyword,omitempty"`
}

type SortField struct {
	Field string
	Order string
}

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

func (p *PageRequest) Offset() int {
	return (p.Page - 1) * p.PageSize
}

func (p *PageRequest) Limit() int {
	return p.PageSize
}

type PageResult[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

func NewPageResult[T any](items []T, total int64, page, pageSize int) *PageResult[T] {
	return &PageResult[T]{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
}

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

func ApplyKeyword(query *gorm.DB, keyword string, fields ...string) *gorm.DB {
	if keyword == "" || len(fields) == 0 {
		return query
	}
	kw := "%" + keyword + "%"
	expr := query.Where(fields[0]+" LIKE ?", kw)
	for _, f := range fields[1:] {
		expr = expr.Or(f + " LIKE ?", kw)
	}
	return query.Where(expr)
}

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type PaginatedResult struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

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

func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

func (p *Pagination) Limit() int {
	return p.PageSize
}

func (p *Pagination) SetTotal(total int) {
	p.Total = total
}

func NewPaginatedResult(data interface{}, page, pageSize, total int) *PaginatedResult {
	pg := NewPagination(page, pageSize)
	pg.SetTotal(total)
	return &PaginatedResult{
		Data:       data,
		Pagination: *pg,
	}
}