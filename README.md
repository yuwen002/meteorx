# MeteorX — 多租户 SaaS 平台

一套基于 Go + Chi + GORM + Redis 构建的 **MaaS (Multi-tenant as a Service)** 多租户 SaaS 平台，内置「平台超级管理员 / 租户管理员 / 普通用户」三级权限模型，完整的用户/租户/文件生命周期管理（软删除、回收站、恢复、批量操作、永久删除），并在同一套租户隔离底座上叠加 **任务协作、数据导出、Wiki 知识库、套餐配额、审计告警、第三方登录、运营看板与可观测性**，配套 Vue 3 + TypeScript 前端管理后台。

---

## ✨ 核心特性

| 模块 | 特性 |
| --- | --- |
| **多租户架构** | 基于 `tenant_id` 行级隔离；支持域名自动识别租户；独立管理员体系；统一 `tenantctx` 上下文管理 |
| **认证鉴权** | JWT（HS256）双 Token + 每次签发唯一 `jti`；注册 / 登录 / 登出 / 忘记密码 / 重置密码 / 邮箱验证；Token 黑名单；API Token（`scopes` 最小权限 + 路径白名单） |
| **第三方登录** | OAuth2（GitHub / Google / 微信 / 飞书 / 钉钉 / QQ / Microsoft / 通用 OIDC）；provider **配置下沉数据库**、AES-256-GCM 加密凭据、免重启热生效 |
| **二次验证** | TOTP（RFC 6238，±1 时间窗）；绑定 / 启用 / 关闭；登录二段式验证码；错码计入账户锁定 |
| **RBAC 权限** | 角色 + 权限 + 角色-权限绑定 + 用户角色分配；支持作用域（`system` / `tenant` / `all`）；自动权限推导中间件 + 显式权限覆盖；权限变更即时失效缓存 |
| **统一错误/响应** | 集中式错误码 (`apperrors`)；`AppError` 结构体；统一成功/分页/错误响应信封；全局异常恢复 |
| **统一分页/排序** | 泛型 `PageRequest` / `PageResult[T]`；Sort 字段白名单防 SQL 注入；关键字多字段搜索 |
| **事务管理** | `TxManager` 统一事务入口；Service 层事务边界；Repository 透明 tx 感知 |
| **统一 ID 生成** | `pkg/idgen` 单一入口；ULID 时间有序；替代分散的 `ulid` / `uuid` 调用 |
| **用户管理** | 个人中心 / 租户内用户 / 跨租户用户三类独立 API；bcrypt 密码；邮箱验证状态 / 手机号 / 头像；角色、排序、注册时间区间筛选；**CSV / XLSX 批量导入**（表头别名、全量事务、服务端生成初始密码、逐行错误定位） |
| **租户管理** | 自主注册开户；后台手动建租户；启用/禁用；软删除/恢复；批量操作；物理销毁；登录侧软着陆拦截 |
| **租户独立配置** | 租户独立 Logo / 主题色 / 语言 / 时区 / 联系方式等设置，实时生效（服务端白标） |
| **文件管理** | 上传/下载/重命名/删除；回收站恢复+永久删除；MD5 去重；租户隔离；**local / OSS / S3 三驱动 + 存储注册表**（按记录 `storage_type` 路由，切换后端后存量文件仍可用） |
| **套餐与配额** | 套餐 CRUD；租户套餐分配；用户数 / 存储 / API 速率 / Wiki 空间四类配额（`-1` 不限）；超限拦截 403 / 429 / 507 / 402；到期禁用与临期提醒 |
| **任务管理** | 个人待办 + 租户协作（`visibility` 区分）；优先级 / 标签 / 截止日 / 重复周期（月末钳制）；批量完成/改状态/指派/删除；回收站；完成率与平均处理耗时；到期提醒（任务级提前量 + 冷却去重，消息链接带任务 ID 直达）；**AI 解析 / 建议 / 扩写 / 摘要** |
| **数据导出** | 异步任务 + 下载中心；四类资源 × CSV / XLSX / JSON / PDF；执行租约 + 心跳 + 僵尸回收；租户队列准入；过期文件清理 |
| **审计日志** | 自动记录所有请求；`auditctx` Service 层丰富（before/after）；批量异步写入；多维度筛选查询；可视化仪表盘；用户操作时间线（支持 `?user_id=` 定位）；异常行为检测；详细统计（小时级/风险等级/用户活跃度）；支持 `?log_id=` 从告警详情直达某条日志记录 |
| **告警管理** | 基于审计日志的实时告警；评估链在日志落库后触发；支持风险等级/操作类型/特定用户触发；按渠道类型路由通知；冷却机制防告警风暴；告警详情可下钻到关联的审计日志、也可直达触发用户的时间线 |
| **会话分析** | 用户会话追踪；操作时间线分析；会话统计（请求数/成功率/平均耗时）；IP 地理位置自动解析 |
| **Wiki 知识库** | 空间/节点/文档/版本/成员 五层模型；分栏 Markdown 编辑 + 实时预览；版本历史与回滚；附件与内嵌图片（带签名 URL 防盗链）；节点移动/排序/节点权限；成员协作；回收站与全局搜索（可选 MeiliSearch）；**发布工作流（草稿→待审核→已发布→归档）**；审核评论；订阅/审核/编辑通知；完整租户隔离 |
| **运营看板** | 平台总览 + 增长趋势（按业务时区分桶补零）+ 租户健康度评分 + 慢查询看板 |
| **通知公告** | 平台公告 CRUD + 发布/下架；全平台或指定租户范围定向推送（WS 按租户过滤，不跨租户泄露） |
| **消息通知** | 多渠道 Manager（邮件 / Webhook / WebSocket）；覆盖公告发布、审计告警、任务指派与到期、订阅到期、注销审批结果 |
| **站内消息中心** | 收件箱持久化（`user_messages` 按「人 × 消息」写行）+ 离线补推 + 已读/删除；WS `unread_count` 实时角标；同一事件按「业务事件 + 自然日」去重（`dedupe_key` 唯一索引 + `ON CONFLICT DO NOTHING`）；审计告警按规则邮箱反查账号落库（`audit` 分类）；按分类关闭接收（`user_message_preferences`，落库前过滤，安全告警强制接收）；平台级公告不落库、租户广播 500 人封顶；保留期清理（默认 90 天）；批量已读 / 批量删除（单次 100 条封顶，缺一条不报错）；「全部已读」跟随当前页签（本分类全部已读）；告警消息链接带告警 ID、任务提醒带任务 ID、订阅临期带租户 ID，从收件箱点开直接定位到具体记录 |
| **注销审批** | 租户注销申请 → 平台审批（通过/驳回）→ 到期自动执行注销的完整闭环 |
| **邮箱验证** | 用户邮箱验证链接发送（Redis 令牌 24h 有效）；一次性令牌；验证状态标记；防止信息泄露的静默响应 |
| **成员邀请** | 管理员邀请新成员加入租户；邮件通知含邀请链接；被邀请人自助注册并自动分配角色；状态管理（pending/accepted/cancelled/expired）；7 天有效期；**批量创建（单次上限 50）**；重发/取消/删除 |
| **安全增强** | 登录失败锁定（显示剩余次数）；密码复杂度策略；接口限流（基于 Redis）；安全响应头 + CSP；凭据加密；release 模式 Redis 不可用直接拒绝启动 |
| **回收站** | 用户 / 租户 / 角色 / 文件 / 任务 / Wiki 均支持软删除 → 回收站查询 → 恢复 → 永久删除的完整闭环（Wiki 按条目事务化过期清理，支持 Redis 分布式锁） |
| **批量操作** | 批量删除 / 批量更新状态 / 批量指派 / 批量绑定权限；幂等返回影响行数 |
| **可观测性** | OpenTelemetry 全链路追踪（含 GORM SQL span）+ Prometheus 指标 + 慢查询看板 + `/health/ready` 运行时与连接池读数 |
| **优雅关闭** | HTTP Server 信号处理；Redis/DB/后台任务资源清理；30s 超时优雅关闭；导出/审计在关闭路径仍能落终态 |
| **工程化** | Viper 配置 + `.env` 覆盖；Chi 路由；GORM 自动迁移 + 脏列只读巡检；ULID 主键；配置启动校验 |
| **前端管理后台** | Vue 3 + TypeScript + Element Plus；RBAC 动态菜单/按钮权限；暗色主题；中英双语；白标品牌 |
| **测试与 CI** | 183 个 Go 测试文件（含进程内集成 + MySQL 方言）+ 21 个前端测试文件 + 9 条浏览器 E2E；10 个 CI job；覆盖率门禁 |

---

## 🧱 技术栈

### 后端

- **语言**：Go 1.25
- **Web 框架**：[go-chi/chi v5](https://github.com/go-chi/chi)
- **ORM**：[GORM v1.31](https://gorm.io/)
- **数据库**：MySQL 8.0+
- **缓存**：Redis 7+（go-redis v9）
- **认证**：JWT（golang-jwt v5）+ TOTP（RFC 6238）
- **校验**：go-playground/validator v10
- **ID 生成**：ULID（按时间有序、适合数据库索引）
- **可观测性**：OpenTelemetry（trace）+ Prometheus（metrics）+ 进程内慢查询采集
- **搜索**：MeiliSearch（可选，未配置时回落数据库 LIKE）
- **AI**：OpenAI 兼容接口（可选叠加层，默认关闭）
- **配置**：Viper + YAML + `.env` 环境变量覆盖

### 前端

- **框架**：Vue 3 + TypeScript
- **UI**：Element Plus（含官方暗色 css-vars）
- **状态管理**：Pinia
- **路由**：Vue Router
- **HTTP**：Axios
- **构建**：Vite
- **测试**：Vitest + Playwright（E2E）

### 部署

- Docker / Docker Compose
- Nginx 反向代理（安全响应头 + CSP）
- Prometheus / Grafana / OTel Collector / Jaeger（可选）
- MinIO（本地 S3 兼容联调）

---

## 📁 项目结构

```
meteorx/
├── cmd/
│   ├── server/
│   │   └── main.go              # 程序入口（含 migrate 子命令）
│   └── migrate/
│       └── main.go              # 独立迁移 CLI（含 -audit-columns 脏列只读巡检）
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
│   │   ├── auth/                # 认证：注册、登录、JWT、2FA、密码重置、邮箱验证、API Token
│   │   ├── oauth/               # 第三方登录：Provider 实现 + 运行时配置（DB 热更新 + 凭据加密）
│   │   ├── user/                # 用户：3 层接口 + 回收站 + 批量 + 批量导入 + 邮箱验证/手机号/头像
│   │   │   └── importer/        # CSV/XLSX 解析：表头别名 + 行数/文件大小上限
│   │   ├── tenant/              # 租户：自助开户 + 后台管理 + 注销审批 + 独立配置
│   │   ├── rbac/                # 角色权限：角色、权限、绑定、自动权限推导、权限缓存
│   │   ├── file/                # 文件：上传、下载、回收站、存储抽象（local/OSS/S3 + 注册表）
│   │   ├── plan/                # 套餐：CRUD + 租户分配 + 四类配额 + 到期任务
│   │   ├── task/                # 任务：个人待办 + 租户协作 + 提醒 + 回收站 + AI 辅助
│   │   ├── audit/               # 审计日志：自动记录 + 统计 + 告警链 + 可视化仪表盘
│   │   ├── export/              # 数据导出：异步入队 + worker 池 + 多格式 + 下载中心
│   │   ├── wiki/                # ⭐ Wiki 知识库：空间/节点/文档/版本/成员/发布流
│   │   ├── dashboard/           # 运营看板：总览 + 趋势 + 健康度 + 慢查询
│   │   ├── notification/        # 通知公告：公告 CRUD + 发布/下架 + 定向推送
│   │   ├── message/             # 站内消息：收件箱 + 离线补推 + 已读 + 未读角标 + 接收偏好
│   │   └── invitation/          # 成员邀请：创建/批量/邮件/接受注册/状态管理
│   │
│   ├── middleware/              # HTTP 中间件
│   │   ├── request_id.go        # 请求 ID + 全局异常恢复
│   │   ├── auth.go              # JWT 认证
│   │   ├── admin_middleware.go  # 超级管理员校验
│   │   ├── auto_permission_middleware.go  # 自动权限推导
│   │   ├── explicit_permission.go         # 显式权限覆盖
│   │   ├── permission_middleware.go       # 细粒度权限校验
│   │   ├── permission_cache.go  # 权限缓存（Redis，5 分钟）
│   │   ├── role_middleware.go   # 角色校验
│   │   ├── audit_middleware.go  # 审计日志记录
│   │   ├── audit_batch.go       # 审计日志批量写入
│   │   ├── rate_limit_middleware.go       # 接口限流
│   │   ├── security_headers.go  # 安全响应头（nosniff / X-Frame-Options / HSTS）
│   │   ├── tracing.go           # OpenTelemetry 追踪
│   │   ├── prometheus.go        # Prometheus 指标
│   │   └── logger.go            # 请求日志
│   │
│   ├── ai/                      # ⭐ 通用 AI 抽象层（Provider + OpenAI 兼容 Client，默认关闭）
│   ├── search/                  # 全文检索（MeiliSearch 引擎 + Wiki 索引器，未配置降级 LIKE）
│   ├── dbmonitor/               # ⭐ 慢查询采集（GORM logger 包装 + 内存环形缓冲）
│   ├── notify/                  # 多渠道通知（Manager + 邮件 / Webhook / WebSocket）
│   ├── ws/                      # WebSocket Hub（按用户 / 租户投递）
│   ├── otel/                    # OpenTelemetry 初始化（stdout / OTLP）
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
│   ├── pkg/                     # ⭐ 内部基础设施包（Framework Hardening）
│   │   ├── apperrors/           # ⭐ 集中式错误码 + AppError 结构体
│   │   │   ├── codes.go         # 错误码常量 + HTTP 状态映射
│   │   │   └── app_error.go     # AppError{Code,Message,StatusCode,RequestID,Details}
│   │   ├── emailer/             # ⭐ 邮件发送（SMTP + 模板：密码重置/邮箱验证/邀请通知）
│   │   └── db/
│   │       └── transaction.go   # ⭐ TxManager 事务管理器（WithTx/GetTx/GetDB）
│   │
│   ├── integration/             # 进程内集成测试（testkit + sqlite + miniredis 全量路由）
│   ├── testkit/                 # 集成测试装配（真实路由 + 迁移 + 种子）
│   │
│   └── cache/                   # Redis 客户端封装
│
├── pkg/                         # 可复用基础库
│   ├── idgen/                   # ⭐ 统一 ID 生成（ULID，替代分散的 ulid/uuid）
│   │   └── idgen.go             # New/NewUUID/MustParse/Parse
│   ├── crypto/                  # bcrypt 密码加密
│   ├── logger/                  # 日志封装
│   ├── pagination/              # ⭐ 统一分页/排序/搜索（含 Sort Whitelist 防注入）
│   ├── security/               # 安全工具（锁定、密码策略、限流、随机密码）
│   │   ├── login_lockout.go
│   │   ├── login_lockout_test.go
│   │   ├── password_gen.go      # 随机初始密码生成（合规字符集 + 必含各字符类）
│   │   ├── password_gen_test.go
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
│   │   ├── idgen.md             # ID/ULID 统一架构
│   │   ├── wiki.md              # Wiki 知识库架构
│   │   ├── oauth.md             # 第三方登录架构（配置下沉 + 凭据加密）
│   │   ├── task.md              # 任务管理架构（访问控制 / 周期 / 提醒）
│   │   ├── export.md            # 数据导出架构（租约 / 多格式 / 清理）
│   │   ├── dashboard.md         # 运营看板架构（趋势 / 健康度 / 慢查询）
│   │   └── message.md           # 站内消息架构（按人写行 / 写放大封顶 / 已读幂等 / 接收偏好）
│   ├── api/                     # ⭐ 各模块接口文档
│   │   ├── auth-api.md          # 认证接口（登录 / 2FA / API Token）
│   │   ├── user-api.md          # 用户管理接口
│   │   ├── tenant-api.md        # 租户管理接口
│   │   ├── rbac-api.md          # RBAC 权限接口
│   │   ├── file-module-api.md   # 文件管理接口
│   │   ├── plan-api.md          # 套餐管理接口
│   │   ├── task-api.md          # 任务管理接口（含 AI 辅助）
│   │   ├── export-api.md        # 数据导出接口
│   │   ├── message-api.md       # 站内消息（收件箱 / 未读数 / 已读 / 偏好）
│   │   ├── oauth-provider-api.md # 第三方登录提供商管理接口
│   │   ├── audit-api.md         # 审计日志接口
│   │   ├── wiki-api.md          # ⭐ Wiki 知识库接口
│   │   ├── dashboard-api.md     # 运营看板接口
│   │   ├── announcement-api.md  # 通知公告接口
│   │   └── invitation-api.md    # 成员邀请接口
│   ├── development/             # 开发手册
│   │   └── testing.md           # 测试指南与覆盖率基线
│   ├── FEATURE_UPGRADE.md       # 功能升级说明
│   ├── DESIGN.md                # 整体设计说明
│   ├── project-structure.md     # 项目结构说明
│   └── openapi-sync.md          # OpenAPI 同步说明
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
| `POST` | `/auth/login` | 登录，返回 JWT + 用户信息 + 权限码列表；账号启用 2FA 时需带 `otp`，否则返回 401 + `require_2fa` |
| `POST` | `/auth/logout` | 登出（Token 加入黑名单） |
| `POST` | `/auth/forgot-password` | 忘记密码（发送重置邮件） |
| `POST` | `/auth/reset-password` | 重置密码（通过邮件令牌） |
| `POST` | `/auth/email/send-verification` | 发送邮箱验证链接（需登录，只能验证自己的邮箱） |
| `POST` | `/auth/email/verify` | 验证邮箱（通过令牌，需登录） |
| `POST` | `/auth/2fa/setup` | 生成 TOTP 密钥与 `otpauth://` 二维码 URI（密钥暂存 Redis，不落库） |
| `POST` | `/auth/2fa/enable` | 提交验证器口令，确认启用 2FA |
| `POST` | `/auth/2fa/disable` | 提交当前口令，关闭 2FA |
| `GET` | `/auth/tokens` | API Token 列表（脱敏） |
| `POST` | `/auth/tokens` | 创建 API Token（`scopes` + 可选有效期） |
| `POST` | `/auth/tokens/revoke` | 撤销 API Token |

> 2FA 说明：基于 RFC 6238（±1 时间窗容错），仅启用后登录才需第二段验证码；错码同样计入登录锁定。
> API Token 说明：`scopes` 最小权限 + 路径白名单，最长有效期由 `auth.api_token_max_ttl` 限制；明文仅在创建时返回一次。

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
| `POST` | `/wiki/documents/{id}/submit-review` | 提交审核（draft → pending_review） |
| `POST` | `/wiki/documents/{id}/approve` | 审核通过（pending_review → published） |
| `POST` | `/wiki/documents/{id}/reject` | 驳回（pending_review → rejected） |
| `POST` | `/wiki/documents/{id}/publish` | 直接发布（Owner/Admin 跳过审核） |
| `POST` | `/wiki/documents/{id}/unpublish` | 取消发布（published → draft） |
| `POST` | `/wiki/documents/{id}/archive` | 归档（published → archived） |
| `GET` | `/wiki/documents/{id}/review-comments` | 审核评论列表 |
| `GET` | `/wiki/pending-reviews` | 待审核文档列表（分页） |

### 12. 运营看板（平台超管）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/admin/dashboard/overview` | 平台运营数据总览（租户/用户/订阅/审计） |
| `GET` | `/admin/dashboard/trend?days=N` | 增长趋势（按业务时区分桶补零，默认 30，上限 365） |
| `GET` | `/admin/dashboard/tenants-health` | 租户健康度列表（分级筛选 + 分页） |
| `GET` | `/admin/dashboard/tenants-health/{id}` | 单租户健康度详情（评分明细） |
| `GET` | `/admin/dashboard/slow-queries` | 慢查询快照（进程内环形缓冲） |
| `POST` | `/admin/dashboard/slow-queries/reset` | 清空慢查询缓冲 |

### 13. 通知公告

**管理员接口**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/admin/announcements` | 公告列表（分页） |
| `POST` | `/admin/announcements` | 创建公告 |
| `GET` | `/admin/announcements/{id}` | 公告详情 |
| `PUT` | `/admin/announcements/{id}` | 更新公告 |
| `PUT` | `/admin/announcements/{id}/status` | 发布 / 下架公告 |
| `DELETE` | `/admin/announcements/{id}` | 删除公告 |

**租户接口**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/announcements` | 当前租户可见的已发布公告 |

### 14. 成员邀请

**认证接口（需登录 + 权限）**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/invitations` | 当前租户邀请列表 |
| `POST` | `/invitations` | 创建邀请（邀请新成员） |
| `PUT` | `/invitations/{id}/cancel` | 取消邀请 |
| `PUT` | `/invitations/{id}/resend` | 重发邀请邮件 |
| `DELETE` | `/invitations/{id}/delete` | 删除邀请记录 |

**公开接口（无需登录）**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/invitations/info?token=xxx` | 通过令牌查询邀请信息 |
| `POST` | `/invitations/accept` | 接受邀请并注册 |

### 15. 任务管理（需登录）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/tasks` | 任务列表（个人 / 租户视图、状态 / 优先级 / 标签 / 负责人筛选） |
| `POST` | `/tasks` | 创建任务（`visibility` 区分 personal / tenant） |
| `GET` | `/tasks/stats` | 完成率、平均处理耗时与分组统计 |
| `GET` | `/tasks/deleted` | 回收站列表 |
| `GET` | `/tasks/{id}` | 任务详情（含创建人 / 负责人展示名） |
| `PUT` | `/tasks/{id}` | 更新任务 |
| `DELETE` | `/tasks/{id}` | 删除（进回收站） |
| `PUT` | `/tasks/{id}/complete` | 完成任务（首次完成时按重复周期生成下一条） |
| `PUT` | `/tasks/{id}/reopen` | 重开任务（清空 `completed_at`，保留 `started_at`） |
| `PUT` | `/tasks/{id}/restore` | ♻️ 从回收站恢复 |
| `DELETE` | `/tasks/{id}/permanent` | ☠️ 永久删除 |
| `POST` | `/tasks/batch/complete` | 批量完成 |
| `POST` | `/tasks/batch/status` | 批量改状态 |
| `POST` | `/tasks/batch/assign` | 批量指派负责人 |
| `POST` | `/tasks/batch/delete` | 批量删除 |

**AI 辅助（可选叠加层）**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/tasks/ai/capabilities` | 查询 AI 能力是否可用 |
| `POST` | `/tasks/ai/parse` | 自然语言解析为任务草稿 |
| `POST` | `/tasks/ai/suggest` | 任务建议 |
| `POST` | `/tasks/ai/expand` | 内容扩写 |
| `POST` | `/tasks/ai/summary` | 任务摘要 |

> AI 未启用或未配置凭据时上述端点返回 503，不阻断任务主流程；全部端点只读不写库。

### 16. 数据导出（需登录）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/exports` | 导出任务列表（下载中心） |
| `POST` | `/exports` | 创建导出任务（资源 × 格式），单租户未完结任务达上限返回 429 |
| `GET` | `/exports/{id}` | 任务详情与进度 |
| `GET` | `/exports/{id}/download` | 下载产物（仅 completed 可下载） |
| `DELETE` | `/exports/{id}` | 删除任务与产物文件 |

> 资源：`user` / `task` / `audit_log` / `wiki_document`；格式：`csv` / `xlsx` / `json` / `pdf`。

### 17. 第三方登录（OAuth2）

**公开接口（免登录）**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/auth/oauth/providers` | 已启用 provider 列表（登录页按钮数据源） |
| `GET` | `/auth/oauth/{provider}/redirect` | 取授权跳转地址（同时写入一次性 state） |
| `POST` | `/auth/oauth/callback` | 授权码回调：登录 / 自动注册 / 待补邮箱 |
| `GET` | `/auth/oauth/tenants` | 回调后查询该账号关联的租户列表 |
| `POST` | `/auth/oauth/token/refresh` | 刷新 OAuth 临时令牌 |
| `POST` | `/auth/oauth/complete-email` | 第三方未返回邮箱时补全 |

**需登录（账号绑定）**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/auth/oauth/accounts` | 当前用户已绑定账号 |
| `POST` | `/auth/oauth/bind` | 绑定新账号 |
| `POST` | `/auth/oauth/unbind` | 解绑 |

**平台超管（配置管理，权限码 `admin:oauth_provider:*`）**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` / `POST` | `/admin/oauth-providers` | 配置列表 / 新建配置 |
| `GET` / `PUT` / `DELETE` | `/admin/oauth-providers/{code}` | 详情 / 更新 / 删除 |
| `PUT` | `/admin/oauth-providers/{code}/status` | 启用 / 禁用 |

> 支持的 `kind`：`google` / `github` / `wechat` / `feishu` / `dingtalk` / `qq` / `microsoft` / `oidc`。
> 配置以 DB 为准（`config.yaml` 仅作为启动种子），CRUD 后立即热生效，多副本 30s 内收敛。

### 18. 站内消息（需登录）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/messages` | 我的收件箱（分页；可按分类 / 仅未读 / 标题关键字筛选） |
| `GET` | `/messages/unread-count` | 未读总数与分类明细（顶栏铃铛 + 页签角标） |
| `PUT` | `/messages/{id}/read` | 标记单条已读（幂等；他人消息返回 404） |
| `PUT` | `/messages/read-all` | 全部已读，`category` 可选（只清该分类） |
| `PUT` | `/messages/batch/read` | 批量标记已读（`{"ids": [...]}`，1~100 条，返回 `updated`） |
| `GET` | `/messages/preferences` | 我的分类接收偏好（全部分类 + `mandatory` 标记） |
| `PUT` | `/messages/preferences` | 修改单个分类的接收开关（`audit` 不可关闭，返回 400） |
| `DELETE` | `/messages/{id}` | 删除单条消息 |
| `DELETE` | `/messages/batch/delete` | 批量删除（`{"ids": [...]}`，1~100 条，返回 `deleted`） |

> 按「收件人」鉴权（不挂 RBAC 权限码），查询与写操作均限定 `user_id = 当前用户`；
> 批量接口对已读 / 已删除 / 他人的 ID 不报 404，只从 `updated` / `deleted` 中扣除；
> 在线时额外收到 WS `unread_count` 角标推送，离线用户登录后主动拉取补齐。
> 偏好是「关闭即不落库」（在 `Deliver` 内批量过滤）而非读时隐藏，因此角标与列表始终一致；`audit` 分类强制接收。

### 19. 运维端点（无业务前缀）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/health` | 存活探针（不查依赖） |
| `GET` | `/health/ready` | 就绪探针：数据库 / Redis / 运行时内存 / 连接池读数 |
| `GET` | `/metrics` | Prometheus 指标（需在采集侧限流或网络隔离） |
| `GET` | `/ws` | WebSocket 实时推送（按用户 / 租户投递，需 Token） |
| `POST` | `/csp-report` | 浏览器 CSP 违规上报接收 |

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

```bash
# 只跑迁移（不启动 HTTP 服务）
go run ./cmd/migrate

# 巡检审计表脏列（只读，不修改结构）
go run ./cmd/migrate -audit-columns

# 探活 / 就绪
curl http://127.0.0.1:8081/health
curl http://127.0.0.1:8081/health/ready
```

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
| `server.test_bypass` | — | `false` | 调试后门（固定 Token 免登录），`release` 模式无效，生产必须关闭 |
| `server.trusted_proxies` | — | 空 | 可信反向代理 CIDR，多跳部署必配（防 XFF 伪造） |
| `database.host` | `METEORX_DB_HOST` | `127.0.0.1` | MySQL 地址 |
| `database.port` | `METEORX_DB_PORT` | `3306` | MySQL 端口 |
| `database.user` | `METEORX_DB_USER` | `root` | MySQL 用户 |
| `database.password` | `METEORX_DB_PASSWORD` | `123456` | MySQL 密码 |
| `database.name` | `METEORX_DB_NAME` | `meteorx` | 数据库名 |
| `database.debug` | `METEORX_DB_DEBUG` | `true` | 是否打印 SQL |
| `database.slow_query_ms` | — | `500` | 慢查询阈值（毫秒）；`0` 回落默认，负值关闭采集 |
| `database.timezone` | — | `Asia/Shanghai` | 业务时区：启动时强制 `time.Local`，隔离容器 TZ |
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
| `client.base_url` | — | `http://localhost:5173` | 前端 Base URL（用于密码重置/邮箱验证/邀请链接/回跳） |
| `auth.api_token_max_ttl` | `METEORX_AUTH_API_TOKEN_MAX_TTL` | `2160h` | API Token 最长有效期（90 天） |
| `oauth.secret_key` | `METEORX_OAUTH_SECRET_KEY` | 空 | Provider 凭据 AES-256-GCM 主密钥；留空时保存含 secret 的配置直接 422 |
| `oauth.state_ttl` | `METEORX_OAUTH_STATE_TTL` | `10m` | OAuth CSRF state 有效期 |
| `oauth.refresh_token_ttl` | `METEORX_OAUTH_REFRESH_TOKEN_TTL` | `168h` | OAuth 临时令牌刷新有效期 |
| `oauth.providers` | — | 空 | 仅作为启动种子（只种填了 `client_id` 的项），之后以 DB 为准 |
| `task_reminder.horizon` | — | `24h` | 任务提醒提前量（截止日进入该窗口即提醒） |
| `task_reminder.interval` | — | `30m` | 提醒扫描间隔 |
| `task_reminder.overdue_cooldown` | — | `24h` | 逾期重复提醒的最小间隔 |
| `task_reminder.batch_limit` | — | `200` | 单轮最多发送的提醒数 |
| `export.dir` | — | `./data/exports` | 导出产物目录 |
| `export.retention_days` | — | `7` | 产物文件保留天数 |
| `export.worker_count` | — | `3` | 导出 worker 池大小 |
| `export.batch_size` | — | `1000` | 流式读库 / 写出分批行数 |
| `export.max_active_jobs` | — | `10` | 单租户未完结导出任务上限（超出 429） |
| `ws.enabled` | — | `true` | 是否启用 WebSocket 实时推送 |
| `ws.max_conn_per_user` | — | `10` | 单用户连接上限（配置保留项，Hub 当前未强制拒绝） |
| `message.retention_days` | — | `90` | 站内消息保留天数；未配置(0)回落 90，负值关闭清理 |
| `notify.webhook.enabled` | — | `false` | 启用 Webhook 通知 |
| `notify.webhook.kind` | — | `generic` | `generic` / `dingtalk` / `wechat` / `feishu` |
| `notify.webhook.url` | — | 空 | Webhook 地址（机器人回调） |
| `notify.webhook.secret` | — | 空 | 签名密钥（部分渠道需要） |
| `notify.webhook.on_events` | — | `alert`、`announcement` | 触发通知的事件列表 |
| `ai.enabled` | — | `false` | AI 叠加层总开关（关闭时相关端点 503、前端按钮禁用） |
| `ai.base_url` | — | 空 | OpenAI 兼容端点基址（留空默认官方地址） |
| `ai.api_key` | `METEORX_AI_API_KEY` | 空 | 访问密钥（建议经环境变量注入） |
| `ai.model` | — | 空 | 默认模型（留空默认 `gpt-4o-mini`） |
| `ai.timeout` | — | `20s` | 上游调用超时 |
| `search.provider` | — | `none` | `meilisearch` / `none`（`none` 降级为数据库 LIKE） |
| `search.host` | — | 空 | MeiliSearch 地址 |
| `otel.enabled` | — | `true` | 是否启用链路追踪 |
| `otel.exporter` | — | `stdout` | `stdout`（开发）/ `otlp`（生产） |
| `otel.endpoint` | — | `localhost:4317` | OTLP gRPC 端点 |
| `otel.sample_rate` | — | `1.0` | 采样率（生产建议 0.1） |

---

## 🔐 安全建议

- **生产环境必须替换 `jwt.secret`** 为 32 位以上随机字符串
- 数据库密码、Redis 密码不要使用默认值
- 建议通过环境变量 / 密钥管理服务注入敏感配置
- 部署时将 `server.mode` 改为 `release`，`database.debug` 改为 `false`，`server.test_bypass` 必须为 `false`
- 建议启用 HTTPS 反向代理（Nginx / Traefik）
- **登录锁定**：连续失败 5 次后账号锁定 30 分钟；2FA 错码同样计入失败次数
- **二次验证**：建议管理员账号开启 TOTP；密钥未验证前只存 Redis（15 分钟），不落库
- **密码策略**：默认要求 8–32 位，包含大小写字母、数字与特殊字符
- **接口限流**：每个 IP 每分钟最多 100 个请求（突发 10）；`release` 模式下 Redis 不可用**直接拒绝启动**，避免限流/锁定静默失效
- **凭据加密**：OAuth provider 的 `client_secret` 以 AES-256-GCM 加密落库，读接口只返回掩码；未配置 `oauth.secret_key` 时拒绝保存
- **文件访问**：`upload_url` 返回的链接附带短时效 HMAC 签名（默认 30 分钟），`file.sign_key` 与 `jwt.secret` 独立，支持双密钥平滑轮换
- **审计日志**：异步批量写入，支持 CSV 导出
- **安全响应头 / CSP**：由后端与 Nginx 双重下发，前端违规上报到 `/csp-report`
- **可观测性端点**：`/metrics` 与 `/health/ready` 不应直接暴露公网，建议在网关层限制或网络隔离
- **邮箱验证**：令牌一次性使用，24h 过期，验证后立即失效；邮箱不存在时静默返回成功（防信息泄露）
- **成员邀请**：邀请令牌 7 天有效期；同租户内同邮箱不重复邀请；接受后自动标记邮箱已验证
- **优雅关闭**：支持 SIGTERM/SIGINT 信号，确保资源清理完成（导出/审计在关闭路径仍能落终态）

---

## 🧪 测试

### 一键全量（推荐）

```powershell
# 后端 vet + build + test（含覆盖率）→ 前端 lint + 类型检查 + 覆盖 + 构建
powershell -File scripts/test-all.ps1

# 额外跑浏览器 E2E（需后端已在 127.0.0.1:8081、MySQL、Redis）
powershell -File scripts/test-all.ps1 -WithE2E
```

| 层级 | 命令 | 内容 |
| --- | --- | --- |
| 后端 | `scripts/test.ps1` | `go vet` + `go build` + `go test -count=1`，写 `coverage.out` |
| 后端 | `scripts/test.sh` | Linux/macOS 与 CI 入口（CI 额外加 `-race`） |
| 前端 | `cd web-admin && npm run test:coverage` | Vitest（21 个测试文件） |
| 前端 | `cd web-admin && npm run lint && npx vue-tsc --noEmit` | ESLint + 类型检查 |
| E2E | `cd e2e && npx playwright test` | 真实 Chrome + 真实 MySQL/Redis（9 条冒烟） |

### 分层执行

```bash
# 全量后端测试
go test ./...

# 单模块
go test -count=1 ./internal/modules/task/...
go test -count=1 ./internal/modules/export/...
go test -count=1 ./internal/modules/oauth/...
go test -count=1 ./internal/modules/dashboard/...

# 进程内集成测试（真实路由 + 迁移 + 种子）
go test -count=1 ./internal/integration/...

# 基准测试
go test -bench=. ./pkg/security/...
```

### MySQL 方言测试

`sqlite` 内存库无法覆盖 MySQL 特有语法（`DATE_FORMAT`、`ON DUPLICATE KEY` 等）。
方言用例由环境变量开关，且**库名必须以 `_test` 结尾**作为防误伤护栏：

```bash
METEORX_MYSQL_DIALECT_TEST=1 go test -count=1 ./internal/modules/task/repository/...
```

### 当前基线

| 指标 | 数值 |
| --- | --- |
| Go 测试文件 | 183 个（全绿） |
| Go 总覆盖率 | **50.9%**（CI 门禁 35%） |
| 前端测试文件 | 21 个（api / utils / composables 覆盖 99.63% / 97.44% / 99.31% / 99.63%） |
| 浏览器 E2E | 9 条（登录、套餐、消息分类持久化、站内信深链、告警详情入口） |

> 单元测试策略：Service 层通过仓库层提供的内存 Mock（如 `task/repository` 接口、`wiki/repository/mock_repository.go`）隔离 DB；
> 仓储层用 `sqlmock` 验证 SQL 形状与事务边界（Begin/Commit）；内存库统一用同名 shared-cache DSN
> （`file:<name>?mode=memory&cache=shared`）并钉死单连接，裸 `:memory:` 在多协程时会随机撞 `no such table`。
> 详细约定见 [测试指南](docs/development/testing.md)。

---

## 📚 接口文档

详细的模块接口文档见 `docs/` 目录：

| 文档 | 说明 |
|------|------|
| [认证 API](docs/api/auth-api.md) | 认证（注册/登录/2FA/API Token/邮箱验证/密码重置） |
| [用户 API](docs/api/user-api.md) | 用户管理（个人中心/租户用户/管理员/手机号/头像） |
| [租户 API](docs/api/tenant-api.md) | 租户管理（注册/后台管理/注销审批/独立配置） |
| [RBAC API](docs/api/rbac-api.md) | RBAC 权限（角色/权限/绑定/用户角色） |
| [文件 API](docs/api/file-module-api.md) | 文件管理（上传/下载/回收站/多存储驱动） |
| [套餐 API](docs/api/plan-api.md) | 套餐管理（CRUD/分配/配额用量） |
| [任务 API](docs/api/task-api.md) | 任务管理（个人/租户协作/批量/回收站/统计/AI 辅助） |
| [导出 API](docs/api/export-api.md) | 数据导出（创建/列表/下载/删除） |
| [站内消息 API](docs/api/message-api.md) | 收件箱（列表/未读数/已读/删除/批量操作/接收偏好）与离线补推 |
| [OAuth 提供商 API](docs/api/oauth-provider-api.md) | 三方登录提供商配置管理（含凭据加密） |
| [审计 API](docs/api/audit-api.md) | 审计日志（查询/导出/仪表盘/告警规则/会话分析） |
| [Wiki API](docs/api/wiki-api.md) | ⭐ Wiki 知识库（空间/节点/文档/版本/成员/发布流） |
| [运营看板 API](docs/api/dashboard-api.md) | 运营看板（总览/趋势/健康度/慢查询） |
| [公告 API](docs/api/announcement-api.md) | 通知公告（CRUD/发布/定向推送） |
| [邀请 API](docs/api/invitation-api.md) | 成员邀请（创建/取消/重发/接受/状态管理） |
| [功能升级记录](docs/FEATURE_UPGRADE.md) | 功能升级说明 |
| [测试指南](docs/development/testing.md) | 测试分层与覆盖率基线 |

**架构设计文档**（`docs/architecture/`）：

| 文档 | 说明 |
|------|------|
| [tenant.md](docs/architecture/tenant.md) | 多租户隔离（tenantctx / FilterQuery） |
| [permission.md](docs/architecture/permission.md) | RBAC 权限（自动推导 + 显式覆盖 + 缓存） |
| [error.md](docs/architecture/error.md) | 错误/响应规范（AppError / 统一信封） |
| [database.md](docs/architecture/database.md) | 数据库/事务（TxManager / ULID / 连接池） |
| [audit.md](docs/architecture/audit.md) | 审计（自动记录 / 批量异步 / 告警评估链） |
| [wiki.md](docs/architecture/wiki.md) | Wiki 知识库（五层模型 / 版本 / 权限 / XSS） |
| [task.md](docs/architecture/task.md) | 任务管理（访问控制 / 状态副作用 / 周期 / 提醒） |
| [export.md](docs/architecture/export.md) | 数据导出（租约 / 心跳 / 多格式 / 清理） |
| [oauth.md](docs/architecture/oauth.md) | 第三方登录（配置下沉 / 凭据加密 / 热重载） |
| [dashboard.md](docs/architecture/dashboard.md) | 运营看板（趋势分桶 / 健康度评分 / 慢查询） |
| [message.md](docs/architecture/message.md) | 站内消息（按人写行 / 写放大封顶 / 已读幂等 / 接收偏好） |
| [pagination.md](docs/architecture/pagination.md) | 分页/排序/过滤（Sort Whitelist 防注入） |
| [config.md](docs/architecture/config.md) | 配置/启动（Viper / 校验 / 优雅关闭） |
| [idgen.md](docs/architecture/idgen.md) | ID 生成（pkg/idgen） |

OpenAPI 规范文件位于 `docs/apifox/`：
- `MeteorX-backend.openapi.json` — 可导入 Swagger / Postman / Apifox
- `MeteorX-backend.apifox.json` — Apifox 项目文件

---

## 🏗️ 前端管理后台

`web-admin/` 目录为配套的 Vue 3 前端管理后台。

### 页面结构

| 页面 | 路由 | 功能 |
|------|------|------|
| 登录 | `/login` | 账号密码登录 + 2FA 验证码段 + 动态第三方登录按钮 |
| 租户注册 | `/register` | 企业自主注册开户（含租户信息） |
| 用户注册 | `/register-user` | 个人用户自助注册 |
| 忘记密码 | `/forgot-password` | 邮箱找回密码 |
| 重置密码 | `/reset-password` | 通过邮件令牌设置新密码 |
| OAuth 回调 | `/auth/:provider/callback` | 第三方登录结果处理 |
| 补全邮箱 | `/oauth/complete-email` | 第三方未返回邮箱时补全 |
| 接受邀请 | `/accept-invitation` | 被邀请人填写信息并加入租户（公开） |
| 邮箱验证 | `/verify-email` | 邮箱验证结果页（公开） |
| 工作台 / 看板 | `/dashboard` | 个人统计；平台管理员额外展示运营总览 + 趋势图 |
| 个人中心 | `/profile` | 修改资料/密码、2FA 开关、API Token、第三方账号绑定 |
| 任务管理 | `/task` | 个人待办 + 租户协作、批量操作、回收站、统计、AI 解析/扩写/摘要；支持 `?task_id=` 从任务提醒直达具体任务 |
| 数据导出 | `/export` | 新建导出（资源 × 格式）、任务进度轮询、下载中心 |
| 消息中心 | `/messages` | 收件箱（分类页签跨会话记住 / 仅看未读 / 关键字 / 分页 / 已读 / 删除 / 全选批量已读与删除 / 本分类全部已读 / 偏好设置，点开消息自动标已读）；顶栏铃铛与侧边栏角标 |
| 租户设置 | `/tenant-settings` | 租户独立配置（Logo/主题色/语言/时区） |
| 成员邀请 | `/invitation` | 邀请创建/列表/取消/重发/删除 |
| 公告（租户侧） | `/announcement` | 当前租户可见的已发布公告 |
| Wiki 知识库 | `/wiki` | 空间列表/创建/搜索与回收站入口 |
| Wiki 空间 | `/wiki/spaces/:id` | 节点树 + 分栏 Markdown 编辑/预览 + 版本恢复 + 附件 + 成员/节点权限 |
| Wiki 搜索 | `/wiki/search` | 全局搜索（标题/正文，高亮摘要） |
| Wiki 回收站 | `/wiki/trash` | 空间/节点/文档恢复与永久删除 |
| Wiki 待审核 | `/wiki/pending-reviews` | 审核队列（通过/驳回 + 评论） |
| Wiki 分享 | `/wiki/share/:token` | 公开分享页（无登录） |
| 用户管理 | `/system/user` | 租户用户 CRUD |
| 系统管理员 | `/system/master-admin` | 管理员 CRUD |
| 管理员回收站 | `/system/master-admin/recycle` | 已删除管理员恢复/永久删除 |
| 租户管理 | `/system/tenant` | 租户 CRUD；支持 `?tenant_id=` 从订阅临期消息直接打开该租户的套餐配置弹窗 |
| 租户回收站 | `/system/tenant/recycle` | 已删除租户恢复/永久删除 |
| 角色管理 | `/system/role` | 角色 CRUD + 权限绑定 |
| 角色回收站 | `/system/role/recycle` | 已删除角色恢复/永久删除 |
| 权限管理 | `/system/permission` | 权限 CRUD |
| 文件管理 | `/system/file` | 上传/下载/重命名/回收站/永久删除 |
| 套餐管理 | `/system/plan` | 套餐 CRUD + 租户套餐分配 |
| 审计日志 | `/system/audit` | 日志查询/导出/可视化仪表盘 |
| 登录历史 | `/system/audit/login-history` | 登录成功/失败与锁定记录 |
| 用户时间线 | `/system/audit/timeline` | 按天分组的用户操作轨迹；支持 `?user_id=` 从告警详情直达该用户时间线 |
| 异常检测 | `/system/audit/anomaly` | 高频失败等异常行为 |
| 告警管理 | `/system/audit/alert` | 告警规则管理 / 告警记录查询与详情；支持 `?alert_id=` 从告警站内信直接定位到具体记录，详情里的审计日志 ID 可跳到日志详情、触发用户可跳到用户时间线 |
| 会话分析 | `/system/audit/session` | 会话追踪 / 操作时间线 / 统计 |
| 租户健康度 | `/system/dashboard/tenant-health` | 分级筛选 + 评分明细（`requireMaster`） |
| 慢查询 | `/system/dashboard/slow-queries` | 指标卡 + 明细表 + 5s 自动刷新（`requireMaster`） |
| 通知公告 | `/system/announcement` | 公告 CRUD / 发布 / 下架 |
| 注销审批 | `/system/cancel-request` | 租户注销申请审批（通过/驳回） |
| 第三方登录 | `/system/oauth-provider` | Provider 配置 CRUD / 启停（`requireMaster`） |

### 权限控制

- 动态菜单：根据用户权限码显示/隐藏侧边栏
- 动态按钮：根据权限码显示/隐藏操作按钮
- 登录时后端返回 `permissions` 数组，前端存入 Pinia Store

---

## 📄 License

MIT