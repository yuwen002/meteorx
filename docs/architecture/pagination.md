# Pagination / Filter / Sort Architecture

## Overview

Unified pagination, filtering, and sorting mechanism for all list endpoints. Prevents SQL injection via sort field whitelist and provides consistent response format.

## Package: `pkg/pagination`

## PageRequest

```go
type PageRequest struct {
    Page      int    `json:"page"`
    PageSize  int    `json:"page_size"`
    SortBy    string `json:"sort_by,omitempty"`
    SortOrder string `json:"sort_order,omitempty"`
    Keyword   string `json:"keyword,omitempty"`
}
```

### Construction

```go
// From request parameters
page, _ := strconv.Atoi(r.URL.Query().Get("page"))
pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
req := pagination.NewPageRequest(page, pageSize)
// Defaults: page=1, pageSize=20, maxPageSize=100
```

### Constants

```go
const (
    DefaultPage     = 1
    DefaultPageSize = 20
    MaxPageSize     = 100
)
```

## PageResult (Generic)

```go
type PageResult[T any] struct {
    Items    []T   `json:"items"`
    Total    int64 `json:"total"`
    Page     int   `json:"page"`
    PageSize int   `json:"page_size"`
}

// Usage
result := pagination.NewPageResult(items, total, page, pageSize)
// result.Items, result.Total, result.Page, result.PageSize
```

## Query Builder Functions

### ApplySort (with Whitelist Protection)

```go
// IMPORTANT: sortBy MUST be in allowedFields map
// This prevents SQL injection via user-controlled sort fields
allowedSortFields := map[string]string{
    "name":        "name",
    "created_at":  "created_at",
    "updated_at":  "updated_at",
    "status":      "status",
}

query = pagination.ApplySort(query, req.SortBy, req.SortOrder, allowedSortFields)
// - If sortBy is empty → no sorting applied
// - If sortBy is not in whitelist → no sorting applied (silently ignored)
// - SortOrder: "ASC" (default) or "DESC"
```

### ApplyPagination

```go
// Applies OFFSET and LIMIT with bounds checking
query = pagination.ApplyPagination(query, req.Page, req.PageSize)
// - Page <= 0 → defaults to 1
// - PageSize <= 0 → defaults to 20
// - PageSize > 100 → capped at 100
```

### ApplyKeyword

```go
// Multi-field LIKE search
query = pagination.ApplyKeyword(query, req.Keyword, "name", "description", "content")
// - keyword is wrapped in %...%
// - All fields are OR'd together
// - Returns unchanged query if keyword is empty
```

## Complete List Pattern

```go
// In repository
func (r *wikiRepository) ListDocuments(ctx context.Context, req *pagination.PageRequest) (*pagination.PageResult[*model.Document], error) {
    var docs []*model.Document
    var total int64

    // 1. Tenant isolation
    query := tenantctx.FilterQuery(ctx, r.db.WithContext(ctx), "tenant_id")

    // 2. Keyword search (if any)
    query = pagination.ApplyKeyword(query, req.Keyword, "title", "content")

    // 3. Count total (before pagination)
    if err := query.Count(&total).Error; err != nil {
        return nil, err
    }

    // 4. Apply sort (with whitelist)
    allowedSorts := map[string]string{
        "title":      "title",
        "created_at": "created_at",
        "updated_at": "updated_at",
    }
    query = pagination.ApplySort(query, req.SortBy, req.SortOrder, allowedSorts)

    // 5. Apply pagination
    query = pagination.ApplyPagination(query, req.Page, req.PageSize)

    // 6. Execute query
    if err := query.Find(&docs).Error; err != nil {
        return nil, err
    }

    // 7. Return typed result
    return pagination.NewPageResult(docs, total, req.Page, req.PageSize), nil
}
```

## Handler Integration

```go
func (h *WikiHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
    page, _ := strconv.Atoi(r.URL.Query().Get("page"))
    pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
    sortBy := r.URL.Query().Get("sort_by")
    sortOrder := r.URL.Query().Get("sort_order")
    keyword := r.URL.Query().Get("keyword")

    req := pagination.NewPageRequest(page, pageSize)
    req.SortBy = sortBy
    req.SortOrder = sortOrder
    req.Keyword = keyword

    tenantID := contextx.GetTenantID(r.Context())
    result, err := h.svc.ListDocuments(r.Context(), tenantID, req)
    if err != nil {
        response.FailError(w, err)
        return
    }

    response.SuccessWithPagination(w, result.Items, result.Page, result.PageSize, result.Total)
}
```

## API Contract

### Query Parameters

| Parameter | Type | Default | Max | Description |
|-----------|------|---------|-----|-------------|
| `page` | int | 1 | - | Page number (1-based) |
| `page_size` | int | 20 | 100 | Items per page |
| `sort_by` | string | - | - | Sort field (must be whitelisted) |
| `sort_order` | string | ASC | - | ASC or DESC |
| `keyword` | string | - | - | Multi-field search |

### Response

```json
{
  "data": [
    {
      "id": "01H...",
      "title": "Document Title",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 150
  }
}
```

## Sort Field Whitelist

### Why?

User-supplied sort fields can be exploited for SQL injection. The whitelist ensures only known, safe column names are used in ORDER BY clauses.

```go
// BAD (VULNERABLE):
query.Order(r.URL.Query().Get("sort_by"))  // Direct user input → SQL injection!

// GOOD (SAFE):
allowedFields := map[string]string{"name": "name"}
sortField, ok := allowedFields[userInput]
if ok {
    query.Order(sortField)  // Only whitelisted column
}
```

### Defining Allowed Sorts

Each repository defines its own sort field mapping:

```go
// In repository or service
var wikiSpaceSortFields = map[string]string{
    "name":       "name",
    "created_at": "created_at",
    "updated_at": "updated_at",
}

var documentSortFields = map[string]string{
    "title":      "title",
    "created_at": "created_at",
    "updated_at": "updated_at",
    "status":     "status",
}
```

## Compatibility

The package maintains backward compatibility with the original `Pagination` and `PaginatedResult` types:

```go
// Old API (still works)
pg := pagination.NewPagination(page, pageSize)
result := pagination.NewPaginatedResult(data, page, pageSize, total)
```

## Best Practices

1. **Always** whitelist sort fields — never pass user input directly to ORDER BY
2. **Always** use `NewPageRequest()` for defaults — don't manually calculate
3. **Always** apply tenant filtering BEFORE pagination
4. **Always** count total rows before pagination (not after)
5. **Use** `PageResult[T]` for type-safe results in new code
6. **Document** allowed sort fields for each list endpoint
7. **Consider** adding cursor-based pagination for very large datasets in the future