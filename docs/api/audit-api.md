# 审计日志模块 API 接口说明

> Base URL: `/api/v1`  
> 所属模块：`internal/modules/audit`  
> 路由注册：`internal/modules/audit/routes.go`  
> Handler：`internal/modules/audit/handler/audit_handler.go`  
> DTO：`internal/modules/audit/dto/audit_dto.go`

---

## 1. 接口总览

### 1.1 审计日志

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/audit/stats` | 审计日志统计（Dashboard） | 需登录 |
| GET | `/audit/dashboard` | 可视化仪表盘（趋势/模块分布/操作统计） | 需登录 |
| GET | `/audit/logs` | 审计日志列表（分页+多条件筛选） | `audit:log:list` |
| GET | `/audit/logs/export` | 导出审计日志 | `audit:log:export` |
| POST | `/audit/logs` | 创建审计日志（通常内部使用） | `audit:log:create` |
| GET | `/audit/logs/{id}` | 审计日志详情 | `audit:log:read` |
| DELETE | `/audit/logs/cleanup` | 清理过期审计日志 | `audit:log:cleanup` |

### 1.2 告警管理

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| POST | `/audit/alert-rules` | 创建告警规则 | `audit:alert-rule:create` |
| GET | `/audit/alert-rules` | 获取所有告警规则 | `audit:alert-rule:list` |
| GET | `/audit/alert-rules/{id}` | 告警规则详情 | `audit:alert-rule:read` |
| PUT | `/audit/alert-rules/{id}` | 更新告警规则 | `audit:alert-rule:update` |
| DELETE | `/audit/alert-rules/{id}` | 删除告警规则 | `audit:alert-rule:delete` |
| GET | `/audit/alerts` | 告警记录列表（分页） | `audit:alert:list` |
| GET | `/audit/alerts/{id}` | 告警记录详情 | `audit:alert:read` |

### 1.3 会话分析

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/audit/sessions` | 会话摘要列表（分页） | `audit:session:list` |
| GET | `/audit/sessions/{id}/logs` | 获取会话的所有日志 | `audit:session:read` |

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

### 2.3 AuditDashboardData（可视化仪表盘响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| total_count | int64 | 总日志数 |
| today_count | int64 | 今日日志数 |
| action_stats | map[string]int64 | 按操作类型统计 |
| module_stats | map[string]int64 | 按模块统计 |
| result_stats | map[string]int64 | 按结果统计 |
| trend | array\<AuditTrendPoint\> | 最近 N 天的趋势数据 |
| top_modules | array\<ModuleCount\> | 操作量 Top 模块列表 |

### 2.4 AuditTrendPoint（趋势数据点）

| 字段 | 类型 | 说明 |
|------|------|------|
| date | string | 日期（YYYY-MM-DD） |
| count | int64 | 当天总操作数 |
| success | int64 | 当天成功操作数 |
| failure | int64 | 当天失败操作数 |

### 2.5 ModuleCount（模块统计）

| 字段 | 类型 | 说明 |
|------|------|------|
| module | string | 模块名称 |
| count | int64 | 操作次数 |
| percentage | float64 | 占比百分比 |

### 2.6 ListAuditLogsQuery（查询参数）

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

### 3.2 可视化仪表盘（趋势/模块分布/操作统计）

`GET /api/v1/audit/dashboard`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| days | int | 趋势数据的天数范围，默认 7，最大 30 |

**说明：** 返回可视化仪表盘所需的全部聚合数据，包括操作趋势、模块分布 Top、操作类型统计、结果分布等。适用于前端渲染图表。

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
    },
    "trend": [
      { "date": "2026-08-15", "count": 180, "success": 175, "failure": 5 },
      { "date": "2026-08-16", "count": 220, "success": 210, "failure": 10 },
      { "date": "2026-08-17", "count": 195, "success": 190, "failure": 5 }
    ],
    "top_modules": [
      { "module": "user", "count": 680, "percentage": 26.8 },
      { "module": "auth", "count": 450, "percentage": 17.8 },
      { "module": "rbac", "count": 320, "percentage": 12.6 }
    ]
  }
}
```

### 3.3 审计日志列表

`GET /api/v1/audit/logs`

**Query 参数：** 参见 2.3 ListAuditLogsQuery

**成功响应（200）：** 分页响应，data 为 AuditLogResp 数组

**说明：**
- 支持多条件组合筛选
- 时间范围查询使用 `start_time` + `end_time`
- 分页最大 page_size 限制为 100

### 3.4 导出审计日志

`GET /api/v1/audit/logs/export`

**Query 参数：** 与列表查询相同（不含分页）

**成功响应：** CSV/Excel 文件流

**说明：**
- 导出数据量较大时建议使用时间范围缩小范围
- 导出的日志字段与列表接口一致

### 3.5 审计日志详情

`GET /api/v1/audit/logs/{id}`

**成功响应（200）：** AuditLogResp

### 3.6 创建审计日志

`POST /api/v1/audit/logs`

**说明：** 通常由审计中间件内部调用，不对外暴露

### 3.7 清理过期审计日志

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
| `audit:alert-rule:create` | 创建告警规则 |
| `audit:alert-rule:list` | 查看告警规则列表 |
| `audit:alert-rule:read` | 查看告警规则详情 |
| `audit:alert-rule:update` | 更新告警规则 |
| `audit:alert-rule:delete` | 删除告警规则 |
| `audit:alert:list` | 查看告警记录列表 |
| `audit:alert:read` | 查看告警记录详情 |
| `audit:session:list` | 查看会话列表 |
| `audit:session:read` | 查看会话详情 |

---

## 5. 告警管理接口详细说明

### 5.1 创建告警规则

`POST /api/v1/audit/alert-rules`

**请求体：**
```json
{
  "name": "高风险操作告警",
  "description": "当出现高风险操作时触发告警",
  "enabled": true,
  "trigger_type": "risk_level",
  "trigger_value": "high",
  "notify_channels": ["email", "dingtalk"],
  "notify_targets": ["admin@example.com", "https://oapi.dingtalk.com/robot/send?access_token=xxx"],
  "notify_template": "【告警】用户 {{.Username}} 执行了 {{.Action}} 操作，风险等级：{{.RiskLevel}}",
  "cooldown_minutes": 30
}
```

**请求参数说明：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 规则名称 |
| description | string | 否 | 规则描述 |
| enabled | bool | 否 | 是否启用，默认 true |
| trigger_type | string | 是 | 触发类型：risk_level/action/user |
| trigger_value | string | 是 | 触发值：high/critical/delete/user_id |
| notify_channels | array[string] | 是 | 通知渠道：email/dingtalk/wechat/webhook |
| notify_targets | array[string] | 是 | 通知目标：邮箱地址或 Webhook URL |
| notify_template | string | 否 | 通知模板（Go 模板语法） |
| cooldown_minutes | int | 否 | 冷却时间（分钟），默认 30 |

**成功响应（201）：**
```json
{
  "code": 201,
  "message": "创建成功",
  "data": {
    "id": "01JXYZ123456789ABCDEFGH",
    "name": "高风险操作告警",
    "enabled": true,
    "trigger_type": "risk_level",
    "trigger_value": "high",
    "notify_channels": ["email", "dingtalk"],
    "cooldown_minutes": 30,
    "created_at": "2026-09-03T10:00:00Z",
    "updated_at": "2026-09-03T10:00:00Z"
  }
}
```

### 5.2 获取所有告警规则

`GET /api/v1/audit/alert-rules`

**成功响应（200）：**
```json
{
  "code": 200,
  "data": [
    {
      "id": "01JXYZ123456789ABCDEFGH",
      "name": "高风险操作告警",
      "description": "当出现高风险操作时触发告警",
      "enabled": true,
      "trigger_type": "risk_level",
      "trigger_value": "high",
      "notify_channels": ["email", "dingtalk"],
      "notify_targets": ["admin@example.com", "https://oapi.dingtalk.com/robot/send?access_token=xxx"],
      "notify_template": "【告警】用户 {{.Username}} 执行了 {{.Action}} 操作",
      "cooldown_minutes": 30,
      "created_at": "2026-09-03T10:00:00Z",
      "updated_at": "2026-09-03T10:00:00Z"
    }
  ]
}
```

### 5.3 告警规则详情

`GET /api/v1/audit/alert-rules/{id}`

**成功响应（200）：** 返回单个告警规则对象

### 5.4 更新告警规则

`PUT /api/v1/audit/alert-rules/{id}`

**请求体：** 与创建接口相同（所有字段可选）

**成功响应（200）：** 返回更新后的告警规则对象

### 5.5 删除告警规则

`DELETE /api/v1/audit/alert-rules/{id}`

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "删除成功"
}
```

### 5.6 告警记录列表

`GET /api/v1/audit/alerts`

**Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20 |
| rule_id | string | 否 | 按规则 ID 筛选 |
| user_id | string | 否 | 按用户 ID 筛选 |
| risk_level | string | 否 | 按风险等级筛选：low/medium/high/critical |

**成功响应（200）：**
```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "id": "01JXYZ987654321ZYXWVUTS",
        "rule_id": "01JXYZ123456789ABCDEFGH",
        "rule_name": "高风险操作告警",
        "audit_log_id": "01JXYZ111222333444555666",
        "user_id": "user-001",
        "username": "admin",
        "risk_level": "high",
        "action": "delete",
        "message": "用户 admin 执行了 delete 操作，风险等级：high",
        "notified": true,
        "notify_time": "2026-09-03T10:05:00Z",
        "created_at": "2026-09-03T10:05:00Z"
      }
    ],
    "total": 1
  }
}
```

### 5.7 告警记录详情

`GET /api/v1/audit/alerts/{id}`

**成功响应（200）：** 返回单个告警记录对象

---

## 6. 会话分析接口详细说明

### 6.1 会话摘要列表

`GET /api/v1/audit/sessions`

**Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20 |
| user_id | string | 否 | 按用户 ID 筛选 |

**成功响应（200）：**
```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "session_id": "01JXYZABC123DEF456GHI789",
        "user_id": "user-001",
        "username": "admin",
        "total_requests": 15,
        "success_count": 14,
        "failure_count": 1,
        "avg_duration": 125,
        "first_request": "2026-09-03T09:00:00Z",
        "last_request": "2026-09-03T09:30:00Z",
        "duration_minutes": 30
      }
    ],
    "total": 1
  }
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| session_id | string | 会话 ID |
| user_id | string | 用户 ID |
| username | string | 用户名 |
| total_requests | int | 总请求数 |
| success_count | int | 成功请求数 |
| failure_count | int | 失败请求数 |
| avg_duration | int | 平均耗时（毫秒） |
| first_request | string | 首次请求时间 |
| last_request | string | 最后请求时间 |
| duration_minutes | int | 会话时长（分钟） |

### 6.2 获取会话的所有日志

`GET /api/v1/audit/sessions/{id}/logs`

**成功响应（200）：**
```json
{
  "code": 200,
  "data": {
    "session_id": "01JXYZABC123DEF456GHI789",
    "user_id": "user-001",
    "username": "admin",
    "total_requests": 15,
    "success_count": 14,
    "failure_count": 1,
    "avg_duration": 125,
    "first_request": "2026-09-03T09:00:00Z",
    "last_request": "2026-09-03T09:30:00Z",
    "duration_minutes": 30,
    "logs": [
      {
        "id": "01JXYZ111222333444555666",
        "user_id": "user-001",
        "username": "admin",
        "module": "user",
        "action": "create",
        "method": "POST",
        "path": "/api/v1/users",
        "status_code": 200,
        "result": "success",
        "client_ip": "192.168.1.100",
        "ip_location": "中国 北京",
        "duration": 120,
        "error_message": "",
        "created_at": "2026-09-03T09:05:00Z"
      }
    ]
  }
}
```

**日志字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 日志 ID |
| user_id | string | 用户 ID |
| username | string | 用户名 |
| module | string | 模块标识 |
| action | string | 操作动作 |
| method | string | HTTP 方法 |
| path | string | 请求路径 |
| status_code | int | HTTP 状态码 |
| result | string | 结果：success/failed |
| client_ip | string | 客户端 IP |
| ip_location | string | IP 地理位置 |
| duration | int | 请求耗时（毫秒） |
| error_message | string | 错误信息 |
| created_at | string | 创建时间 |

---

## 7. 审计机制说明

### 7.1 自动审计

通过 `internal/middleware/audit_middleware.go` 和 `internal/middleware/audit_batch.go` 实现：

- 每个 HTTP 请求自动记录操作日志
- 异步批量写入（BatchProcessor），减少数据库压力
- 记录请求耗时、IP、User-Agent、请求/响应体
- 自动解析 IP 地理位置（使用 ip-api.com）

### 7.2 脱敏处理

- 密码字段自动脱敏（`******`）
- Token 字段自动脱敏
- 敏感字段通过 `AuditIgnore` 标签标记后不记录

### 7.3 日志模块标识

| 模块 | 标识 |
|------|------|
| 认证 | auth |
| 用户 | user |
| 租户 | tenant |
| RBAC | rbac |
| 文件 | file |
| 计划 | plan |
| 审计 | audit |

### 7.4 告警机制

#### 7.4.1 触发类型

| 类型 | 说明 | 触发值示例 |
|------|------|------------|
| risk_level | 按风险等级触发 | low/medium/high/critical |
| action | 按操作类型触发 | create/update/delete/login |
| user | 按特定用户触发 | user_id |

#### 7.4.2 通知渠道

| 渠道 | 说明 | 目标格式 |
|------|------|----------|
| email | 邮件通知 | 邮箱地址 |
| dingtalk | 钉钉机器人 | Webhook URL |
| wechat | 企业微信机器人 | Webhook URL |
| webhook | 自定义 Webhook | HTTP URL |

#### 7.4.3 冷却机制

- 每个规则可设置冷却时间（默认 30 分钟）
- 冷却期内同一规则不会重复触发告警
- 避免告警风暴

### 7.5 会话追踪

- 每次登录生成新的 session_id
- 同一会话的所有操作自动关联
- 支持按会话 ID 查询完整操作时间线
- 提供会话统计（请求数、成功率、平均耗时）