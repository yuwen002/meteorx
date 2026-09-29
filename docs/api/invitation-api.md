# 成员邀请模块 API 接口说明

> Base URL: `/api/v1`  
> 所属模块：`internal/modules/invitation`  
> 路由注册：`internal/modules/invitation/module.go`  
> Handler：`internal/modules/invitation/handler/invitation_handler.go`  
> Service：`internal/modules/invitation/service/invitation_service.go`  
> DTO：`internal/modules/invitation/dto/invitation_dto.go`

---

## 1. 接口总览

### 1.1 认证接口（需登录 + 权限）

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/invitations` | 获取当前租户邀请列表 | `invitation:list` |
| POST | `/invitations` | 创建邀请（邀请新成员） | `invitation:create` |
| PUT | `/invitations/{id}/cancel` | 取消邀请 | `invitation:cancel` |
| PUT | `/invitations/{id}/resend` | 重发邀请邮件 | `invitation:resend` |
| DELETE | `/invitations/{id}/delete` | 删除邀请记录 | `invitation:delete` |

### 1.2 公开接口（无需登录）

| 方法 | 路径 | 功能 | 说明 |
|------|------|------|------|
| GET | `/invitations/info?token=xxx` | 通过令牌查询邀请信息 | 被邀请人打开邮件链接时调用 |
| POST | `/invitations/accept` | 接受邀请并注册 | 被邀请人填写注册信息后调用 |

---

## 2. 数据结构

### 2.1 InvitationResp（邀请响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 邀请 ID |
| tenant_id | string | 租户 ID |
| tenant_name | string | 租户名称 |
| email | string | 被邀请人邮箱 |
| role_ids | []string | 分配的角色 ID 列表 |
| status | string | 状态：`pending` / `accepted` / `cancelled` / `expired` |
| invited_by | string | 邀请人用户 ID |
| expires_at | string | 过期时间 |
| accepted_at | string | 接受时间（仅已接受时有值） |
| created_at | string | 创建时间 |

### 2.2 CreateInvitationReq（创建邀请请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| email | string | 是 | 邮箱格式 | 被邀请人邮箱 |
| role_ids | []string | 是 | min=1 | 分配的角色 ID 列表 |

### 2.3 AcceptInvitationReq（接受邀请请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| token | string | 是 | - | 邀请令牌（邮件链接中携带） |
| username | string | 是 | alphanum, 4-50 | 设置用户名 |
| password | string | 是 | 6-32 | 设置密码 |
| nickname | string | 是 | max=50 | 设置昵称 |

### 2.4 AcceptInvitationResp（接受邀请响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| message | string | 提示信息 |

---

## 3. 接口详细说明

### 3.1 获取邀请列表

`GET /api/v1/invitations`

**请求头：** `Authorization: Bearer <token>`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页条数，默认 10 |
| keyword | string | 按邮箱搜索 |
| status | string | 按状态筛选：pending / accepted / cancelled / expired |

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
        "tenant_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
        "email": "newuser@example.com",
        "role_ids": ["role-id-1", "role-id-2"],
        "status": "pending",
        "invited_by": "admin-user-id",
        "expires_at": "2026-10-06 10:00:00",
        "created_at": "2026-09-29 10:00:00"
      }
    ],
    "total": 1
  }
}
```

**说明：** 仅返回当前租户的邀请记录。

---

### 3.2 创建邀请

`POST /api/v1/invitations`

**请求头：** `Authorization: Bearer <token>`

**请求体：**
```json
{
  "email": "newuser@example.com",
  "role_ids": ["01ARZ3NDEKTSV4RRFFQ69G5FAV"]
}
```

**业务规则：**
- 同一租户内，同一邮箱不能有多个待处理邀请
- 如果该邮箱已是本租户成员，返回 `409` 错误
- 邀请默认有效期为 **7 天**
- 创建成功后自动发送邀请邮件到目标邮箱
- 邮件中包含邀请链接：`{client.base_url}/accept-invitation?token={token}`
- 邮件内容包含邀请人昵称和租户名称

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "tenant_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "email": "newuser@example.com",
    "role_ids": ["01ARZ3NDEKTSV4RRFFQ69G5FAV"],
    "status": "pending",
    "invited_by": "admin-user-id",
    "expires_at": "2026-10-06 10:00:00",
    "created_at": "2026-09-29 10:00:00"
  }
}
```

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 参数校验失败 |
| 409 | 该邮箱已有待处理邀请 / 该邮箱已是本租户成员 |
| 500 | 创建失败 / 邮件发送失败 |

---

### 3.3 取消邀请

`PUT /api/v1/invitations/{id}/cancel`

**请求头：** `Authorization: Bearer <token>`

**业务规则：**
- 仅 `pending` 状态的邀请可以取消
- 取消后邀请状态变为 `cancelled`
- 取消后邀请链接失效，被邀请人无法再接受

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 邀请已处理，无法取消 |
| 404 | 邀请不存在 |

---

### 3.4 重发邀请邮件

`PUT /api/v1/invitations/{id}/resend`

**请求头：** `Authorization: Bearer <token>`

**业务规则：**
- 仅 `pending` 状态的邀请可以重发
- 重发使用原邀请令牌，不生成新令牌
- 重发不重置过期时间

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "status": "pending",
    "expires_at": "2026-10-06 10:00:00"
  }
}
```

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 邀请已处理，无法重发 |
| 404 | 邀请不存在 |
| 500 | 邮件服务未配置 / 邮件发送失败 |

---

### 3.5 删除邀请记录

`DELETE /api/v1/invitations/{id}/delete`

**请求头：** `Authorization: Bearer <token>`

**说明：** 物理删除邀请记录，不可恢复。

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 404 | 邀请不存在 |

---

### 3.6 通过令牌查询邀请信息（公开）

`GET /api/v1/invitations/info?token=xxx`

**说明：** 被邀请人打开邮件中的邀请链接时调用，用于展示邀请详情。

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "tenant_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "tenant_name": "示例公司",
    "email": "newuser@example.com",
    "role_ids": ["01ARZ3NDEKTSV4RRFFQ69G5FAV"],
    "status": "pending",
    "expires_at": "2026-10-06 10:00:00",
    "created_at": "2026-09-29 10:00:00"
  }
}
```

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 缺少邀请令牌 |
| 404 | 邀请不存在 |

---

### 3.7 接受邀请并注册（公开）

`POST /api/v1/invitations/accept`

**请求体：**
```json
{
  "token": "8f1a3c5d-e2b4-4f6a-9c8d-1e2f3a4b5c6d",
  "username": "zhangsan",
  "password": "Pass@123456",
  "nickname": "张三"
}
```

**业务规则：**
- 令牌必须有效且对应 `pending` 状态的邀请
- 邀请过期后自动标记为 `expired` 状态
- 密码需符合安全策略（6-32 位，包含字母和数字）
- 接受后自动完成以下操作：
  1. 创建新用户（关联到邀请的租户）
  2. 设置邮箱为已验证（`email_verified = true`）
  3. 分配邀请中指定的角色
  4. 标记邀请状态为 `accepted`
- 用户名在租户内唯一

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "message": "邀请接受成功，请使用新账号登录"
  }
}
```

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 参数校验失败 / 邀请已过期 / 邀请已被使用 |
| 404 | 邀请不存在 |
| 500 | 服务器内部错误 |

---

## 4. 状态流转

```
  创建邀请
      │
      ▼
  ┌─────────┐
  │ pending  │ ← 初始状态
  └────┬────┘
       │
  ┌────┼────────────────────┐
  │    │                    │
  ▼    ▼                    ▼
┌──────┐  ┌──────────┐  ┌─────────┐
│accepted│  │cancelled │  │ expired │
└──────┘  └──────────┘  └─────────┘
  接受邀请    取消邀请      超时过期
```

**状态说明：**

| 状态 | 说明 | 可执行操作 |
|------|------|------------|
| `pending` | 待处理，等待被邀请人接受 | 取消、重发、删除 |
| `accepted` | 已接受，被邀请人已注册 | 删除 |
| `cancelled` | 已取消，邀请人主动取消 | 删除 |
| `expired` | 已过期，超过有效期 | 删除 |

---

## 5. 完整邀请流程

```
邀请人（管理员）                  系统                    被邀请人
     │                            │                         │
     │  POST /invitations         │                         │
     │  { email, role_ids }       │                         │
     │ ──────────────────────────>│                         │
     │                            │  生成 token             │
     │                            │  存入 DB (pending)      │
     │                            │  发送邀请邮件 ─────────>│
     │                            │                         │
     │                            │                  打开邮件链接
     │                            │                         │
     │                            │<── GET /invitations/info│
     │                            │    ?token=xxx           │
     │                            │                         │
     │                            │    填写注册信息          │
     │                            │                         │
     │                            │<── POST /invitations/   │
     │                            │    accept               │
     │                            │    { token, username,   │
     │                            │      password, nickname}│
     │                            │                         │
     │                            │  创建用户 + 分配角色    │
     │                            │  标记邀请 accepted      │
     │                            │ ──────────────────────> │
     │                            │                   注册成功
     │                            │                   去登录
```

---

## 6. 配置说明

邀请功能依赖以下配置项：

```yaml
email:
  enabled: true              # 必须启用邮件服务
  host: "smtp.example.com"
  port: 587
  username: "noreply@example.com"
  password: "smtp-password"
  from: "noreply@example.com"
  from_name: "MeteorX"

client:
  base_url: "https://app.example.com"  # 用于生成邀请链接
```

**注意：** 如果 `email.enabled` 为 `false`，创建邀请仍会成功（记录写入数据库），但不会发送邮件。可通过"重发"功能在配置邮件服务后补发。

---

## 7. 权限码

| 权限码 | 说明 |
|--------|------|
| `invitation:list` | 查看邀请列表 |
| `invitation:create` | 创建邀请 |
| `invitation:cancel` | 取消邀请 |
| `invitation:resend` | 重发邀请邮件 |
| `invitation:delete` | 删除邀请记录 |

> 公开接口（`/invitations/info` 和 `/invitations/accept`）无需权限码，任何人都可访问。