# 审计架构

## 概述

框架级审计能力，用于追踪所有模块的操作。审计日志记录每个重要操作的操作者、内容、地点、时间和结果。

## 审计日志模型

### 包路径：`internal/modules/audit/model`

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 日志 ID（ULID） |
| user_id | string | 操作者用户 ID |
| username | string | 操作者用户名（反规范化） |
| tenant_id | string | 租户 ID（用于隔离） |
| module | string | 模块名（user、tenant、auth、wiki 等） |
| action | string | 操作类型（create、update、delete、login 等） |
| resource | string | 资源路径（如 /api/v1/users） |
| resource_id | string | 资源 ID（如用户 ID） |
| method | string | HTTP 方法（GET/POST/PUT/DELETE） |
| path | string | 请求路径 |
| request_body | string | 请求载荷（JSON，敏感信息已脱敏） |
| response_body | string | 响应数据（JSON，敏感信息已脱敏） |
| status_code | int | HTTP 状态码 |
| result | string | 操作结果（成功/失败） |
| error_message | string | 失败时的错误详情 |
| client_ip | string | 客户端 IP |
| user_agent | string | 浏览器 UA |
| duration | int64 | 请求耗时（毫秒） |
| created_at | time.Time | 时间戳 |

### 索引

```
idx_audit_logs_tenant_id  (tenant_id)
idx_audit_logs_user_id    (user_id)
idx_audit_logs_module     (module)
idx_audit_logs_action     (action)
idx_audit_logs_result     (result)
idx_audit_logs_created_at (created_at)
```

## 审计上下文

### 包路径：`internal/common/auditctx`

审计上下文允许服务层通过 before/after 值丰富审计条目：

```go
// 创建审计操作
action := auditctx.NewAction("wiki", "UPDATE_DOCUMENT", "/documents/"+docID, docID)

// 设置用户上下文
action.WithUser(userID, username, tenantID)

// 捕获 before/after
action.WithBefore(oldDoc).WithAfter(newDoc)

// 设置请求元数据
action.WithRequest(clientIP, userAgent, statusCode)

// 标记结果
action.Succeeded()
// 或
action.Failed(err)

// 通过上下文传递
ctx := auditctx.WithAction(r.Context(), action)
```

### 操作命名约定

```
{模块}_{操作}

模块：
  AUTH, USER, ROLE, PERMISSION, TENANT, PLAN,
  WIKI_SPACE, WIKI_NODE, WIKI_DOCUMENT, WIKI_REVISION,
  ANNOUNCEMENT, FILE, SYSTEM

操作：
  CREATE, UPDATE, DELETE, RESTORE, PERMANENT_DELETE,
  LOGIN, LOGOUT, EXPORT, IMPORT, APPROVE, REJECT,
  BATCH_CREATE, BATCH_UPDATE, BATCH_DELETE
```

示例：
```
WIKI_DOCUMENT_CREATE
WIKI_DOCUMENT_UPDATE
WIKI_DOCUMENT_DELETE
WIKI_DOCUMENT_RESTORE
WIKI_REVISION_ROLLBACK
USER_LOGIN
USER_LOGOUT
TENANT_PLAN_ASSIGN
```

## 两种审计模式

### 1. 自动审计（中间件）

`AuditMiddleware` 自动记录所有 HTTP 请求：

```go
// 在中间件链中
r.Use(middleware.AuditMiddleware(auditLogger))

// 自动捕获：
// - 谁（user_id, username, tenant_id）
// - 什么（method, path, request_body）
// - 在哪里（client_ip, user_agent）
// - 何时（duration, created_at）
// - 结果（status_code, success/failure, error_message）
```

### 2. 手动审计（服务层）

使用 `auditctx` 丰富 before/after 值：

```go
func (s *WikiService) UpdateDocument(ctx context.Context, id, userID string, req *dto.UpdateDocumentReq) (*model.Document, error) {
    oldDoc, _ := s.repo.GetDocumentByID(ctx, id)

    // ... 业务逻辑 ...

    // 通过 before/after 丰富审计
    action := auditctx.NewAction("wiki", "UPDATE_DOCUMENT", "/documents/"+id, id)
    action.WithUser(userID, username, tenantID)
    action.WithBefore(oldDoc).WithAfter(newDoc)
    ctx = auditctx.WithAction(ctx, action)

    return newDoc, nil
}
```

## 审计流程

```
HTTP 请求
    ↓
AuditMiddleware（自动捕获）
    ↓
服务层（可选：通过 auditctx 丰富）
    ↓
Handler 响应
    ↓
AuditBatchProcessor（异步写入）
    ↓
数据库（audit_logs 表）
```

### 批量处理

```go
// AuditBatchProcessor
间隔：5 秒
批量大小：100 条
超时：10 秒（context.WithTimeout）
```

日志在内存中缓冲，定期批量写入数据库（批量INSERT），性能提升10-100倍。

**Context传递优化**：
- 异步审计日志使用带超时的Context（10秒），避免goroutine泄漏
- 服务器关闭时自动取消未完成的批量写入
- 通过 `context.WithTimeout(context.Background(), 10*time.Second)` 实现

**批量插入实现**：
- `AuditService.BatchCreateLogs()` 方法支持批量创建
- `AuditRepository.BatchCreate()` 使用单条SQL批量插入
- 从O(n)数据库操作优化为O(1)

## 查询审计日志

### 包路径：`internal/modules/audit/service`

```go
// 按条件过滤
query := &model.AuditLogQuery{
    UserID:    "user_001",
    Module:    "wiki",
    Action:    "UPDATE",
    Result:    "failure",
    StartTime: "2024-01-01T00:00:00Z",
    EndTime:   "2024-12-31T23:59:59Z",
    Page:      1,
    PageSize:  20,
}

// Dashboard 统计
dashboard, err := auditService.GetDashboard(ctx, tenantID)
// 返回：总数、今日数量、操作统计、模块统计、趋势、热门模块

// 趋势分析
trend, err := auditService.GetTrend(ctx, tenantID, 30)
// 返回最近 30 天的每日数量
```

## 租户隔离

审计日志自动按 `tenant_id` 过滤：

```go
// 在仓储中
func (r *auditRepository) ListLogs(ctx context.Context, query *model.AuditLogQuery) ([]*model.AuditLog, int64, error) {
    db := tenantctx.FilterQuery(ctx, r.db.WithContext(ctx), "tenant_id")
    // ... 应用查询条件 ...
}
```

### 系统管理员例外

系统管理员可以查看跨租户的审计日志以满足合规和调试需求：

```go
// 当 IsMaster=true 时，FilterQuery() 不添加 tenant_id 过滤
// 允许查看所有租户日志
```

## 数据保留

```
// 保留策略
- 超过 90 天的日志可归档
- 清理定时任务删除旧日志
- 可通过 config.yaml 配置
```

## 安全

- 请求体敏感数据脱敏（密码、Token、密钥）
- 响应体敏感数据脱敏
- 记录 IP 和 User Agent 用于安全分析
- 所有写入不可变（只追加）

## 实现清单

| 功能 | 状态 |
|------|------|
| 自动审计中间件 | ✅ 已完成 |
| 手动审计上下文（auditctx） | ✅ 已完成 |
| Before/after 值追踪 | ✅ 已完成 |
| 操作命名约定 | ✅ 已完成 |
| 基于模块的操作 | ✅ 已完成 |
| 批量异步处理 | ✅ 已完成 |
| 查询时租户隔离 | ✅ 已完成 |
| Dashboard 统计 | ✅ 已完成 |
| 趋势分析 | ✅ 已完成 |
| 热门模块 | ✅ 已完成 |
| 数据清理 | ✅ 已完成 |
| 请求/响应脱敏 | ⏳ 进行中 |
| 保留策略 | ⏳ 计划中 |

## 最佳实践

1. 在服务层对重要操作**始终**使用 `auditctx`
2. **禁止**在审计日志中包含敏感数据（密码、Token）
3. 跨模块**使用**一致的操作名称
4. 为可追溯性**包含** `resource_id`
5. 对变更操作**捕获** before/after 以支持回滚分析
6. 查询审计日志时**遵守**租户隔离
7. **监控**审计日志量以防止存储问题