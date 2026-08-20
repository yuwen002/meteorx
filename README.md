# MeteorX — 多租户 SaaS 平台

一套基于 Go + Chi + GORM + Redis 构建的 **MaaS (Multi-tenant as a Service)** 多租户 SaaS 平台，内置「平台超级管理员 / 租户管理员 / 普通用户」三级权限模型，完整的用户/租户/文件生命周期管理（软删除、回收站、恢复、批量操作、永久删除），配套的 Vue 3 + TypeScript 前端管理后台。

---

## ✨ 核心特性

| 模块 | 特性 |
| --- | --- |
| **多租户架构** | 基于 `tenant_id` 行级隔离；支持域名自动识别租户；独立管理员体系 |
| **认证鉴权** | JWT（HS256）双 Token；注册 / 登录 / 登出；Token 自动注入用户上下文；登出黑名单 |
| **RBAC 权限** | 角色 + 权限 + 角色-权限绑定 + 用户角色分配；支持作用域（`system` / `tenant` / `all`）；自动权限推导中间件 |
| **用户管理** | 个人中心 / 租户内用户 / 跨租户用户三类独立 API；密码加密（bcrypt） |
| **租户管理** | 自主注册开户；后台手动建租户；启用/禁用；软删除/恢复；批量操作 |
| **文件管理** | 上传/下载/重命名/删除；回收站恢复+永久删除；MD5 去重；租户隔离；本地/云存储可扩展 |
| **套餐管理** | 套餐 CRUD；租户套餐分配；用量限制（用户数上限）；到期提醒 |
| **审计日志** | 自动记录所有请求；支持多维度筛选查询；敏感信息脱敏；定时清理；CSV 导出 |
| **安全增强** | 登录失败锁定（显示剩余次数）；密码复杂度策略；接口限流（基于 Redis） |
| **回收站** | 用户 / 租户 / 角色 / 文件 均支持软删除 → 回收站查询 → 恢复 → 永久删除的完整闭环 |
| **批量操作** | 批量删除 / 批量更新状态；幂等返回影响行数 |
| **优雅关闭** | HTTP Server 信号处理；Redis/DB/后台任务资源清理 |
| **工程化** | Viper 配置 + `.env` 覆盖；Chi 路由；GORM 自动迁移；ULID 主键 |
| **前端管理后台** | Vue 3 + TypeScript + Element Plus；RBAC 动态菜单/按钮权限 |
| **测试覆盖** | 审计模块、安全模块单元测试；Mock 仓库实现 |

---

## 🧱 技术栈

### 后端

- **语言**：Go 1.25
- **Web 框架**：[go-chi/chi v5](https://github.com/go-chi/chi)
- **ORM**：[GORM v1.31](https://gorm.io/)
- **数据库**：MySQL 8.0+
- **缓存**：Redis 7+（go-redis v9）
- **认证**：JWT（golang-jwt v5）
- **校验**：go-playground/validator v10
- **ID 生成**：ULID（按时间有序、适合数据库索引）
- **配置**：Viper + YAML + `.env` 环境变量覆盖

### 前端

- **框架**：Vue 3 + TypeScript
- **UI**：Element Plus
- **状态管理**：Pinia
- **路由**：Vue Router
- **HTTP**：Axios
- **构建**：Vite

### 部署

- Docker / Docker Compose
- Nginx 反向代理

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
│   │   ├── app.go               # 应用初始化（模块装配 + 优雅关闭）
│   │   ├── config.go            # Viper 配置加载
│   │   ├── database.go          # GORM 数据库初始化 + 连接池
│   │   ├── migrate.go           # 自动建表 + 种子数据
│   │   ├── router.go            # 路由注册（公共/鉴权/后台）
│   │   └── middleware.go        # 中间件装配
│   │
│   ├── modules/                 # ⭐ 业务模块（核心代码）
│   │   ├── auth/                # 认证：注册、登录、JWT
│   │   ├── user/                # 用户：3 层接口 + 回收站 + 批量
│   │   ├── tenant/              # 租户：自助开户 + 后台管理
│   │   ├── rbac/                # 角色权限：角色、权限、绑定
│   │   ├── file/                # 文件：上传、下载、回收站、存储抽象
│   │   ├── plan/                # 套餐：CRUD + 租户分配 + 用量检查
│   │   └── audit/               # 审计日志：自动记录 + 统计 + 导出
│   │
│   ├── middleware/              # HTTP 中间件
│   │   ├── auth.go              # JWT 认证
│   │   ├── admin_middleware.go  # 超级管理员校验
│   │   ├── auto_permission_middleware.go  # 自动权限推导
│   │   ├── auto_permission_middleware_test.go  # 中间件单元测试
│   │   ├── permission_middleware.go        # 细粒度权限校验
│   │   ├── role_middleware.go   # 角色校验
│   │   ├── audit_middleware.go  # 审计日志记录
│   │   ├── audit_batch.go       # 审计日志批量写入
│   │   ├── rate_limit_middleware.go        # 接口限流
│   │   └── logger.go            # 请求日志
│   │
│   ├── common/                  # 通用组件
│   │   ├── contextx/            # 上下文扩展（TenantID/UserID 注入）
│   │   ├── jwt/                 # JWT 工具
│   │   ├── response/            # 统一响应封装
│   │   └── validator/           # 参数校验
│   │
│   ├── config/                  # 配置结构体 + YAML
│   └── cache/                   # Redis 客户端封装
│
├── pkg/                         # 可复用基础库
│   ├── crypto/                  # bcrypt 密码加密
│   ├── logger/                  # 日志封装
│   ├── pagination/              # 统一分页请求/响应
│   ├── security/               # 安全工具（锁定、密码策略、限流）
│   │   ├── login_lockout.go
│   │   ├── login_lockout_test.go
│   │   ├── password_policy.go
│   │   ├── password_policy_test.go
│   │   ├── rate_limiter.go
│   │   └── rate_limiter_test.go
│   ├── ulid/                    # ULID 生成
│   └── uuid/                    # UUID 工具
│
├── scripts/sql/
│   ├── init.sql                 # 建表脚本
│   └── seed.sql                 # 种子数据（默认管理员等）
│
├── docs/                        # 接口文档
│   ├── apifox/                  # Apifox / OpenAPI
│   ├── auth-api.md              # 认证接口
│   ├── user-api.md              # 用户管理接口
│   ├── tenant-api.md            # 租户管理接口
│   ├── rbac-api.md              # RBAC 权限接口
│   ├── file-module-api.md       # 文件管理接口
│   ├── plan-api.md              # 套餐管理接口
│   └── audit-api.md             # 审计日志接口
│
├── web-admin/                   # ⭐ 前端管理后台（Vue 3）
│   ├── src/
│   │   ├── api/                 # API 请求封装
│   │   ├── views/               # 页面组件
│   │   ├── stores/              # Pinia 状态管理
│   │   ├── router/              # 路由配置
│   │   └── layouts/             # 布局组件
│   └── dist/                    # 构建产物
│
├── .env                         # 环境变量（覆盖 YAML 默认值）
├── .env.example                 # 环境变量示例
├── .gitignore                   # Git 忽略规则
├── LICENSE                      # 开源许可证（MIT）
├── Dockerfile
├── docker-compose.yml
├── docker-compose.prod.yml      # 生产环境 Docker Compose
├── deploy.sh                    # 部署脚本
├── go.mod
├── go.sum
├── qodana.yaml                  # Qodana 代码扫描配置
└── README.md
```

---

## 🔌 API 总览（`/api/v1`）

完整接口文档见 `docs/` 目录下各模块说明文档。

### 1. 认证（公开）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/auth/register` | 用户自主注册 |
| `POST` | `/auth/login` | 登录，返回 JWT + 用户信息 + 权限码列表 |
| `POST` | `/auth/logout` | 登出（Token 加入黑名单） |

### 2. 用户个人中心（需登录）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/profile/stats` | Dashboard 统计 |
| `GET` | `/profile` | 查看当前用户信息 |
| `PUT` | `/profile` | 修改昵称/邮箱等 |
| `PUT` | `/profile/password` | 修改当前用户密码 |

### 3. 租户内用户管理（需登录 · 本租户）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/users` | 分页查询本租户用户 |
| `POST` | `/users` | 在本租户下创建用户 |
| `GET` | `/users/deleted` | 🗑️ 已删除用户列表 |
| `GET` | `/users/{id}/detail` | 用户详情 |
| `PUT` | `/users/{id}/update` | 更新用户 |
| `PUT` | `/users/{id}/reset-password` | 重置用户密码 |
| `DELETE` | `/users/{id}/delete` | 软删除用户 |
| `PUT` | `/users/{id}/restore` | ♻️ 恢复用户 |
| `DELETE` | `/users/{id}/permanent` | ☠️ 永久删除 |

### 4. 系统管理员管理（需超级管理员）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/admin/stats` | 全平台 Dashboard |
| `GET` | `/admin/users` | 系统管理员列表 |
| `POST` | `/admin/users` | 创建系统管理员 |
| `GET` | `/admin/users/deleted` | 已删除管理员列表 |
| `GET` | `/admin/users/{id}/detail` | 管理员详情 |
| `PUT` | `/admin/users/{id}/update` | 更新管理员 |
| `PUT` | `/admin/users/{id}/status` | 启用/禁用 |
| `DELETE` | `/admin/users/{id}/delete` | 软删除 |
| `PUT` | `/admin/users/{id}/restore` | 恢复 |
| `DELETE` | `/admin/users/{id}/permanent` | 永久删除 |
| `PUT` | `/admin/users/batch/status` | 🔁 批量更新状态 |
| `DELETE` | `/admin/users/batch/delete` | 🔁 批量删除 |

### 5. 跨租户用户管理（需超级管理员）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/admin/tenant-users/all` | 全量租户用户列表 |
| `POST` | `/admin/tenant-users` | 为指定租户创建用户 |
| `GET` | `/admin/tenant-users/{tenantID}/list` | 指定租户用户列表 |
| `PUT` | `/admin/tenant-users/{tenantID}/{userID}/update` | 更新租户用户 |
| `PUT` | `/admin/tenant-users/{tenantID}/{userID}/status` | 启用/禁用 |
| `DELETE` | `/admin/tenant-users/{tenantID}/{userID}/delete` | 删除 |
| `GET` | `/admin/tenant-users/deleted/all` | 全量已删除租户用户 |
| `PUT` | `/admin/tenant-users/{tenantID}/{userID}/restore` | 恢复 |
| `DELETE` | `/admin/tenant-users/{tenantID}/{userID}/permanent` | 永久删除 |
| `PUT` | `/admin/tenant-users/{tenantID}/batch/status` | 批量更新状态 |
| `DELETE` | `/admin/tenant-users/{tenantID}/batch/delete` | 批量删除 |

### 6. 租户管理

**公开接口**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/tenants/register` | 企业自主注册开户 |

**租户私有接口**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/tenants/current` | 当前租户详情 |
| `PUT` | `/tenants/current` | 修改当前租户信息 |
| `GET` | `/tenants/current/status` | 查询初始化状态 |
| `POST` | `/tenants/current/cancel` | 申请自主注销 |

**管理员接口**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/admin/tenants` | 全量租户列表 |
| `POST` | `/admin/tenants` | 后台新建租户 |
| `GET` | `/admin/tenants/{id}/detail` | 租户详情 |
| `PUT` | `/admin/tenants/{id}/update` | 更新租户 |
| `PUT` | `/admin/tenants/{id}/status` | 启用/禁用 |
| `DELETE` | `/admin/tenants/{id}/delete` | 软删除 |
| `GET` | `/admin/tenants/deleted` | 已删除租户列表 |
| `PUT` | `/admin/tenants/{id}/restore` | 恢复 |
| `PUT` | `/admin/tenants/batch/status` | 批量更新状态 |
| `DELETE` | `/admin/tenants/batch` | 批量删除 |

### 7. RBAC 角色与权限

**角色管理**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/rbac/roles` | 角色列表 |
| `GET` | `/rbac/roles/select` | 角色下拉选项（不分页） |
| `GET` | `/rbac/roles/system-admin` | 系统管理员角色列表 |
| `POST` | `/rbac/roles` | 创建角色 |
| `GET` | `/rbac/roles/{id}/detail` | 角色详情 |
| `PUT` | `/rbac/roles/{id}/update` | 更新角色 |
| `PUT` | `/rbac/roles/{id}/status` | 启用/禁用 |
| `DELETE` | `/rbac/roles/{id}/delete` | 删除角色 |
| `GET` | `/rbac/roles/deleted` | 已删除角色列表 |
| `PUT` | `/rbac/roles/{id}/restore` | 恢复角色 |
| `PUT` | `/rbac/roles/{id}/permissions` | 绑定权限（覆盖式） |
| `GET` | `/rbac/roles/{id}/permissions` | 获取角色权限 |
| `DELETE` | `/rbac/roles/{id}/permissions` | 解绑单个权限 |
| `DELETE` | `/rbac/roles/{id}/permissions/batch` | 批量解绑权限 |
| `PUT` | `/rbac/roles/batch/status` | 批量更新状态 |
| `DELETE` | `/rbac/roles/batch/delete` | 批量删除 |
| `PUT` | `/rbac/roles/batch/permissions` | 批量绑定权限 |
| `DELETE` | `/rbac/roles/batch/permissions` | 批量解绑权限 |

**权限管理**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/rbac/permissions` | 权限列表 |
| `POST` | `/rbac/permissions` | 创建权限 |
| `GET` | `/rbac/permissions/{id}/detail` | 权限详情 |
| `PUT` | `/rbac/permissions/{id}/update` | 更新权限 |
| `PUT` | `/rbac/permissions/{id}/status` | 启用/禁用 |
| `DELETE` | `/rbac/permissions/{id}/delete` | 删除权限 |
| `PUT` | `/rbac/permissions/batch/status` | 批量更新状态 |
| `DELETE` | `/rbac/permissions/batch/delete` | 批量删除 |

**用户角色管理**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/rbac/user-roles` | 用户角色关系列表 |
| `POST` | `/rbac/user-roles/batch/assign` | 批量分配用户角色 |
| `GET` | `/rbac/user-roles/{user_id}/roles` | 获取用户角色 |
| `POST` | `/rbac/user-roles/{user_id}/roles` | 分配角色 |
| `DELETE` | `/rbac/user-roles/{user_id}/roles` | 移除所有角色 |
| `DELETE` | `/rbac/user-roles/{user_id}/roles/{role_id}` | 移除单个角色 |
| `GET` | `/rbac/user-roles/roles/{role_id}/users` | 获取角色下用户 |

### 8. 文件管理

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/files/upload` | 上传文件（multipart/form-data） |
| `GET` | `/files` | 租户文件列表 |
| `GET` | `/files/my` | 当前用户文件列表 |
| `GET` | `/files/{id}` | 文件详情 |
| `GET` | `/files/{id}/download` | 文件下载 |
| `PUT` | `/files/{id}` | 重命名 |
| `DELETE` | `/files/{id}` | 软删除（进回收站） |
| `POST` | `/files/batch/delete` | 批量删除 |
| `GET` | `/files/deleted` | 回收站列表 |
| `PUT` | `/files/{id}/restore` | 恢复文件 |
| `DELETE` | `/files/{id}/permanent` | ☠️ 永久删除 |

### 9. 套餐管理（管理员）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/admin/plans` | 套餐列表 |
| `GET` | `/admin/plans/select` | 启用的套餐下拉 |
| `POST` | `/admin/plans` | 创建套餐 |
| `PUT` | `/admin/plans/{id}/update` | 更新套餐 |
| `DELETE` | `/admin/plans/{id}/delete` | 删除套餐 |
| `GET` | `/admin/tenants-plan/{id}` | 查询租户套餐 |
| `PUT` | `/admin/tenants-plan/{id}` | 为租户分配/变更套餐 |
| `GET` | `/tenant/current/plan` | 当前租户套餐与用量（租户侧） |

### 10. 审计日志

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/audit/stats` | Dashboard 统计 |
| `GET` | `/audit/logs` | 日志列表（多维度筛选） |
| `GET` | `/audit/logs/export` | 导出为 CSV |
| `POST` | `/audit/logs` | 创建日志（内部） |
| `GET` | `/audit/logs/{id}` | 日志详情 |
| `DELETE` | `/audit/logs/cleanup` | 清理过期日志 |

---

## 🚀 快速开始

### 前置条件

- Go 1.25+
- Node.js 18+（前端构建）
- MySQL 8.0+
- Redis 7+
- Docker（可选）

### 后端运行

```bash
# 1. 克隆项目
git clone <your-repo-url> && cd meteorx

# 2. 复制并编辑配置
cp .env.example .env

# 3. 安装依赖
go mod download

# 4. 确保 MySQL / Redis 已启动，并在 .env 中配置好连接信息

# 5. 运行（首次启动会自动建表并插入种子数据）
go run cmd/server/main.go
```

服务默认启动在 `http://127.0.0.1:8081`，所有 API 统一前缀 `/api/v1`。

### 前端运行

```bash
cd web-admin

# 安装依赖
npm install

# 开发模式
npm run dev

# 构建生产版本
npm run build
```

默认登录：`admin / admin123`（超级管理员）

### Docker 方式

```bash
# 使用 docker-compose 一键启动 MySQL + Redis + 应用
docker-compose up -d

# 查看日志
docker-compose logs -f api
```

---

## 📝 统一响应格式

**成功响应**
```json
{
  "code": 200,
  "message": "success",
  "data": { ... }
}
```

**分页响应**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [...],
    "pagination": { "page": 1, "page_size": 10, "total": 128 }
  }
}
```

**错误响应**
```json
{
  "code": 400,
  "message": "参数校验失败",
  "errors": { "username": "用户名不能为空" }
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
| `jwt.secret` | `METEORX_JWT_SECRET` | `your-secret-key` | Token 签名密钥 |
| `jwt.expiration` | `METEORX_JWT_EXPIRATION` | `24h` | Token 有效期 |
| `jwt.issuer` | `METEORX_JWT_ISSUER` | `meteorx-auth` | Token Issuer |
| `file.upload_path` | — | `./uploads` | 文件存储路径 |
| `file.upload_url` | — | `/uploads` | 文件访问 URL 前缀 |
| `file.max_file_size` | — | `10485760` | 最大文件大小（10MB） |
| `file.storage_type` | — | `local` | 存储类型（local/oss/s3） |
| `log.level` | — | `info` | 日志等级 |
| `log.format` | — | `json` | `json` / `text` |
| `security.login_lockout.enabled` | — | `true` | 启用登录锁定 |
| `security.login_lockout.max_attempts` | — | `5` | 最大失败次数 |
| `security.login_lockout.lockout_duration` | — | `30m` | 锁定持续时间 |

---

## 🔐 安全建议

- **生产环境必须替换 `jwt.secret`** 为 32 位以上随机字符串
- 数据库密码、Redis 密码不要使用默认值
- 建议通过环境变量 / 密钥管理服务注入敏感配置
- 部署时将 `server.mode` 改为 `release`，`database.debug` 改为 `false`
- 建议启用 HTTPS 反向代理（Nginx / Traefik）
- **登录锁定**：连续失败 5 次后账号锁定 30 分钟
- **密码策略**：默认要求 8 位以上，包含大小写字母和数字
- **接口限流**：每个 IP 每分钟最多 100 个请求
- **审计日志**：异步批量写入，支持 CSV 导出
- **Redis 降级**：Redis 不可用时应用仍可运行（限流/锁定功能降级）
- **优雅关闭**：支持 SIGTERM/SIGINT 信号，确保资源清理完成

---

## 🧪 测试

```bash
# 运行所有测试
go test ./...

# 运行特定模块测试
go test ./internal/modules/audit/service/... -v
go test ./internal/modules/plan/service/... -v
go test ./pkg/security/... -v

# 运行基准测试
go test -bench=. ./pkg/security/...
```

---

## 📚 接口文档

详细的模块接口文档见 `docs/` 目录：

| 文档 | 说明 |
|------|------|
| [auth-api.md](docs/auth-api.md) | 认证模块接口（注册/登录/登出） |
| [user-api.md](docs/user-api.md) | 用户管理接口（个人中心/租户用户/管理员） |
| [tenant-api.md](docs/tenant-api.md) | 租户管理接口（注册/后台管理） |
| [rbac-api.md](docs/rbac-api.md) | RBAC 权限接口（角色/权限/绑定） |
| [file-module-api.md](docs/file-module-api.md) | 文件管理接口（上传/下载/回收站） |
| [plan-api.md](docs/plan-api.md) | 套餐管理接口（CRUD/分配/用量） |
| [audit-api.md](docs/audit-api.md) | 审计日志接口（查询/导出/清理） |

OpenAPI 规范文件位于 `docs/apifox/`：
- `MeteorX-backend.openapi.json` — 可导入 Swagger / Postman / Apifox
- `MeteorX-backend.apifox.json` — Apifox 项目文件

---

## 🏗️ 前端管理后台

`web-admin/` 目录为配套的 Vue 3 前端管理后台。

### 页面结构

| 页面 | 路由 | 功能 |
|------|------|------|
| 登录 | `/login` | 用户名密码登录 |
| Dashboard | `/dashboard` | 数据统计总览 |
| 个人中心 | `/profile` | 修改个人信息/密码 |
| 用户管理 | `/system/user` | 租户用户 CRUD |
| 系统管理员 | `/system/master-admin` | 管理员 CRUD |
| 管理员回收站 | `/system/master-admin/recycle` | 已删除管理员恢复/永久删除 |
| 租户管理 | `/system/tenant` | 租户 CRUD |
| 租户回收站 | `/system/tenant/recycle` | 已删除租户恢复/永久删除 |
| 角色管理 | `/system/role` | 角色 CRUD + 权限绑定 |
| 角色回收站 | `/system/role/recycle` | 已删除角色恢复/永久删除 |
| 权限管理 | `/system/permission` | 权限 CRUD |
| 文件管理 | `/system/file` | 上传/下载/重命名/回收站/永久删除 |
| 套餐管理 | `/system/plan` | 套餐 CRUD |
| 审计日志 | `/system/audit` | 日志查询/导出 |

### 权限控制

- 动态菜单：根据用户权限码显示/隐藏侧边栏
- 动态按钮：根据权限码显示/隐藏操作按钮
- 登录时后端返回 `permissions` 数组，前端存入 Pinia Store

---

## 📄 License

MIT