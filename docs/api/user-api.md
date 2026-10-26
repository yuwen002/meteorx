# 用户管理模块 API 接口说明

> Base URL: `/api/v1`  
> 所属模块：`internal/modules/user`  
> 路由注册：`internal/modules/user/routes.go`  
> Handler：`internal/modules/user/handler/user_handler.go`、`handler/user_import.go`（批量导入）  
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
| POST | `/users/import` | 批量导入用户（CSV/XLSX） | `user:create` |
| GET | `/users/deleted` | 已删除用户列表 | `user:list` |
| GET | `/users/{id}/detail` | 用户详情 | `user:read` |
| PUT | `/users/{id}/update` | 更新用户信息 | `user:update` |
| PUT | `/users/{id}/reset-password` | 重置用户密码 | `user:reset_password` |
| DELETE | `/users/{id}/delete` | 删除用户（软删除） | `user:delete` |
| PUT | `/users/{id}/restore` | 恢复已删除用户 | `user:restore` |
| DELETE | `/users/{id}/permanent` | 永久删除用户 | `user:permanent_delete` |
| PUT | `/users/batch/status` | 批量启用/禁用用户 | `user:batch_status` |
| DELETE | `/users/batch/delete` | 批量删除用户（软删除） | `user:batch_delete` |

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
| POST | `/admin/tenant-users/import` | 为指定租户批量导入用户 | `admin:tenant_user:create` |
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

> 创建接口的请求体为 `AdminCreateTenantUserReq`，在 `CreateUserReq` 基础上多一个必填 `tenant_id`；`role_ids` 仍为必填（至少一个），且只能选 `scope` 为 `all` / `tenant` 的角色。系统管理员必须走这组接口——调租户端 `POST /users`、`POST /users/import` 会被拒（见 3.5、3.11）。

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
| email_verified | bool | 邮箱是否已验证 |
| phone | string | 手机号 |
| avatar | string | 头像 URL |
| department | string | 所属部门 |
| position | string | 岗位/职务 |
| remark | string | 管理员备注 |
| roles | []string | 角色编码列表（兼容） |
| role_ids | []string | 角色 ID 列表 |
| role_list | []UserRoleInfo | 角色详细信息列表 |
| status | int | 状态：`1` 启用，`0` 禁用 |
| is_master | bool | 是否为系统管理员 |
| last_login_at | string | 最后登录时间（登录成功后回写，可为空） |
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
| department | string | 否 | max=100 | 所属部门 |
| position | string | 否 | max=100 | 岗位/职务 |
| role_ids | []string | 是 | min=1 | 分配的角色 ID 列表 |

### 2.4 UpdateUserReq（更新用户请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| nickname | string | 否 | max=50 | 昵称 |
| email | string | 否 | 邮箱格式 | 邮箱 |
| phone | string | 否 | max=20 | 手机号 |
| avatar | string | 否 | max=500 | 头像 URL |
| department | string | 否 | max=100 | 所属部门 |
| position | string | 否 | max=100 | 岗位/职务 |
| remark | string | 否 | max=255 | 管理员备注 |
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

### 2.10 ImportResult（批量导入结果）

| 字段 | 类型 | 说明 |
|------|------|------|
| total | int | 文件中提交的数据行数 |
| created | int | 实际创建的用户数（全量事务下只有 0 或 total） |
| users | []ImportUserResult | 逐行创建结果，按文件行序排列 |
| warnings | []string | 非阻断性提示（未知列被忽略、角色回落到默认角色等） |

**ImportUserResult**

| 字段 | 类型 | 说明 |
|------|------|------|
| row | int | 文件内物理行号（表头为第 1 行） |
| user_id | string | 新用户 ID |
| username | string | 用户名 |
| nickname | string | 昵称 |
| email | string | 邮箱 |
| role_codes | []string | 实际分配的角色编码 |
| initial_password | string | 服务端生成的初始密码，仅本次响应可见（不落库、不进审计日志） |

### 2.11 ImportRowError（行级校验错误）

| 字段 | 类型 | 说明 |
|------|------|------|
| row | int | 文件内物理行号（表头为第 1 行） |
| field | string | 出问题的列：`username` / `nickname` / `email` / `department` / `position` / `roles` |
| message | string | 面向操作者的中文提示 |

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
| status | int | 按状态筛选（非法值保守降级为 0-禁用） |
| role | string | 按角色筛选（关联 `user_roles`，仅返回持有该角色的用户） |
| sort | string | 排序列：`created_at`（默认）/`updated_at`/`username`/`status`/`email`/`last_login_at`；非法值回退默认 |
| order | string | 排序方向：`asc` / `desc`（大小写不敏感，默认 `asc`） |
| created_from | string | 注册时间下界，支持 `2006-01-02`、`2006-01-02 15:04:05` 或 RFC3339；非法返回 400 |
| created_to | string | 注册时间上界，格式同上；仅传日期时自动补足到当天 23:59:59 |

**成功响应（200）：** 分页响应，data 为 UserResp 数组

### 3.5 创建用户

`POST /api/v1/users`

**请求体：** CreateUserReq

**业务规则：** 用户名在当前租户内唯一；`role_ids` 必填且至少一个，角色需是启用中且可分配给租户用户的角色（`scope` 为 `all` 或 `tenant`）。前端的「新增用户」弹窗会拦截未选角色的提交（本地提示「请至少选择一个角色」，不发请求）。

**系统管理员请走跨租户接口：** 系统管理员的登录态租户是虚拟租户 `SYSTEM_ROOT`（不可登录），在本接口建号只会把用户落到这个虚拟租户里。此时返回 400 `系统管理员请使用跨租户接口为指定租户创建用户`，请改用 `POST /api/v1/admin/tenant-users` 并在请求体带上 `tenant_id`。前端的新增用户弹窗在系统管理员模式下会自动展示「目标租户」选择器，并改走该接口。

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

### 3.11 批量导入用户

`POST /api/v1/users/import`（租户管理员，租户取自登录态）  
`POST /api/v1/admin/tenant-users/import`（系统管理员，租户取自表单 `tenant_id`）  
`Content-Type: multipart/form-data`

**表单字段：**

| 字段 | 必填 | 说明 |
|------|------|------|
| file | 是 | 导入文件，仅支持 `.csv` / `.xlsx`，大小 ≤ 5MB，数据行 ≤ 500 |
| role_ids | 否 | 默认角色 ID，多个用逗号分隔；文件 `roles` 列为空时使用。**管理端入口忽略该字段**（角色 ID 不跨租户通用） |
| tenant_id | 管理端必填 | 目标租户 ID（租户端入口忽略该字段） |

**两端入口差异：**

- 租户端 `/users/import`：租户取自登录态，可用 `role_ids` 指定默认角色。
- 管理端 `/admin/tenant-users/import`：`tenant_id` 必填；角色 ID 属于具体租户，跨租户复用会把别的租户的角色挂到目标租户用户上，因此该入口**忽略 `role_ids`**，请通过文件 `roles` 列填写角色编码。前端在系统管理员模式下会要求先选择目标租户，并隐藏默认角色选择器。

**文件列（表头，均不区分大小写，支持中英文别名）：**

| 列 | 必填 | 别名 | 说明 |
|----|------|------|------|
| username | 是 | 用户名 / 登录名 / 账号 | 字母数字，4-50 位，全局唯一 |
| nickname | 是 | 昵称 / 姓名 | ≤ 50 字符 |
| email | 否 | 邮箱 / 电子邮件 | 邮箱格式 |
| department | 否 | 部门 | ≤ 100 字符 |
| position | 否 | 职位 / 岗位 | ≤ 100 字符 |
| roles | 否 | 角色 / 角色编码 | 角色**编码**，可用中英文逗号、分号、竖线分隔多个 |

未知列会被忽略，并在 `warnings` 中提示。

**业务规则：**

1. **全量事务**：先完成整份文件的校验，任一行不通过则一条都不写库；写入阶段用户与 `user_roles` 关联在同一事务内，失败整体回滚。
2. **用户名唯一**：同时校验整个平台现存用户、文件内重复行，以及**当前租户回收站中的同名账号**（回收站的记录仍占用唯一索引，需先彻底删除）。
3. **角色解析**：文件填了 `roles` 则按编码查本租户角色，找不到时回落平台级角色（`scope=all`）；禁用或不允许分配给租户用户的角色会报行级错误。
4. **套餐配额**：整批行数必须能放进剩余名额，否则直接拒绝（不会写入一部分）。
5. **初始密码**：由服务端随机生成（默认 12 位，密码策略开启时取 `min_length + 2` 与策略上限的较小值），只在本次响应中回显一次，**不落库明文、不写入审计日志**。
6. **审计**：请求体为 multipart，审计中间件不采集正文（避免文件内的明文信息落库），仅记录路径、操作人、结果与 `import` 标签。
7. **系统管理员走管理端入口**：系统管理员的登录态租户是虚拟租户 `SYSTEM_ROOT`，调租户端 `/users/import` 直接返回 400 `系统管理员请使用跨租户接口为指定租户导入用户`，必须改用 `/admin/tenant-users/import` 并显式指定 `tenant_id`。

**成功响应（200）：** ImportResult

```json
{
  "code": 200,
  "data": {
    "total": 2,
    "created": 2,
    "users": [
      {
        "row": 2,
        "user_id": "01JBX0000000000000000000A",
        "username": "alice",
        "nickname": "Alice",
        "email": "alice@example.com",
        "role_codes": ["member"],
        "initial_password": "Xy12!abcdEFG"
      }
    ],
    "warnings": ["已忽略无法识别的列：手机号"]
  }
}
```

**行级校验失败（400）：** 明细在 `details` 中，前端据此逐行展示

```json
{
  "code": "INVALID_PARAM",
  "message": "导入校验未通过，共 2 处问题，未写入任何用户",
  "details": [
    { "row": 3, "field": "username", "message": "用户名已被使用" },
    { "row": 5, "field": "roles", "message": "角色编码 'ghost' 不存在" }
  ]
}
```

**其他错误：**

| 场景 | 状态码 | 说明 |
|------|--------|------|
| 文件为空 / 表头缺列 / 扩展名不符 / 超过 500 行 | 400 | 直接返回面向操作者的文案 |
| 套餐剩余名额不足 | 400 | 提示当前已用、剩余可导入数 |
| 未选择文件、非 multipart 请求 | 400 | `未找到上传文件` |
| 写库失败（已回滚） | 500 | `批量写入失败（已整体回滚）` |
| 系统管理员调用租户端入口 | 400 | `系统管理员请使用跨租户接口为指定租户导入用户` |

### 3.12 批量启用/禁用与批量删除

`PUT /api/v1/users/batch/status`（`user:batch_status`）  
`DELETE /api/v1/users/batch/delete`（`user:batch_delete`）

**请求体：** BatchUpdateUserStatusReq / BatchDeleteUsersReq（`ids` 为 `required,min=1`）

**成功响应（200）：**

```json
{ "code": 200, "data": { "updated": 2 } }
```

批量删除返回 `{"code": 200, "data": {"deleted": 2}}`。

**业务规则：**

1. **租户取自登录态**：两个接口只操作登录租户下的用户，仓库层额外约束 `tenant_id = 登录租户` 与 `is_master = 0`，跨租户 ID 不会生效。
2. **整批拒绝**：先逐条校验 `ids` 全部存在且属于当前租户，任一不通过即整批拒绝、一条都不改，避免返回「部分成功」的模糊结果。
3. **批量删除**先解除 `user_roles` 绑定再软删除，可在回收站恢复（角色需重新分配）。
4. **系统管理员走管理端**：系统管理员的登录态租户是虚拟租户 `SYSTEM_ROOT`，调这两个接口直接返回 400 `系统管理员请使用跨租户接口批量操作用户`，必须改用 `/admin/tenant-users/{tenantID}/batch/status` 与 `/admin/tenant-users/{tenantID}/batch/delete`（租户 ID 在路径上）。
5. **前端分组提交**：用户列表勾选后批量栏按权限出现；系统管理员模式下列表返回多租户数据，前端按行上的 `tenant_id` 分组，逐租户调用管理端接口，没有 `tenant_id` 的行不可勾选。

**错误响应：**

| 场景 | 状态码 | 说明 |
|------|--------|------|
| `ids` 为空 | 400 | 校验失败，不落库 |
| 含不属于当前租户的用户 | 400 | `所选用户不属于当前租户` |
| 含不存在的用户 | 404 | `所选用户不存在` |
| 系统管理员调用租户端接口 | 400 | `系统管理员请使用跨租户接口批量操作用户` |

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