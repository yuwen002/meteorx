# RBAC 权限管理模块 API 接口说明

> Base URL: `/api/v1`  
> 所属模块：`internal/modules/rbac`  
> 路由注册：`internal/modules/rbac/routes.go`  
> DTO：`internal/modules/rbac/dto/`  
> 权限码定义：`internal/modules/rbac/permissions.go`

---

## 1. 接口总览

### 1.1 Dashboard 统计

| 方法 | 路径 | 功能 | 认证 |
|------|------|------|------|
| GET | `/rbac/stats` | RBAC 模块统计 | 需登录 |

### 1.2 角色管理（`/api/v1/rbac/roles`）

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/rbac/roles` | 角色列表（分页） | `rbac:role:list` |
| GET | `/rbac/roles/select` | 角色下拉列表（不分页） | `rbac:role:list_select` |
| GET | `/rbac/roles/system-admin` | 系统管理员角色列表 | `rbac:role:list_system_admin` |
| POST | `/rbac/roles` | 创建角色 | `rbac:role:create` |
| GET | `/rbac/roles/{id}/detail` | 角色详情 | `rbac:role:read` |
| PUT | `/rbac/roles/{id}/update` | 更新角色 | `rbac:role:update` |
| PUT | `/rbac/roles/{id}/status` | 更新角色状态 | `rbac:role:status` |
| DELETE | `/rbac/roles/{id}/delete` | 删除角色 | `rbac:role:delete` |
| PUT | `/rbac/roles/{id}/restore` | 恢复已删除角色 | `rbac:role:restore` |
| PUT | `/rbac/roles/{id}/permissions` | 绑定角色权限（覆盖式） | `rbac:role:bind_perm` |
| GET | `/rbac/roles/{id}/permissions` | 获取角色已绑定权限 | `rbac:role:get_perms` |
| DELETE | `/rbac/roles/{id}/permissions` | 解绑角色单个权限 | `rbac:role:unbind_perm` |
| DELETE | `/rbac/roles/{id}/permissions/batch` | 批量解绑角色权限 | `rbac:role:batch_unbind_perm` |
| GET | `/rbac/roles/deleted` | 已删除角色列表 | `rbac:role:list_deleted` |
| PUT | `/rbac/roles/batch/status` | 批量更新角色状态 | `rbac:role:batch_status` |
| DELETE | `/rbac/roles/batch/delete` | 批量删除角色 | `rbac:role:batch_delete` |
| PUT | `/rbac/roles/batch/permissions` | 批量为多个角色绑定权限 | `rbac:role:batch_bind` |
| DELETE | `/rbac/roles/batch/permissions` | 批量为多个角色解绑权限 | `rbac:role:batch_unbind` |

### 1.3 权限管理（`/api/v1/rbac/permissions`）

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/rbac/permissions` | 权限列表（分页） | `rbac:perm:list` |
| POST | `/rbac/permissions` | 创建权限 | `rbac:perm:create` |
| GET | `/rbac/permissions/{id}/detail` | 权限详情 | `rbac:perm:read` |
| PUT | `/rbac/permissions/{id}/update` | 更新权限 | `rbac:perm:update` |
| PUT | `/rbac/permissions/{id}/status` | 更新权限状态 | `rbac:perm:status` |
| DELETE | `/rbac/permissions/{id}/delete` | 删除权限 | `rbac:perm:delete` |
| PUT | `/rbac/permissions/batch/status` | 批量更新权限状态 | `rbac:perm:batch_status` |
| DELETE | `/rbac/permissions/batch/delete` | 批量删除权限 | `rbac:perm:batch_delete` |

### 1.4 角色权限关系（`/api/v1/rbac/role-permissions`）

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/rbac/role-permissions` | 角色权限关系列表 | `rbac:role_perm:list` |

### 1.5 用户角色管理（`/api/v1/rbac/user-roles`）

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/rbac/user-roles` | 用户角色关系列表 | `rbac:user_role:list` |
| POST | `/rbac/user-roles/batch/assign` | 批量分配用户角色 | `rbac:user_role:batch_assign` |
| GET | `/rbac/user-roles/{user_id}/roles` | 获取用户角色列表 | `rbac:user_role:get_roles` |
| POST | `/rbac/user-roles/{user_id}/roles` | 分配用户角色 | `rbac:user_role:assign` |
| DELETE | `/rbac/user-roles/{user_id}/roles` | 移除用户所有角色 | `rbac:user_role:remove_all` |
| DELETE | `/rbac/user-roles/{user_id}/roles/{role_id}` | 移除用户单个角色 | `rbac:user_role:remove_one` |
| GET | `/rbac/user-roles/roles/{role_id}/users` | 获取角色下用户列表 | `rbac:user_role:get_users` |

---

## 2. 数据结构

### 2.1 RoleResp（角色响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 角色 ID |
| name | string | 角色名称 |
| code | string | 角色编码（唯一，如 `admin`, `tenant_admin`） |
| description | string | 角色描述 |
| tenant_id | string | 租户 ID（空则为系统级） |
| is_system | bool | 是否系统内置（内置角色不可删除） |
| scope | string | 作用域：`system`/`tenant`/`all` |
| status | int | 状态：1-启用，0-禁用 |
| created_at | string | 创建时间 |
| updated_at | string | 更新时间 |

### 2.2 PermissionResp（权限响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 权限 ID |
| name | string | 权限名称（中文） |
| code | string | 权限编码（如 `file:upload`） |
| description | string | 权限描述 |
| resource | string | 资源标识（如 `file`, `user`） |
| action | string | 操作标识（如 `list`, `create`） |
| status | int | 状态：1-启用，0-禁用 |
| created_at | string | 创建时间 |
| updated_at | string | 更新时间 |

### 2.3 RolePermissionResp（角色-权限关系）

| 字段 | 类型 | 说明 |
|------|------|------|
| role_id | string | 角色 ID |
| permission_id | string | 权限 ID |
| created_at | string | 绑定时间 |
| role | SimpleRoleResp | 角色简要信息 |
| permission | SimplePermissionResp | 权限简要信息 |

### 2.4 UserRoleResp（用户-角色关系）

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | string | 用户 ID |
| role_id | string | 角色 ID |
| created_at | string | 绑定时间 |
| user | SimpleUserInfoResp | 用户简要信息 |
| role | SimpleRoleResp | 角色简要信息 |

### 2.5 CreateRoleReq（创建角色请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| name | string | 是 | max=50 | 角色名称 |
| code | string | 是 | max=50 | 角色编码（唯一） |
| description | string | 否 | max=255 | 描述 |
| tenant_id | string | 否 | max=26 | 租户 ID（空为系统级） |
| is_system | bool | 否 | - | 是否系统内置 |
| scope | string | 否 | system/tenant/all | 作用域，默认 tenant |
| status | int | 否 | 0 或 1 | 状态，默认 1 |

### 2.6 BindRolePermissionsReq（绑定角色权限请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| permission_ids | []string | 是 | 权限 ID 列表（覆盖式） |

### 2.7 AssignUserRolesReq（分配用户角色）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| role_ids | []string | 是 | 角色 ID 列表（全覆盖式更新） |

### 2.8 BatchAssignUserRolesReq（批量分配）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| assignments | []UserRoleAssignment | 是 | 分配列表 |

**UserRoleAssignment：**

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | string | 用户 ID |
| role_ids | []string | 角色 ID 列表 |

---

## 3. 接口详细说明

### 3.1 角色列表

`GET /api/v1/rbac/roles`

**Query 参数：** page, page_size, keyword（按名称/编码搜索）

**成功响应（200）：** 分页响应

### 3.2 创建角色

`POST /api/v1/rbac/roles`

**请求体：** CreateRoleReq

**说明：** 系统内置角色（is_system=true）不可删除

### 3.3 更新角色

`PUT /api/v1/rbac/roles/{id}/update`

### 3.4 删除角色

`DELETE /api/v1/rbac/roles/{id}/delete`

**说明：** 系统内置角色不可删除

### 3.5 绑定角色权限（覆盖式）

`PUT /api/v1/rbac/roles/{id}/permissions`

**请求体：**
```json
{ "permission_ids": ["perm_id_1", "perm_id_2"] }
```

**说明：** 覆盖式更新，传入的列表为最终结果

### 3.6 获取角色权限

`GET /api/v1/rbac/roles/{id}/permissions`

**成功响应（200）：** 权限 ID 列表

### 3.7 解绑角色单个权限

`DELETE /api/v1/rbac/roles/{id}/permissions?permission_id=xxx`

### 3.8 批量解绑角色权限

`DELETE /api/v1/rbac/roles/{id}/permissions/batch`

**请求体：**
```json
{ "permission_ids": ["perm_id_1", "perm_id_2"] }
```

### 3.9 分配用户角色

`POST /api/v1/rbac/user-roles/{user_id}/roles`

**请求体：** AssignUserRolesReq

### 3.10 获取用户角色

`GET /api/v1/rbac/user-roles/{user_id}/roles`

### 3.11 移除用户所有角色

`DELETE /api/v1/rbac/user-roles/{user_id}/roles`

### 3.12 移除用户单个角色

`DELETE /api/v1/rbac/user-roles/{user_id}/roles/{role_id}`

### 3.13 获取角色下用户

`GET /api/v1/rbac/user-roles/roles/{role_id}/users`

### 3.14 批量分配用户角色

`POST /api/v1/rbac/user-roles/batch/assign`

**请求体：**
```json
{
  "assignments": [
    { "user_id": "user_1", "role_ids": ["role_1"] },
    { "user_id": "user_2", "role_ids": ["role_2", "role_3"] }
  ]
}
```

---

## 4. 权限码完整列表

### 角色管理

| 权限码 | 说明 |
|--------|------|
| `rbac:role:list` | 角色列表 |
| `rbac:role:list_select` | 角色下拉选项 |
| `rbac:role:list_system_admin` | 系统管理员角色列表 |
| `rbac:role:list_deleted` | 已删除角色列表 |
| `rbac:role:create` | 创建角色 |
| `rbac:role:read` | 角色详情 |
| `rbac:role:update` | 更新角色 |
| `rbac:role:status` | 更新角色状态 |
| `rbac:role:delete` | 删除角色 |
| `rbac:role:restore` | 恢复已删除角色 |
| `rbac:role:batch_status` | 批量更新状态 |
| `rbac:role:batch_delete` | 批量删除 |
| `rbac:role:bind_perm` | 绑定权限 |
| `rbac:role:get_perms` | 获取权限 |
| `rbac:role:unbind_perm` | 解绑权限 |
| `rbac:role:batch_unbind_perm` | 批量解绑 |
| `rbac:role:batch_bind` | 批量绑定权限 |
| `rbac:role:batch_unbind` | 批量解绑权限 |

### 权限管理

| 权限码 | 说明 |
|--------|------|
| `rbac:perm:list` | 权限列表 |
| `rbac:perm:create` | 创建权限 |
| `rbac:perm:read` | 权限详情 |
| `rbac:perm:update` | 更新权限 |
| `rbac:perm:status` | 更新权限状态 |
| `rbac:perm:delete` | 删除权限 |
| `rbac:perm:batch_status` | 批量更新状态 |
| `rbac:perm:batch_delete` | 批量删除 |

### 角色权限关系

| 权限码 | 说明 |
|--------|------|
| `rbac:role_perm:list` | 关系列表 |

### 用户角色关系

| 权限码 | 说明 |
|--------|------|
| `rbac:user_role:list` | 关系列表 |
| `rbac:user_role:assign` | 分配角色 |
| `rbac:user_role:batch_assign` | 批量分配 |
| `rbac:user_role:get_roles` | 获取用户角色 |
| `rbac:user_role:remove_all` | 移除所有角色 |
| `rbac:user_role:remove_one` | 移除单个角色 |
| `rbac:user_role:get_users` | 获取角色下用户 |

---

## 5. 自动权限推导机制

路由使用 `AutoRequirePermission` 中间件，根据 HTTP Method + 路径自动推导权限码：

| HTTP Method | 权限 Action 映射 |
|-------------|------------------|
| GET | `list` / `read` |
| POST | `create` / `assign` |
| PUT | `update` / `status` / `restore` |
| DELETE | `delete` |

推导规则：`{module}:{resource}:{action}`

**示例：**
- `GET /rbac/roles` → `rbac:role:list`
- `POST /rbac/roles` → `rbac:role:create`
- `PUT /rbac/roles/{id}/update` → `rbac:role:update`