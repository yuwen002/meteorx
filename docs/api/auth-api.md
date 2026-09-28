# 认证模块 API 接口说明

> Base URL: `/api/v1`  
> 所属模块：`internal/modules/auth`  
> 路由注册：`internal/modules/auth/routes.go`  
> Handler：`internal/modules/auth/handler/auth_handler.go`  
> Service：`internal/modules/auth/service/auth_service.go`

---

## 1. 接口总览

| 方法 | 路径 | 功能 | 认证 |
|------|------|------|------|
| POST | `/auth/register` | 用户注册（加入已有租户） | 公开 |
| POST | `/tenants/register` | 租户注册（创建新租户） | 公开 |
| POST | `/auth/login` | 用户登录 | 公开 |
| POST | `/auth/logout` | 用户登出 | 需 Token |
| POST | `/auth/forgot-password` | 忘记密码（发送重置邮件） | 公开 |
| POST | `/auth/reset-password` | 重置密码（通过邮件令牌） | 公开 |
| GET | `/auth/oauth/:provider/redirect` | 获取 OAuth2 跳转链接 | 公开 |
| POST | `/auth/oauth/callback` | OAuth2 登录回调 | 公开 |
| GET | `/auth/oauth/tenants` | 获取可用租户列表（OAuth登录时选择租户） | 公开 |
| POST | `/auth/oauth/token/refresh` | 刷新 Token | 公开 |
| GET | `/auth/oauth/accounts` | 获取已绑定的第三方账号列表 | 需 Token |
| POST | `/auth/oauth/unbind` | 解绑第三方账号 | 需 Token |
| POST | `/auth/oauth/bind` | 绑定新的第三方账号 | 需 Token |
| GET | `/auth/tokens` | 获取 API Token 列表 | 需 Token |
| POST | `/auth/tokens` | 创建 API Token | 需 Token |
| POST | `/auth/tokens/revoke` | 撤销 API Token | 需 Token |

---

## 2. 数据结构

### 2.1 RegisterUserReq（用户注册请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| tenant_id | string | 是 | - | 租户 ID |
| username | string | 是 | 4-20 字符 | 用户名 |
| password | string | 是 | 6-32 字符 | 密码 |
| nickname | string | 是 | - | 昵称 |
| email | string | 是 | 邮箱格式 | 邮箱 |

### 2.2 RegisterTenantReq（租户注册请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| name | string | 是 | 2-50 字符 | 租户名称 |
| domain | string | 是 | 小写字母/数字/连字符 | 租户域名标识 |
| contact_email | string | 否 | 邮箱格式 | 联系邮箱 |
| admin_user | object | 是 | - | 管理员信息（见 2.3） |

### 2.3 AdminUserInfo（管理员信息）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| username | string | 是 | 4-20 字符 | 管理员用户名 |
| password | string | 是 | 6-32 字符 | 管理员密码 |
| nickname | string | 是 | - | 管理员昵称 |
| email | string | 是 | 邮箱格式 | 管理员邮箱 |

### 2.4 LoginReq（登录请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| tenant_id | string | 否 | 租户 ID（超级管理员登录时无需传） |
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |

### 2.5 LoginResp（登录成功响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| token | string | JWT Token，后续请求放入 Authorization Header |
| user | UserResp | 用户信息（参见用户模块） |
| permissions | []string | 用户拥有的所有权限码列表（前端用于按钮/菜单权限控制） |

### 2.6 LoginErrorResp（登录失败响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| message | string | 错误信息 |
| remaining_attempts | int | 剩余尝试次数 |
| locked | bool | 账户是否已锁定 |
| lockout_duration | int64 | 锁定剩余时间（秒） |

### 2.7 ForgotPasswordReq（忘记密码请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| email | string | 是 | 邮箱格式 | 用户注册时使用的邮箱 |

### 2.8 ResetPasswordReq（重置密码请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| token | string | 是 | - | 邮件中携带的重置令牌 |
| new_password | string | 是 | 8-32 字符 | 新密码 |

---

## 3. 接口详细说明

### 3.1 用户注册

`POST /api/v1/auth/register`

**请求体：**
```json
{
  "tenant_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "username": "zhangsan",
  "password": "123456",
  "nickname": "张三",
  "email": "zhangsan@example.com"
}
```

**业务规则：**
- 用户名 4-20 字符
- 密码 6-32 字符
- 同一租户内用户名唯一
- 邮箱格式校验

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "username": "zhangsan",
    "nickname": "张三",
    "email": "zhangsan@example.com"
  }
}
```

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 参数校验失败 |
| 409 | 用户名已存在 |
| 500 | 服务器内部错误 |

---

### 3.1.1 租户注册

`POST /api/v1/tenants/register`

**请求体：**
```json
{
  "name": "示例公司",
  "domain": "example-corp",
  "contact_email": "admin@example.com",
  "admin_user": {
    "username": "admin",
    "password": "123456",
    "nickname": "管理员",
    "email": "admin@example.com"
  }
}
```

**业务规则：**
- 租户名称 2-50 字符
- 域名标识仅允许小写字母、数字、连字符
- 域名全局唯一
- 管理员用户名 4-20 字符
- 密码 6-32 字符
- 邮箱格式校验
- 注册成功后自动创建租户和初始管理员账号

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "tenant": {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
      "name": "示例公司",
      "domain": "example-corp",
      "status": 1
    },
    "admin_user": {
      "id": "02BRZ3NDEKTSV4RRFFQ69G5FBW",
      "username": "admin",
      "nickname": "管理员",
      "email": "admin@example.com"
    }
  }
}
```

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 参数校验失败 |
| 409 | 域名已存在 |
| 500 | 服务器内部错误 |

---

### 3.2 用户登录

`POST /api/v1/auth/login`

**请求体：**
```json
{
  "tenant_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "username": "zhangsan",
  "password": "123456"
}
```

**业务规则：**
- 连续失败超过阈值后账户锁定
- 锁定时间内返回 `LoginErrorResp` 含锁定信息
- 超级管理员（master admin）登录时无需 `tenant_id`

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
      "tenant_id": "...",
      "username": "zhangsan",
      "nickname": "张三",
      "email": "zhangsan@example.com",
      "status": 1,
      "is_master": false
    },
    "permissions": ["file:list", "file:upload", "user:list"]
  }
}
```

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 401 | 用户名或密码错误 |
| 423 | 账户已锁定 |
| 500 | 服务器内部错误 |

---

### 3.3 用户登出

`POST /api/v1/auth/logout`

**请求头：** `Authorization: Bearer <token>`

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

**说明：** 服务端将 Token 加入黑名单（Redis），登出后 Token 立即失效。

---

### 3.4 忘记密码（发送重置邮件）

`POST /api/v1/auth/forgot-password`

**请求体：**
```json
{
  "email": "user@example.com"
}
```

**业务规则：**
- 如果邮箱存在，系统向该邮箱发送一封包含重置密码链接的邮件
- 为防止信息泄露，无论邮箱是否存在，接口都返回相同的成功响应
- 重置令牌有效期为 30 分钟
- 令牌通过 Redis 存储，过期自动失效

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "重置邮件已发送",
  "data": { "message": "如果该邮箱已注册,重置密码链接已发送" }
}
```

**配置说明：** 需在 `config.yaml` 或 `.env` 中正确配置 `email.*` SMTP 参数和 `client.base_url`。

---

### 3.5 重置密码

`POST /api/v1/auth/reset-password`

**请求体：**
```json
{
  "token": "8f1a3c5d-e2b4-4f6a-9c8d-1e2f3a4b5c6d",
  "new_password": "NewPass@123456"
}
```

**业务规则：**
- `token` 为邮箱中携带的重置令牌（通过 Redis 验证有效性和过期时间）
- `new_password` 需符合密码策略（8-32 字符）
- 密码重置成功后，令牌立即失效（Redis 删除）
- 用户需重新登录

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "密码重置成功",
  "data": { "message": "密码已重置,请使用新密码登录" }
}
```

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 参数校验失败（如密码不符合策略） |
| 404 | 令牌无效或已过期 |
| 500 | 服务器内部错误 |

---

## 4. 安全机制

| 机制 | 说明 |
|------|------|
| 登录锁定 | 连续失败 5 次锁定账户 30 分钟 |
| 密码哈希 | 使用 bcrypt 存储密码，不存储明文 |
| Token 有效期 | Access Token 默认 24 小时（`jwt.expiration`） |
| Refresh Token | 有效期由 `oauth.refresh_token_ttl` 控制，默认 168h（7 天），一次性使用，轮转机制 |
| OAuth CSRF State | 有效期由 `oauth.state_ttl` 控制，默认 10m，一次性使用 |
| Token 黑名单 | 登出后 Token 加入 Redis 黑名单 |
| API Token | 长期令牌（`mxat_` 前缀），SHA256 哈希存储，明文仅创建时返回一次，最长有效期由 `auth.api_token_max_ttl` 控制（默认 2160h/90天），每用户上限 10 个，支持撤销 |
| XSS 防护 | 用户名/昵称等输出时自动转义 |
| SQL 注入防护 | 使用参数化查询（GORM） |
| 密码重置令牌 | Redis 存储，30 分钟过期，一次性使用 |
| 信息泄露防护 | 忘记密码接口对不存在的邮箱返回相同响应 |
| 邮件发送失败 | 即使邮件发送失败，仍返回统一响应避免暴露信息 |

---

## 5. OAuth2 第三方登录

### 5.1 获取 OAuth2 跳转链接

`GET /api/v1/auth/oauth/{provider}/redirect`

**路径参数：**
| 参数 | 类型 | 说明 |
|------|------|------|
| provider | string | 第三方登录提供商，支持 `google`、`github` |

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "url": "https://accounts.google.com/o/oauth2/v2/auth?client_id=xxx&redirect_uri=xxx&response_type=code&scope=email+profile&state=xxx",
    "state": "xxx"
  }
}
```

**业务规则：**
- 支持的 provider：`google`、`github`
- 服务端生成随机 `state` 并存入 Redis（有效期由 `oauth.state_ttl` 控制，默认 10 分钟），用于 CSRF 防护
- 前端收到 `url` 和 `state` 后，需保存 `state`，然后跳转到 `url`
- 跳转后用户授权，第三方平台回调到前端 `OAuthCallback.vue` 页面
- 回调时必须将保存的 `state` 传回服务端进行验证

### 5.2 获取可用租户列表

`GET /api/v1/auth/oauth/tenants`

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "tenants": [
      {
        "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
        "name": "租户A"
      },
      {
        "id": "02BRZ3NDEKTSV4RRFFQ69G5FBW",
        "name": "租户B"
      }
    ]
  }
}
```

**业务规则：**
- 仅返回状态为启用（status=1）的租户
- 用于 OAuth 登录前让用户选择所属租户
- 无需认证即可访问

---

### 5.3 OAuth2 登录回调

`POST /api/v1/auth/oauth/callback`

**请求体：**
```json
{
  "provider": "google",
  "code": "4/0AX4XfWi...",
  "tenant_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV"
}
```

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| provider | string | 是 | OAuth 提供商，支持 `google`、`github` |
| code | string | 是 | 第三方返回的授权码 |
| state | string | 是 | 防 CSRF 状态码（从跳转链接响应中获取，必填） |
| tenant_id | string | 是 | 租户ID（通过 GET /api/v1/auth/oauth/tenants 获取） |

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "a1b2c3d4-e5f6-...",
    "user": {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
      "tenant_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
      "username": "google_12345",
      "nickname": "张三",
      "email": "zhangsan@gmail.com",
      "avatar": "https://...",
      "status": 1,
      "is_master": false
    },
    "permissions": ["file:list", "user:list"],
    "is_new_user": true
  }
}
```

**业务规则：**
- 必须传入 `state`（CSRF 防护），从跳转链接响应中获取
- 必须传入 `tenant_id`，否则返回错误
- 首次登录自动创建用户并关联到指定租户（用户名自动生成：`{provider}_{email前8位}`）
- 重复登录优先通过 `oauth_accounts` 表的 `provider+provider_id` 匹配已有用户，其次通过邮箱匹配
- 自动分配租户默认角色（tenant_user）
- 返回 `is_new_user` 字段标识是否为新注册用户
- 返回 `refresh_token` 用于长期会话（有效期由 `oauth.refresh_token_ttl` 控制，默认 7 天，一次性使用）

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 缺少必要参数 / CSRF state 验证失败 / 第三方账号未提供邮箱 |
| 401 | 第三方授权码无效或已过期 / 租户不存在或已禁用 |
| 500 | 服务器内部错误 |

---

### 5.4 刷新 Token

`POST /api/v1/auth/oauth/token/refresh`

**请求体：**
```json
{
  "refresh_token": "a1b2c3d4-e5f6-..."
}
```

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "new-refresh-token-..."
  }
}
```

**业务规则：**
- 刷新令牌一次性使用，使用后立即失效
- 新的刷新令牌同时返回，形成轮转链
- 刷新令牌有效期由 `oauth.refresh_token_ttl` 控制，默认 168h（7 天）
- 用户被禁用时无法刷新

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 参数校验失败 |
| 401 | 刷新令牌无效或已过期 / 用户已禁用 |
| 500 | 服务器内部错误 |

---

### 5.5 获取已绑定的第三方账号列表

`GET /api/v1/auth/oauth/accounts`

**请求头：** `Authorization: Bearer <token>`

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "accounts": [
      {
        "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
        "provider": "google",
        "email": "zhangsan@gmail.com",
        "created_at": "2025-01-15T10:30:00Z"
      },
      {
        "id": "02BRZ3NDEKTSV4RRFFQ69G5FBW",
        "provider": "github",
        "email": "zhangsan@users.noreply.github.com",
        "created_at": "2025-02-20T14:00:00Z"
      }
    ]
  }
}
```

---

### 5.6 解绑第三方账号

`POST /api/v1/auth/oauth/unbind`

**请求头：** `Authorization: Bearer <token>`

**请求体：**
```json
{
  "provider": "google"
}
```

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
| 404 | 未找到该绑定的第三方账号 |
| 500 | 服务器内部错误 |

---

### 5.7 绑定新的第三方账号

`POST /api/v1/auth/oauth/bind`

**请求头：** `Authorization: Bearer <token>`

**请求体：**
```json
{
  "provider": "github",
  "code": "4/0AX4XfWi...",
  "state": "csrf-state-from-redirect"
}
```

**业务规则：**
- 需要先调用 `GET /auth/oauth/{provider}/redirect` 获取跳转链接和 state
- 同一提供商不能重复绑定
- 同一第三方账号不能绑定到不同用户

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
| 400 | CSRF state 验证失败 |
| 409 | 该提供商已绑定到当前账号 |
| 500 | 服务器内部错误 |

---

## 6. API Token（长期令牌）

API Token 用于 CI/CD 脚本、自动化工具、第三方系统集成等场景，无需通过浏览器登录即可调用 API。

**鉴权方式：** 与 JWT 相同，使用 `Authorization: Bearer <api_token>` 头。中间件先尝试 JWT 解析，失败后自动 fallback 查 API Token（`mxat_` 前缀识别），对调用方完全透明。

**存储方式：** 数据库存储 SHA256 哈希值，Redis 缓存加速查询。撤销时同时删除 DB 和 Redis，立即生效。

### 6.1 创建 API Token

`POST /api/v1/auth/tokens`

**请求头：** `Authorization: Bearer <token>`

**请求体：**
```json
{
  "name": "ci-deploy-token",
  "expires_in": "720h"
}
```

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| name | string | 是 | 1-64 字符 | 令牌名称（便于识别用途） |
| expires_in | string | 否 | Go Duration 格式 | 有效期，如 `720h`（30天）、`2160h`（90天）；空则使用默认最长有效期 |

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "name": "ci-deploy-token",
    "token": "mxat_vKx8mN2pQ5rT7wY9aB3cD6fG0hJ4kL",
    "expires_at": "2026-12-28T10:00:00Z",
    "created_at": "2026-09-28T10:00:00Z"
  }
}
```

**⚠️ 重要：** `token` 字段仅在创建时返回一次，后续无法再查看明文。请立即保存。

**业务规则：**
- 每用户最多 10 个有效 API Token
- 有效期不能超过 `auth.api_token_max_ttl`（默认 2160h/90天）
- Token 以 `mxat_` 前缀标识，便于区分 JWT
- 数据库仅存储 SHA256 哈希值，不存明文

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 参数校验失败 / expires_in 格式错误 / 超过最大有效期 |
| 401 | 未授权 |
| 409 | 同名令牌已存在 / 已达到数量上限 |
| 500 | 服务器内部错误 |

---

### 6.2 获取 API Token 列表

`GET /api/v1/auth/tokens`

**请求头：** `Authorization: Bearer <token>`

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "tokens": [
      {
        "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
        "name": "ci-deploy-token",
        "expires_at": "2026-12-28T10:00:00Z",
        "last_used_at": "2026-09-27T08:15:00Z",
        "created_at": "2026-09-28T10:00:00Z",
        "revoked": false
      },
      {
        "id": "02BRZ3NDEKTSV4RRFFQ69G5FBW",
        "name": "old-token",
        "expires_at": "2026-06-15T10:00:00Z",
        "created_at": "2026-03-15T10:00:00Z",
        "revoked": true
      }
    ]
  }
}
```

**说明：** 列表按创建时间倒序排列，包含已撤销和已过期的令牌（`revoked: true`）。不返回令牌明文。

---

### 6.3 撤销 API Token

`POST /api/v1/auth/tokens/revoke`

**请求头：** `Authorization: Bearer <token>`

**请求体：**
```json
{
  "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | string | 是 | 要撤销的 API Token ID |

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

**业务规则：**
- 撤销后立即生效（删除 Redis 缓存 + DB 标记 revoked_at）
- 只能撤销自己的 Token
- 已撤销的 Token 再次撤销返回错误

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 令牌已被撤销 |
| 404 | 令牌不存在 |
| 500 | 服务器内部错误 |

---

### 6.4 使用 API Token 调用 API

API Token 的使用方式与 JWT 完全相同：

```bash
# 示例：使用 API Token 调用文件上传接口
curl -X POST https://api.example.com/api/v1/files/upload \
  -H "Authorization: Bearer mxat_vKx8mN2pQ5rT7wY9aB3cD6fG0hJ4kL" \
  -F "file=@document.pdf"
```

**鉴权流程：**
1. 中间件收到 `Authorization: Bearer <token>` 头
2. 先尝试 JWT 解析
3. JWT 解析失败 且 token 以 `mxat_` 开头 → 查 API Token
4. Redis 缓存命中 → 直接注入 Context（<1ms）
5. Redis 未命中 → 查 DB → 回填 Redis → 注入 Context
6. API Token 验证通过后，注入 userID、tenantID、roles，后续逻辑与 JWT 完全一致

**与 JWT 的区别：**
| 维度 | JWT | API Token |
|------|-----|-----------|
| 前缀 | 无 | `mxat_` |
| 有效期 | 24h（可配） | 最长 90 天（可配） |
| 轮转 | 无 | 无 |
| 撤销 | 加入黑名单 | 标记 revoked_at + 删缓存 |
| 体积 | ~500 字节 | ~48 字节 |
| 权限 | 自包含 roles | 查 DB 获取当前 roles（实时） |