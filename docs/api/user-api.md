# 用户管理模块 API 接口说明

> Base URL: `/api/v1`  
> 所属模块：`internal/modules/user`  
> 路由注册：`internal/modules/user/routes.go`  
> Handler：`internal/modules/user/handler/user_handler.go`  
> DTO：`internal/modules/user/dto/user_dto.go`

---

## 1. 接口总览

### 1.1 当前用户个人信息（`/api/v1/profile`）

| 方法 | 路径 | 功能 | 权限 |
|------|------|------|------|
| GET | `/profile/stats` | 获取 Dashboard 统计（用户总数） | 需登录 |
| GET | `/profile` | 获取当前用户个人信息 | 需登录 |
| PUT | `/profile` | 更新当前用户个人信息 | 需登录 |
| PUT | `/profile/password` | 修改当前用户密码 | 需登录 |

### 1.2 租户用户管理（`/api/v1/users`）

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/users` | 用户列表 | `user:list` |
| POST | `/users` | 创建用户 | `user:create` |
| GET | `/users/deleted` | 已删除用户列表 | `user:list` |
| GET | `/users/{id}/detail` | 用户详情 | `user:read` |
| PUT | `/users/{id}/update` | 更新用户信息 | `user:update` |
| PUT | `/users/{id}/reset-password` | 重置用户密码 | `user:reset_password` |
| DELETE | `/users/{id}/delete` | 删除用户（软删除） | `user:delete` |
| PUT | `/users/{id}/restore` | 恢复已删除用户 | `user:restore` |
| DELETE | `/users/{id}/permanent` | 永久删除用户 | `user:permanent_delete` |

### 1.3 系统管理员管理（`/api/v1/admin`）

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/admin/stats` | 全平台 Dashboard 统计 | 超级管理员 |
| GET | `/admin/users` | 系统管理员列表 | `admin:master:list` |
| POST | `/admin/users` | 创建系统管理员 | `admin:master:create` |
| GET | `/admin/users/deleted` | 已删除系统管理员列表 | `admin:master:list` |
| GET | `/admin/users/{id}/detail` | 系统管理员详情 | `admin:master:read` |
| PUT | `/admin/users/{id}/update` | 更新系统管理员 | `admin:master:update` |
| PUT | `/admin/users/{id}/status` | 更新管理员状态 | `admin:master:status` |
| DELETE | `/admin/users/{id}/delete` | 删除管理员（软删除） | `admin:master:delete` |
| PUT | `/admin/users/{id}/restore` | 恢复已删除管理员 | `admin:master:restore` |
| DELETE | `/admin/users/{id}/permanent` | 永久删除管理员 | `admin:master:permanent_delete` |
| PUT | `/admin/users/batch/status` | 批量更新管理员状态 | `admin:master:batch_status` |
| DELETE | `/admin/users/batch/delete` | 批量删除管理员 | `admin:master:batch_delete` |

### 1.4 跨租户用户管理（`/api/v1/admin/tenant-users`）

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| POST | `/admin/tenant-users` | 为指定租户创建用户 | `admin:tenant_user:create` |
| GET | `/admin/tenant-users/all` | 所有租户用户列表 | `admin:tenant_user:list` |
| GET | `/admin/tenant-users/deleted/all` | 所有租户已删除用户 | `admin:tenant_user:list` |
| GET | `/admin/tenant-users/{tenantID}/list` | 指定租户用户列表 | `admin:tenant_user:list` |
| GET | `/admin/tenant-users/{tenantID}/deleted` | 指定租户已删除用户 | `admin:tenant_user:list` |
| PUT | `/admin/tenant-users/{tenantID}/{userID}/update` | 更新租户用户 | `admin:tenant_user:update` |
| PUT | `/admin/tenant-users/{tenantID}/{userID}/status` | 更新租户用户状态 | `admin:tenant_user:status` |
| PUT | `/admin/tenant-users/{tenantID}/{userID}/reset-password` | 重置租户用户密码 | `admin:tenant_user:reset_password` |
| PUT | `/admin/tenant-users/{tenantID}/{userID}/restore` | 恢复租户用户 | `admin:tenant_user:restore` |
| DELETE | `/admin/tenant-users/{tenantID}/{userID}/delete` | 删除租户用户 | `admin:tenant_user:delete` |
| DELETE | `/admin/tenant-users/{tenantID}/{userID}/permanent` | 永久删除租户用户 | `admin:tenant_user:permanent_delete` |
| PUT | `/admin/tenant-users/{tenantID}/batch/status` | 批量更新租户用户状态 | `admin:tenant_user:batch_status` |
| DELETE | `/admin/tenant-users/{tenantID}/batch/delete` | 批量删除租户用户 | `admin:tenant_user:batch_delete` |

---

## 2. 数据结构

### 2.1 UserResp（用户响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 用户 ID (ULID) |
| tenant_id | string | 租户 ID |
| tenant_name | string | 租户名称 |
| username | string | 用户名 |
| nickname | string | 昵称 |
| email | string | 邮箱 |
| roles | []string | 角色编码列表（兼容） |
| role_ids | []string | 角色 ID 列表 |
| role_list | []UserRoleInfo | 角色详细信息列表 |
| status | int | 状态：`1` 启用，`0` 禁用 |
| is_master | bool | 是否为系统管理员 |
| created_at | string | 创建时间 |
| updated_at | string | 更新时间 |
| deleted_at | string | 软删除时间（仅回收站有值） |

### 2.2 UserRoleInfo（角色简化信息）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 角色 ID |
| name | string | 角色名称 |
| code | string | 角色编码 |

### 2.3 CreateUserReq（创建用户请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| username | string | 是 | alphanum, 4-50 | 用户名 |
| password | string | 是 | 6-32 | 初始密码 |
| nickname | string | 是 | max=50 | 昵称 |
| email | string | 否 | 邮箱格式 | 邮箱 |
| role_ids | []string | 是 | min=1 | 分配的角色 ID 列表 |

### 2.4 UpdateUserReq（更新用户请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| nickname | string | 否 | max=50 | 昵称 |
| email | string | 否 | 邮箱格式 | 邮箱 |
| role_ids | []string | 否 | min=1 | 角色 ID 列表（覆盖式更新） |
| status | *int | 否 | 0 或 1 | 启用/禁用 |

### 2.5 ChangePasswordReq（修改密码请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| old_password | string | 是 | 6-32 | 原密码 |
| new_password | string | 是 | 6-32 | 新密码 |
| confirm_password | string | 是 | 与 new_password 一致 | 确认密码 |

### 2.6 ResetPasswordReq（重置密码请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| new_password | string | 是 | 新密码 |
| confirm_password | string | 是 | 确认密码 |

### 2.7 UpdateUserStatusReq（更新状态请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| status | int | 是 | 1-启用，0-禁用 |

### 2.8 BatchUpdateUserStatusReq（批量更新状态）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | []string | 是 | 用户 ID 列表 |
| status | int | 是 | 1-启用，0-禁用 |

### 2.9 BatchDeleteUsersReq（批量删除）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | []string | 是 | 用户 ID 列表 |

---

## 3. 接口详细说明

### 3.1 获取当前用户个人信息

`GET /api/v1/profile`

**成功响应（200）：** UserResp

### 3.2 更新当前用户个人信息

`PUT /api/v1/profile`

**请求体：** UpdateUserReq（nickname、email 可选择性更新）

**成功响应（200）：**
```json
{ "code": 200, "data": null }
```

### 3.3 修改当前用户密码

`PUT /api/v1/profile/password`

**请求体：** ChangePasswordReq

**业务规则：** 需验证原密码，新密码不能与原密码相同

**成功响应（200）：**
```json
{ "code": 200, "message": "密码修改成功", "data": null }
```

### 3.4 用户列表

`GET /api/v1/users`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页条数，默认 10 |
| keyword | string | 按用户名/昵称/邮箱搜索 |
| role_id | string | 按角色筛选 |
| status | int | 按状态筛选 |

**成功响应（200）：** 分页响应，data 为 UserResp 数组

### 3.5 创建用户

`POST /api/v1/users`

**请求体：** CreateUserReq

**业务规则：** 用户名在当前租户内唯一

**成功响应（200）：** 创建的 UserResp

### 3.6 更新用户信息

`PUT /api/v1/users/{id}/update`

**请求体：** UpdateUserReq

**说明：** 当前租户管理员仅能管理本租户用户

### 3.7 重置用户密码

`PUT /api/v1/users/{id}/reset-password`

**请求体：** ResetPasswordReq

**说明：** 管理员重置密码，无需原密码验证

### 3.8 删除用户（软删除）

`DELETE /api/v1/users/{id}/delete`

**说明：** 软删除，用户可在回收站恢复

### 3.9 恢复已删除用户

`PUT /api/v1/users/{id}/restore`

### 3.10 永久删除用户

`DELETE /api/v1/users/{id}/permanent`

**说明：** 物理删除，不可恢复

---

## 4. 权限码列表

### 租户管理员权限

| 权限码 | 说明 |
|--------|------|
| `user:list` | 查看用户列表 |
| `user:create` | 创建用户 |
| `user:read` | 查看用户详情 |
| `user:update` | 更新用户信息 |
| `user:reset_password` | 重置用户密码 |
| `user:delete` | 删除用户 |
| `user:restore` | 恢复已删除用户 |
| `user:permanent_delete` | 永久删除用户 |
| `user:batch_status` | 批量更新状态 |
| `user:batch_delete` | 批量删除 |

### 系统管理员权限

| 权限码 | 说明 |
|--------|------|
| `admin:master:list` | 查看系统管理员列表 |
| `admin:master:create` | 创建系统管理员 |
| `admin:master:read` | 查看管理员详情 |
| `admin:master:update` | 更新管理员 |
| `admin:master:status` | 更新管理员状态 |
| `admin:master:delete` | 删除管理员 |
| `admin:master:restore` | 恢复已删除管理员 |
| `admin:master:permanent_delete` | 永久删除管理员 |
| `admin:master:batch_status` | 批量更新管理员状态 |
| `admin:master:batch_delete` | 批量删除管理员 |
| `admin:tenant_user:create` | 跨租户创建用户 |
| `admin:tenant_user:list` | 跨租户查看用户 |
| `admin:tenant_user:update` | 跨租户更新用户 |
| `admin:tenant_user:status` | 跨租户更新状态 |
| `admin:tenant_user:reset_password` | 跨租户重置密码 |
| `admin:tenant_user:restore` | 跨租户恢复用户 |
| `admin:tenant_user:delete` | 跨租户删除用户 |
| `admin:tenant_user:permanent_delete` | 跨租户永久删除 |
| `admin:tenant_user:batch_status` | 跨租户批量更新状态 |
| `admin:tenant_user:batch_delete` | 跨租户批量删除 |