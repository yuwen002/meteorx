# 更新日志（Changelog）

本项目的所有重要变更都记录在此文件中。

格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 新增
- **双因素认证（2FA / TOTP）**：纯标准库实现 `pkg/totp`（RFC 6238，±1 时间窗容错 + `otpauth://` provisioning URI）；`User` 增 `totp_secret`/`totp_enabled` 字段与 `UpdateTOTP` 仓储方法；登录流程在租户软着陆校验后追加 TOTP 分支（缺码/错码返回 `require_2fa`，错码计入账户锁定）；新增 `POST /api/v1/auth/2fa/setup|enable|disable` 三端点，绑定密钥经 Redis 暂存（TTL 15 分钟）不落库直至验证通过；前端登录页二段式揭示验证码输入、个人中心三步绑定/关闭
- **慢查询看板**：新增 `internal/dbmonitor` 以 GORM `logger.Interface` 包装器在进程内采集超阈值 SQL 至内存环形缓冲（默认容量 200），无需外部慢查询日志；配置 `database.slow_query_ms`（未配置默认 500ms，负值关闭）；新增 `GET /api/v1/admin/dashboard/slow-queries` 与 `POST /api/v1/admin/dashboard/slow-queries/reset`（权限 `admin:dashboard:list`/`create`）；前端 `#/system/dashboard/slow-queries` 看板页（四指标卡 + 明细表 + 5 秒自动刷新 + 重置）
- **前端暗色主题**：新增 `useTheme` composable（localStorage 持久化 + 切换 `html.dark`），引入 Element Plus 官方 dark css-vars 使组件自动适配，布局 chrome 改用 CSS 变量，顶栏加主题切换按钮
- **前端国际化落地**：`useI18n` 增加语言持久化，`el-config-provider` 随当前语言联动 Element Plus 内置文案，顶栏加中/EN 切换；进一步**铺开至全局外壳**——侧边栏菜单/面包屑/顶栏提示/用户下拉/退出确认/品牌对话框统一改用 `t()`（此前 `nav.*` 等 key 已存在却未使用），并补齐相关缺失 key
- **全局状态迁移 Pinia**：`useTheme`/`useBrand`/`useI18n` 由“模块级单例 + localStorage”迁至 `stores/theme|brand|i18n`，composable 保留为 `storeToRefs` 薄封装（消费方零改动）；`main.ts` 挂载前以纯函数引导主题/品牌/语言，消除首屏闪烁
- **白标自定义品牌**：新增 `useBrand` composable（产品名/Logo/主色 localStorage 持久化，应用 `document.title` 与 `--el-color-primary`），仅管理员可经顶栏「品牌设置」对话框编辑；**服务端级多租户白标已落地**——`tenant_settings` 下发 `tenant_name`/Logo/主色，登录后拉取应用（服务端优先、本地回退），品牌名复用 `Tenant.Name`
- 导出队列上限可配置：`export.max_active_jobs`（默认 10）覆盖原硬编码常量，未完结任务达上限新建返回 429
- **API Token 权限作用域（Scope）**：`api_tokens` 新增 `scopes` 字段（JSON 权限码列表）。令牌未配 scope 时沿用继承属主全部 RBAC 权限的历史行为（向后兼容）；配置非空 scope 时进入**严格最小权限模式**——有效权限收敛为「属主实际权限 ∩ scopes」（属主为超管时即 scopes），并在 `RequirePermission`/`AutoRequirePermission`/`RequireExplicitPermission` 三个权限中间件中强制生效、**忽略超管旁路**，杜绝长期令牌默认继承全部权限。`Validate` 改为返回 `contextx.TokenPrincipal`，收敛结果经请求上下文透传；同时修正 `matchAllowedPath` 由裸前缀匹配改为路径段对齐（`/users` 不再误配 `/users-admin`），Redis 令牌缓存值改为 JSON 结构以容纳 scopes 与解析后的权限集合。新增 `POST /api/v1/auth/tokens` 的 `scopes`/`allowed_paths` 请求与响应字段
- **前端键盘导航与无障碍（a11y）基线**：全局外壳 `DefaultLayout` 补齐地标语义（侧边栏 `role=navigation`、顶栏 `role=banner`、主内容 `role=main` 及各自 `aria-label`），新增「跳到主内容」skip-link（聚焦时才可见）；将折叠、主题切换两个「图标即按钮」由不可聚焦的 `el-icon` 改为原生 `<button>`（自动可 Tab、可回车/空格触发）并补 `aria-label`/`aria-expanded`，语言切换、用户菜单、WS 连接状态点补 `role` 与 `aria-label`（状态色点标 `aria-hidden`）；移除顶栏操作 `.header-action/.lang-toggle` 上 `outline:none` 的焦点反模式，改由 `App.vue` 全局 `:focus-visible` 统一高亮键盘焦点；新增 a11y 文案的 i18n 中英文 key 及 `useI18n` 断言（`<html lang>` 联动此前已由 `applyLocaleToDOM` 覆盖，本次未重复实现）
- **云存储后端（阿里 OSS + AWS S3）**：`Storage` 接口新增两个驱动实现 `OSSStorage`（`github.com/aliyun/aliyun-oss-go-sdk`）与 `S3Storage`（`github.com/aws/aws-sdk-go-v2`，兼容 MinIO 等 S3 协议端点），与本地存储同走一套 `Upload/Download/Delete/Exists/GetURL/PresignURL/Type` 契约，`FileService` 无需改动即可切换；`NewStorage` 改为返回 `(Storage, error)` 并按 `file.storage_type` 分派 local/oss/s3，云存储初始化失败时沿用与 IP 定位器一致的降级策略（记告警并回退本地，不阻断启动）。私有桶下 `GetURL` 尽力返回预签名 URL（默认 3600s，可配 `cloud.presign_ttl`），配 `cloud.public_base_url` 则走公开/CDN 直链；对象键沿用「唯一 ID + 原扩展名」并支持 `cloud.path_prefix` 前缀隔离。新增 `file.cloud.*` 配置项（config.yaml 默认注释）与对应 `METEORX_FILE_CLOUD_*` 环境变量绑定；补工厂分支、预签名 URL、对象键构建的单测（预签名均为本地计算、无需联网）
- **本地 MinIO 联调配套**：`docker-compose.dev.yml` 新增 `minio`（S3 兼容对象存储，API `9000`/控制台 `9001`）与 `minio-init`（就绪后自动建 `meteorx` 桶）两个服务，`app` 服务内置默认注释的 `METEORX_FILE_STORAGE_TYPE=s3` 变量组；新增 `docs/minio-local-s3.md` 零云账号跑通 `S3Storage` 驱动的完整步骤（上传/下载/预签名均真实生效），并在 `docs/README.md` 与 `file-module-api.md` 引用

### 变更
- OpenTelemetry `service_version` 默认值与发布版本对齐（1.0.0 → 1.1.0），发版时随本文件一并更新

### 修复
- **路由平台级门禁失效**：全局 `beforeEach` 此前只校验 `meta.permission`，而大量平台页声明的 `meta.requireMaster` 从未被检查，导致被授予 `admin:*` 权限码的租户管理员可越级进入租户/审计/套餐/注销/公告等平台页。现抽出纯函数 `hasRouteAccess` 在守卫中强制校验 `requireMaster`（仅 `is_master` 放行），并补 `router/guard.test.ts` 回归
- **白标标题被覆盖**：路由 `afterEach` 曾把 `document.title` 硬编码为 `xxx - MeteorX`，在任何导航后覆盖掉租户白标品牌名；现改用当前品牌名 `xxx - {brand}`
- 套餐配额遗留数据归一化：配额列引入前创建的套餐行其 `storage_limit`/`api_rate_limit`/`wiki_space_limit`/`user_limit` 数据库默认值为 0，此前读路径原样返回 0，导致前端展示为「0 B / 0 次/分 / 0 个」，且消费端以 `limit < 0` 判不限时会把 0 误当作「上限为 0」而拦截全部上传/请求/建空间。现于 repository `toDomain` 读路径统一将 0 归一为 -1（不限），与写路径 `CreatePlan` 归一化口径对齐；补 `plan_repository_test.go` 覆盖遗留 0 值归一与显式值保留两条路径
- **云存储切换后存量文件不可用**：`FileService` 此前只持有单一存储实例，虽在每条记录写入 `StorageType` 却从未用于路由；切换 `file.storage_type`（如 local→s3）后，历史文件的 `FilePath` 在新后端不存在，下载/URL/删除全部失效。新增 `storage.Registry`：上传走默认后端，读取/删除按记录 `StorageType` 惰性解析到原后端（无法构造时告警并回退默认），`FileService` 改经 `BackendResolver` 取后端（`NewFileService` 签名不变，内部包单后端垫片，旧测试零改动）；补 `registry_test.go` 覆盖路由/缓存/回退
- **OSS 删除不幂等**：`OSSStorage.Delete` 对不存在对象直接返回错误（阿里云 OSS 删不存在 key 返回 404 NoSuchKey），与本地（忽略 `IsNotExist`）/ S3（`DeleteObject` 幂等）行为不一致，删除已消失对象会刷无意义告警。现识别 404/NoSuchKey 并视为成功；补 `oss_storage_test.go`
- **S3 仅支持静态凭证**：`S3Storage` 原先强制要求 `access_key`/`secret_key`，EC2/ECS/EKS（IRSA）等实例角色场景无法使用。现两者均留空即走 AWS 默认凭证链（环境变量/共享配置/IMDS/IRSA），仅填其一报错；补凭证三态单测
- **中文文件名下载乱码**：下载响应 `Content-Disposition` 此前直接拼接原始文件名，非 ASCII（中文）名未编码导致部分浏览器乱码。现纯 ASCII 走 `filename="..."`，含非 ASCII 时附加 RFC 5987 `filename*=UTF-8''<percent-encoded>` 并提供 ASCII 兜底名；补 handler 测试

### 测试
- 新增导出模块 handler 层 HTTP 测试（桩服务 + chi 路由），覆盖创建/列表/详情/下载/删除 5 端点的鉴权、参数透传与 400/401/403/404/409/429/500 错误映射
- 前端补齐 export/plan/dashboard/tenant 四个 API 模块测试（37 用例，均 100% 覆盖），消除 v1.1.0 新页面接口层零测试盲区

### 构建
- CI 覆盖率门槛（阶梯式防回退）：后端 `go tool cover` 总覆盖率 ≥35%（基线 ~37%）；前端 `vitest coverage` 门槛内置于 vitest.config.ts（仅统计 api/utils/composables 层，lines/statements 85、branches 95、functions 70，基线 86.24/97.84/73.21），前端 job 改用 `npm run test:coverage`

## [1.1.0] - 2026-10-13

### 新增
- **异步数据导出与下载中心**：CSV 流式异步导出（支持 user / task / audit_log / wiki_document 四类资源），下载中心页面（`#/export`）；5 个 REST 端点（创建 / 列表 / 详情 / 下载 / 删除），状态机 pending → running → completed/failed；worker 池 2s 轮询消费，过期文件每 6 小时清理（单轮 200 条）；单租户未完结任务队列上限准入（默认 10），达到上限返回 429
- **租户健康度看板**：`/api/v1/admin/dashboard/tenants-health[/{id}]` 两个管理端点与前端看板页面（`#/system/dashboard/tenant-health`）；评分模型为订阅 40 分 + 近 7 天活跃度 30 分 + 资源足迹 30 分，≥70 healthy / ≥40 moderate / <40 at_risk；批量 `IN + GROUP BY` 聚合避免 N+1
- **租户套餐配额与 API 限流**：套餐新增 `storage_limit` / `api_rate_limit` / `wiki_space_limit` 三类配额（-1 表示不限）；超限拦截映射——用户数超限 403、API 速率超限 429（含 `Retry-After`，Redis 自然分钟窗口，不可用时 fail-open）、存储超限 507、Wiki 空间超限 402
- 文档：新增 `docs/api/export-api.md`；补齐 `dashboard-api.md`（tenants-health）、`plan-api.md`（三类配额字段与拦截说明）；`docs/README.md` 索引同步

### 变更
- `config.yaml`：`otel.service_version` 更正为 1.1.0

## [1.0.0] 及更早

v1.0.0 基线：多租户 SaaS 后端（认证 / RBAC / 用户 / 租户 / 审计 / 文件 / 任务 / Wiki / 计划套餐 / 邀请 / 通知 / OAuth / 仪表盘）与 web-admin 管理前端。本自 v1.1.0 起维护变更日志。

[Unreleased]: https://github.com/yuwen002/meteorx/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/yuwen002/meteorx/releases/tag/v1.1.0
