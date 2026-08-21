# 分页 / 过滤 / 排序架构

## 概述

为所有列表接口提供统一的分页、过滤和排序机制。通过排序字段白名单防止 SQL 注入，提供一致的响应格式。

## 包路径：`pkg/pagination`

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

### 构造方式

```go
// 从请求参数构建
page, _ := strconv.Atoi(r.URL.Query().Get("page"))
pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
req := pagination.NewPageRequest(page, pageSize)
// 默认值：page=1, pageSize=20, maxPageSize=100
```

### 常量

```go
const (
    DefaultPage     = 1
    DefaultPageSize = 20
    MaxPageSize     = 100
)
```

## PageResult（泛型）

```go
type PageResult[T any] struct {
    Items    []T   `json:"items"`
    Total    int64 `json:"total"`
    Page     int   `json:"page"`
    PageSize int   `json:"page_size"`
}

// 使用方式
result := pagination.NewPageResult(items, total, page, pageSize)
// result.Items, result.Total, result.Page, result.PageSize
```

## 查询构建器函数

### ApplySort（带白名单保护）

```go
// 重要：sortBy 必须在 allowedFields 映射中
// 这可以防止用户可控的排序字段导致 SQL 注入
allowedSortFields := map[string]string{
    "name":        "name",
    "created_at":  "created_at",
    "updated_at":  "updated_at",
    "status":      "status",
}

query = pagination.ApplySort(query, req.SortBy, req.SortOrder, allowedSortFields)
// - 如果 sortBy 为空 → 不应用排序
// - 如果 sortBy 不在白名单 → 不应用排序（静默忽略）
// - SortOrder: "ASC"（默认）或 "DESC"
```

### ApplyPagination

```go
// 应用 OFFSET 和 LIMIT，带边界检查
query = pagination.ApplyPagination(query, req.Page, req.PageSize)
// - Page <= 0 → 默认为 1
// - PageSize <= 0 → 默认为 20
// - PageSize > 100 → 限制为 100
```

### ApplyKeyword

```go
// 多字段 LIKE 搜索
query = pagination.ApplyKeyword(query, req.Keyword, "name", "description", "content")
// - keyword 被包裹在 %...% 中
// - 所有字段以 OR 方式组合
// - 如果 keyword 为空，返回不变的 query
```

## 完整列表模式

```go
// 在 repository 中
func (r *wikiRepository) ListDocuments(ctx context.Context, req *pagination.PageRequest) (*pagination.PageResult[*model.Document], error) {
    var docs []*model.Document
    var total int64

    // 1. 租户隔离
    query := tenantctx.FilterQuery(ctx, r.db.WithContext(ctx), "tenant_id")

    // 2. 关键字搜索（如果有）
    query = pagination.ApplyKeyword(query, req.Keyword, "title", "content")

    // 3. 统计总数（分页前）
    if err := query.Count(&total).Error; err != nil {
        return nil, err
    }

    // 4. 应用排序（带白名单）
    allowedSorts := map[string]string{
        "title":      "title",
        "created_at": "created_at",
        "updated_at": "updated_at",
    }
    query = pagination.ApplySort(query, req.SortBy, req.SortOrder, allowedSorts)

    // 5. 应用分页
    query = pagination.ApplyPagination(query, req.Page, req.PageSize)

    // 6. 执行查询
    if err := query.Find(&docs).Error; err != nil {
        return nil, err
    }

    // 7. 返回类型化结果
    return pagination.NewPageResult(docs, total, req.Page, req.PageSize), nil
}
```

## Handler 层集成

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

## API 契约

### 查询参数

| 参数 | 类型 | 默认值 | 最大值 | 说明 |
|------|------|--------|--------|------|
| `page` | int | 1 | - | 页码（从 1 开始） |
| `page_size` | int | 20 | 100 | 每页条数 |
| `sort_by` | string | - | - | 排序字段（必须在白名单中） |
| `sort_order` | string | ASC | - | ASC 或 DESC |
| `keyword` | string | - | - | 多字段搜索 |

### 响应

```json
{
  "data": [
    {
      "id": "01H...",
      "title": "文档标题",
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

## 排序字段白名单

### 为什么需要白名单？

用户提供的排序字段可能被利用进行 SQL 注入。白名单确保只有已知的、安全的列名被用于 ORDER BY 子句。

```go
// 危险（有漏洞）：
query.Order(r.URL.Query().Get("sort_by"))  // 直接使用用户输入 → SQL 注入！

// 安全（推荐）：
allowedFields := map[string]string{"name": "name"}
sortField, ok := allowedFields[userInput]
if ok {
    query.Order(sortField)  // 仅使用白名单中的列
}
```

### 定义允许的排序字段

每个 repository 定义自己的排序字段映射：

```go
// 在 repository 或 service 中
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

## 兼容性

该包与原来的 `Pagination` 和 `PaginatedResult` 类型保持向后兼容：

```go
// 旧 API（仍然可用）
pg := pagination.NewPagination(page, pageSize)
result := pagination.NewPaginatedResult(data, page, pageSize, total)
```

## 最佳实践

1. **始终**使用排序字段白名单 — 绝不要将用户输入直接传给 ORDER BY
2. **始终**使用 `NewPageRequest()` 获取默认值 — 不要手动计算
3. **始终**在分页之前应用租户过滤
4. **始终**在分页之前统计总行数（而不是之后）
5. **使用** `PageResult[T]` 在新代码中实现类型安全的结果
6. **文档化**每个列表接口允许的排序字段
7. **考虑**将来为超大数据集添加游标分页