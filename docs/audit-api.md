# 审计日志模块 API 接口说明

> Base URL: `/api/v1`  
> 所属模块：`internal/modules/audit`  
> 路由注册：`internal/modules/audit/routes.go`  
> Handler：`internal/modules/audit/handler/audit_handler.go`  
> DTO：`internal/modules/audit/dto/audit_dto.go`

---

## 1. 接口总览

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/audit/stats` | 审计日志统计（Dashboard） | 需登录 |
| GET | `/audit/logs` | 审计日志列表（分页+多条件筛选） | `audit:log:list` |
| GET | `/audit/logs/export` | 导出审计日志 | `audit:log:export` |
| POST | `/audit/logs` | 创建审计日志（通常内部使用） | `audit:log:create` |
| GET | `/audit/logs/{id}` | 审计日志详情 | `audit:log:read` |
| DELETE | `/audit/logs/cleanup` | 清理过期审计日志 | `audit:log:cleanup` |

---

## 2. 数据结构

### 2.1 AuditLogResp（审计日志响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 日志 ID |
| user_id | string | 操作人用户 ID |
| username | string | 操作人用户名 |
| tenant_id | string | 租户 ID |
| module | string | 模块标识（如 `auth`, `user`, `file`） |
| action | string | 操作动作（如 `login`, `create`, `delete`） |
| resource | string | 操作资源（如 `user`, `role`, `file`） |
| resource_id | string | 操作资源 ID |
| method | string | HTTP Method |
| path | string | 请求路径 |
| request_body | string | 请求体（可能截断） |
| response_body | string | 响应体（可能截断） |
| status_code | int | HTTP 状态码 |
| result | string | 操作结果：`success` / `failed` |
| error_message | string | 错误信息（仅失败时） |
| client_ip | string | 客户端 IP |
| user_agent | string | User-Agent |
| duration | int64 | 请求耗时（毫秒） |
| created_at | string | 创建时间 |

### 2.2 AuditLogStatsResp（统计响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| total_count | int64 | 总日志数 |
| today_count | int64 | 今日日志数 |
| action_stats | map[string]int64 | 按操作类型统计（key=action, value=count） |
| module_stats | map[string]int64 | 按模块统计（key=module, value=count） |
| result_stats | map[string]int64 | 按结果统计（key=result, value=count） |

### 2.3 ListAuditLogsQuery（查询参数）

| 字段 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页条数，默认 10 |
| user_id | string | 按用户 ID 筛选 |
| username | string | 按用户名筛选 |
| tenant_id | string | 按租户 ID 筛选 |
| module | string | 按模块筛选 |
| action | string | 按操作动作筛选 |
| resource | string | 按资源筛选 |
| result | string | 按结果筛选：success/failed |
| start_time | string | 开始时间（格式：2006-01-02 15:04:05） |
| end_time | string | 结束时间（格式：2006-01-02 15:04:05） |
| keyword | string | 全文搜索（匹配 path/request_body 等） |

---

## 3. 接口详细说明

### 3.1 审计日志统计

`GET /api/v1/audit/stats`

**说明：** 返回 Dashboard 统计数据，仅需登录无需细粒度权限

**成功响应（200）：**
```json
{
  "code": 200,
  "data": {
    "total_count": 12580,
    "today_count": 320,
    "action_stats": {
      "login": 450,
      "create": 120,
      "update": 340,
      "delete": 80
    },
    "module_stats": {
      "auth": 450,
      "user": 680,
      "rbac": 320,
      "file": 200
    },
    "result_stats": {
      "success": 12000,
      "failed": 580
    }
  }
}
```

### 3.2 审计日志列表

`GET /api/v1/audit/logs`

**Query 参数：** 参见 2.3 ListAuditLogsQuery

**成功响应（200）：** 分页响应，data 为 AuditLogResp 数组

**说明：**
- 支持多条件组合筛选
- 时间范围查询使用 `start_time` + `end_time`
- 分页最大 page_size 限制为 100

### 3.3 导出审计日志

`GET /api/v1/audit/logs/export`

**Query 参数：** 与列表查询相同（不含分页）

**成功响应：** CSV/Excel 文件流

**说明：**
- 导出数据量较大时建议使用时间范围缩小范围
- 导出的日志字段与列表接口一致

### 3.4 审计日志详情

`GET /api/v1/audit/logs/{id}`

**成功响应（200）：** AuditLogResp

### 3.5 创建审计日志

`POST /api/v1/audit/logs`

**说明：** 通常由审计中间件内部调用，不对外暴露

### 3.6 清理过期审计日志

`DELETE /api/v1/audit/logs/cleanup`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| days | int | 清理多少天前的日志（默认 90） |

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "清理完成",
  "data": { "deleted_count": 15230 }
}
```

**说明：** 物理删除过期日志，不可恢复

---

## 4. 权限码列表

| 权限码 | 说明 |
|--------|------|
| `audit:log:list` | 查看审计日志列表 |
| `audit:log:read` | 查看审计日志详情 |
| `audit:log:export` | 导出审计日志 |
| `audit:log:create` | 创建审计日志（内部） |
| `audit:log:cleanup` | 清理过期日志 |

---

## 5. 审计机制说明

### 5.1 自动审计

通过 `internal/middleware/audit_middleware.go` 和 `internal/middleware/audit_batch.go` 实现：

- 每个 HTTP 请求自动记录操作日志
- 异步批量写入（BatchProcessor），减少数据库压力
- 记录请求耗时、IP、User-Agent、请求/响应体

### 5.2 脱敏处理

- 密码字段自动脱敏（`******`）
- Token 字段自动脱敏
- 敏感字段通过 `AuditIgnore` 标签标记后不记录

### 5.3 日志模块标识

| 模块 | 标识 |
|------|------|
| 认证 | auth |
| 用户 | user |
| 租户 | tenant |
| RBAC | rbac |
| 文件 | file |
| 计划 | plan |
| 审计 | audit |