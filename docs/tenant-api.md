# 租户管理模块 API 接口说明

> Base URL: `/api/v1`  
> 所属模块：`internal/modules/tenant`  
> 路由注册：`internal/modules/tenant/routes.go`  
> Handler：`internal/modules/tenant/handler/tenant_handler.go`  
> DTO：`internal/modules/tenant/dto/tenant_dto.go`

---

## 1. 接口总览

### 1.1 公开接口（无需认证）

| 方法 | 路径 | 功能 |
|------|------|------|
| POST | `/tenants/register` | 租户自主注册（开户） |

### 1.2 租户侧接口（需租户用户 Token）

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/tenants/current` | 获取当前租户详情 |
| PUT | `/tenants/current` | 更新当前租户信息 |
| GET | `/tenants/current/status` | 查询租户初始化/开通状态 |
| POST | `/tenants/current/cancel` | 申请租户注销 |

### 1.3 平台管理员接口（需超级管理员）

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| POST | `/admin/tenants` | 创建租户 | `admin:tenant:create` |
| GET | `/admin/tenants` | 租户列表 | `admin:tenant:list` |
| GET | `/admin/tenants/deleted` | 已删除租户列表 | `admin:tenant:list` |
| GET | `/admin/tenants/{id}/detail` | 租户详情 | `admin:tenant:read` |
| PUT | `/admin/tenants/{id}/update` | 更新租户信息 | `admin:tenant:update` |
| PUT | `/admin/tenants/{id}/status` | 更新租户状态 | `admin:tenant:status` |
| DELETE | `/admin/tenants/{id}/delete` | 删除租户 | `admin:tenant:delete` |
| PUT | `/admin/tenants/{id}/restore` | 恢复已删除租户 | `admin:tenant:restore` |
| PUT | `/admin/tenants/batch/status` | 批量更新租户状态 | `admin:tenant:batch_status` |
| DELETE | `/admin/tenants/batch` | 批量删除租户 | `admin:tenant:batch_delete` |

---

## 2. 数据结构

### 2.1 RegisterTenantReq（租户注册请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| name | string | 是 | 2-100 | 租户名称（企业名称） |
| domain | string | 是 | username, 3-30 | 租户域名（唯一） |
| description | string | 否 | max=255 | 租户描述 |
| contact_email | string | 否 | 邮箱格式 | 联系邮箱 |
| region | string | 否 | max=50 | 所在地区 |
| logo | string | 否 | URL 格式 | Logo 图片地址 |
| extra | string | 否 | max=1000 | 扩展字段（JSON） |
| admin_user | object | 是 | - | 初始管理员信息 |
| admin_user.username | string | 是 | 4-50 | 管理员用户名 |
| admin_user.password | string | 是 | 6-32 | 管理员密码 |
| admin_user.nickname | string | 是 | max=50 | 管理员昵称 |
| admin_user.email | string | 否 | 邮箱格式 | 管理员邮箱 |

### 2.2 UpdateTenantReq（管理员更新租户）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 租户名称 |
| status | int | 是 | 0-禁用，1-正常 |
| description | string | 否 | 描述 |

### 2.3 UpdateCurrentTenantReq（租户侧更新）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 租户名称 |
| logo | string | 否 | Logo URL |
| description | string | 否 | 描述 |
| contact_email | string | 否 | 联系邮箱 |
| region | string | 否 | 地区 |
| extra | string | 否 | 扩展字段 |

### 2.4 TenantResp（租户响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 租户 ID (ULID) |
| name | string | 租户名称 |
| domain | string | 租户域名 |
| status | int | 状态：1-正常，0-禁用 |
| description | string | 描述 |
| contact_email | string | 联系邮箱 |
| region | string | 地区 |
| logo | string | Logo URL |
| extra | string | 扩展字段 |
| created_at | string | 创建时间 |
| updated_at | string | 更新时间 |

### 2.5 GetInitStatusResp（初始化状态响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| status | string | pending/initializing/completed/failed |
| message | string | 状态描述 |
| progress | int | 进度百分比 0-100 |
| initialized | bool | 是否已完成 |

### 2.6 ApplyCancellationReq（注销申请）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| reason | string | 是 | 注销原因（max=500） |

### 2.7 ApplyCancellationResp（注销响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| applied_at | string | 申请时间 |
| status | string | pending/approved/rejected |
| estimated_day | int | 预计注销天数 |

---

## 3. 接口详细说明

### 3.1 租户自主注册

`POST /api/v1/tenants/register`

**请求体：** RegisterTenantReq

**业务规则：** domain 全局唯一；自动创建租户+初始管理员账户

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "注册成功",
  "data": {
    "tenant_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "user_id": "01ARZ3NDEKTSV4RRFFQ69G5FAW",
    "initialized": false
  }
}
```

### 3.2 获取当前租户详情

`GET /api/v1/tenants/current`

**成功响应（200）：** TenantResp

### 3.3 更新当前租户信息

`PUT /api/v1/tenants/current`

**请求体：** UpdateCurrentTenantReq

### 3.4 查询租户初始化状态

`GET /api/v1/tenants/current/status`

**成功响应（200）：** GetInitStatusResp

### 3.5 申请租户注销

`POST /api/v1/tenants/current/cancel`

**请求体：** ApplyCancellationReq

**说明：** 需填写注销原因；宽限期结束后数据物理删除

### 3.6 管理员创建租户

`POST /api/v1/admin/tenants`

**请求体：** RegisterTenantReq

### 3.7 管理员租户列表

`GET /api/v1/admin/tenants`

**Query 参数：** page, page_size, keyword, status

**成功响应（200）：** 分页响应

### 3.8 管理员更新租户

`PUT /api/v1/admin/tenants/{id}/update`

### 3.9 管理员更新租户状态

`PUT /api/v1/admin/tenants/{id}/status`

### 3.10 管理员删除租户

`DELETE /api/v1/admin/tenants/{id}/delete`

### 3.11 管理员恢复租户

`PUT /api/v1/admin/tenants/{id}/restore`

### 3.12 批量操作

- `PUT /api/v1/admin/tenants/batch/status`
- `DELETE /api/v1/admin/tenants/batch`

---

## 4. 权限码列表

| 权限码 | 说明 |
|--------|------|
| `admin:tenant:create` | 创建租户 |
| `admin:tenant:list` | 查看租户列表 |
| `admin:tenant:read` | 查看租户详情 |
| `admin:tenant:update` | 更新租户信息 |
| `admin:tenant:status` | 更新租户状态 |
| `admin:tenant:delete` | 删除租户 |
| `admin:tenant:restore` | 恢复已删除租户 |
| `admin:tenant:batch_status` | 批量更新状态 |
| `admin:tenant:batch_delete` | 批量删除租户 |