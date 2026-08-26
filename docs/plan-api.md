# 套餐管理模块 API 接口说明

> Base URL: `/api/v1`  
> 所属模块：`internal/modules/plan`  
> 路由注册：`internal/modules/plan/routes.go`  
> Handler：`internal/modules/plan/handler/plan_handler.go`  
> DTO：`internal/modules/plan/dto/plan_dto.go`

---

## 1. 接口总览

### 1.1 平台管理员接口（`/api/v1/admin/plans`）

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/admin/plans` | 套餐列表 | `admin:plan:list` |
| GET | `/admin/plans/select` | 启用的套餐下拉列表 | `admin:plan:list` |
| POST | `/admin/plans` | 创建套餐 | `admin:plan:create` |
| PUT | `/admin/plans/{id}/update` | 更新套餐 | `admin:plan:update` |
| DELETE | `/admin/plans/{id}/delete` | 删除套餐 | `admin:plan:delete` |

### 1.2 租户套餐分配（`/api/v1/admin`）

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/admin/tenants-plan/{id}` | 查询租户套餐 | `admin:plan:list` |
| PUT | `/admin/tenants-plan/{id}` | 为租户分配/变更套餐（Plan 模块入口） | `admin:plan:assign` |
| PUT | `/admin/tenants/{id}/plan` | 为租户分配/变更套餐（Tenant 模块入口） | `admin:tenant:update_plan` |

### 1.3 租户侧接口（`/api/v1/tenant/current/plan`）

| 方法 | 路径 | 功能 | 认证 |
|------|------|------|------|
| GET | `/tenant/current/plan` | 获取当前租户套餐与用量 | 需登录 |

---

## 2. 数据结构

### 2.1 PlanResp（套餐响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 套餐 ID |
| name | string | 套餐名称 |
| code | string | 套餐编码（唯一） |
| description | string | 套餐描述 |
| user_limit | int | 用户数上限（-1 表示不限） |
| price | float64 | 月费价格 |
| status | int | 状态：1-启用，0-禁用 |
| subscriber_cnt | int64 | 正在使用该套餐的租户数 |
| created_at | string | 创建时间 |
| updated_at | string | 更新时间 |

### 2.2 CreatePlanReq（创建套餐请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| name | string | 是 | 2-100 | 套餐名称 |
| code | string | 是 | 2-50 | 套餐编码（唯一） |
| description | string | 否 | max=255 | 描述 |
| user_limit | int | 是 | >=-1 | 用户数上限（-1=不限） |
| price | float64 | 是 | >=0 | 月费价格 |
| status | int | 是 | 0 或 1 | 状态 |

### 2.3 UpdatePlanReq（更新套餐请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 套餐名称 |
| description | string | 否 | 描述 |
| user_limit | int | 否 | 用户数上限 |
| price | float64 | 否 | 月费价格 |
| status | int | 否 | 状态 |

### 2.4 AssignPlanReq（分配套餐请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| plan_id | string | 是 | 目标套餐 ID |
| expires_at | string | 否 | 到期时间（格式：2006-01-02 15:04:05） |

### 2.5 CurrentPlanResp（当前套餐响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| tenant_id | string | 租户 ID |
| plan_id | string | 套餐 ID |
| plan_name | string | 套餐名称 |
| plan_code | string | 套餐编码 |
| user_limit | int | 用户数上限 |
| current_users | int64 | 当前用户数 |
| started_at | string | 生效时间 |
| expires_at | string | 到期时间（null=永不过期） |
| status | int | 状态 |
| effective_days | int | 剩余有效天数（负数表示已过期） |

### 2.6 TenantPlanBrief（租户套餐摘要）

| 字段 | 类型 | 说明 |
|------|------|------|
| tenant_id | string | 租户 ID |
| plan_name | string | 套餐名称 |
| expired | bool | 是否已过期 |

---

## 3. 接口详细说明

### 3.1 套餐列表

`GET /api/v1/admin/plans`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页条数 |
| keyword | string | 名称/编码搜索 |
| status | int | 状态筛选 |

**成功响应（200）：** 分页响应，data 为 PlanResp 数组

### 3.2 创建套餐

`POST /api/v1/admin/plans`

**请求体：** CreatePlanReq

**业务规则：**
- `code` 全局唯一
- `user_limit` 为 -1 时表示不限用户数
- 创建后默认为启用状态

### 3.3 更新套餐

`PUT /api/v1/admin/plans/{id}/update`

**请求体：** UpdatePlanReq

### 3.4 删除套餐

`DELETE /api/v1/admin/plans/{id}/delete`

**业务规则：**
- 若有租户正在使用该套餐，不允许删除
- 建议先将使用中的租户迁移到其他套餐后再删除

### 3.5 启用套餐下拉列表

`GET /api/v1/admin/plans/select`

**说明：** 返回所有启用状态的套餐，用于下拉选择框（不分页）

### 3.6 查询租户套餐

`GET /api/v1/admin/tenants-plan/{id}`

**说明：** 查询指定租户当前的套餐信息和用量

**成功响应（200）：**
```json
{
  "code": 200,
  "data": {
    "tenant_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "plan_id": "01ARZ3NDEKTSV4RRFFQ69G5FAX",
    "plan_name": "企业版",
    "plan_code": "enterprise",
    "user_limit": 100,
    "current_users": 45,
    "started_at": "2024-01-01 00:00:00",
    "expires_at": "2025-01-01 00:00:00",
    "status": 1,
    "effective_days": 180
  }
}
```

### 3.7 为租户分配/变更套餐

两个入口等价，均调用 `PlanService.AssignPlan`：

- **Plan 模块入口：** `PUT /api/v1/admin/tenants-plan/{id}`
- **Tenant 模块入口：** `PUT /api/v1/admin/tenants/{id}/plan`（`admin:tenant:update_plan`）

**请求体：** AssignPlanReq

**业务规则：**
- 可指定到期时间，不指定则永久有效
- 变更套餐时旧套餐的订阅记录归档
- 新套餐立即生效

### 3.8 获取当前租户套餐与用量

`GET /api/v1/tenant/current/plan`

**说明：** 租户侧用户可查看当前套餐信息、已使用用户数、剩余有效期

**成功响应（200）：** CurrentPlanResp

---

## 4. 权限码列表

| 权限码 | 说明 |
|--------|------|
| `admin:plan:list` | 查看套餐列表 |
| `admin:plan:create` | 创建套餐 |
| `admin:plan:update` | 更新套餐 |
| `admin:plan:delete` | 删除套餐 |
| `admin:plan:assign` | 为租户分配/变更套餐 |

---

## 5. 套餐与用户限制

系统在用户管理接口中自动检查套餐用户上限：

- 当 `current_users >= user_limit` 时，创建新用户接口返回 403
- 套餐过期后用户登录返回 403（需续费）
- `user_limit = -1` 表示不限制用户数

检查逻辑位置：`internal/modules/user/service/user_service.go`