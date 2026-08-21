# 错误与响应架构

## 概述

标准化的错误处理和响应信封。所有 HTTP 响应遵循统一格式，使前端能够一致地处理错误。

## 响应格式

### 成功（单对象）

```json
{
  "data": { ... },
  "request_id": "req_01H..."
}
```

### 成功（分页列表）

```json
{
  "data": [ ... ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 100
  },
  "request_id": "req_01H..."
}
```

### 成功（无内容）

HTTP 204，空响应体

### 错误

```json
{
  "code": "WIKI_DOCUMENT_NOT_FOUND",
  "message": "文档不存在",
  "request_id": "req_01H...",
  "details": { ... }
}
```

## 错误码

### 包路径：`internal/pkg/apperrors`

```go
// 预定义错误码
apperrors.ErrBadRequest        // 400
apperrors.ErrUnauthorized      // 401
apperrors.ErrForbidden         // 403
apperrors.ErrResourceNotFound  // 404
apperrors.ErrConflict          // 409
apperrors.ErrInternal          // 500

// 业务域错误码
ErrWikiSpaceNotFound
ErrWikiNodeNotFound
ErrDocumentNotFound
ErrWikiPermissionDenied
ErrInvalidWikiState
```

### 错误码约定

```
{模块}_{资源}_{操作}_{状态}

示例：
WIKI_SPACE_NOT_FOUND
WIKI_DOCUMENT_NOT_FOUND
WIKI_PERMISSION_DENIED
WIKI_INVALID_STATE
USER_DUPLICATE_EMAIL
TENANT_NOT_ACTIVE
```

## AppError 结构

```go
type AppError struct {
    Code       ErrorCode `json:"code"`
    Message    string    `json:"message"`
    StatusCode int       `json:"-"`
    RequestID  string    `json:"request_id,omitempty"`
    Details    any       `json:"details,omitempty"`
    Err        error     `json:"-"` // 底层错误
}
```

## 辅助函数

### 包路径：`internal/pkg/apperrors`

```go
// 使用默认 HTTP 状态码创建
err := apperrors.New("MY_ERROR_CODE", "错误信息")

// 指定状态码创建
err := apperrors.NewWithStatus("MY_CODE", "消息", http.StatusBadRequest)

// 预定义快捷方法
err := apperrors.ErrBadRequest("输入无效")     // 400
err := apperrors.ErrNotFound("资源不存在")    // 404
err := apperrors.ErrForbidden("访问被拒绝")      // 403
err := apperrors.ErrInternal("服务器错误")        // 500

// 包装已有错误
err := apperrors.Wrap(err, "CONTEXT", "操作失败")
```

### 包路径：`internal/common/response`

```go
// 新 API（推荐）
response.Success(w, data)
response.SuccessWithPagination(w, data, page, pageSize, total)
response.SuccessNoContent(w)
response.FailError(w, appErr)    // 从 AppError 转换
response.FailWithRequestID(w, appErr, requestID)

// 快捷方法
response.BadRequest(w, "消息")
response.NotFound(w, "消息")
response.Forbidden(w, "消息")
response.Unauthorized(w, "消息")
response.InternalError(w, "消息")

// 旧版 API（向后兼容）
response.Fail(w, httpStatus, "消息")
response.JSON(w, httpStatus, code, "消息", data)
```

## 错误流程

```
内部错误 / AppError
    ↓
response.FailError(w, err)
    ↓
apperrors.FromError(err) → 带状态码的 AppError
    ↓
返回 JSON 响应 {code, message, request_id, details}
    ↓
GlobalErrorHandler 捕获 panic → INTERNAL_ERROR
```

## 请求 ID

每个响应都包含 `request_id` 用于追踪：

```go
// 中间件生成请求 ID
RequestIDMiddleware → 设置 X-Request-ID 请求头

// Handler 可读取
requestID := middleware.GetRequestID(r.Context())

// 响应中包含
response.FailWithRequestID(w, err, requestID)
```

## 全局错误处理

`GlobalErrorHandler` 捕获 panic 并转换为错误响应：

```go
// 在中间件链中
r.Use(middleware.GlobalErrorHandler)
r.Use(middleware.RequestIDMiddleware)
```

## 前端集成

```typescript
// axios 拦截器
axios.interceptors.response.use(
  response => response.data,
  error => {
    if (error.response) {
      const { code, message, request_id } = error.response.data
      console.error(`[${request_id}] ${code}: ${message}`)
      // 处理特定错误码
      if (code === 'SESSION_EXPIRED') redirectToLogin()
      if (code === 'PERMISSION_DENIED') showForbiddenModal()
    }
    return Promise.reject(error)
  }
)
```

## HTTP 状态码映射

| 分类 | 错误码 | HTTP 状态 |
|------|--------|-----------|
| 客户端错误 | `INVALID_PARAM` | 400 |
| 客户端错误 | `SESSION_EXPIRED` | 401 |
| 客户端错误 | `PERMISSION_DENIED` | 403 |
| 客户端错误 | `RESOURCE_NOT_FOUND` | 404 |
| 客户端错误 | `CONFLICT` | 409 |
| 客户端错误 | `VALIDATION_ERROR` | 422 |
| 服务端错误 | `INTERNAL` | 500 |
| 服务端错误 | `DB_ERROR` | 500 |

## 最佳实践

1. **始终**使用 `apperrors.New()` 创建新错误，不要使用 `errors.New()`
2. **禁止**在 Handler 中直接使用 `http.Error()` 或 `w.WriteHeader()`
3. **必须**在所有错误响应中包含 `request_id`
4. 服务端**记录**底层错误，仅向客户端暴露 `code` + `message`
5. 尽可能**使用**具体错误码而非通用的 `INTERNAL`