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
| POST | `/auth/register` | 用户注册（含租户创建） | 公开 |
| POST | `/auth/login` | 用户登录 | 公开 |
| POST | `/auth/logout` | 用户登出 | 需 Token |

---

## 2. 数据结构

### 2.1 RegisterUserReq（注册请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| tenant_id | string | 是 | - | 租户 ID |
| username | string | 是 | 4-20 字符 | 用户名 |
| password | string | 是 | 6-32 字符 | 密码 |
| nickname | string | 是 | - | 昵称 |
| email | string | 是 | 邮箱格式 | 邮箱 |

### 2.2 LoginReq（登录请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| tenant_id | string | 否 | 租户 ID（超级管理员登录时无需传） |
| username | string | 是 | 用户名 |
| password | string | 是 | 密码 |

### 2.3 LoginResp（登录成功响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| token | string | JWT Token，后续请求放入 Authorization Header |
| user | UserResp | 用户信息（参见用户模块） |
| permissions | []string | 用户拥有的所有权限码列表（前端用于按钮/菜单权限控制） |

### 2.4 LoginErrorResp（登录失败响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| message | string | 错误信息 |
| remaining_attempts | int | 剩余尝试次数 |
| locked | bool | 账户是否已锁定 |
| lockout_duration | int64 | 锁定剩余时间（秒） |

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

## 4. 安全机制

| 机制 | 说明 |
|------|------|
| 登录锁定 | 连续失败 5 次锁定账户 30 分钟 |
| 密码哈希 | 使用 bcrypt 存储密码，不存储明文 |
| Token 有效期 | Access Token 默认 24 小时 |
| Token 黑名单 | 登出后 Token 加入 Redis 黑名单 |
| XSS 防护 | 用户名/昵称等输出时自动转义 |
| SQL 注入防护 | 使用参数化查询（GORM） |