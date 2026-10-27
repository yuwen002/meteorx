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
| GET | `/audit/user-timeline` | 用户操作时间线（按天分组） | 需登录 |
| GET | `/audit/detailed-stats` | 详细统计（小时级/风险等级/用户活跃度） | 需登录 |
| GET | `/audit/anomalies` | 异常检测（高频失败等） | 需登录 |
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
| GET | `/audit/alerts/stats` | 告警统计数据 | `audit:alert:stats` |
| GET | `/audit/alerts/{id}` | 告警记录详情 | `audit:alert:read` |

### 1.3 会话分析

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/audit/sessions` | 会话摘要列表（分页） | `audit:session:list` |
| GET | `/audit/sessions/{id}/logs` | 获取会话的所有日志 | `audit:session:read` |

### 1.4 完整性校验

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| POST | `/audit/integrity/verify` | 校验审计日志完整性（行摘要 + 锚点链，只读） | `audit:integrity:verify` |
| POST | `/audit/integrity/rehash` | 密钥轮换重算：把旧 pepper 历史行迁到新 pepper 并重建锚点链 | `audit:integrity:rehash` |

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
| ip_location | string | IP 地理位置 |
| user_agent | string | User-Agent |
| device_info | string | 设备信息 |
| duration | int64 | 请求耗时（毫秒） |
| session_id | string | 会话 ID |
| risk_level | string | 风险等级 |
| tags | string | 标签（JSON 数组） |
| created_at | string | 创建时间 |

### 2.2 AuditLogStatsResp（统计响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| total_count | int64 | 总日志数 |
| today_count | int64 | 今日日志数 |
| action_stats | map[string]int64 | 按操作类型统计（key=action, value=count） |
| module_stats | map[string]int64 | 按模块统计（key=module, value=count） |
| result_stats | map[string]int64 | 按结果统计（key=result, value=count） |

### 2.3 DashboardResp（可视化仪表盘响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| total_count | int64 | 总日志数 |
| today_count | int64 | 今日日志数 |
| action_stats | map[string]int64 | 按操作类型统计 |
| module_stats | map[string]int64 | 按模块统计 |
| result_stats | map[string]int64 | 按结果统计 |
| trend | array\<TrendPoint\> | 最近 N 天的趋势数据 |
| top_modules | array\<ModuleCount\> | 操作量 Top 模块列表 |

### 2.4 TrendPoint（趋势数据点）

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
| action | string | 按操作动作筛选（支持逗号分隔多值，如 `login,logout`） |
| resource | string | 按资源筛选 |
| result | string | 按结果筛选：success/failed |
| start_time | string | 开始时间（格式：2006-01-02 15:04:05） |
| end_time | string | 结束时间（格式：2006-01-02 15:04:05） |
| keyword | string | 全文搜索（匹配 path/request_body 等） |

### 2.7 UserTimelineReq（用户时间线查询参数）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | string | 是 | 用户 ID **或用户名**（仓储按 `user_id = ? OR username = ?` 匹配） |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 20 |
| start_time | string | 否 | 开始时间 |
| end_time | string | 否 | 结束时间 |

### 2.8 TimelineItem（时间线条目）

| 字段 | 类型 | 说明 |
|------|------|------|
| date | string | 日期（YYYY-MM-DD） |
| count | int64 | 当天总操作数 |
| success | int64 | 当天成功操作数 |
| failure | int64 | 当天失败操作数 |
| logs | array\<AuditLogResp\> | 当日操作日志列表 |

### 2.9 DetailedStatsResp（详细统计响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| hourly_stats | map[string]int64 | 小时级统计（key=小时, value=操作数） |
| risk_level_stats | map[string]int64 | 风险等级分布（key=等级, value=数量） |
| user_activity | array\<UserActivityStat\> | 用户活跃度排行 |

### 2.10 AnomalyLogResp（异常日志响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | string | 用户 ID |
| username | string | 用户名 |
| anomaly_type | string | 异常类型：`high_failure_rate` / `geo_anomaly` / `brute_force` |
| anomaly_label | string | 异常类型中文标签 |
| failure_count | int64 | 失败次数 |
| total_count | int64 | 总操作次数 |
| window_minutes | int | 检测时间窗口（分钟） |
| first_seen | string | 首次出现时间 |
| last_seen | string | 最后出现时间 |
| risk_level | string | 风险等级：low / medium / high / critical |
| details | string | 详情描述 |

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

### 3.2 可视化仪表盘

`GET /api/v1/audit/dashboard`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| days | int | 趋势数据的天数范围，默认 7，最大 30 |

**成功响应（200）：**
```json
{
  "code": 200,
  "data": {
    "total_count": 12580,
    "today_count": 320,
    "trend": [
      { "date": "2026-08-15", "count": 180, "success": 175, "failure": 5 },
      { "date": "2026-08-16", "count": 220, "success": 210, "failure": 10 }
    ],
    "top_modules": [
      { "module": "user", "count": 680, "percentage": 26.8 }
    ]
  }
}
```

### 3.3 用户操作时间线

`GET /api/v1/audit/user-timeline`

**Query 参数：** 参见 2.7 UserTimelineReq

**说明：** 按天分组展示用户操作记录，每日聚合显示总数、成功数、失败数

**前端定位（`?user_id=`）：** 用户时间线页 `views/system/audit/timeline.vue` 读 `route.query.user_id`
后直接查这个人的时间线（日期筛选与页码重置，避免旧筛选把结果裁成空页）；无 query 时退回当前登录用户，
保持「进去就看到自己」的默认行为。入口来自告警详情里的「触发用户」（优先传 `user_id`，
无 ID 时传 `username`）；监听用 `watch` 而非 `onMounted`，已在本页时再点另一个人的告警才会生效。

**成功响应（200）：**
```json
{
  "code": 200,
  "data": {
    "items": [
      {
        "date": "2026-09-08",
        "count": 15,
        "success": 12,
        "failure": 3,
        "logs": [
          {
            "id": "01JXYZ...",
            "action": "login",
            "result": "success",
            "method": "POST",
            "path": "/api/v1/auth/login",
            "client_ip": "192.168.1.100",
            "duration": 120,
            "created_at": "2026-09-08T10:00:00Z"
          }
        ]
      }
    ],
    "total": 15,
    "pages": 1
  }
}
```

### 3.4 详细统计

`GET /api/v1/audit/detailed-stats`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| days | int | 统计天数，默认 7 |

**说明：** 返回小时级操作分布、风险等级分布、用户活跃度排行

**成功响应（200）：**
```json
{
  "code": 200,
  "data": {
    "hourly_stats": {
      "09:00": 45,
      "10:00": 78,
      "11:00": 62
    },
    "risk_level_stats": {
      "low": 5000,
      "medium": 300,
      "high": 50,
      "critical": 5
    },
    "user_activity": [
      { "user_id": "user-001", "username": "admin", "count": 1200, "failures": 5 }
    ]
  }
}
```

### 3.5 异常检测

`GET /api/v1/audit/anomalies`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| threshold | int | 失败次数阈值，默认 5 |
| window_minutes | int | 检测时间窗口（分钟），默认 30 |

**说明：** 检测短时高频失败等异常行为

**成功响应（200）：**
```json
{
  "code": 200,
  "data": [
    {
      "user_id": "user-001",
      "username": "admin",
      "anomaly_type": "high_failure_rate",
      "anomaly_label": "高频失败",
      "failure_count": 10,
      "total_count": 12,
      "window_minutes": 30,
      "first_seen": "2026-09-08T10:00:00Z",
      "last_seen": "2026-09-08T10:25:00Z",
      "risk_level": "high",
      "details": ""
    }
  ]
}
```

### 3.6 审计日志列表

`GET /api/v1/audit/logs`

**Query 参数：** 参见 2.6 ListAuditLogsQuery

**说明：**
- 支持多条件组合筛选
- `action` 支持逗号分隔多值查询（如 `action=login,logout`）
- 时间范围查询使用 `start_time` + `end_time`
- 分页最大 page_size 限制为 100

### 3.7 导出审计日志

`GET /api/v1/audit/logs/export`

**Query 参数：** 与列表查询相同（不含分页）

**成功响应：** CSV 文件流

> **自身留痕**：本路径命中 `shouldSkipAudit` 的 `/api/v1/audit/logs` 前缀，不被自动审计；批量导出审计数据属于
> 数据外流，handler 内手动补记为 `module=audit / action=export / risk=high`，**成功与失败都入库**，
> 只记导出筛选条件与条数（`{exported_count:N}`），不落 CSV 正文。

### 3.8 审计日志详情

`GET /api/v1/audit/logs/{id}`

**权限码：** `audit:log:read`

**成功响应（200）：** AuditLogResp（见 2.1）。

**前端定位（`?log_id=`）：** 审计日志页 `views/system/audit/index.vue` 读 `route.query.log_id`
后调本接口直接弹出该条详情，不要求这条日志已在当前筛选 / 分页里；日志已被保留期清理时接口返回 404，
页面展示“日志不存在或已被清理”的空态。入口来自告警详情里的「审计日志 ID」（告警只存 ID，
正文与请求参数都在审计日志里）。

### 3.9 创建审计日志

`POST /api/v1/audit/logs`

**说明：** 通常由审计中间件内部调用，不对外暴露

### 3.10 清理过期审计日志

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

> **自身留痕**：删除审计历史是最需留痕的高危操作，但本路径同样被自动审计前缀跳过；handler 内手动补记为
> `module=audit / action=delete / risk=critical`，**成功与非超管越权（403）/ 服务失败（500）都入库**，
> 只记参数与结果摘要（`{days}` / `{deleted_count}`）。

---

### 3.11 完整性校验

`POST /api/v1/audit/integrity/verify`

**权限码：** `audit:integrity:verify`（仅平台超管）

**说明：** 只读重放行级摘要与锚点链，**不修改任何数据**。需开启完整性（配置 `audit.integrity.enabled`
与 `secret_key`）；未配置 pepper 时返回 `not_configured`。处于密钥轮换过渡期（额外配 `previous_secret_key`）时，
仅匹配旧 pepper 的历史行会被归入 `pending_rehash_rows`（非篡改），提示可执行 rehash。

**成功响应（200）：**
```json
{
  "code": 200,
  "data": {
    "enabled": true,
    "verdict": "verified",
    "checked_rows": 12580,
    "legacy_rows": 0,
    "tampered_rows": 0,
    "tampered_samples": [],
    "pending_rehash_rows": 0,
    "pending_rehash_samples": [],
    "anchors_checked": 36,
    "anomaly_anchor_ids": [],
    "anomalies": [],
    "covered_from": "2026-09-01T00:00:00Z",
    "covered_to": "2026-10-01T00:00:00Z",
    "message": "完整性校验通过：12580 行、36 个锚点均一致"
  }
}
```

**`verdict` 取值：**

| 值 | 含义 |
|------|------|
| `verified` | 行级重算全部相符且锚点链完整 |
| `tampered` | 检出篡改（内容被改）或异常（删/插/重序/断链），`tampered_samples` / `anomaly_anchor_ids` 给出定位样本 |
| `pending_rehash` | 处于密钥轮换过渡期，`pending_rehash_rows` 行仅匹配旧 pepper（非篡改），可执行 rehash 迁移 |
| `not_configured` | 未配置 pepper，无法校验（`enabled=false`） |

> `record_hash` 为空的历史行视为 legacy（计入 `legacy_rows`），**不视为篡改**。

---

### 3.12 密钥轮换重算（rehash）

`POST /api/v1/audit/integrity/rehash`

**权限码：** `audit:integrity:rehash`（仅平台超管）

**说明：** 密钥（pepper）轮换后的安全迁移。仅重写**旧 pepper 可验证**的历史行（`RowPrevious`）到当前
pepper，真篡改行（两把 key 都对不上）不会被洗白；随后清空并从创世按当前 key **全量重建锚点链**。
需同时配置 `secret_key`（新）与 `previous_secret_key`（旧）；**非轮换期（未配旧 pepper）返回 400**。
迁移完成、`verify` 回到 `verified` 后，清空 `previous_secret_key` 即回单密钥常态。

**成功响应（200）：**
```json
{
  "code": 200,
  "data": {
    "rehashed_rows": 9,
    "anchors_rebuilt": 36,
    "message": "已重算 9 行至当前密钥并重建 36 个锚点"
  }
}
```

**失败响应（400）：** 未启用完整性或未配置旧 pepper 时返回，`message` 为对应原因（如“未配置旧 pepper，无待迁移密钥”）。

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
| `audit:alert:stats` | 查看告警统计 |
| `audit:session:list` | 查看会话列表 |
| `audit:session:read` | 查看会话详情 |
| `audit:integrity:verify` | 校验审计日志完整性（仅平台超管） |
| `audit:integrity:rehash` | 密钥轮换后重算历史行并重建锚点链（仅平台超管） |

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
  "notify_targets": ["admin@example.com"],
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
| notify_targets | array[string] | 是 | 通知目标 |
| notify_template | string | 否 | 通知模板（Go 模板语法） |
| cooldown_minutes | int | 否 | 冷却时间（分钟），默认 30 |

### 5.2 获取所有告警规则

`GET /api/v1/audit/alert-rules`

### 5.3 告警规则详情

`GET /api/v1/audit/alert-rules/{id}`

### 5.4 更新告警规则

`PUT /api/v1/audit/alert-rules/{id}`

### 5.5 删除告警规则

`DELETE /api/v1/audit/alert-rules/{id}`

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

### 5.7 告警统计数据

`GET /api/v1/audit/alerts/stats`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| days | int | 统计天数，默认 7 |

### 5.8 告警记录详情

`GET /api/v1/audit/alerts/{id}`

**权限码：** `audit:alert:read`

**关键字段：** `audit_log_id`（触发这条告警的审计日志 ID）、`rule_name`、`risk_level`、`action`、
`username` / `user_id`、`message`、`notified` / `notify_time`、`created_at`。

**两段跳转闭环（页面间不靠用户自己找）：**

1. 站内信 → 告警：收件箱里告警消息的 `link` 为 `/system/audit/alert?alert_id={告警ID}`，
   告警页 `views/system/audit/alert.vue` 读 `alert_id` 后调本接口直接弹出详情；
2. 告警 → 审计日志：详情里的 `audit_log_id` 可点，跳 `/system/audit?log_id={日志ID}`，
   审计日志页据此调 3.8 拉详情（见 3.8 的前端定位）。

另外详情里的**「触发用户」**也可点：跳 `/system/audit/timeline?user_id={用户ID}`，
时间线页据此查这个人的全部操作（无 ID 时退回 `username`）。告警只存「谁触发的」，
这个人还做过什么全在时间线里——留一段不可点的文本等于让用户退出去重新敲一遍用户名。

三处都用 `watch` 监听 query 而不是只在 `onMounted` 读一次：已在目标页时再点另一条只会改 query，
不重新 mount，只读一次会“没反应”。

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

### 6.2 获取会话的所有日志

`GET /api/v1/audit/sessions/{id}/logs`

---

## 7. 审计机制说明

### 7.1 自动审计

通过 `internal/middleware/audit_middleware.go` 和 `internal/middleware/audit_batch.go` 实现：

- 每个 HTTP 请求自动记录操作日志
- 异步批量写入（BatchProcessor），减少数据库压力
- 记录请求耗时、IP、User-Agent、请求/响应体
- 自动解析 IP 地理位置

### 7.2 脱敏处理

- 按**字段名包含关键字**匹配脱敏（命中即替换为 `***`）：`password`/`passwd`/`pwd`、`token`/`secret`/`authorization`、
  `api_key`/`apikey`/`access_key`/`private_key`/`secret_key`、`credit_card`/`card_no`/`cvv`、`id_card`/`idcard`、
  `totp`/`otp`/`mfa`——用包含匹配以覆盖 `old_password`/`refresh_token` 这类命名变体
- **递归脱敏**：不止处理 JSON 顶层 key，`sanitizeValue` 递归下钻嵌套对象与数组元素（顶层是数组的批量请求同样处理），
  `config.db_password`、`users[].api_key` 这类嵌套明文也会被掩码
- multipart 请求体不采集，上传文件内容不进入审计日志
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

#### 7.4.1 触发时机与触发类型

**触发时机**：告警由审计日志**落库成功后**自动评估触发（写入路径——含批量 flush 与直写——接入告警评估链），
无需手动调用；日志行写入失败时不评估（告警必须指向真实存在的审计记录）。
批量路径按批摊销：每轮 flush 整批只查一次启用规则，匹配与冷却判定逐条生效。

**评估延迟（SLA）**：告警评估依附于批量写入节奏，从请求产生审计日志到告警触发的延迟上限为一次 flush 间隔
（默认 5 秒，即 `InitAuditBatchProcessor` 固定的 100 条 / 5 秒配置；缓冲区写满 100 条会提前 flush，延迟随之缩短）；
无批量处理器时的直写路径在落库后立即评估。该延迟不含邮件 / Webhook 的渠道投递耗时（取决于外部服务），
属批量聚合换取数据库压力的**最终一致**设计取舍，不承诺秒级实时；冷却按告警创建时间判定，与评估延迟无关。

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

**收件人按类型路由**：`notify_targets` 允许混装邮箱与 Webhook URL（如 `["admin@example.com", "https://oapi.dingtalk.com/…"]`），
发送时按地址类型拆分——含 `@` 的进邮件渠道，`http` 前缀的进对应 Webhook 渠道，类型不匹配的目标不会误发到其它渠道。

**送达与 notified 回写**：告警记录先落库再发送；只有**至少一个渠道真实送达**时，才以 UPDATE 回写
`notified=true` 与 `notify_time`。全部渠道失败时记录保持 `notified=false` 并在服务端日志留痕
（需要人工确认通知配置）；因此看板的 `notified_count` 反映的是真实送达数，不是触发数。
冷却判定基于告警创建时间，与 notified 无关（未送达不会自动重发）。

**收件箱补推**：`notify_targets` 里的邮箱若对应某个平台账号，该账号还会收到一条 `audit` 分类的站内信
（标题 `[告警] {规则名}`，跳转 `/system/audit/alert?alert_id={告警ID}`），使收件人在邮件没看到 / 当时离线的情况下
重新登录仍能看到同一条告警，并可直接定位到该条记录（无记录 ID 时链接不带 query）。收件箱收件人**只来自「邮箱反查得到的账号」**：未注册账号的外部值班邮箱
不会产生收件箱记录，也不会因此按租户或全平台展开——告警正文含用户行为明细，广播等于泄露审计信息。
站内信落库不参与 `notified` 回写判定（与其它通知同一口径：写库成功不等于送达）；
同一条告警按告警 ID 去重，渠道重试 / 多副本只落一条。

#### 7.4.3 冷却机制

- 每个规则可设置冷却时间（默认 30 分钟）
- 冷却期内同一规则不会重复触发告警
- 避免告警风暴

### 7.5 会话追踪

- 每次登录生成新的 session_id
- 同一会话的所有操作自动关联
- 支持按会话 ID 查询完整操作时间线
- 提供会话统计（请求数、成功率、平均耗时）

### 7.6 WebSocket 实时告警推送

#### 7.6.1 连接说明

通过 WebSocket 实时推送告警通知到前端：

| 项目 | 说明 |
|------|------|
| 连接地址 | `ws://{host}/api/v1/ws` |
| 认证方式 | URL Query 参数 `?token={jwt_token}` 或 `Authorization: Bearer {jwt_token}` |
| 协议 | JSON 文本帧 |
| 心跳间隔 | 客户端每 30 秒发送 `{"type":"ping"}`，服务端回复 `{"type":"pong"}` |
| 自动重连 | 前端 SDK 支持断线自动重连 |

#### 7.6.2 消息格式

**服务端推送消息：**
```json
{
  "type": "alert",
  "payload": {
    "id": "alert-001",
    "rule_name": "高风险操作告警",
    "risk_level": "high",
    "action": "delete",
    "message": "用户 admin 执行了删除操作，风险等级：high",
    "username": "admin",
    "time": "2026-09-08T10:00:00Z"
  }
}
```

**消息类型：**

| 类型 | 说明 | 方向 |
|------|------|------|
| `alert` | 告警通知 | 服务端 → 客户端 |
| `announcement` | 公告推送 | 服务端 → 客户端 |
| `ping` | 心跳请求 | 双向 |
| `pong` | 心跳响应 | 双向 |

#### 7.6.3 告警推送流程

1. 告警规则触发 → 创建告警记录
2. 服务端通过 WebSocket **定向推送给触发该告警的用户**（`alert.user_id`，即审计日志的操作人），
   不做全量广播——告警内容可能包含其他用户的行为明细，广播会给全体在线用户泄露审计信息
3. 邮件/Webhook 渠道通知规则配置的目标收件人，与站内 WS 推送互不混用收件人列表
4. 规则邮箱能反查到平台账号时，同一告警另落一条 `audit` 分类站内信（见 7.4.2 收件箱补推），
   供收件人离线查看
5. 前端收到消息后更新通知栏红点
6. 用户点击红点跳转至告警管理页面查看详情

#### 7.6.4 前端集成

**WebSocket 连接管理：** `web-admin/src/composables/useWebSocket.ts`

```typescript
// 使用示例
const { connected, connect, disconnect, onMessage } = useWebSocket()

// 监听告警消息
onMessage((msg) => {
  if (msg.type === 'alert') {
    // 更新通知栏
    notificationStore.addAlert(msg.payload)
  }
})
```

### 7.7 数据保留与完整性校验配置

均位于 `config.yaml` 的 `audit` 段（可用环境变量覆盖），**默认全部关闭**（向后兼容）：

| 配置项 | env | 默认 | 说明 |
|------|-----|------|------|
| `audit.retention_days` | `METEORX_AUDIT_RETENTION_DAYS` | `0` | 自动保留天数；`<= 0` 关闭（不自动删）。开启后每小时清理一次并补记自指审计 |
| `audit.integrity.enabled` | `METEORX_AUDIT_INTEGRITY_ENABLED` | `false` | 防篡改完整性总开关 |
| `audit.integrity.secret_key` | `METEORX_AUDIT_INTEGRITY_SECRET_KEY` | 空 | HMAC pepper（不落库）；未配置则整卡降级关闭、不阻断写入 |
| `audit.integrity.previous_secret_key` | `METEORX_AUDIT_INTEGRITY_PREVIOUS_SECRET_KEY` | 空 | 轮换过渡期的旧 pepper；仅在换 key 迁移期临时配置，配合 rehash 使用，迁移完成后清空即回单密钥常态 |
| `audit.integrity.anchor_interval` | `METEORX_AUDIT_INTEGRITY_ANCHOR_INTERVAL` | `1h` | 锚点链封口周期 |
| `audit.integrity.anchor_grace` | `METEORX_AUDIT_INTEGRITY_ANCHOR_GRACE` | `1m` | 锚点封口安全边界：窗口右端取 `now - grace`，给异步 flush 迟到行留落库时间；周期任务与 rehash 重建共用同一口径 |

> 脱敏（见 7.2）与本节无关：`record_hash` 只覆盖已落库的字段，不改变任何脱敏行为。