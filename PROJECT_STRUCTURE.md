# MeteorX 项目目录结构

```
meteorx/
├── .env                           # 环境变量配置文件（覆盖 YAML 默认值）
├── .env.example                   # 环境变量示例
├── .gitignore                     # Git 忽略规则
├── Dockerfile                     # Docker 镜像构建文件
├── docker-compose.yml             # Docker Compose 编排（MySQL + Redis + App）
├── docker-compose.prod.yml        # 生产环境 Docker Compose
├── LICENSE                        # 开源许可证（MIT）
├── README.md                      # 项目说明文档（含快速开始、API 总览）
├── PROJECT_STRUCTURE.md           # 项目结构说明（本文件）
├── go.mod                         # Go 模块依赖
├── go.sum                         # Go 依赖校验
├── qodana.yaml                    # Qodana 代码扫描配置
├── deploy.sh                      # 部署脚本
├── cmd/                           # 应用程序入口
│   └── server/
│       └── main.go               # 服务器启动入口（加载配置→初始化模块→启动 HTTP Server）
│
├── internal/                      # 内部包（不对外暴露）
│   │
│   ├── bootstrap/                 # ⭐ 启动引导（核心编排层）
│   │   ├── app.go                # 应用初始化 + 优雅关闭（信号处理、资源清理）
│   │   ├── config.go             # Viper 配置加载（YAML + .env 合并）
│   │   ├── database.go           # GORM 连接池初始化（含连接池配置校验）
│   │   ├── middleware.go         # 全局中间件装配
│   │   ├── migrate.go            # 数据库自动迁移 + 种子数据（RBAC 权限等）
│   │   └── router.go             # Chi 路由注册（公共/鉴权/后台三级路由）
│   │
│   ├── cache/                     # 缓存层
│   │   └── redis.go              # Redis 客户端封装（含降级处理）
│   │
│   ├── common/                    # 通用组件
│   │   ├── contextx/             # 上下文扩展
│   │   │   ├── constants.go       # 上下文 Key 常量
│   │   │   └── contextx.go       # 上下文工具（获取 UserID/TenantID/Role 等）
│   │   ├── jwt/                  # JWT 工具封装
│   │   │   └── jwt.go            # Token 生成/解析/验证
│   │   ├── response/             # 统一响应封装
│   │   │   └── response.go       # Success/Fail/Paginated 响应
│   │   └── validator/            # 参数校验
│   │       └── validator.go      # 基于 go-playground 的校验器
│   │
│   ├── config/                    # 配置管理
│   │   ├── config.go             # 配置结构体定义（Server/Database/Redis/JWT/File/Security）
│   │   └── config.yaml           # YAML 默认配置
│   │
│   ├── middleware/                # HTTP 中间件
│   │   ├── auth.go               # JWT 认证中间件（解析 Token 注入上下文）
│   │   ├── admin_middleware.go   # 超级管理员权限校验
│   │   ├── auto_permission_middleware.go  # 自动权限推导（Method+Path → Permission Code）
│   │   ├── auto_permission_middleware_test.go  # 中间件单元测试
│   │   ├── permission_middleware.go       # 细粒度权限校验
│   │   ├── role_middleware.go    # 角色校验
│   │   ├── audit_middleware.go   # 审计日志记录（同步）
│   │   ├── audit_batch.go        # 审计日志批量写入（异步，缓冲 100 条 / 5s 刷新）
│   │   ├── rate_limit_middleware.go       # 接口限流（基于 Redis）
│   │   └── logger.go             # 请求日志中间件
│   │
│   └── modules/                   # ⭐ 业务模块（核心代码，按领域划分）
│       │
│       ├── auth/                  # 认证模块
│       │   ├── dto/
│       │   │   └── auth_dto.go   # LoginReq/RegisterUserReq/LoginResp
│       │   ├── handler/
│       │   │   └── auth_handler.go  # HTTP Handler
│       │   ├── service/
│       │   │   └── auth_service.go # 业务逻辑（注册/登录/登出）
│       │   ├── module.go          # 模块装配
│       │   └── routes.go          # 路由注册：/auth/*
│       │
│       ├── user/                  # 用户管理模块
│       │   ├── dto/
│       │   │   ├── user_converter.go  # Model ↔ DTO 转换
│       │   │   └── user_dto.go   # CreateUserReq/UpdateUserReq/UserResp 等
│       │   ├── handler/
│       │   │   └── user_handler.go   # HTTP Handler
│       │   ├── model/
│       │   │   └── user.go       # User 模型（GORM 映射）
│       │   ├── repository/
│       │   │   ├── interface.go   # Repository 接口定义
│       │   │   └── user_repository.go # Repository 实现
│       │   ├── service/
│       │   │   └── user_service.go   # 业务逻辑
│       │   ├── module.go          # 模块装配
│       │   └── routes.go          # 路由注册：/profile/*, /users/*, /admin/users/*, /admin/tenant-users/*
│       │
│       ├── tenant/                # 租户管理模块
│       │   ├── dto/
│       │   │   ├── admin_tenant_dto.go  # 管理员 DTO
│       │   │   ├── tenant_converter.go  # Model ↔ DTO 转换
│       │   │   └── tenant_dto.go # RegisterTenantReq/TenantResp/注销审批 DTO 等
│       │   ├── handler/
│       │   │   └── tenant_handler.go
│       │   ├── model/
│       │   │   ├── tenant.go     # Tenant 模型
│       │   │   └── cancel_request.go # CancelRequest（租户注销申请）模型
│       │   ├── repository/
│       │   │   ├── interface.go
│       │   │   ├── tenant_repository.go
│       │   │   └── cancel_request_repository.go # 注销申请仓储
│       │   ├── service/
│       │   │   └── tenant_service.go # 业务逻辑 + 注销申请/审批/执行
│       │   ├── cancel_cleanup_job.go # 到期注销后台任务
│       │   ├── module.go
│       │   └── routes.go         # 路由注册：/tenants/*, /admin/tenants/*, /admin/cancel-requests/*
│       │
│       ├── rbac/                  # RBAC 权限模块
│       │   ├── dto/
│       │   │   ├── permission_dto.go  # 权限 DTO
│       │   │   ├── role_dto.go    # 角色 DTO + RolePermissionResp
│       │   │   └── user_role_dto.go # 用户角色 DTO
│       │   ├── handler/
│       │   │   └── rbac_handler.go
│       │   ├── model/
│       │   │   ├── permission.go  # Permission 模型
│       │   │   ├── role.go        # Role 模型
│       │   │   ├── role_permission.go # RolePermission 关联模型
│       │   │   └── user_role.go   # UserRole 关联模型
│       │   ├── repository/
│       │   │   ├── interface.go
│       │   │   ├── migrate.go     # RBAC 表迁移
│       │   │   ├── permission_repository.go
│       │   │   ├── role_repository.go
│       │   │   ├── role_permission_repository.go
│       │   │   └── user_role_repository.go
│       │   ├── service/
│       │   │   └── rbac_service.go     # 业务逻辑 + PermissionChecker 实现
│       │   ├── module.go
│       │   ├── permissions.go    # 权限码常量定义 + 种子数据
│       │   └── routes.go         # 路由注册：/rbac/*
│       │
│       ├── file/                  # 文件管理模块
│       │   ├── dto/
│       │   │   ├── file_converter.go  # Model ↔ DTO 转换
│       │   │   └── file_dto.go   # FileListReq/FileUpdateReq 等
│       │   ├── handler/
│       │   │   └── file_handler.go
│       │   ├── model/
│       │   │   └── file.go       # File 模型（含 TenantID 隔离）
│       │   ├── repository/
│       │   │   ├── interface.go
│       │   │   └── file_repository.go # 含 GetByIDUnscoped（回收站支持）
│       │   ├── service/
│       │   │   └── file_service.go     # 业务逻辑 + 租户隔离校验
│       │   ├── storage/          # ⭐ 存储抽象层
│       │   │   ├── storage.go    # Storage 接口（Upload/Download/Delete/PresignURL）
│       │   │   └── local_storage.go # 本地存储实现（预留 OSS/S3 扩展）
│       │   ├── module.go
│       │   └── routes.go         # 路由注册：/files/*
│       │
│       ├── plan/                  # 套餐管理模块
│       │   ├── dto/
│       │   │   └── plan_dto.go   # CreatePlanReq/AssignPlanReq/CurrentPlanResp
│       │   ├── handler/
│       │   │   └── plan_handler.go
│       │   ├── model/
│       │   │   ├── plan.go       # Plan 模型
│       │   │   └── subscription.go # Subscription（租户套餐订阅）模型
│       │   ├── repository/
│       │   │   ├── interface.go
│       │   │   ├── mock_repository.go  # Mock 实现（测试用）
│       │   │   ├── mock_subscription_repository.go
│       │   │   └── plan_repository.go
│       │   ├── service/
│       │   │   ├── plan_service.go    # 业务逻辑
│       │   │   └── plan_service_test.go # 单元测试
│       │   ├── expiry_job.go     # 套餐到期检查后台任务
│       │   ├── seeds.go          # 默认套餐种子数据
│       │   ├── module.go
│       │   └── routes.go         # 路由注册：/admin/plans/*, /tenant/current/plan
│       │
│       └── audit/                 # 审计日志模块
│           ├── dto/
│           │   └── audit_dto.go # ListAuditLogsQuery/AuditLogResp 等
│           ├── handler/
│           │   └── audit_handler.go
│           ├── model/
│           │   ├── audit_log.go # AuditLog 模型
│           │   └── stats.go     # Stats 模型
│           ├── repository/
│           │   ├── interface.go
│           │   ├── audit_repository.go
│           │   └── mock_repository.go  # Mock 实现
│           ├── service/
│           │   ├── audit_service.go
│           │   └── audit_service_test.go
│           ├── module.go
│           └── routes.go        # 路由注册：/audit/*
│
│       ├── dashboard/            # 运营看板模块
│       │   ├── dto/
│       │   │   └── dashboard_dto.go # OverviewResp（运营数据总览）
│       │   ├── handler/
│       │   │   └── dashboard_handler.go
│       │   ├── repository/
│       │   │   ├── interface.go
│       │   │   └── dashboard_repository.go # 聚合统计查询
│       │   ├── service/
│       │   │   └── dashboard_service.go
│       │   ├── module.go
│       │   └── routes.go        # 路由注册：/admin/dashboard/*（仅平台管理员）
│       │
│       └── notification/          # 通知公告模块
│           ├── dto/
│           │   └── announcement_dto.go # CreateAnnouncementReq/AnnouncementResp 等
│           ├── handler/
│           │   └── announcement_handler.go
│           ├── model/
│           │   └── announcement.go # Announcement（公告）模型
│           ├── repository/
│           │   ├── interface.go
│           │   └── announcement_repository.go
│           ├── service/
│           │   └── announcement_service.go
│           ├── module.go
│           └── routes.go        # 路由注册：/admin/announcements/*（仅平台管理员）
│
├── pkg/                           # 可复用基础库（可对外暴露）
│   ├── crypto/
│   │   └── crypto.go             # bcrypt 密码哈希
│   ├── logger/
│   │   └── logger.go             # 结构化日志
│   ├── pagination/
│   │   └── pagination.go         # 统一分页请求/响应
│   ├── security/                 # 安全工具集
│   │   ├── login_lockout.go      # 登录锁定逻辑
│   │   ├── login_lockout_test.go
│   │   ├── password_policy.go    # 密码复杂度策略
│   │   ├── password_policy_test.go
│   │   ├── rate_limiter.go       # 限流工具
│   │   └── rate_limiter_test.go
│   ├── ulid/
│   │   └── ulid.go               # ULID 生成器封装
│   └── uuid/
│       └── uuid.go               # UUID 工具
│
├── scripts/
│   └── sql/
│       ├── init.sql              # 建表 DDL
│       └── seed.sql              # 种子数据（默认管理员、角色、权限）
│
├── docs/
│   ├── apifox/
│   │   ├── MeteorX-backend.apifox.json
│   │   └── MeteorX-backend.openapi.json
│   ├── auth-api.md
│   ├── user-api.md
│   ├── tenant-api.md
│   ├── rbac-api.md
│   ├── file-module-api.md
│   ├── plan-api.md
│   ├── audit-api.md
│   ├── dashboard-api.md
│   ├── announcement-api.md
│   └── FEATURE_UPGRADE.md
│
└── web-admin/                    # ⭐ 前端管理后台（Vue 3 + TypeScript）
    ├── Dockerfile
    ├── README.md
    ├── index.html
    ├── nginx.conf
    ├── package.json
    ├── vite.config.ts
    ├── tsconfig.json
    ├── .env.development
    ├── .env.production
    ├── dist/
    └── src/
        ├── api/
        │   ├── request.ts        # Axios 实例（拦截器、Token 注入）
        │   ├── auth.ts           # 登录/登出 API
        │   └── modules/          # 按模块划分的 API
        │       ├── audit.ts
        │       ├── file.ts
        │       ├── permission.ts
        │       ├── announcement.ts
        │       ├── cancel-request.ts
        │       ├── dashboard.ts
        │       ├── plan.ts
        │       ├── role.ts
        │       ├── tenant.ts
        │       └── user.ts
        ├── layouts/
        │   └── DefaultLayout.vue # 侧边栏 + Header + 主内容区
        ├── router/
        │   └── index.ts          # 路由配置
        ├── stores/
        │   ├── app.ts            # 全局应用状态
        │   └── user.ts           # 用户状态 + 权限码列表
        ├── utils/
        │   └── auth.ts           # Token 存储工具
        ├── views/
        │   ├── login/index.vue
        │   ├── dashboard/index.vue
        │   ├── profile/index.vue
        │   └── system/
        │       ├── user/index.vue
        │       ├── master-admin/
        │       │   ├── index.vue
        │       │   └── recycle.vue
        │       ├── tenant/
        │       │   ├── index.vue
        │       │   └── recycle.vue
        │       ├── role/
        │       │   ├── index.vue
        │       │   └── recycle.vue
        │       ├── permission/index.vue
        │       ├── file/index.vue
        │       ├── plan/index.vue
        │       ├── audit/index.vue
        │       ├── announcement/index.vue
        │       └── cancel-request/index.vue
        ├── App.vue
        ├── env.d.ts
        └── main.ts
```

---

## 架构说明

### 分层架构

```
┌─────────────────────────────────────────────┐
│  Handler Layer (handler/)                    │
│  HTTP 请求处理 · 参数验证 · 响应封装          │
├─────────────────────────────────────────────┤
│  Service Layer (service/)                    │
│  业务逻辑 · 事务管理 · DTO 转换 · 权限校验    │
├─────────────────────────────────────────────┤
│  Repository Layer (repository/)              │
│  数据访问抽象 · 数据库操作 · 事务执行          │
├─────────────────────────────────────────────┤
│  Model Layer (model/)                        │
│  业务实体 · GORM 映射 · 生命周期钩子          │
├─────────────────────────────────────────────┤
│  DTO Layer (dto/)                            │
│  数据传输对象 · 请求/响应结构定义              │
└─────────────────────────────────────────────┘
```

### 设计模式

- **Repository Pattern** — 数据访问抽象层，便于 Mock 测试
- **Dependency Injection** — 依赖注入（构造函数注入）
- **Clean Architecture** — 清洁架构原则
- **Multi-tenancy** — 基于 `tenant_id` 的行级数据隔离
- **Onion Architecture** — 洋葱架构（依赖方向向内）
- **Strategy Pattern** — 存储抽象（local/oss/s3 可切换）

### 核心中间件流程

```
请求 → Logger → RateLimit → Auth → AutoPermission → Permission → Handler
         │          │          │          │                │
         │          │          │          │                └→ 业务逻辑
         │          │          │          └→ 权限码校验
         │          │          └→ 自动推导权限码
         │          └→ 限流检查
         └→ 请求日志
```

### 技术栈

| 分类 | 技术 | 说明 |
|------|------|------|
| Web Framework | Chi v5 | 轻量高性能 Go 路由框架 |
| ORM | GORM v1 | Go 生态主流 ORM |
| Database | MySQL 8.0 | 主数据存储 |
| Cache | Redis v9 | 会话/限流/黑名单 |
| Authentication | JWT v5 | HS256 签名 |
| Validation | Validator v10 | 结构体标签校验 |
| ID 生成 | ULID | 时间有序，适合索引 |
| Configuration | Viper v1 | 多源配置合并 |
| Frontend | Vue 3 + TS + Element Plus | SPA 管理后台 |
| Deployment | Docker Compose | 容器化部署 |

### 业务模块说明

| 模块 | 功能 | 状态 |
|------|------|------|
| **auth** | 注册、登录、登出、Token 管理 | ✅ 已实现 |
| **user** | 个人中心、租户用户 CRUD、系统管理员 CRUD、跨租户用户管理、回收站、批量操作 | ✅ 已实现 |
| **tenant** | 自助开户、租户信息管理、后台租户 CRUD、注销申请→审批→执行闭环 | ✅ 已实现 |
| **rbac** | 角色 CRUD、权限 CRUD、角色权限绑定、用户角色分配、自动权限推导 | ✅ 已实现 |
| **file** | 文件上传/下载/重命名、MD5 去重、回收站、永久删除、存储抽象 | ✅ 已实现 |
| **plan** | 套餐 CRUD、租户套餐分配、用量限制、到期检查 | ✅ 已实现 |
| **audit** | 自动记录审计日志、多维度筛选、CSV 导出、过期清理 | ✅ 已实现 |
| **dashboard** | 平台运营数据总览：租户/用户/订阅/审计多维统计 | ✅ 已实现 |
| **notification** | 公告 CRUD、发布/下架、全平台或指定租户定向推送 | ✅ 已实现 |

### 公共包说明

| 包 | 功能 |
|----|------|
| `crypto` | bcrypt 密码哈希（Generate/Compare） |
| `logger` | 结构化日志（基于 log/slog） |
| `pagination` | 统一分页请求/响应结构体 |
| `security` | 登录锁定、密码策略、IP 限流 |
| `ulid` | ULID 生成器封装 |
| `uuid` | UUID v4 工具 |

### 中间件说明

| 中间件 | 功能 |
|--------|------|
| `auth` | JWT Token 解析，注入 UserID/TenantID/IsMaster 到上下文 |
| `admin_middleware` | 超级管理员权限校验（RequiresMasterAdmin） |
| `auto_permission_middleware` | 根据 HTTP Method + Path 自动推导权限码 |
| `permission_middleware` | 校验当前用户是否拥有指定权限码 |
| `role_middleware` | 角色校验（检查用户是否属于指定角色） |
| `audit_middleware` | 同步记录每次请求的审计日志 |
| `audit_batch` | 异步批量刷新审计日志（100 条 / 5 秒） |
| `rate_limit_middleware` | 基于 Redis 的 IP 限流 |
| `logger` | 请求日志（Method/Path/Duration/Status） |

### 配置管理

配置加载优先级（从低到高）：

1. `internal/config/config.yaml` — 默认值
2. `.env` 文件 — 环境变量覆盖
3. 系统环境变量 — 运行时覆盖

### 数据库设计

- **主键策略**：ULID（26 字符，按时间有序，适合 B+Tree 索引）
- **多租户隔离**：所有业务表带 `tenant_id` 字段，Service 层强制校验
- **软删除**：GORM `gorm.DeletedAt` 支持，回收站可恢复
- **物理删除**：回收站中的记录支持永久删除（Unscoped）
- **事务支持**：GORM Transaction 封装，用于租户创建等复杂场景
- **自动迁移**：启动时 `AutoMigrate` 同步表结构
- **种子数据**：首次启动自动插入默认管理员、角色、权限

### 文件存储设计

```
Storage Interface (storage/storage.go)
├── LocalStorage   — 本地文件系统（已实现）
├── OSSStorage     — 阿里云 OSS（预留扩展点）
└── S3Storage      — AWS S3（预留扩展点）

存储路径结构：
{upload_path}/{tenant_id}/{year}/{month}/{day}/{ulid}_{filename}
```

### API 端点汇总

| 模块 | 前缀 | 主要端点 |
|------|------|----------|
| 认证 | `/api/v1/auth` | register, login, logout |
| 个人中心 | `/api/v1/profile` | stats, profile, password |
| 用户管理 | `/api/v1/users` | CRUD, deleted, restore, permanent |
| 系统管理员 | `/api/v1/admin/users` | CRUD, batch, recycle |
| 跨租户用户 | `/api/v1/admin/tenant-users` | CRUD, batch |
| 租户（公开） | `/api/v1/tenants` | register |
| 租户（租户侧） | `/api/v1/tenants/current` | GET/PUT, status, cancel |
| 租户（管理员） | `/api/v1/admin/tenants` | CRUD, batch, recycle |
| 注销审批 | `/api/v1/admin/cancel-requests` | list, approve, reject |
| RBAC | `/api/v1/rbac` | roles, permissions, user-roles |
| 文件 | `/api/v1/files` | upload, CRUD, batch, recycle |
| 套餐（管理员） | `/api/v1/admin/plans` | CRUD |
| 套餐（租户） | `/api/v1/tenant/current/plan` | GET |
| 审计 | `/api/v1/audit` | stats, logs, export, cleanup |
| 运营看板 | `/api/v1/admin/dashboard` | overview |
| 通知公告 | `/api/v1/admin/announcements` | CRUD, status |
| 健康检查 | `/health` | GET |