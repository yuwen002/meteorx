# MeteorX — 多租户 SaaS 平台后端

一套基于 Go + Chi + GORM + Redis 构建的 **MaaS (Multi-tenant as a Service)** 多租户后端脚手架，内置「平台超级管理员 / 租户管理员 / 普通用户」三级权限模型，以及完整的用户/租户生命周期管理（软删除、回收站、恢复、批量操作）。

---

## ✨ 核心特性

| 模块 | 特性 |
| --- | --- |
| **多租户架构** | 基于 `tenant_id` 行级隔离；支持域名自动识别租户；独立管理员体系 |
| **认证鉴权** | JWT（HS256）双 Token；注册 / 登录；Token 自动注入用户上下文 |
| **用户管理** | 个人中心 / 租户内用户 / 跨租户用户三类独立 API；密码加密（bcrypt） |
| **租户管理** | 自主注册开户；后台手动建租户；启用/禁用；软删除/恢复；批量操作 |
| **RBAC 权限** | 角色 + 权限 + 角色-权限绑定；支持作用域（`tenant` / `master` / `all`） |
| **回收站** | 用户 / 租户 / 角色 均支持软删除 → 回收站查询 → 恢复的完整闭环 |
| **批量操作** | 批量删除 / 批量更新状态；幂等返回影响行数 |
| **审计日志** | 自动记录所有请求；支持多维度筛选查询；敏感信息脱敏；定时清理 |
| **安全增强** | 登录失败锁定；密码复杂度策略；接口限流（基于 Redis） |
| **工程化** | Viper 配置 + `.env` 覆盖；Chi 路由；GORM 自动迁移；ULID 主键 |
| **测试覆盖** | 审计模块、安全模块单元测试；Mock 仓库实现 |

---

## 🧱 技术栈

- **语言**：Go 1.25
- **Web 框架**：[go-chi/chi v5.2](https://github.com/go-chi/chi)
- **ORM**：[GORM v1.31](https://gorm.io/)
- **数据库**：MySQL 8.0+
- **缓存**：Redis 7+（go-redis v9）
- **认证**：JWT（golang-jwt v5）
- **校验**：go-playground/validator v10
- **ID 生成**：ULID（按时间有序、适合数据库索引）
- **配置**：Viper + YAML + `.env` 环境变量覆盖

---

## 📁 项目结构

```
meteorx/
├── cmd/
│   └── server/
│       └── main.go              # 程序入口
│
├── internal/
│   ├── bootstrap/               # ⭐ 启动编排
│   │   ├── app.go               # 应用初始化（模块装配）
│   │   ├── config.go            # Viper 配置加载
│   │   ├── database.go          # GORM 数据库初始化
│   │   ├── migrate.go           # 自动建表 + 种子数据
│   │   ├── router.go            # 路由注册（公共/鉴权/后台）
│   │   └── middleware.go        # 中间件装配
│   │
│   ├── modules/                 # ⭐ 业务模块（核心代码）
│   │   ├── auth/                # 认证：注册、登录、JWT
│   │   ├── user/                # 用户：3 层接口 + 回收站 + 批量
│   │   ├── tenant/              # 租户：自助开户 + 后台管理
│   │   ├── rbac/                # 角色权限：角色、权限、绑定
│   │   └── audit/               # 审计日志（预留）
│   │
│   ├── middleware/              # 认证、超级管理员、权限等中间件
│   ├── common/                  # 响应封装、上下文工具、JWT、校验器
│   ├── config/                  # 配置结构体 + 默认 YAML
│   └── cache/                   # Redis 客户端封装
│
├── pkg/                         # 可复用基础库
│   ├── crypto/                  # bcrypt 密码加密
│   ├── logger/                  # 日志封装
│   ├── pagination/              # 统一分页请求/响应
│   ├── ulid/                    # ULID 生成
│   └── uuid/                    # UUID 工具
│
├── scripts/sql/
│   ├── init.sql                 # 建表脚本
│   └── seed.sql                 # 种子数据（默认管理员等）
│
├── docs/apifox/
│   ├── MeteorX-backend.apifox.json       # Apifox 项目文件
│   └── MeteorX-backend.openapi.json      # OpenAPI 3.0 规范
│
├── deployments/
│   ├── docker/
│   └── nginx/
│
├── .env                         # 环境变量（覆盖 YAML 默认值）
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── README.md
```

---

## 🔌 API 总览（`/api/v1`）

### 1. 认证（公开）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/auth/register` | 用户自主注册 |
| `POST` | `/auth/login` | 登录，返回 JWT |

### 2. 用户个人中心（需登录）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/profile` | 查看当前用户信息 |
| `PUT` | `/profile` | 修改昵称/邮箱等 |
| `PUT` | `/profile/password` | 修改当前用户密码 |

### 3. 租户内用户管理（需登录 · 本租户）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/users` | 分页查询本租户用户（支持 keyword） |
| `POST` | `/users` | 在本租户下创建用户 |
| `GET` | `/users/{id}/detail` | 用户详情 |
| `PUT` | `/users/{id}/update` | 更新用户 |
| `DELETE` | `/users/{id}/delete` | 软删除用户 |

### 4. 平台超级管理员 — 系统管理员

> 需通过 `RequiresMasterAdmin` 中间件

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/admin/users` | 系统管理员列表 |
| `POST` | `/admin/users` | 创建系统管理员 |
| `GET` | `/admin/users/{id}/detail` | 管理员详情 |
| `PUT` | `/admin/users/{id}/update` | 更新管理员 |
| `PUT` | `/admin/users/{id}/status` | 启用/禁用管理员 |
| `DELETE` | `/admin/users/{id}/delete` | 软删除管理员 |
| `GET` | `/admin/users/deleted` | 🗑️ 回收站：已删除管理员列表 |
| `PUT` | `/admin/users/{id}/restore` | ♻️ 恢复已删除管理员 |
| `PUT` | `/admin/users/batch/status` | 🔁 批量启用/禁用管理员 |
| `DELETE` | `/admin/users/batch/delete` | 🔁 批量删除管理员 |

### 5. 平台超级管理员 — 跨租户用户管理

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/admin/tenant-users/all` | 全量租户用户列表（排除系统管理员） |
| `POST` | `/admin/tenant-users` | 为指定租户创建用户 |
| `GET` | `/admin/tenant-users/{tenantID}/list` | 指定租户用户列表 |
| `PUT` | `/admin/tenant-users/{tenantID}/{userID}/update` | 更新租户用户 |
| `PUT` | `/admin/tenant-users/{tenantID}/{userID}/status` | 启用/禁用租户用户 |
| `DELETE` | `/admin/tenant-users/{tenantID}/{userID}/delete` | 删除租户用户 |
| `GET` | `/admin/tenant-users/deleted/all` | 🗑️ 全量已删除租户用户 |
| `GET` | `/admin/tenant-users/{tenantID}/deleted` | 🗑️ 指定租户已删除用户 |
| `PUT` | `/admin/tenant-users/{tenantID}/{userID}/restore` | ♻️ 恢复已删除租户用户 |
| `PUT` | `/admin/tenant-users/{tenantID}/batch/status` | 🔁 批量更新状态 |
| `DELETE` | `/admin/tenant-users/{tenantID}/batch/delete` | 🔁 批量删除 |

### 6. 租户管理

**公开接口**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/tenants/register` | 企业自主注册开户 |

**租户私有接口（需登录 · 本租户）**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/tenants/current` | 当前租户详情 |
| `PUT` | `/tenants/current` | 修改当前租户信息 |
| `GET` | `/tenants/current/status` | 查询租户初始化状态 |
| `POST` | `/tenants/current/cancel` | 申请自主注销 |

**平台超级管理员接口**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/admin/tenants` | 全量租户列表（支持 keyword） |
| `POST` | `/admin/tenants` | 后台手动新建租户 |
| `GET` | `/admin/tenants/{id}/detail` | 租户详情 |
| `PUT` | `/admin/tenants/{id}/update` | 更新租户 |
| `PUT` | `/admin/tenants/{id}/status` | 启用/禁用租户 |
| `DELETE` | `/admin/tenants/{id}/delete` | 软删除租户 |
| `GET` | `/admin/tenants/deleted` | 🗑️ 回收站：已删除租户列表 |
| `PUT` | `/admin/tenants/{id}/restore` | ♻️ 恢复已删除租户 |
| `PUT` | `/admin/tenants/batch/status` | 🔁 批量启用/禁用 |
| `DELETE` | `/admin/tenants/batch` | 🔁 批量删除 |

### 7. RBAC 角色与权限（平台超级管理员专属）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/rbac/roles` | 角色列表 |
| `POST` | `/rbac/roles` | 创建角色 |
| `GET` | `/rbac/roles/{id}/detail` | 角色详情 |
| `PUT` | `/rbac/roles/{id}/update` | 更新角色 |
| `PUT` | `/rbac/roles/{id}/status` | 启用/禁用角色 |
| `DELETE` | `/rbac/roles/{id}/delete` | 软删除角色 |
| `GET` | `/rbac/roles/deleted` | 🗑️ 回收站：已删除角色 |
| `PUT` | `/rbac/roles/{id}/restore` | ♻️ 恢复角色 |
| `PUT` | `/rbac/roles/{id}/permissions` | 绑定权限到角色 |
| `GET` | `/rbac/roles/{id}/permissions` | 查询角色拥有的权限 |
| `PUT` | `/rbac/roles/batch/status` | 🔁 批量更新角色状态 |
| `DELETE` | `/rbac/roles/batch/delete` | 🔁 批量删除角色 |
| `GET` | `/rbac/permissions` | 权限列表 |
| `POST` | `/rbac/permissions` | 创建权限 |
| `GET` | `/rbac/permissions/{id}/detail` | 权限详情 |
| `PUT` | `/rbac/permissions/{id}/update` | 更新权限 |
| `DELETE` | `/rbac/permissions/{id}/delete` | 删除权限 |

### 8. 审计日志（平台超级管理员专属）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/audit/stats` | 审计日志统计（总日志数、今日日志、操作分布等） |
| `GET` | `/audit/logs` | 审计日志列表（支持多维度筛选） |
| `POST` | `/audit/logs` | 创建审计日志（内部API，中间件自动调用） |
| `GET` | `/audit/logs/{id}` | 审计日志详情 |
| `DELETE` | `/audit/logs/cleanup?days=30` | 清理指定天数前的日志 |

---

## 🚀 快速开始

### 前置条件

- Go 1.25+
- MySQL 8.0+
- Redis 7+
- Docker（可选）

### 本地运行

```bash
# 1. 克隆项目
git clone <your-repo-url> && cd meteorx

# 2. 复制并编辑配置
cp .env.example .env   # （如无 .env.example，直接编辑 .env）

# 3. 安装依赖
go mod download

# 4. 确保 MySQL / Redis 已启动，并在 .env 中配置好连接信息

# 5. 运行（首次启动会自动建表并插入种子数据）
go run cmd/server/main.go
```

服务默认启动在 `http://127.0.0.1:8081`，所有 API 统一前缀 `/api/v1`。

### Docker 方式

```bash
# 使用 docker-compose 一键启动 MySQL + Redis + 应用
docker-compose up -d

# 查看日志
docker-compose logs -f api
```

---

## 📝 统一响应格式

```json
{
  "code": 0,
  "message": "ok",
  "data": { ... }
}
```

**分页响应**

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "items": [...],
    "page": 1,
    "page_size": 10,
    "total": 128
  }
}
```

**请求参数**

- 查询列表：`?page=1&page_size=10&keyword=xxx`
- 批量操作 body：`{"ids": ["xxx"], "status": 0}`

---

## ⚙️ 配置

配置文件位于 `internal/config/config.yaml`，可通过 `.env` 环境变量覆盖（前缀 `METEORX_`）。

| YAML 路径 | .env 变量 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `server.port` | `METEORX_APP_PORT` | 8080 | 服务端口 |
| `server.mode` | `METEORX_APP_MODE` | `debug` | `debug` / `release` |
| `database.host` | `METEORX_DB_HOST` | `127.0.0.1` | MySQL 地址 |
| `database.port` | `METEORX_DB_PORT` | `3306` | MySQL 端口 |
| `database.user` | `METEORX_DB_USER` | `root` | MySQL 用户 |
| `database.password` | `METEORX_DB_PASSWORD` | `123456` | MySQL 密码 |
| `database.name` | `METEORX_DB_NAME` | `meteorx` | 数据库名 |
| `database.debug` | `METEORX_DB_DEBUG` | `true` | 是否打印 SQL |
| `redis.host` | `METEORX_REDIS_HOST` | `127.0.0.1` | Redis 地址 |
| `redis.port` | `METEORX_REDIS_PORT` | `6379` | Redis 端口 |
| `redis.password` | `METEORX_REDIS_PASSWORD` | `""` | Redis 密码 |
| `redis.db` | `METEORX_REDIS_DB` | `0` | Redis DB 索引 |
| `jwt.secret` | `METEORX_JWT_SECRET` | `your-secret-key` | Token 签名密钥（生产必须替换） |
| `jwt.expiration` | `METEORX_JWT_EXPIRATION` | `24h` | Token 有效期 |
| `jwt.issuer` | `METEORX_JWT_ISSUER` | `meteorx-auth` | Token Issuer |
| `log.level` | — | `info` | 日志等级 |
| `log.format` | — | `json` | `json` / `text` |
| `security.login_lockout.enabled` | — | `true` | 启用登录失败锁定 |
| `security.login_lockout.max_attempts` | — | `5` | 最大失败次数 |
| `security.login_lockout.lockout_duration` | — | `30m` | 锁定持续时间 |
| `security.password_policy.enabled` | — | `true` | 启用密码策略 |
| `security.password_policy.min_length` | — | `8` | 密码最小长度 |
| `security.password_policy.require_uppercase` | — | `true` | 需要大写字母 |
| `security.rate_limit.enabled` | — | `true` | 启用接口限流 |
| `security.rate_limit.requests` | — | `100` | 每窗口最大请求数 |
| `security.rate_limit.window` | — | `1m` | 限流窗口时长 |

---

## 🔐 安全建议

- **生产环境必须替换 `jwt.secret`** 为 32 位以上随机字符串
- 数据库密码、Redis 密码不要使用默认值
- 建议通过环境变量 / 密钥管理服务注入敏感配置，不要写入 YAML
- 部署时将 `server.mode` 改为 `release`，`database.debug` 改为 `false`
- 建议启用 HTTPS 反向代理（Nginx / Traefik）
- **登录失败锁定**：连续失败 5 次后账号锁定 30 分钟，防止暴力破解
- **密码策略**：默认要求 8 位以上，包含大小写字母、数字和特殊字符
- **接口限流**：每个 IP 每分钟最多 100 个请求，防止 DDoS 攻击

---

## 🧪 测试

```bash
# 运行所有测试
go test ./...

# 运行特定模块测试
go test ./internal/modules/audit/service/... -v
go test ./pkg/security/... -v

# 运行基准测试
go test -bench=. ./pkg/security/...
```

---

## 📚 接口文档

项目自带 API 文档，位于 `docs/apifox/`：

- `MeteorX-backend.openapi.json` — OpenAPI 3.0 规范，可直接导入 Swagger / Postman / Apifox
- `MeteorX-backend.apifox.json` — Apifox 项目文件，导入后可直接调试

---

## 📖 更多

详细的目录设计与模块说明见 `PROJECT_STRUCTURE.md`。

---

## 📄 License

MIT