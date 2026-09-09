# MeteorX — 多租户 SaaS 平台

一套基于 Go + Chi + GORM + Redis 构建的 **MaaS (Multi-tenant as a Service)** 多租户 SaaS 平台，内置「平台超级管理员 / 租户管理员 / 普通用户」三级权限模型，完整的用户/租户/文件生命周期管理（软删除、回收站、恢复、批量操作、永久删除），配套的 Vue 3 + TypeScript 前端管理后台。

---

## ✨ 核心特性

| 模块 | 特性 |
| --- | --- |
| **多租户架构** | 基于 `tenant_id` 行级隔离；支持域名自动识别租户；独立管理员体系；统一 `tenantctx` 上下文管理 |
| **认证鉴权** | JWT（HS256）双 Token；注册 / 登录 / 登出 / 忘记密码 / 重置密码；Token 自动注入用户上下文；登出黑名单；邮件找回密码 |
| **RBAC 权限** | 角色 + 权限 + 角色-权限绑定 + 用户角色分配；支持作用域（`system` / `tenant` / `all`）；自动权限推导中间件 + 显式权限覆盖 |
| **统一错误/响应** | 集中式错误码 (`apperrors`)；`AppError` 结构体；统一成功/分页/错误响应信封；全局异常恢复 |
| **统一分页/排序** | 泛型 `PageRequest` / `PageResult[T]`；Sort 字段白名单防 SQL 注入；关键字多字段搜索 |
| **事务管理** | `TxManager` 统一事务入口；Service 层事务边界；Repository 透明 tx 感知 |
| **统一 ID 生成** | `pkg/idgen` 单一入口；ULID 时间有序；替代分散的 `ulid` / `uuid` 调用 |
| **用户管理** | 个人中心 / 租户内用户 / 跨租户用户三类独立 API；密码加密（bcrypt） |
| **租户管理** | 自主注册开户；后台手动建租户；启用/禁用；软删除/恢复；批量操作 |
| **租户独立配置** | 租户独立 Logo / 主题色 / 语言 / 时区 / 联系方式等设置，实时生效 |
| **文件管理** | 上传/下载/重命名/删除；回收站恢复+永久删除；MD5 去重；租户隔离；本地/云存储可扩展 |
| **套餐管理** | 套餐 CRUD；租户套餐分配；用量限制（用户数上限）；到期提醒 |
| **审计日志** | 自动记录所有请求；`auditctx` Service 层丰富（before/after）；批量异步写入；多维度筛选查询；可视化仪表盘；用户操作时间线；异常行为检测；详细统计（小时级/风险等级/用户活跃度） |
| **告警管理** | 基于审计日志的实时告警；支持风险等级/操作类型/特定用户触发；邮件/钉钉/企业微信/Webhook 通知；冷却机制防告警风暴 |
| **会话分析** | 用户会话追踪；操作时间线分析；会话统计（请求数/成功率/平均耗时）；IP 地理位置自动解析 |
| **Wiki 知识库** | 空间/节点/文档/版本/成员 五层模型；分栏 Markdown 编辑 + 实时预览；版本历史与回滚；附件与内嵌图片（带签名 URL 防盗链）；节点移动/排序/节点权限；成员协作；回收站与全局搜索；完整租户隔离 |
| **运营看板** | 平台运营数据总览：租户/用户/订阅/审计多维统计，实时掌握平台健康状况 |
| **通知公告** | 平台公告 CRUD + 发布/下架；支持全平台或指定租户范围定向推送 |
| **注销审批** | 租户注销申请 → 平台审批（通过/驳回）→ 到期自动执行注销的完整闭环 |
| **安全增强** | 登录失败锁定（显示剩余次数）；密码复杂度策略；接口限流（基于 Redis）；邮件找回密码 |
| **回收站** | 用户 / 租户 / 角色 / 文件 均支持软删除 → 回收站查询 → 恢复 → 永久删除的完整闭环 |
| **批量操作** | 批量删除 / 批量更新状态；幂等返回影响行数 |
| **优雅关闭** | HTTP Server 信号处理；Redis/DB/后台任务资源清理；30s 超时优雅关闭 |
| **工程化** | Viper 配置 + `.env` 覆盖；Chi 路由；GORM 自动迁移；ULID 主键；配置启动校验 |
| **前端管理后台** | Vue 3 + TypeScript + Element Plus；RBAC 动态菜单/按钮权限 |
| **测试覆盖** | 审计模块、安全模块单元测试；Mock 仓库实现；`tenantctx` 隔离测试 |

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
│   │   ├── auth/                # 认证：注册、登录、JWT、忘记密码/重置密码
│   │   ├── user/                # 用户：3 层接口 + 回收站 + 批量
│   │   ├── tenant/              # 租户：自助开户 + 后台管理 + 注销审批 + 独立配置
│   │   ├── rbac/                # 角色权限：角色、权限、绑定、自动权限推导
│   │   ├── file/                # 文件：上传、下载、回收站、存储抽象
│   │   ├── plan/                # 套餐：CRUD + 租户分配 + 用量检查
│   │   ├── audit/               # 审计日志：自动记录 + 统计 + 导出 + 可视化仪表盘
│   │   ├── wiki/                # ⭐ Wiki 知识库：空间/节点/文档/版本/成员
│   │   ├── dashboard/           # 运营看板：平台数据总览统计
│   │   └── notification/        # 通知公告：公告 CRUD + 发布/下架 + 定向推送
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
│   │   ├── tenantctx/           # ⭐ 租户上下文（From/FilterQuery/CanAccessTenant）
│   │   ├── auditctx/            # ⭐ 审计上下文（NewAction/WithBefore/WithAfter）
│   │   ├── jwt/                 # JWT 工具
│   │   ├── response/            # ⭐ 统一响应封装（Success/FailError/FailWithPagination）
│   │   └── validator/           # 参数校验
│   │
│   ├── config/                  # 配置结构体 + YAML + 启动校验
│   │   ├── config.go
│   │   ├── config.yaml
│   │   └── validator.go         # ⭐ 配置验证器（启动时校验必填项）
│   │
│   ├── middleware/              # HTTP 中间件
│   │   ├── request_id.go        # ⭐ 请求 ID + 全局异常恢复
│   │   ├── auth.go              # JWT 认证
│   │   ├── admin_middleware.go  # 超级管理员校验
│   │   ├── auto_permission_middleware.go  # 自动权限推导
│   │   ├── explicit_permission.go # ⭐ 显式权限覆盖
│   │   ├── permission_middleware.go       # 细粒度权限校验
│   │   ├── role_middleware.go   # 角色校验
│   │   ├── audit_middleware.go  # 审计日志记录
│   │   ├── audit_batch.go       # 审计日志批量写入
│   │   ├── rate_limit_middleware.go       # 接口限流
│   │   └── logger.go            # 请求日志
│   │
│   ├── pkg/                     # ⭐ 内部基础设施包（Framework Hardening）
│   │   ├── apperrors/           # ⭐ 集中式错误码 + AppError 结构体
│   │   │   ├── codes.go         # 错误码常量 + HTTP 状态映射
│   │   │   └── app_error.go     # AppError{Code,Message,StatusCode,RequestID,Details}
│   │   └── db/
│   │       └── transaction.go   # ⭐ TxManager 事务管理器（WithTx/GetTx/GetDB）
│   │
│   └── cache/                   # Redis 客户端封装
│
├── pkg/                         # 可复用基础库
│   ├── idgen/                   # ⭐ 统一 ID 生成（ULID，替代分散的 ulid/uuid）
│   │   └── idgen.go             # New/NewUUID/MustParse/Parse
│   ├── crypto/                  # bcrypt 密码加密
│   ├── logger/                  # 日志封装
│   ├── pagination/              # ⭐ 统一分页/排序/搜索（含 Sort Whitelist 防注入）
│   ├── security/               # 安全工具（锁定、密码策略、限流）
│   │   ├── login_lockout.go
│   │   ├── login_lockout_test.go
│   │   ├── password_policy.go
│   │   ├── password_policy_test.go
│   │   ├── rate_limiter.go
│   │   └── rate_limiter_test.go
│   ├── ulid/                    # （向后兼容）使用 pkg/idgen 替代
│   │   └── ulid.go
│   └── uuid/                    # （向后兼容）使用 pkg/idgen 替代
│       └── uuid.go
│
├── scripts/sql/
│   ├── init.sql                 # 建表脚本
│   └── seed.sql                 # 种子数据（默认管理员等）
│
├── docs/                        # 接口文档 + 架构文档
│   ├── apifox/                  # Apifox / OpenAPI
│   ├── architecture/            # ⭐ 架构设计文档
│   │   ├── tenant.md            # 多租户隔离架构
│   │   ├── permission.md        # RBAC 权限架构
│   │   ├── error.md             # 错误/响应规范
│   │   ├── database.md          # 数据库/事务架构
│   │   ├── audit.md             # 审计架构
│   │   ├── pagination.md        # 分页/排序/过滤架构
│   │   ├── config.md            # 配置/启动架构
│   │   └── idgen.md             # ID/ULID 统一架构
│   ├── auth-api.md              # 认证接口
│   ├── user-api.md              # 用户管理接口
│   ├── tenant-api.md            # 租户管理接口
│   ├── rbac-api.md              # RBAC 权限接口
│   ├── file-module-api.md       # 文件管理接口
│   ├── plan-api.md              # 套餐管理接口
│   ├── audit-api.md             # 审计日志接口
│   ├── wiki-api.md              # ⭐ Wiki 知识库接口
│   ├── dashboard-api.md         # 运营看板接口
│   ├── announcement-api.md      # 通知公告接口
│   └── FEATURE_UPGRADE.md       # 功能升级说明
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
| `POST` | `/auth/forgot-password` | 忘记密码（发送重置邮件） |
| `POST` | `/auth/reset-password` | 重置密码（通过邮件令牌） |

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
| `GET` | `/tenant-settings` | 获取当前租户独立配置 |
| `PUT` | `/tenant-settings` | 更新当前租户独立配置 |

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
| `DELETE` | `/admin/tenants/{id}/hard` | ☠️ 物理删除租户（彻底销毁，同步取消生效订阅） |
| `PUT` | `/admin/tenants/{id}/plan` | 为租户分配/变更套餐（等价于第 9 节 `/admin/tenants-plan/{id}`） |

**注销审批接口（需超级管理员）**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/admin/cancel-requests` | 注销申请列表（分页） |
| `PUT` | `/admin/cancel-requests/{id}/approve` | 审批通过（可指定生效时间） |
| `PUT` | `/admin/cancel-requests/{id}/reject` | 审批驳回 |

### 7. RBAC 角色与权限（需超级管理员）

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

### 10. 审计日志（需超级管理员）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/audit/stats` | Dashboard 统计 |
| `GET` | `/audit/dashboard` | 可视化仪表盘数据（趋势/模块分布/操作统计） |
| `GET` | `/audit/user-timeline` | 用户操作时间线（按天分组） |
| `GET` | `/audit/detailed-stats` | 详细统计（小时级/风险等级/用户活跃度） |
| `GET` | `/audit/anomalies` | 异常检测（高频失败等） |
| `GET` | `/audit/logs` | 日志列表（多维度筛选） |
| `GET` | `/audit/logs/export` | 导出为 CSV |
| `POST` | `/audit/logs` | 创建日志（内部） |
| `GET` | `/audit/logs/{id}` | 日志详情 |
| `DELETE` | `/audit/logs/cleanup` | 清理过期日志 |

### 10.1 告警管理

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/audit/alert-rules` | 创建告警规则 |
| `GET` | `/audit/alert-rules` | 获取所有告警规则 |
| `GET` | `/audit/alert-rules/{id}` | 告警规则详情 |
| `PUT` | `/audit/alert-rules/{id}` | 更新告警规则 |
| `DELETE` | `/audit/alert-rules/{id}` | 删除告警规则 |
| `GET` | `/audit/alerts` | 告警记录列表（分页） |
| `GET` | `/audit/alerts/stats` | 告警统计数据 |
| `GET` | `/audit/alerts/{id}` | 告警记录详情 |

### 10.2 会话分析

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/audit/sessions` | 会话摘要列表（分页） |
| `GET` | `/audit/sessions/{id}/logs` | 获取会话的所有日志 |

### 11. Wiki 知识库

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/wiki/stats` | Wiki 统计（空间数/文档数/节点数） |
| `GET` | `/wiki/spaces` | 空间列表（分页，支持 keyword） |
| `POST` | `/wiki/spaces` | 创建空间 |
| `GET` | `/wiki/spaces/{id}` | 空间详情 |
| `PUT` | `/wiki/spaces/{id}` | 更新空间 |
| `DELETE` | `/wiki/spaces/{id}` | 删除空间（进回收站） |
| `POST` | `/wiki/spaces/{spaceId}/nodes` | 创建节点（文件夹/文档节点） |
| `GET` | `/wiki/spaces/{spaceId}/nodes/tree` | 节点树（完整层级） |
| `GET` | `/wiki/spaces/{spaceId}/nodes/{id}` | 节点详情 |
| `PUT` | `/wiki/spaces/{spaceId}/nodes/{id}` | 更新节点 |
| `DELETE` | `/wiki/spaces/{spaceId}/nodes/{id}` | 删除节点（进回收站） |
| `POST` | `/wiki/spaces/{spaceId}/nodes/{id}/move` | 移动节点（可跨文件夹） |
| `PUT` | `/wiki/spaces/{spaceId}/nodes/{id}/sort` | 同级节点排序 |
| `GET` | `/wiki/spaces/{spaceId}/nodes/{id}/permissions` | 节点权限列表 |
| `POST` | `/wiki/spaces/{spaceId}/nodes/{id}/permissions` | 设置节点权限（view/edit/delete） |
| `DELETE` | `/wiki/spaces/{spaceId}/nodes/{id}/permissions/{userId}/{permission}` | 移除节点权限 |
| `POST` | `/wiki/documents/nodes/{nodeId}` | 创建文档（Markdown） |
| `GET` | `/wiki/documents/nodes/{nodeId}` | 获取文档内容 |
| `PUT` | `/wiki/documents/{id}` | 更新文档（自动创建新版本） |
| `DELETE` | `/wiki/documents/{id}` | 删除文档 |
| `POST` | `/wiki/documents/preview` | Markdown 实时预览（离线渲染+净化，返回 HTML） |
| `GET` | `/wiki/documents/{documentId}/revisions` | 版本历史列表 |
| `GET` | `/wiki/documents/{documentId}/revisions/{version}` | 指定版本详情 |
| `POST` | `/wiki/documents/{documentId}/revisions/{version}/restore` | 恢复到指定版本 |
| `POST` | `/wiki/documents/attachments` | 上传文档附件（登记 file_id） |
| `GET` | `/wiki/documents/{documentId}/attachments` | 文档附件列表 |
| `DELETE` | `/wiki/documents/attachments/{id}` | 删除文档附件 |
| `GET` | `/wiki/spaces/{spaceId}/members` | 空间成员列表 |
| `POST` | `/wiki/spaces/{spaceId}/members` | 添加成员（角色：owner/admin/editor/viewer） |
| `DELETE` | `/wiki/spaces/{spaceId}/members/{userId}` | 移除成员 |
| `GET` | `/wiki/search` | Wiki 全局搜索（标题+内容，返回高亮摘要） |
| `GET` | `/wiki/trash` | 回收站列表（按 space_id/item_type 过滤） |
| `POST` | `/wiki/trash/{id}/restore` | 从回收站恢复 |
| `DELETE` | `/wiki/trash/{id}` | 永久删除回收站项目 |

### 12. 运营看板（管理员）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/admin/dashboard/overview` | 平台运营数据总览（租户/用户/订阅/审计） |

### 13. 通知公告（管理员）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/admin/announcements` | 公告列表（分页） |
| `POST` | `/admin/announcements` | 创建公告 |
| `GET` | `/admin/announcements/{id}` | 公告详情 |
| `PUT` | `/admin/announcements/{id}` | 更新公告 |
| `PUT` | `/admin/announcements/{id}/status` | 发布 / 下架公告 |
| `DELETE` | `/admin/announcements/{id}` | 删除公告 |

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

所有 API 响应均包含 `request_id` 用于请求追踪。错误响应使用结构化错误码而非 HTTP 状态码。

**成功响应（单对象）**
```json
{
  "data": { ... },
  "request_id": "req_01H7K3N5P8..."
}
```

**成功响应（分页列表）**
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

**成功响应（无内容）**
```
HTTP 204 No Content
```

**错误响应**
```json
{
  "code": "WIKI_DOCUMENT_NOT_FOUND",
  "message": "Document not found",
  "request_id": "req_01H7K3N5P8...",
  "details": null
}
```

**请求参数**

- 查询列表：`?page=1&page_size=20&keyword=xxx&sort_by=created_at&sort_order=DESC`
- 批量操作 body：`{"ids": ["xxx"], "status": 0}`

**错误码分类**

| 分类 | 错误码 | HTTP 状态 |
|------|--------|-----------|
| 通用 | `INVALID_PARAM` | 400 |
| 通用 | `RESOURCE_NOT_FOUND` | 404 |
| 通用 | `PERMISSION_DENIED` | 403 |
| 通用 | `CONFLICT` | 409 |
| 通用 | `RATE_LIMITED` | 429 |
| 通用 | `INTERNAL_ERROR` | 500 |
| 认证 | `SESSION_EXPIRED` | 401 |
| 认证 | `AUTH_LOCKED` | 423 |
| 租户 | `TENANT_NOT_FOUND` | 404 |
| 租户 | `CROSS_TENANT_ACCESS_DENIED` | 403 |
| Wiki | `WIKI_SPACE_NOT_FOUND` | 404 |
| Wiki | `WIKI_DOCUMENT_NOT_FOUND` | 404 |
| Wiki | `WIKI_REVISION_NOT_FOUND` | 404 |

---

## ⚙️ 配置

配置文件位于 `internal/config/config.yaml`，可通过 `.env` 环境变量覆盖（前缀 `METEORX_`）。

| YAML 路径 | .env 变量 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `server.port` | `METEORX_APP_PORT` | 8081 | 服务端口 |
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
| `file.upload_url` | — | `http://localhost:8081/uploads` | 文件访问 URL 前缀 |
| `file.max_file_size` | — | `10485760` | 最大文件大小（10MB） |
| `file.storage_type` | — | `local` | 存储类型（local/oss/s3） |
| `log.level` | — | `info` | 日志等级 |
| `log.format` | — | `json` | `json` / `text` |
| `security.login_lockout.enabled` | — | `true` | 启用登录锁定 |
| `security.login_lockout.max_attempts` | — | `5` | 最大失败次数 |
| `security.login_lockout.lockout_duration` | — | `30m` | 锁定持续时间 |
| `email.enabled` | — | `false` | 是否启用邮件服务 |
| `email.host` | — | `smtp.example.com` | SMTP 服务器地址 |
| `email.port` | — | `587` | SMTP 端口 |
| `email.username` | — | `noreply@example.com` | SMTP 用户名 |
| `email.password` | — | `your-email-password` | SMTP 密码 |
| `email.from` | — | `noreply@example.com` | 发件人邮箱 |
| `email.from_name` | — | `MeteorX 平台` | 发件人名称 |
| `client.base_url` | — | `http://localhost:5173` | 前端 Base URL（用于密码重置链接） |

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
go test ./internal/modules/wiki/service/... -v
go test ./pkg/security/... -v

# 运行基准测试
go test -bench=. ./pkg/security/...
```

> 单元测试策略：Service 层通过仓库层提供的内存 Mock（如 `wiki/repository/mock_repository.go`）隔离 DB 依赖，
> 覆盖权限校验、业务校验（节点移动/回收站/搜索摘要等）与错误路径；`sqlmock` 仅用于验证事务边界（Begin/Commit）。

---

## 📚 接口文档

详细的模块接口文档见 `docs/` 目录：

| 文档 | 说明 |
|------|------|
| [认证 API](docs/api/auth-api.md) | 认证模块接口（注册/登录/登出/忘记密码/重置密码） |
| [用户 API](docs/api/user-api.md) | 用户管理接口（个人中心/租户用户/管理员） |
| [租户 API](docs/api/tenant-api.md) | 租户管理接口（注册/后台管理/注销审批） |
| [RBAC API](docs/api/rbac-api.md) | RBAC 权限接口（角色/权限/绑定） |
| [文件 API](docs/api/file-module-api.md) | 文件管理接口（上传/下载/回收站） |
| [套餐 API](docs/api/plan-api.md) | 套餐管理接口（CRUD/分配/用量） |
| [审计 API](docs/api/audit-api.md) | 审计日志接口（查询/导出/清理/仪表盘/告警管理/会话分析） |
| [Wiki API](docs/api/wiki-api.md) | ⭐ Wiki 知识库接口（空间/节点/文档/版本/成员） |
| [运营看板 API](docs/api/dashboard-api.md) | 运营看板接口（平台数据总览） |
| [公告 API](docs/api/announcement-api.md) | 通知公告接口（CRUD/发布/定向推送） |
| [功能升级记录](docs/FEATURE_UPGRADE.md) | 功能升级说明（看板/公告/注销审批） |

**架构设计文档**（`docs/architecture/`）：

| 文档 | 说明 |
|------|------|
| [tenant.md](docs/architecture/tenant.md) | 多租户隔离架构（tenantctx/FilterQuery） |
| [permission.md](docs/architecture/permission.md) | RBAC 权限架构（自动推导 + 显式覆盖） |
| [error.md](docs/architecture/error.md) | 错误/响应规范（AppError/统一响应信封） |
| [database.md](docs/architecture/database.md) | 数据库/事务架构（TxManager/ULID） |
| [audit.md](docs/architecture/audit.md) | 审计架构（自动/手动/批量异步） |
| [audit-enhanced.md](docs/architecture/audit-enhanced.md) | ⭐ 审计增强功能（IP地理位置/告警管理/会话分析） |
| [wiki.md](docs/architecture/wiki.md) | ⭐ Wiki 知识库架构（五层模型/节点树/版本/权限/XSS/回收站） |
| [pagination.md](docs/architecture/pagination.md) | 分页/排序/过滤架构（Sort Whitelist 防注入） |
| [config.md](docs/architecture/config.md) | 配置/启动架构（Viper/优雅关闭） |
| [idgen.md](docs/architecture/idgen.md) | ID/ULID 统一架构（pkg/idgen） |

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
| 忘记密码 | `/forgot-password` | 邮箱找回密码 |
| 重置密码 | `/reset-password` | 通过邮件令牌设置新密码 |
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
| 审计日志 | `/system/audit` | 日志查询/导出/可视化仪表盘 |
| 告警管理 | `/system/audit/alert` | 告警规则管理/告警记录查询/通知配置 |
| 会话分析 | `/system/audit/session` | 会话追踪/操作时间线/会话统计 |
| Wiki 知识库 | `/wiki` | Wiki 空间列表/创建/管理（含搜索与回收站入口） |
| Wiki 空间 | `/wiki/spaces/:id` | 节点树 + 分栏 Markdown 编辑/实时预览 + 版本历史恢复 + 附件 + 成员/节点权限弹窗 |
| Wiki 搜索 | `/wiki/search` | Wiki 全局搜索（标题/正文，高亮摘要） |
| Wiki 回收站 | `/wiki/trash` | 已删除空间/节点/文档 恢复与永久删除 |
| 租户设置 | `/tenant-settings` | 租户独立配置（Logo/主题色/语言等） |
| 通知公告 | `/system/announcement` | 公告 CRUD / 发布 / 下架 |
| 注销审批 | `/system/cancel-request` | 租户注销申请审批（通过/驳回） |

### 权限控制

- 动态菜单：根据用户权限码显示/隐藏侧边栏
- 动态按钮：根据权限码显示/隐藏操作按钮
- 登录时后端返回 `permissions` 数组，前端存入 Pinia Store

---

## 📄 License

MIT