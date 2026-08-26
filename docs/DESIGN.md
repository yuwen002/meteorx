# MeteorX 项目设计文档

## 设计哲学

MeteorX 是一个基于 Go 构建的多租户 SaaS 平台后端框架。核心设计哲学：

1. **Clean Architecture** — 严格的分层架构，依赖方向向内
2. **Multi-tenancy First** — 多租户隔离是基础设施，不是业务逻辑
3. **Security by Default** — 安全是默认状态，不是可选配置
4. **Developer Experience** — 统一错误响应、分页、事务，降低心智负担
5. **Production Ready** — 从一开始就面向生产环境设计

---

## 架构分层

### 五层业务架构

```
┌─────────────────────────────────────────────┐
│  Handler Layer (handler/)                    │
│  HTTP 请求处理 · 参数验证 · 响应封装          │
│  依赖: Service 接口                          │
├─────────────────────────────────────────────┤
│  Service Layer (service/)                    │
│  业务逻辑 · 事务管理 · DTO 转换 · 权限校验    │
│  依赖: Repository 接口 + TxManager           │
├─────────────────────────────────────────────┤
│  Repository Layer (repository/)              │
│  数据访问抽象 · 数据库操作 · 事务执行          │
│  依赖: GORM DB + tenantctx                   │
├─────────────────────────────────────────────┤
│  Model Layer (model/)                        │
│  业务实体 · GORM 映射 · 生命周期钩子          │
├─────────────────────────────────────────────┤
│  DTO Layer (dto/)                            │
│  数据传输对象 · 请求/响应结构定义              │
└─────────────────────────────────────────────┘
```

### 依赖方向规则

```
Handler → Service → Repository → Model
                    ↘
                     TxManager

所有依赖通过接口注入，禁止跨层直接依赖。
```

### 分层职责

| 层 | 禁止做什么 | 应该做什么 |
|----|------------|------------|
| Handler | 写业务逻辑、直接操作数据库 | 参数验证、调用 Service、封装响应 |
| Service | 写 SQL、处理 HTTP 请求 | 业务编排、事务控制、权限校验 |
| Repository | 包含业务规则 | 纯数据访问、GORM 查询构建 |
| Model | 包含业务逻辑 | GORM 映射定义、常量定义 |

---

## 请求数据流

### 完整请求链路

```
HTTP Request
    │
    ▼
┌─ Middleware Chain ──────────────────────────────────────┐
│                                                          │
│  1. RequestIDMiddleware                                  │
│     ├── 生成 request_id                                   │
│     ├── 注入上下文                                        │
│     └── recover panic → AppError                         │
│                                                          │
│  2. LoggerMiddleware                                     │
│     └── 记录 Method/Path/Duration/Status                 │
│                                                          │
│  3. RateLimitMiddleware                                  │
│     └── Redis IP 限流（100 req/min）                     │
│                                                          │
│  4. AuthMiddleware                                       │
│     ├── 解析 JWT Token                                   │
│     ├── 注入 UserID/TenantID/Role/IsMaster 到上下文      │
│     └── Token 黑名单检查                                 │
│                                                          │
│  5. AutoPermissionMiddleware                             │
│     ├── Method + Path → Permission Code                  │
│     └── 自动推导权限码 (e.g., GET /wiki/spaces → wiki:wiki_space:list) │
│                                                          │
│  6. ExplicitPermissionMiddleware                         │
│     └── 对自动推导结果进行特殊覆盖                       │
│                                                          │
│  7. PermissionMiddleware                                 │
│     └── 校验用户是否拥有推导的权限码                      │
│                                                          │
│  8. AuditMiddleware                                      │
│     └── 同步记录审计日志                                 │
│                                                          │
└──────────────────────────────────────────────────────────┘
    │
    ▼
┌─ Handler ─────────────────────────────────────────────┐
│  1. validator.ValidateJSON() — 参数校验                 │
│  2. contextx.GetXxx() — 获取上下文数据                   │
│  3. svc.Method() — 调用 Service                        │
│  4. response.Success/FailError() — 统一响应             │
└────────────────────────────────────────────────────────┘
    │
    ▼
┌─ Service ─────────────────────────────────────────────┐
│  1. 权限校验 (CheckXxxPermission)                       │
│  2. 业务逻辑执行                                        │
│  3. s.tx.WithTx() — 事务包裹（可选）                    │
│  4. repo.Method() — 数据访问                            │
└────────────────────────────────────────────────────────┘
    │
    ▼
┌─ Repository ──────────────────────────────────────────┐
│  1. tenantctx.FilterQuery() — 自动租户隔离              │
│  2. GORM 查询构建                                       │
│  3. 返回结果 / 错误                                     │
└────────────────────────────────────────────────────────┘
    │
    ▼
  MySQL (带 tenant_id 行级隔离)
```

---

## 多租户架构

### 隔离策略

```
┌─────────────────────────────────────────────────────┐
│                  租户 A (ID: tenant_a)               │
│  ┌───────────┐  ┌───────────┐  ┌───────────────┐  │
│  │  Users    │  │  Wiki      │  │  Files        │  │
│  └───────────┘  └───────────┘  └───────────────┘  │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│                  租户 B (ID: tenant_b)               │
│  ┌───────────┐  ┌───────────┐  ┌───────────────┐  │
│  │  Users    │  │  Wiki      │  │  Files        │  │
│  └───────────┘  └───────────┘  └───────────────┘  │
└─────────────────────────────────────────────────────┘

所有业务数据通过 tenant_id 字段行级隔离
```

### 核心机制

```go
// tenantctx.FilterQuery — 自动注入租户过滤
query := tenantctx.FilterQuery(ctx, db, "tenant_id")
// 当 IsMaster=false 时: WHERE tenant_id = 当前租户ID
// 当 IsMaster=true 时: 不过滤（系统管理员跨租户访问）
// 缺少租户上下文: WHERE 1=0（防止数据泄露）
```

### 租户上下文注入

```go
// AuthMiddleware 解析 JWT 后注入
contextx.SetVars(ctx, userID, tenantID, roles, isMaster)

// 后续全链路通过 contextx 获取
tenantID := contextx.GetTenantID(ctx)
userID := contextx.GetUserID(ctx)
isMaster := contextx.IsMaster(ctx)
```

---

## RBAC 权限体系

### 三级权限模型

```
┌─────────────────────────────────────────────────┐
│  Platform Level (超级管理员)                     │
│  - 管理所有租户                                  │
│  - 分配套餐                                      │
│  - 审计全平台日志                                │
├─────────────────────────────────────────────────┤
│  Tenant Level (租户管理员)                       │
│  - 管理本租户用户                                │
│  - 配置租户独立设置                              │
│  - 管理本租户 Wiki / 文件                        │
├─────────────────────────────────────────────────┤
│  Business Level (业务权限)                       │
│  - 基于角色的细粒度权限控制                      │
│  - 自动权限推导 + 显式覆盖                       │
│  - 节点级补充授权                                │
└─────────────────────────────────────────────────┘
```

### 自动权限推导

```
HTTP Method + API Path → Permission Code

GET    /wiki/spaces           → wiki:wiki_space:list
POST   /wiki/spaces           → wiki:wiki_space:create
GET    /wiki/spaces/{id}      → wiki:wiki_space:read
PUT    /wiki/spaces/{id}      → wiki:wiki_space:update
DELETE /wiki/spaces/{id}      → wiki:wiki_space:delete
```

### 权限推导规则

| HTTP Method | 权限动作 |
|-------------|----------|
| GET | list / read |
| POST | create |
| PUT | update |
| DELETE | delete |

特殊路径有特殊推导规则（如 `/wiki/stats` → `wiki:wiki_space:stats`）。

---

## 事务管理

### TxManager 设计

```go
// 统一事务入口
err := txManager.WithTx(ctx, func(txCtx context.Context, tx *gorm.DB) error {
    // txCtx 绑定了事务
    // 所有 Repository 方法接受 context.Context
    // 在事务内调用时自动使用 tx
    repo.CreateSpace(txCtx, space)
    repo.AddMember(txCtx, member)
    return nil
})
```

### 事务边界

| 场景 | 是否需要事务 | 原因 |
|------|-------------|------|
| 创建空间 + 添加 Owner | 是 | 保证原子性 |
| 更新文档 + 创建 Revision | 是 | 保证版本一致性 |
| 删除节点 + 级联删除 | 是 | 防止孤儿数据 |
| 恢复版本 + 保存快照 | 是 | 防止状态不一致 |
| 列出数据 | 否 | 只读操作 |
| 单个实体创建 | 可选 | 单操作无需事务 |

### 嵌套事务处理

```
WithTx 包裹的最外层事务内：
  - 递归调用 deleteNodeRecursive 时传递同一个 txCtx
  - 避免在递归中产生嵌套事务
  - 所有操作在同一个事务中完成
```

---

## 错误处理

### 集中式错误码

```go
// internal/pkg/apperrors/codes.go
var (
    ErrBadRequest      = &AppError{Code: "INVALID_PARAM", StatusCode: 400}
    ErrNotFound        = &AppError{Code: "RESOURCE_NOT_FOUND", StatusCode: 404}
    ErrForbidden       = &AppError{Code: "PERMISSION_DENIED", StatusCode: 403}
    ErrConflict        = &AppError{Code: "CONFLICT", StatusCode: 409}
    ErrRateLimited     = &AppError{Code: "RATE_LIMITED", StatusCode: 429}
    ErrInternal        = &AppError{Code: "INTERNAL_ERROR", StatusCode: 500}
    ErrSessionExpired  = &AppError{Code: "SESSION_EXPIRED", StatusCode: 401}
    ErrAuthLocked      = &AppError{Code: "AUTH_LOCKED", StatusCode: 423}
)
```

### 错误响应格式

```json
{
  "code": "WIKI_DOCUMENT_NOT_FOUND",
  "message": "Document not found",
  "request_id": "req_01H7K3N5P8...",
  "details": null
}
```

### 全局异常恢复

```go
// RequestIDMiddleware 中捕获 panic
// recover → AppError{Code: "INTERNAL_ERROR", StatusCode: 500}
// 保证所有异常都有统一响应
```

---

## 统一响应规范

### 成功响应（单对象）

```json
{
  "data": { ... },
  "request_id": "req_01H7K3N5P8..."
}
```

### 成功响应（分页列表）

```json
{
  "data": [ ... ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 150
  },
  "request_id": "req_01H7K3N5P8..."
}
```

### 成功响应（无内容）

```
HTTP 204 No Content
```

### 错误响应

```json
{
  "code": "WIKI_DOCUMENT_NOT_FOUND",
  "message": "Document not found",
  "request_id": "req_01H7K3N5P8...",
  "details": null
}
```

---

## ID 生成策略

### ULID 统一入口

```go
// pkg/idgen/idgen.go
idgen.New()      → 生成 ULID（26 字符，按时间有序）
idgen.NewUUID()  → 生成 UUID v4
idgen.MustParse() → 解析并验证 ID 格式
```

### 为什么选择 ULID

| 特性 | UUID | ULID |
|------|------|------|
| 长度 | 36 字符 | 26 字符 |
| 时间有序 | 否 | 是（适合 B+Tree 索引） |
| 可排序 | 否 | 是 |
| 可读性 | 低 | 高 |

### 使用规范

- **禁止**直接使用 `ulid.New()` 或 `uuid.New()`
- **必须**通过 `pkg/idgen` 统一入口
- 所有主键使用 ULID

---

## 安全设计

### 认证安全

- **JWT HS256**：双 Token 机制（Access + Refresh）
- **Token 黑名单**：登出后 Token 加入 Redis 黑名单
- **密码加密**：bcrypt（cost=10）
- **登录锁定**：连续 5 次失败后锁定 30 分钟
- **密码策略**：8 位以上，含大小写字母和数字

### 传输安全

- HTTPS 强制（生产环境）
- 接口限流：100 req/min（基于 Redis）
- 请求 ID 追踪：每个请求唯一 ID

### 内容安全

- Markdown XSS 防护：HTML 转义 + Sanitize
- SQL 注入防护：Sort 字段白名单 + 参数化查询
- 文件上传：MIME 类型检查 + 大小限制

### 审计安全

- 所有写操作自动记录审计日志
- 审计日志不可篡改
- 支持 CSV 导出和多维度查询

---

## 配置管理

### 配置加载优先级

```
1. internal/config/config.yaml    — 默认值（最低优先级）
2. .env 文件                       — 环境变量覆盖
3. 系统环境变量                    — 运行时覆盖（最高优先级）
```

### 配置验证

```go
// internal/config/validator.go
// 启动时校验必填项
// 缺失必填项则拒绝启动
```

### 配置项分类

| 分类 | 配置项 | 说明 |
|------|--------|------|
| server | port, mode | 服务端口和运行模式 |
| database | host, port, user, password, name | MySQL 连接 |
| redis | host, port, password, db | Redis 连接 |
| jwt | secret, expiration, issuer | JWT 配置 |
| file | upload_path, max_file_size, storage_type | 文件存储 |
| security | login_lockout.* | 安全策略 |
| email | host, port, username, password | SMTP 邮件 |

---

## 数据库设计

### 主键策略

- **类型**：ULID（26 字符）
- **生成**：`pkg/idgen` 统一入口
- **索引**：按时间有序，适合 B+Tree 聚簇索引

### 多租户隔离

- 所有业务表包含 `tenant_id` 字段
- 通过 `tenantctx.FilterQuery()` 自动注入过滤条件
- 系统管理员可跨租户访问（`IsMaster=true`）

### 软删除

- GORM `gorm.DeletedAt` 支持
- 回收站可恢复
- 支持永久删除（Unscoped）

### 自动迁移

- 启动时 `AutoMigrate` 同步表结构
- 首次启动自动插入种子数据

---

## 模块边界

### 业务模块

| 模块 | 职责 | 对外暴露 |
|------|------|----------|
| auth | 认证、注册、JWT | Handler + Service |
| user | 用户 CRUD、回收站 | Handler + Service |
| tenant | 租户管理、独立配置 | Handler + Service |
| rbac | 角色、权限、绑定 | Handler + Service |
| file | 文件上传/下载/存储抽象 | Handler + Service |
| plan | 套餐 CRUD、分配、用量 | Handler + Service |
| audit | 审计日志、统计、导出 | Handler + Service |
| wiki | 知识库核心功能 | Handler + Service |
| dashboard | 运营数据统计 | Handler + Service |
| notification | 公告 CRUD、定向推送 | Handler + Service |

### 基础设施模块

| 模块 | 职责 | 路径 |
|------|------|------|
| idgen | 统一 ID 生成 | pkg/idgen/ |
| pagination | 分页/排序/搜索 | pkg/pagination/ |
| apperrors | 集中式错误码 | internal/pkg/apperrors/ |
| db | 事务管理 | internal/pkg/db/ |
| tenantctx | 租户上下文 | internal/common/tenantctx/ |
| auditctx | 审计上下文 | internal/common/auditctx/ |
| response | 统一响应封装 | internal/common/response/ |

---

## 技术选型

### 后端技术栈

| 技术 | 版本 | 选型理由 |
|------|------|----------|
| Go | 1.25+ | 高性能、静态类型、部署简单 |
| Chi | v5 | 轻量级、高性能路由框架 |
| GORM | v1.31 | Go 生态主流 ORM，功能完善 |
| MySQL | 8.0+ | 成熟稳定的关系型数据库 |
| Redis | 7+ | 高性能缓存、会话存储、限流 |
| JWT | golang-jwt v5 | 标准的 JWT 实现 |
| Viper | v1 | 多源配置合并 |
| ULID | - | 时间有序 ID，适合索引 |

### 前端技术栈

| 技术 | 版本 | 选型理由 |
|------|------|----------|
| Vue | 3.x | Composition API、性能优异 |
| TypeScript | - | 类型安全、开发体验好 |
| Element Plus | - | 成熟的 UI 组件库 |
| Pinia | - | Vue 3 官方推荐状态管理 |
| Vite | - | 快速的构建工具 |

### 部署技术栈

| 技术 | 说明 |
|------|------|
| Docker | 容器化部署 |
| Docker Compose | 编排 MySQL + Redis + App |
| Nginx | 反向代理、SSL 终结 |

---

## 优雅关闭

### 信号处理

```go
// 监听 SIGTERM/SIGINT
// 1. 停止接收新请求
// 2. 等待进行中请求完成（30s 超时）
// 3. 刷新审计日志缓冲区
// 4. 关闭数据库连接池
// 5. 关闭 Redis 连接
// 6. 退出进程
```

### 资源清理顺序

```
1. HTTP Server (停止接受新请求)
2. Audit Batch (刷新缓冲区)
3. Database (关闭连接池)
4. Redis (关闭连接)
5. Logger (同步日志)
```

---

## 测试策略

### 测试分层

| 层 | 测试类型 | 说明 |
|----|----------|------|
| Unit | Service 层业务逻辑 | Mock Repository 实现 |
| Integration | Repository 层数据访问 | 真实数据库操作 |
| Middleware | 中间件逻辑 | Mock HTTP 请求/响应 |
| E2E | API 端到端 | 完整请求链路测试 |

### 测试覆盖重点

- 权限检查逻辑
- 多租户隔离
- 事务边界
- 错误处理
- 并发安全

### Mock 实现

每个模块的 Repository 层提供 Mock 实现：

```go
// internal/modules/wiki/repository/mock_repository.go
type MockWikiRepository struct {
    spaces    []*model.WikiSpace
    nodes     []*model.WikiNode
    // ...
}
```