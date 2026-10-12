# 运营看板 API 文档

数据看板模块（`internal/modules/dashboard`）提供平台运营数据总览，供超级管理员掌握平台健康状况（租户 / 用户 / 订阅 / 审计多维统计）。

**权限：** 需超级管理员角色，且具备 `admin:dashboard:list` 权限码（`RequiresMasterAdmin` + `AutoRequirePermission`）。

**基础路径：** `/api/v1`

---

## 1. 接口列表

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/admin/dashboard/overview` | 平台运营数据总览 | `admin:dashboard:list` |
| GET | `/admin/dashboard/tenants-health` | 租户健康度列表（含分级汇总） | `admin:dashboard:list` |
| GET | `/admin/dashboard/tenants-health/{id}` | 单租户健康度详情 | `admin:dashboard:list` |

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

### 3.3 单租户健康度详情

`GET /api/v1/admin/dashboard/tenants-health/{id}`

**成功响应（200）：** TenantHealthItemResp

**错误响应：** 租户不存在返回 404。

---

## 4. 权限码列表

| 权限码 | 说明 |
|--------|------|
| `admin:dashboard:list` | 查看平台运营数据总览 |
