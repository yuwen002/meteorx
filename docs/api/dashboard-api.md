# 运营看板 API 文档

数据看板模块（`internal/modules/dashboard`）提供平台运营数据总览，供超级管理员掌握平台健康状况（租户 / 用户 / 订阅 / 审计多维统计）。

**权限：** 需超级管理员角色，且具备 `admin:dashboard:list` 权限码（`RequiresMasterAdmin` + `AutoRequirePermission`）。

**基础路径：** `/api/v1`

---

## 1. 接口列表

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/admin/dashboard/overview` | 平台运营数据总览 | `admin:dashboard:list` |

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

---

## 4. 权限码列表

| 权限码 | 说明 |
|--------|------|
| `admin:dashboard:list` | 查看平台运营数据总览 |
