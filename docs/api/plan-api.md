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
| storage_limit | int64 | 存储配额（字节，-1=不限） |
| api_rate_limit | int | API 速率上限（次/分钟，-1=不限） |
| wiki_space_limit | int | Wiki 空间上限（个，-1=不限） |
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
| storage_limit | int64 | 否 | >=-1 | 存储配额（字节），传 0 或不传归一化为 -1（不限） |
| api_rate_limit | int | 否 | >=-1 | API 速率上限（次/分钟），0 归一化为 -1 |
| wiki_space_limit | int | 否 | >=-1 | Wiki 空间上限，0 归一化为 -1 |
| price | float64 | 是 | >=0 | 月费价格 |
| status | int | 是 | 0 或 1 | 状态 |

### 2.3 UpdatePlanReq（更新套餐请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 套餐名称 |
| description | string | 否 | 描述 |
| user_limit | int | 否 | 用户数上限 |
| storage_limit | int64 | 否 | 存储配额（字节，-1=不限） |
| api_rate_limit | int | 否 | API 速率上限（次/分钟，-1=不限） |
| wiki_space_limit | int | 否 | Wiki 空间上限（-1=不限） |
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
| storage_limit | int64 | 存储配额（字节，-1=不限） |
| current_storage | int64 | 当前存储用量（字节，best-effort 回填） |
| wiki_space_limit | int | Wiki 空间上限（-1=不限） |
| current_wiki_spaces | int64 | 当前 Wiki 空间数（best-effort 回填） |
| api_rate_limit | int | API 速率上限（次/分钟，-1=不限） |
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
    "storage_limit": 10737418240,
    "current_storage": 524288000,
    "wiki_space_limit": 10,
    "current_wiki_spaces": 2,
    "api_rate_limit": 300,
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

**说明：** 租户侧用户可查看当前套餐信息、已使用用户数/存储/Wiki 空间、剩余有效期

**成功响应（200）：** CurrentPlanResp

**业务规则：**
- 当前租户未开通套餐时返回业务错误「当前租户尚未开通套餐」
- `current_storage` / `current_wiki_spaces` 为 best-effort 回填：用量提供者异常时保持零值，不阻断接口返回

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

## 5. 套餐配额与限制

系统在相关接口中自动检查套餐配额：

### 5.1 用户数限制（user_limit）

- 当 `current_users >= user_limit` 时，创建新用户接口返回 403
- 套餐过期后用户登录返回 403（需续费）
- `user_limit = -1` 表示不限制用户数

检查逻辑位置：`internal/modules/user/service/user_service.go`

### 5.2 API 速率限制（api_rate_limit）

- 中间件：`internal/middleware/tenant_rate_limit.go`，挂载于全部受保护路由（`internal/bootstrap/router.go`）
- 以自然分钟为窗口，Redis key `tenant:rate:{tenantID}:{minute}`，INCR + 首次 EXPIRE 60s
- 超限返回 **429**，响应头携带 `Retry-After`（距本分钟结束的秒数）
- fail-open：无租户上下文、配额为 -1、Redis 不可用或配额查询失败时均直接放行
- 配额本地缓存 5 分钟，避免每请求查库

### 5.3 存储配额（storage_limit）

- 文件上传前比对已用存储（`SumSizeByTenant`）+ 本次大小与配额，超限返回 **507**（Insufficient Storage）
- 检查逻辑位置：`internal/modules/file/handler/file_handler.go` / `FileService.Upload`

### 5.4 Wiki 空间配额（wiki_space_limit）

- 创建空间前比对现有空间数（`CountSpacesByTenant`）与配额，超限返回 **402**（`ErrTenantLimit`）
- 检查逻辑位置：`internal/modules/wiki` `CreateSpace`

### 5.5 配额归一化规则

- 创建套餐时三类配额未显式设置（传 0 或不传）统一归一化为 **-1（不限）**；显式传 -1 同样为不限
- 前端展示约定：`-1` 统一显示「不限」；存储配额以 GB 输入，与字节互转

### 5.6 订阅到期自动处理（后台任务）

由 `StartPlanExpiryJob`（`internal/bootstrap/router.go`）挂载，`internal/modules/plan/expiry_job.go`，**扫描间隔 5 分钟**，只处理 `status = active` 且 `expires_at IS NOT NULL AND expires_at < now` 的订阅（永久有效订阅永不参与）。

每轮顺序固定为「先禁用租户，后翻转订阅状态」，两步不可颠倒：

1. 禁用租户（`tenant.status = disabled`，可幂等重放）
2. 翻转订阅为 `expired`

原因：驱动本轮扫描的条件是 `status = active`。若先翻转，一旦随后禁用失败，这行订阅再也不会被扫到——租户停在 `enabled`，等于到期后免费续期。按现顺序，任一步失败都会打日志并保持 `active`，由下一轮（5 分钟后）自动重入，无需人工清库。

**到期提醒：** 同一轮扫描后无条件执行，窗口 7 天（`expiryRemindWindow`），向租户 `contact_email` 推送剩余天数。提醒去重键为「订阅 ID + 到期日」，去重集合是**进程内**的：

- 服务重启后，同一到期日会再发一次提醒
- 多副本部署时各副本独立去重，会各发一份

**外部可见影响：** 租户被禁用后其成员登录返回 403（见 5.1）。重新分配套餐（3.7）**不会**自动恢复启用，需平台管理员走租户状态变更接口重新启用（见 tenant-api.md《3.9 管理员更新租户状态》）。