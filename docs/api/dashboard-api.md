# 运营看板 API 文档

数据看板模块（`internal/modules/dashboard`）分两组接口：平台运营总览（租户 / 用户 / 订阅 / 审计多维统计，供超级管理员）与租户工作台总览（成员 / 任务 / Wiki / 存储，供租户内登录用户）。

**权限：**
- 平台侧（`/admin/dashboard/*`）：需超级管理员角色，且具备 `admin:dashboard:list` 权限码（`RequiresMasterAdmin` + `AutoRequirePermission`）。
- 租户侧（`/dashboard/tenant-overview`）：仅需登录，不挂 RBAC 权限码，租户 ID 取自登录上下文。

**基础路径：** `/api/v1`

---

## 1. 接口列表

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/admin/dashboard/overview` | 平台运营数据总览 | `admin:dashboard:list` |
| GET | `/admin/dashboard/trend` | 增长趋势时间序列（按自然日分桶） | `admin:dashboard:list` |
| GET | `/admin/dashboard/tenants-health` | 租户健康度列表（含分级汇总） | `admin:dashboard:list` |
| GET | `/admin/dashboard/tenants-health/{id}` | 单租户健康度详情 | `admin:dashboard:list` |
| GET | `/admin/dashboard/slow-queries` | 慢查询看板快照（统计 + 明细） | `admin:dashboard:list` |
| POST | `/admin/dashboard/slow-queries/reset` | 重置慢查询统计缓冲区 | `admin:dashboard:create` |
| GET | `/dashboard/tenant-overview` | 当前租户工作台总览（成员/任务/Wiki/存储） | 需登录 |

---

## 2. 数据结构

### 2.1 DashboardOverviewResp（总览响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| tenant_stats | TenantStatsResp | 租户统计 |
| user_stats | UserStatsResp | 用户统计 |
| subscription_stats | SubscriptionStatsResp | 订阅统计 |
| audit_stats | AuditStatsResp | 审计统计 |

### 2.2 TenantStatsResp（租户统计）

| 字段 | 类型 | 说明 |
|------|------|------|
| total | int64 | 租户总数 |
| enabled | int64 | 启用中租户（status=1） |
| disabled | int64 | 已禁用租户（status=0） |
| today_new | int64 | 今日新增 |
| week_new | int64 | 本周新增 |
| month_new | int64 | 本月新增 |

### 2.3 UserStatsResp（用户统计）

| 字段 | 类型 | 说明 |
|------|------|------|
| total | int64 | 用户总数 |
| today_new | int64 | 今日新增 |
| week_new | int64 | 本周新增 |
| month_new | int64 | 本月新增 |

### 2.4 SubscriptionStatsResp（订阅统计）

| 字段 | 类型 | 说明 |
|------|------|------|
| total | int64 | 订阅总数 |
| active | int64 | 生效中（status=1） |
| expired | int64 | 已过期（status=2） |
| cancelled | int64 | 已取消（status=3） |
| active_tenant | int64 | 有生效订阅的租户数 |

### 2.5 AuditStatsResp（审计统计）

| 字段 | 类型 | 说明 |
|------|------|------|
| total | int64 | 审计记录总数 |
| today | int64 | 今日请求数 |
| success | int64 | 成功数（result=success） |
| failure | int64 | 失败数（result=failure） |
| action_stats | map\<string,int64\> | 按操作类型统计 |
| module_stats | map\<string,int64\> | 按模块统计 |

### 2.6 TenantHealthItemResp（单租户健康度）

| 字段 | 类型 | 说明 |
|------|------|------|
| tenant_id | string | 租户 ID |
| tenant_name | string | 租户名称 |
| status | int | 租户状态：1-启用 0-禁用 |
| plan_name | string | 当前套餐名称（无订阅时为空） |
| subscription_status | int | 订阅状态：0-无订阅 1-生效 2-到期 3-取消 |
| subscription_expires_at | string\|null | 订阅到期时间（null=长期有效） |
| days_remaining | int | 订阅剩余天数（-1=长期有效） |
| user_count | int64 | 用户总数 |
| active_user_count | int64 | 有效用户数 |
| task_count | int64 | 任务总数 |
| task_completed_count | int64 | 已完成任务数 |
| wiki_doc_count | int64 | Wiki 文档总数 |
| storage_used | int64 | 存储占用（字节） |
| audit_count_7d | int64 | 近 7 天审计日志数 |
| last_activity_at | string\|null | 最近活动时间 |
| health_score | int | 健康度评分（0-100） |
| health_level | string | 健康分级：`healthy` / `moderate` / `at_risk` |

### 2.7 TenantHealthListResp（健康度列表响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| items | TenantHealthItemResp[] | 当前页健康度明细 |
| total | int64 | 租户总数 |
| page | int | 页码 |
| page_size | int | 每页条数 |
| healthy | int64 | healthy 分级租户数（全量统计） |
| moderate | int64 | moderate 分级租户数（全量统计） |
| at_risk | int64 | at_risk 分级租户数（全量统计） |

### 2.8 GrowthTrendResp（增长趋势响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| days | int | 统计窗口天数（含今日） |
| list | TrendDayResp[] | 按日期升序的时间序列，缺失日期以零值补齐 |

### 2.9 TrendDayResp（单日数据点）

| 字段 | 类型 | 说明 |
|------|------|------|
| date | string | 业务时区自然日，格式 `YYYY-MM-DD` |
| new_tenants | int64 | 当日新增租户数（软删除不计入） |
| new_users | int64 | 当日新增用户数（软删除不计入） |
| audit_logs | int64 | 当日审计日志数（活跃度指标） |

### 2.10 TenantOverviewResp（租户工作台总览）

| 字段 | 类型 | 说明 |
|------|------|------|
| member | TenantMemberOverviewResp | 成员统计 |
| task | TenantTaskOverviewResp | 任务状态分布 |
| wiki | TenantWikiOverviewResp | Wiki 统计 |
| storage_used | int64 | 存储占用（字节，排除软删） |

**TenantMemberOverviewResp**

| 字段 | 类型 | 说明 |
|------|------|------|
| total | int64 | 成员总数（排除软删） |
| active | int64 | 启用状态成员数（status=1） |

**TenantTaskOverviewResp**

| 字段 | 类型 | 说明 |
|------|------|------|
| pending | int64 | 待办数（status=pending） |
| in_progress | int64 | 进行中数（status=in_progress） |
| completed | int64 | 已完成数（status=completed） |
| overdue | int64 | 逾期待办数（未完成且 `due_date < now`） |
| total | int64 | 任务总数（排除软删） |
| completion_rate | float64 | 完成率（0～1，保留 4 位小数；总数为 0 时为 0） |

**TenantWikiOverviewResp**

| 字段 | 类型 | 说明 |
|------|------|------|
| spaces | int64 | 知识空间数（排除软删） |
| documents | int64 | 文档数（排除软删） |
| views | int64 | 文档累计浏览量 |

---

## 3. 接口详细说明

### 3.1 平台运营数据总览

`GET /api/v1/admin/dashboard/overview`

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "tenant_stats": {
      "total": 128,
      "enabled": 120,
      "disabled": 8,
      "today_new": 3,
      "week_new": 21,
      "month_new": 68
    },
    "user_stats": {
      "total": 2048,
      "today_new": 32,
      "week_new": 210,
      "month_new": 890
    },
    "subscription_stats": {
      "total": 150,
      "active": 118,
      "expired": 27,
      "cancelled": 5,
      "active_tenant": 118
    },
    "audit_stats": {
      "total": 52000,
      "today": 1300,
      "success": 51500,
      "failure": 500,
      "action_stats": {
        "create": 3200,
        "read": 41000,
        "update": 5600,
        "delete": 2200
      },
      "module_stats": {
        "tenant": 15000,
        "user": 12000,
        "file": 9800
      }
    }
  }
}
```

**业务说明：**

- 租户状态：`1` 启用、`0` 禁用。
- 订阅状态：`1` active、`2` expired、`3` cancelled。
- 审计结果：`success` / `failure`。
- 该接口聚合多张表，数据量较大时建议配合定时任务缓存（当前为实时聚合）。

### 3.2 租户健康度列表

`GET /api/v1/admin/dashboard/tenants-health`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页条数 |
| keyword | string | 按租户名称模糊匹配 |

**成功响应（200）：** TenantHealthListResp（含分页与 healthy/moderate/at_risk 三级分布汇总）

**评分模型（0-100，三项相加）：**

| 维度 | 满分 | 规则 |
|------|------|------|
| 订阅状态 | 40 | 生效 40 / 到期 15 / 取消或无订阅 0 |
| 近 7 天活跃 | 30 | 30 × min(1, 审计日志数 / 20) |
| 资源足迹 | 30 | 30 × min(1, (用户+任务+Wiki文档) / 100) |

**分级：** `health_score ≥ 70` → healthy；`≥ 40` → moderate；`< 40` → at_risk。

**业务说明：** 全部指标基于既有数据表聚合直查（租户/订阅/用户/任务/Wiki/文件/审计），无新增表。

### 3.2b 增长趋势时间序列

`GET /api/v1/admin/dashboard/trend`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| days | int | 统计窗口天数，默认 30，取值 1-365；非法值返回 400 |

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "days": 3,
    "list": [
      { "date": "2026-10-12", "new_tenants": 0, "new_users": 1, "audit_logs": 15 },
      { "date": "2026-10-13", "new_tenants": 1, "new_users": 3, "audit_logs": 20 },
      { "date": "2026-10-14", "new_tenants": 2, "new_users": 5, "audit_logs": 40 }
    ]
  }
}
```

**业务说明：**

- 以今日为终点向前回推 `days-1` 天构成闭区间，逐日按 `created_at` 分桶（`DATE(created_at)`），沿用业务时区自然日边界。
- 租户/用户统计附加 `deleted_at IS NULL`，软删除记录不计入增长；审计日志为活跃度指标，全量统计。
- 沿用总览接口的容错策略：单表查询失败仅记日志、对应指标置零，不阻断整体返回。
- 前端以零依赖内联 SVG 柱状图渲染，支持在“新增用户 / 新增租户 / 活跃日志”三个指标间切换，并提供近 7/30/90 天窗口选择。

### 3.3 单租户健康度详情

`GET /api/v1/admin/dashboard/tenants-health/{id}`

**成功响应（200）：** TenantHealthItemResp

**错误响应：** 租户不存在返回 404。

### 3.4 慢查询看板快照

`GET /api/v1/admin/dashboard/slow-queries`

返回进程内采集的慢查询统计与最近明细（`internal/dbmonitor` 环形缓冲）。

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "stats": {
      "threshold_ms": 500,
      "total_queries": 1200,
      "slow_queries": 15,
      "max_duration_ms": 2300,
      "avg_slow_ms": 820.5,
      "capacity": 200
    },
    "entries": [
      {
        "sql": "SELECT * FROM tasks WHERE ...",
        "duration_ms": 2300,
        "rows": 512,
        "trace_id": "0af7651916cd43dd",
        "time": "2026-10-13T03:21:00Z"
      }
    ]
  }
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| stats.threshold_ms | int64 | 慢查询判定阈值（由 `database.slow_query_ms` 配置，默认 500） |
| stats.total_queries | int64 | 采样期间捕获的 SQL 总数 |
| stats.slow_queries | int64 | 超过阈值的慢查询数 |
| stats.max_duration_ms | int64 | 观测到的最大耗时 |
| stats.avg_slow_ms | float | 慢查询平均耗时 |
| stats.capacity | int | 缓冲区容量（默认 200） |
| entries[].sql | string | SQL 语句 |
| entries[].duration_ms | int64 | 耗时（毫秒） |
| entries[].rows | int64 | 影响/返回行数 |
| entries[].error | string | 错误信息（仅出错时携带） |
| entries[].trace_id | string | 关联链路追踪 ID |
| entries[].time | string | 查询发生时间（UTC，倒序） |

**业务说明：** 采集仅在 `database.slow_query_ms > 0` 时启用；缓冲区仅驻内存，进程重启即清零。

### 3.5 重置慢查询统计

`POST /api/v1/admin/dashboard/slow-queries/reset`

清空当前进程的慢查询统计与明细。返回 `{"message": "慢查询统计已重置"}`。

### 3.6 租户工作台总览

`GET /api/v1/dashboard/tenant-overview`

**权限：** 仅需登录（无 RBAC 权限码），租户 ID 从登录上下文获取，不接收跨租户参数。

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "member": { "total": 18, "active": 16 },
    "task": {
      "pending": 12,
      "in_progress": 5,
      "completed": 40,
      "overdue": 3,
      "total": 57,
      "completion_rate": 0.7018
    },
    "wiki": { "spaces": 4, "documents": 126, "views": 8600 },
    "storage_used": 104857600
  }
}
```

**业务说明：**

- 四个维度（成员 / 存储 / Wiki / 任务）互不依赖，经 `errgroup` 并行下发；单维度查询失败仅记日志并按零值降级，不影响其余维度返回。
- 结果按租户写入 Redis 短缓存（**TTL 30 秒**），命中直接返回；未注入 Redis 时整体降级为实时聚合。
- SQL 均为 `COUNT` / `SUM` 条件聚合，兼容 MySQL 与 SQLite；均排除软删行。
- 前端工作台首屏消费该接口，另叠加「待我处理」与快捷入口。

---

## 4. 权限码列表

| 权限码 | 说明 |
|--------|------|
| `admin:dashboard:list` | 查看平台运营数据总览、租户健康度与慢查询看板 |
| `admin:dashboard:create` | 重置慢查询统计缓冲区 |

> `/dashboard/tenant-overview` 为租户侧登录态接口，不参与上表权限码校验。
