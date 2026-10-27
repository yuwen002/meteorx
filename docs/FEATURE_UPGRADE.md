# MeteorX 平台功能升级说明

按既定排序完成了三大功能升级：**① 平台运营数据看板 ② 平台通知/公告系统 ③ 租户注销审批闭环**。

---

## 一、平台运营数据看板（Dashboard）

**后端新增 `internal/modules/dashboard` 模块**
- `GET /api/v1/admin/dashboard/overview`（平台超级管理员专属）→ 权限码 `admin:dashboard:list`
- 聚合统计：租户总数/启用/禁用/今日新增、用户总数/新增、订阅总数/生效/到期/取消、审计日志总数/成功/失败/按操作与模块分布
- 分层：`model → repository → service → handler → dto`，全部基于 GORM 直查既有数据表（tenants / users / tenant_subscriptions / audit_logs）

**前端增强 `web-admin/src/views/dashboard/index.vue`**
- 平台管理员看到：四类核心指标卡片 + 新增概览表（今日/本周/本月）+ 订阅分布 + 操作/模块访问统计
- 普通租户管理员仍看到原工作台统计，二者自动区分

---

## 二、平台通知/公告系统（Notification）

**后端新增 `internal/modules/notification` 模块**（平台超级管理员专属）
- 路由：
  - `GET /admin/announcements` → `admin:announcement:list`
  - `POST /admin/announcements` → `admin:announcement:create`
  - `GET /admin/announcements/{id}` → `admin:announcement:read`
  - `PUT /admin/announcements/{id}` → `admin:announcement:update`
  - `PUT /admin/announcements/{id}/status` → `admin:announcement:status`（发布/下架）
  - `DELETE /admin/announcements/{id}` → `admin:announcement:delete`
- 公告支持：草稿/已发布/已下架状态、全平台/指定租户范围、发布时间/过期时间
- 新建表 `announcements`（AutoMigrate 自动迁移）
- 在权限推导中间件 `singularize` 增加 `announcements → announcement` 映射

**前端新增 `web-admin/src/views/system/announcement/index.vue`**
- 公告列表（搜索/状态筛选/分页）、新建/编辑弹窗、发布/下架/删除操作、详情弹窗
- 路由 `/system/announcement` + 侧边栏「公告管理」菜单（仅平台管理员可见）

---

## 三、租户注销审批闭环（Tenant）

将原有的注销「占位实现」升级为完整审批闭环。

**后端（改造 `internal/modules/tenant`）**
- 新增领域模型与表 `cancel_requests`（AutoMigrate 迁移）
- 状态机：`待审批 → 已通过 → 已完成` 或 `待审批 → 已驳回`
- 租户侧申请 `ApplyCancellation`：真实写入待审批申请，重复申请会被拦截
- 平台审批接口：
  - `GET /admin/cancel-requests` → `admin:cancel_request:list`
  - `PUT /admin/cancel-requests/{id}/approve` → `admin:cancel_request:approve`（可设生效天数，0=立即执行）
  - `PUT /admin/cancel-requests/{id}/reject` → `admin:cancel_request:reject`
- 定时任务 `StartCancelCleanupJob`（5 分钟间隔）：自动执行已到期的注销申请 → 软删除租户 + 取消生效订阅 + 标记完成
- 在权限推导中间件增加 `cancel-requests → cancel_request` 映射

**前端新增 `web-admin/src/views/system/cancel-request/index.vue`**
- 注销申请列表（租户名称/原因/状态/计划执行时间筛选）
- 通过（选择立即/3天/7天生效 + 备注）与驳回（必填原因）操作
- 路由 `/system/cancel-request` + 侧边栏「注销审批」菜单（仅平台管理员可见）

---

## 四、租户物理删除与套餐配给（Tenant）

补充两个平台管理员高频操作接口：物理删除租户（彻底销毁）和为租户分配/变更套餐。

**后端（`internal/modules/tenant`）**
- 新增 `HardDelete` 仓储方法，通过 GORM `Unscoped().Delete()` 绕过软删除
- 新增 `AdminHardDelete` 服务方法：物理删除租户 + 同步取消生效订阅
- 新增 `AdminUpdatePlan` 服务方法：通过接口注入调用 `PlanService.AssignPlan`
- 注入方式：`SetPlanAssignProvider` 接口，避免 Tenant 模块直接依赖 Plan 模块
- 路由：
  - `DELETE /admin/tenants/{id}/hard` → `admin:tenant:hard_delete`
  - `PUT /admin/tenants/{id}/plan` → `admin:tenant:update_plan`

---

## 五、权限系统完善（RBAC 审计与缓存）

**后端改造 `internal/modules/rbac` 模块**

### 5.1 权限审计日志

**涉及文件：**
- `internal/middleware/permission_audit.go` - 定义权限审计日志结构体和记录器接口
- `internal/modules/rbac/service/permission_audit_storage.go` - 实现审计日志存储到审计模块
- `internal/modules/rbac/service/rbac_service.go` - 在角色/权限分配方法中集成审计日志
- `internal/modules/rbac/module.go` - 初始化审计日志记录器

**改进效果：**
- 每次角色分配/移除 → 自动写入审计日志
- 每次权限绑定/解绑 → 自动写入审计日志
- 支持批量操作审计

### 5.2 权限缓存

**涉及文件：**
- `internal/middleware/permission_cache.go` - 实现基于 Redis 的权限缓存（TTL=5分钟）
- `internal/modules/rbac/module.go` - Redis 可用时启用缓存
- `internal/modules/rbac/service/rbac_service.go` - 权限变更后自动失效缓存

**改进效果：**
- 权限查询性能提升（Redis 缓存命中）
- 支持用户级和全局级缓存失效
- 权限变更后即时生效

---

## 六、指标监控系统（Prometheus + Grafana）

**后端新增 Prometheus 指标暴露**

### 6.1 HTTP 请求指标

**涉及文件：**
- `internal/middleware/prometheus.go` - 实现 Prometheus 格式指标采集
- `internal/bootstrap/router.go` - 注册 `/metrics` 端点

**指标列表：**

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `meteorx_http_requests_total` | Counter | method, path, status | 请求总数 |
| `meteorx_http_request_duration_seconds` | Histogram | method, path | 请求延迟分布 |
| `meteorx_uptime_seconds` | Gauge | - | 服务运行时间 |
| `go_memstats_alloc_bytes` | Gauge | - | 已分配堆内存 |
| `go_goroutines` | Gauge | - | 当前 goroutine 数量 |
| `go_memstats_sys_bytes` | Gauge | - | 系统内存占用 |
| `go_memstats_gc_cpu_fraction` | Gauge | - | GC CPU 占比 |
| `go_threads` | Gauge | - | 系统线程数 |
| `process_start_time_seconds` | Gauge | - | 进程启动时间戳 |

### 6.2 监控部署栈

**新增文件：**
- `deploy/prometheus/prometheus.yml` - Prometheus 采集配置
- `deploy/prometheus/alerts/meteorx_alerts.yml` - 告警规则配置
- `deploy/grafana/datasources/datasource.yml` - Grafana 数据源配置
- `deploy/grafana/dashboard-providers.yml` - Grafana 面板配置
- `deploy/grafana/dashboards/meteorx-dashboard.json` - 服务监控面板

**Docker Compose 新增服务（`docker-compose.prod.yml`）：**

| 服务 | 端口 | 说明 |
|------|------|------|
| Prometheus | 9090 | 指标存储（30天保留） |
| Grafana | 3000 | 可视化面板（admin/admin） |
| Node Exporter | 9100 | 主机指标 |
| MySQL Exporter | 9104 | 数据库指标 |
| Redis Exporter | 9121 | Redis 指标 |
| cAdvisor | 8082 | 容器指标 |

**告警规则：**
- 服务宕机：`meteorx_up == 0` 持续 1 分钟
- 高错误率：5xx 错误率 > 5% 持续 5 分钟
- 高延迟：平均延迟 > 2 秒持续 5 分钟
- 高内存：内存使用 > 80% 持续 5 分钟
- 频繁重启：`process_start_time_seconds` 变化

---

## 七、前端全局通知中心面板

**前端新增 `web-admin/src/views/wiki/components/GlobalNotificationCenter.vue`**

### 7.1 功能描述

将原有的告警按钮升级为完整的全局通知中心，整合公告通知和告警通知。

### 7.2 涉及文件

| 文件 | 变更 |
|------|------|
| `web-admin/src/views/wiki/components/GlobalNotificationCenter.vue` | **新增** - 全局通知中心组件 |
| `web-admin/src/stores/notification.ts` | **修改** - 新增公告管理状态和方法 |
| `web-admin/src/layouts/DefaultLayout.vue` | **修改** - 替换告警按钮为通知中心 |
| `web-admin/src/App.vue` | **修改** - WebSocket 联动刷新公告 |

### 7.3 功能点

- **通知铃铛**：顶部导航栏显示未读总数角标（公告+告警）
- **公告标签**：显示最近 5 条公告，未读标记蓝色圆点，点击跳转
- **告警标签**：显示最近 10 条告警，按风险级别着色
- **全部标为已读**：一键清空所有未读标识
- **WebSocket 联动**：收到新公告通知时自动刷新列表
- **智能时间**：显示"刚刚/X分钟前/X小时前/X天前"

### 7.4 数据流

```
后端发布公告 → 广播 WebSocket announcement 消息
        │
        ▼
前端 App.vue 收到消息 → 调用 notificationStore.loadAnnouncements()
        │
        ▼
   通知中心自动刷新 → 更新未读角标
        │
        ▼
   用户点击铃铛 → 弹出通知面板
        │
        ▼
   点击公告 → 标记已读 → 跳转公告列表页
```

---

## 八、系统优化与架构改进

### 8.1 审计日志批量处理优化

**性能提升：10-100倍**

**涉及文件：**
- `internal/modules/audit/repository/interface.go` - 添加 `BatchCreate` 接口方法
- `internal/modules/audit/repository/audit_repository.go` - 实现真正的批量SQL插入
- `internal/modules/audit/service/audit_service.go` - 添加 `BatchCreateLogs` 方法
- `internal/middleware/audit_batch.go` - 使用批量插入替代逐个写入
- `internal/middleware/audit_middleware.go` - 优化Context传递

**改进效果：**
- 从 O(n) 数据库操作优化为 O(1) 批量插入
- 批量大小：100条，刷新间隔：5秒
- 减少数据库连接开销和磁盘IO

### 8.2 Context生命周期管理

**涉及文件：**
- `internal/middleware/audit_middleware.go` - 异步审计日志使用带超时的Context（10秒）
- `internal/bootstrap/app.go` - 提前创建应用级context
- `internal/bootstrap/router.go` - 更新 `InitRouter` 签名接受context

**改进效果：**
- 避免服务器关闭时 goroutine 泄漏
- 异步操作支持优雅取消
- 符合 Go 语言最佳实践

### 8.3 统一错误处理

**涉及文件：**
- `internal/modules/tenant/handler/tenant_handler.go` - 替换硬编码状态码为 `http.StatusXXX` 常量
- `internal/modules/auth/handler/auth_handler.go` - 统一错误处理模式
- `internal/common/validator/validator.go` - 标准化状态码使用

**改进效果：**
- 代码可读性和可维护性提升
- 符合 Go 语言最佳实践
- 禁止使用硬编码HTTP状态码

### 8.4 日志标准化

**涉及文件：**
- `internal/modules/tenant/handler/tenant_handler.go` - 替换 `fmt.Println` → `log.Printf`
- `internal/modules/user/handler/user_handler.go` - 替换 `fmt.Println` → `log.Printf`
- `internal/modules/file/service/file_service.go` - 替换 `fmt.Printf` → `log.Printf`

**改进效果：**
- 所有日志输出使用标准 `log` 包
- 添加模块前缀（如 `[TenantHandler]`、`[UserHandler]`、`[FileService]`）
- 便于日志收集、分析和监控

### 8.5 JWT密钥安全强化

**涉及文件：**
- `internal/config/config.yaml` - 添加强密钥警告注释

**改进效果：**
- 提醒开发者在生产环境使用强随机密钥
- 提供生成强密钥的方法说明：`openssl rand -base64 64`

### 8.6 健康检查增强

**涉及文件：**
- `internal/bootstrap/router.go` - 新增 `/health/ready` 深度检查端点
- `internal/cache/redis.go` - 添加 `Ping` 方法

**改进效果：**
- `/health` - 基础检查（负载均衡器使用）
- `/health/ready` - 深度检查（数据库+Redis连接状态，K8s就绪探针使用）
- 返回详细的JSON健康状态

### 8.7 角色永久删除功能

**涉及文件：**
- `internal/modules/rbac/repository/interface.go` - 添加 `PermanentDelete` 和 `BatchPermanentDelete` 接口
- `internal/modules/rbac/repository/role_repository.go` - 实现永久删除方法
- `internal/modules/rbac/service/rbac_service.go` - 添加业务逻辑
- `internal/modules/rbac/handler/rbac_handler.go` - 添加HTTP处理器
- `internal/modules/rbac/routes.go` - 注册路由
- `web-admin/src/api/modules/role.ts` - 添加前端API
- `web-admin/src/views/system/role/recycle.vue` - 实现前端功能

**新增API端点：**
- `DELETE /api/v1/rbac/roles/{id}/permanent` - 永久删除单个角色
- `DELETE /api/v1/rbac/roles/batch/permanent` - 批量永久删除角色

**改进效果：**
- 角色回收站功能完整闭环（软删除 → 回收站查询 → 恢复 → 永久删除）
- 完成前端TODO项

---

## 权限注册汇总（`internal/modules/rbac/permissions.go`）

| 权限码 | 说明 |
|---|---|
| `admin:dashboard:list` | 查看运营数据看板 |
| `admin:announcement:list/read/create/update/status/delete` | 公告管理 |
| `admin:cancel_request:list/approve/reject` | 注销审批 |
| `admin:tenant:hard_delete` | 物理删除租户（彻底销毁） |
| `admin:tenant:update_plan` | 为租户分配/变更套餐 |
| `rbac:role:permanent_delete` | 永久删除角色 |
| `rbac:role:batch_permanent_delete` | 批量永久删除角色 |

全部通过幂等 `SeedPermissions` 注册，superadmin 直接放行。

---

## 九、数据库迁移自动化

**新增 `cmd/migrate/main.go`** - 独立的迁移 CLI

```bash
# 执行数据库迁移（不启动服务器）
go run ./cmd/migrate

# 或通过 server 命令
go run ./cmd/server migrate
```

**CI/CD 集成：**
- 新增 `migrate` job：在 `test` 之后、`build` 之前运行，验证数据库迁移
- 生产部署 `deploy` job 中增加迁移步骤：`docker compose exec backend ./meteorx migrate`
- 配合 `main.go` 的 `migrate` 参数，实现单二进制多用途

---

## 十、集成测试基础设施

**新增 `internal/testutil` 包** - 共享测试工具

| 函数 | 说明 |
|------|------|
| `NewTestRouter` | 创建测试用 chi 路由器 |
| `TestContext` | 创建包含认证信息的测试上下文 |
| `ExecuteRequest` | 执行 HTTP 测试请求 |
| `ParseResponse` | 解析 JSON 响应 |
| `AssertStatusCode` | 断言 HTTP 状态码 |
| `NewJSONBody` | 将结构体编码为 JSON 请求体 |

**新增 `test/mockdata` 包** - Mock API 响应数据

| 文件 | 说明 |
|------|------|
| `api/tenant_get.json` | 获取单个租户 |
| `api/tenant_list.json` | 租户列表 |
| `api/user_list.json` | 用户列表 |
| `api/audit_dashboard.json` | 审计仪表盘 |

---

## 十一、前后端联调优化

**WebSocket 重连机制 ( `web-admin/src/composables/useWebSocket.ts` )**

| 特性 | 说明 |
|------|------|
| **指数退避重连** | 1s → 2s → 4s → 8s ... 最大 30s |
| **最大重试次数** | 10 次后停止，避免无限重连 |
| **随机抖动** | 每次重连延迟 +0~1s 随机值，避免大量客户端同时重连 |
| **心跳超时检测** | 10s 无 pong 响应即触发重连 |
| **离线消息队列** | 断连期间的消息缓存到 localStorage，重连后自动发送 |
| **连接状态** | 新增 `connecting` / `reconnectAttempts` / `lastConnectedTime` 状态 |

**审计日志详情增强 ( `web-admin/src/views/system/audit/timeline.vue` )**

| 特性 | 说明 |
|------|------|
| **标签页切换** | 基本信息 / 详细信息 / 请求参数 / 响应内容 |
| **详细信息** | 会话ID、链路追踪ID、用户代理、设备信息、标签、租户ID |
| **请求参数** | JSON 格式化显示请求体 |
| **响应内容** | JSON 格式化显示响应体 |
| **新增字段** | referer（来源页面）、resource_id（资源ID） |

---

## 十二、多渠道消息通知系统

### 架构设计

```
                    ┌─────────────────┐
                    │   notify.Manager │  ← 统一入口
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
     ┌────────────┐ ┌────────────┐ ┌────────────┐
     │  Email     │ │  WebSocket │ │  Webhook   │
     │  Notifier  │ │  Notifier  │ │  Notifier  │
     └────────────┘ └────────────┘ └────────────┘
                                           │
                          ┌────────────────┼────────────────┐
                          ▼                ▼                ▼
                   ┌──────────┐    ┌──────────┐    ┌──────────┐
                   │ 钉钉     │    │ 企业微信  │    │ 飞书     │
                   │ 机器人   │    │ 机器人    │    │ 机器人   │
                   └──────────┘    └──────────┘    └──────────┘
```

### 新增文件

| 文件 | 说明 |
|------|------|
| `internal/notify/notifier.go` | 核心接口 (`Notifier`)、消息结构 (`Message`)、优先级、渠道类型 |
| `internal/notify/email_notifier.go` | 邮件通知渠道 - HTML 模板，支持多收件人 |
| `internal/notify/webhook_notifier.go` | Webhook 渠道 - 钉钉/企微/飞书/通用格式 |
| `internal/notify/ws_notifier.go` | WebSocket 站内信（封装 ws.Hub） |
| `internal/notify/composite_notifier.go` | 组合通知器 - 并发发送到多个渠道 |
| `internal/notify/manager.go` | 统一管理器 - 提供业务语义化接口 + 全局单例 |
| `internal/bootstrap/notify.go` | 启动初始化 - 自动注册所有可用渠道 |

### 核心接口

```go
// Notifier 通知渠道接口
type Notifier interface {
    Type() ChannelType           // 渠道类型
    Send(ctx, *Message) error    // 发送通知
    Name() string                // 渠道名称
}

// Message 统一通知消息
type Message struct {
    ID         string
    Title      string
    Content    string
    Channel    ChannelType    // email/webhook/websocket
    Priority   Priority       // low/normal/high/urgent
    Recipients []string       // 收件人列表
    Extra      map[string]interface{}
}
```

### 业务集成点

| 模块 | 事件 | 通知渠道 | 说明 |
|------|------|----------|------|
| **通知公告** | 公告发布 | WebSocket + Webhook | 推送给所有在线用户 |
| **审计告警** | 告警触发 | Email + Webhook + WebSocket | 发送给规则配置的目标 |
| **租户注销** | 审批通过/驳回 | Email + WebSocket | ✅ 已落地：`ApproveCancellation`/`RejectCancellation` 调用 `notify.NotifyCancelRequestStatus` |
| **订阅到期** | 即将到期 | Email + WebSocket | ✅ 已落地：`ExpiryJob` 扫描 7 天内到期订阅并调用 `notify.NotifySubscriptionExpiry` |

### 配置说明 (`config.yaml`)

```yaml
notify:
  webhook:
    enabled: false
    kind: "generic"              # generic / dingtalk / wechat / feishu
    url: ""                      # Webhook URL
    secret: ""                   # 签名密钥
    on_events:                   # 触发事件
      - "alert"
      - "announcement"
```

---

## 十三、OpenTelemetry 可观测性集成

### 现状与痛点

| 方面 | 当前实现 | 问题 |
|------|----------|------|
| **日志** | 自建 `pkg/logger` 结构化日志 | 无 OTel Logs Bridge |
| **指标** | 手写 `PrometheusMetrics`（纯内存） | 重启丢失，无直方图，无百分位 |
| **追踪** | ❌ 不存在 | 跨模块调用完全黑盒 |

### 架构

```
┌─ MeteorX API ─────────────────────┐
│                                    │
│  HTTP Request                      │
│    │                               │
│    ├─ TracingMiddleware ──┐        │
│    │  (Chi Route Pattern) │ Span   │
│    ├─ GORM OTel Plugin ───┤        │
│    │  (SQL 查询追踪)      │        │
│    └─ ...                 ┘        │
│                                    │
│  TracerProvider (全局)             │
│    │                               │
│    ├─ stdout Exporter (开发)       │
│    └─ OTLP gRPC Exporter (生产)    │
└──────────┬─────────────────────────┘
           │ OTLP gRPC (port 4317)
           ▼
┌─ OTel Collector ───────────────────┐
│  receivers: otlp (gRPC + HTTP)    │
│  processors: batch, memory_limiter│
│  exporters: debug, otlp (->Jaeger)│
└──────────┬─────────────────────────┘
           │ OTLP gRPC
           ▼
┌─ Jaeger (All-in-One) ─────────────┐
│  UI: http://localhost:16686        │
│  Store: in-memory (开发/演示)      │
└────────────────────────────────────┘
```

### 新增/修改文件

| 文件 | 说明 |
|------|------|
| `internal/otel/config.go` | OTel 配置结构体 |
| `internal/otel/provider.go` | SDK 初始化（TracerProvider + Exporter + Sampler） |
| `internal/middleware/tracing.go` | Chi 路由追踪中间件（自动提取 W3C Trace Context） |
| `internal/bootstrap/database.go` | 新增 GORM OTel 追踪插件注册 |
| `internal/bootstrap/app.go` | 集成 OTel 初始化 + 优雅关闭 |
| `internal/bootstrap/router.go` | 注册 TracingMiddleware |
| `internal/config/config.go` | 新增 `OTelConfig` |
| `internal/config/config.yaml` | 新增 OTel 配置节 |
| `deploy/otel-collector/config.yaml` | OTel Collector 管道配置 |
| `docker-compose.prod.yml` | 新增 OTel Collector + Jaeger 服务 |

### 配置说明

```yaml
otel:
  enabled: true
  exporter: "stdout"           # stdout（开发）/ otlp（生产）
  endpoint: "localhost:4317"   # OTLP gRPC 端点
  insecure: true               # 开发环境跳过 TLS
  sample_rate: 1.0             # 采样率（生产建议 0.1）
  service_name: "meteorx"
  service_version: "1.0.0"
```

### 集成效果

- **Trace ID 响应头**：每个 HTTP 响应都会返回 `X-Trace-ID`
- **Span 命名**：`GET /api/v1/tenants/{id}` 格式（含 Chi 路由模式）
- **GORM 追踪**：每次 SQL 查询自动关联到当前 Trace
- **上下文传播**：支持 W3C Trace Context（`traceparent` 头）
- **优雅关闭**：服务退出前刷新所有 Span 缓冲区

### 开发调试

启动后可以在控制台看到类似输出（stdout 模式下）：

```
Span #1
    Trace ID:      b1a9c8d7...
    Span ID:       e5f4a3b2...
    Attributes:
         - http.request.method:  GET
         - http.route:           /api/v1/tenants
         - http.response.status_code: 200
```

生产环境下可通过 Jaeger UI（`http://localhost:16686`）查看完整的 Trace 调用链。

---

## 验证状态

- ✅ 后端 `go build ./...` 全量编译通过
- ✅ `go vet ./internal/notify/...` 新增模块无静态问题
- ✅ 前端 `vue-tsc --noEmit` 类型检查通过（新文件无报错）
- ✅ 新文件均执行 `gofmt` 格式化
- ✅ 修复 7 处 `NewRBACService` 签名变更编译错误
- ✅ OTel 集成：TracingMiddleware + GORM 插件 + OTLP 导出 + Jaeger 部署

> 提示：需重新启动后端服务并执行数据库迁移（AutoMigrate 会自动创建 `announcements`、`cancel_requests` 两张新表）。
> 提示：OTel stdout 模式开机即用；生产环境需 `docker-compose -f docker-compose.prod.yml up -d opentelemetry-collector jaeger` 启动追踪后端。

---

## 十四、邮箱验证与成员邀请

### 14.1 邮箱验证

**后端新增 `auth` 模块扩展**

| 文件 | 变更 |
|------|------|
| `internal/modules/auth/service/auth_service.go` | 新增 `SendEmailVerification` 和 `VerifyEmail` 方法 |
| `internal/modules/auth/handler/auth_handler.go` | 新增 `SendEmailVerification` 和 `VerifyEmail` HTTP handler |
| `internal/modules/auth/routes.go` | 注册 `/auth/email/send-verification` 和 `/auth/email/verify` 路由 |
| `internal/modules/auth/dto/auth_dto.go` | 新增 `SendEmailVerificationReq/Resp` 和 `VerifyEmailReq/Resp` |
| `internal/pkg/emailer/emailer.go` | 新增 `SendEmailVerificationLink` 邮件模板方法 |
| `internal/modules/user/model/user.go` | User 模型新增 `EmailVerified` 字段 |
| `internal/modules/user/repository/user_repository.go` | 新增 `UpdateEmailVerified` 和 `GetByPhone` 方法 |

**业务流程：**
1. 用户在个人中心点击"发送验证邮件"
2. 系统生成 UUID 令牌，存入 Redis（前缀 `email:verify:`，有效期 24 小时）
3. 通过邮件发送验证链接：`{client.base_url}/verify-email?token={token}`
4. 用户点击链接，前端调用 `POST /auth/email/verify` 验证
5. 验证成功：更新 `email_verified = true`，删除 Redis 令牌

**安全机制：**
- 令牌一次性使用，验证后立即失效
- 令牌有效期 24 小时，过期自动失效
- 邮箱已验证时返回 `409` 错误
- 邮箱不存在时静默返回成功（防止信息泄露）

**前端新增：**
- `web-admin/src/views/verify-email/index.vue` — 邮箱验证结果页面（公开路由 `/verify-email`）
- `web-admin/src/api/auth.ts` — 新增 `sendEmailVerification` 和 `verifyEmail` API
- `web-admin/src/views/profile/index.vue` — 邮箱旁显示验证状态标签 + "发送验证邮件"按钮

---

### 14.2 租户成员邀请

**后端新增 `internal/modules/invitation` 模块**

| 文件 | 说明 |
|------|------|
| `model/invitation.go` | 邀请数据模型，含 pending/accepted/cancelled/expired 四种状态 |
| `repository/interface.go` | 仓储接口定义 |
| `repository/invitation_repository.go` | GORM 实现：CRUD + 按租户/令牌/邮箱查询 |
| `service/invitation_service.go` | 业务逻辑：创建→发邮件→接受→创建用户→分配角色 |
| `handler/invitation_handler.go` | RESTful API handler |
| `dto/invitation_dto.go` | 请求/响应 DTO |
| `module.go` | 模块初始化 + 路由注册（认证路由 + 公开路由） |

**路由注册：**

| 方法 | 路径 | 功能 | 认证 |
|------|------|------|------|
| GET | `/invitations` | 邀请列表 | 需 Token |
| POST | `/invitations` | 创建邀请 | 需 Token |
| PUT | `/invitations/{id}/cancel` | 取消邀请 | 需 Token |
| PUT | `/invitations/{id}/resend` | 重发邀请邮件 | 需 Token |
| DELETE | `/invitations/{id}/delete` | 删除邀请 | 需 Token |
| GET | `/invitations/info?token=xxx` | 查询邀请信息 | 公开 |
| POST | `/invitations/accept` | 接受邀请并注册 | 公开 |

**业务流程：**
1. 管理员在"成员邀请"页面填写邮箱和角色，点击"发送邀请"
2. 系统校验邮箱不重复（同租户无待处理邀请、非已有成员）
3. 生成 UUID 令牌，创建邀请记录（状态 `pending`，有效期 7 天）
4. 发送邀请邮件：`{client.base_url}/accept-invitation?token={token}`
5. 被邀请人打开链接，查看邀请详情（租户名、邮箱）
6. 填写用户名、昵称、密码，点击"接受邀请并注册"
7. 系统创建用户（`email_verified = true`）、分配角色、标记邀请 `accepted`

**状态流转：**
```
pending → accepted  （被邀请人接受）
pending → cancelled （管理员取消）
pending → expired   （超过有效期）
```

**前端新增：**
- `web-admin/src/api/modules/invitation.ts` — 邀请 API 模块
- `web-admin/src/views/invitation/index.vue` — 邀请管理页面（列表/创建/取消/重发/删除）
- `web-admin/src/views/accept-invitation/index.vue` — 接受邀请公开页面
- `web-admin/src/router/index.ts` — 新增 `/invitation` 管理路由 + `/accept-invitation` 公开路由

---

### 14.3 用户模型扩展

User 模型新增以下字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `email_verified` | bool | 邮箱是否已验证，默认 `false` |
| `phone` | string | 手机号（最长 20 字符） |
| `avatar` | string | 头像 URL（最长 500 字符） |

**前端个人中心增强：**
- 编辑资料弹窗新增手机号字段
- 个人信息展示新增邮箱验证状态标签、手机号
- `LoginUserInfo` 类型新增 `email_verified`、`phone`、`avatar` 字段

---

### 14.4 配置依赖

邮箱验证和成员邀请均依赖邮件服务配置：

```yaml
email:
  enabled: true
  host: "smtp.example.com"
  port: 587
  username: "noreply@example.com"
  password: "smtp-password"
  from: "noreply@example.com"
  from_name: "MeteorX"

client:
  base_url: "https://app.example.com"
```

> 如果 `email.enabled = false`，邀请仍可创建（记录写入数据库），但不会发送邮件。可在配置邮件后通过"重发"功能补发。

---

## 十五、通知预留项落地与邀请过期清理

本节将此前标注为「预留」的通知能力真正接入业务链路，并补齐邀请过期自动清理。

### 15.1 租户注销审批结果通知

- `internal/modules/tenant/service/tenant_service.go`
  - 新增私有方法 `notifyCancelRequest`，通过 `notify.GetGlobalManager()` 投递 `NotifyCancelRequestStatus`（Email + WebSocket）
  - `ApproveCancellation` 审批通过后（含立即执行与延迟执行两种）通知租户管理员，状态 `approved`
  - `RejectCancellation` 驳回后通知租户管理员，状态 `rejected`
  - 管理员邮箱取自租户 `ContactEmail`；通知器未初始化或查询失败时静默跳过，不影响主流程

### 15.2 订阅到期提醒

- `internal/modules/plan/repository/interface.go` + `plan_repository.go` + `mock_subscription_repository.go`
  - 新增 `FindExpiringSoon(ctx, within)`：查询 `within` 时间内即将到期且仍生效的订阅
- `internal/modules/plan/expiry_job.go`
  - `ExpiryJob` 在禁用已到期订阅后，扫描 7 天内到期订阅并调用 `notify.NotifySubscriptionExpiry`
  - 进程内 `notified` 集合按「订阅ID + 到期日」去重，避免周期任务重复提醒

### 15.3 邀请过期清理定时任务

- `internal/modules/invitation/repository/{interface.go,invitation_repository.go}`：新增 `ExpireOutdated(ctx, now)`，将过期 `pending` 邀请批量置为 `expired`
- `internal/modules/invitation/service/invitation_service.go`：新增 `ExpireOutdatedInvitations`
- `internal/modules/invitation/expiry_job.go`：新增 `InvitationExpiryJob`（结构对齐 `CancelCleanupJob`）
- `internal/bootstrap/{router.go,app.go}`：注册并在启动时拉起 `StartInvitationExpiryJob`（间隔 1 小时，绑定 app context 支持优雅取消）

### 15.4 前端测试补充

- `web-admin/src/api/modules/role.test.ts`：角色 CRUD / 批量 / 回收站接口 14 个用例
- `web-admin/src/api/modules/invitation.test.ts`：邀请管理与公开端接口 7 个用例

### 验证状态

- ✅ `go build ./...` / `go vet ./internal/...` 全量通过
- ✅ `go test ./internal/...` 全量通过（新增邀请过期单测）
- ✅ 前端 `vitest run` 新增 21 个用例全部通过

---

## 十六、任务管理模块（个人待办 + 团队协作）

新增 `internal/modules/task` 模块，将「个人待办」与「租户团队协作」统一为一套模型，通过 `visibility` 字段区分；不接入细粒度 RBAC，仅要求登录，靠数据归属实现访问控制。

### 16.1 后端分层

- `model/task.go`：`Task` 领域模型 + 状态（`pending`/`in_progress`/`completed`）、优先级（`low`/`normal`/`high`/`urgent`）、可见范围（`personal`/`tenant`）三组常量与校验函数；新增重复周期（`daily`/`weekly`/`monthly`）常量与 `IsValidRecurrence`/`NextOccurrence`
- `repository/`：GORM 仓储，`TaskPO` 以 JSON 文本存储标签，`gorm.DeletedAt` 软删除；`applyVisibility` 统一处理个人/团队查询隔离，默认按「未完成优先 + 优先级 + 创建时间」排序
- `service/`：业务编排，`canView`/`canEdit` 归属校验，个人任务负责人强制为创建人，截止时间宽松解析
- `handler/` + `module.go` + `dto/`：RESTful 接口，挂载于受保护分组
- `bootstrap/`：`router.go` 接入 `task.InitModule`；`migrate.go` 注册 `tasks` 非关键迁移

### 16.2 回收站与批量操作

- 回收站：`GET /tasks/deleted`（仅列出当前用户自己删除的任务）、`PUT /tasks/{id}/restore`（恢复）、`DELETE /tasks/{id}/permanent`（物理删除），均限创建人或 `superadmin`
- 批量：`POST /tasks/batch/complete`、`POST /tasks/batch/delete`，逐条按单条归属规则校验，无权/不存在项静默跳过，返回 `affected` 生效数
- 仓储层新增 `GetByIDUnscoped`/`ListDeleted`/`Restore`/`PermanentDelete`（基于 `Unscoped()`）
- 修复：仓储 `Create` 因传入结构体值导致 `reflect.Value.Set` panic，改为传指针

### 16.3 前端

- `web-admin/src/api/modules/task.ts`：完整任务 API（含回收站与批量）
- `web-admin/src/views/task/index.vue`：统计卡片 + 筛选 + 表格 + 新建/编辑弹窗 + 回收站抽屉 + 多选批量完成/删除
- 路由与侧边栏菜单接入（`/task`，「任务管理」）

### 16.4 任务提醒

- 指派提醒：创建/更新任务将负责人指向他人时，通过全局 `notify.Manager` 向新负责人推送 `task_assigned` 站内信（WebSocket），通知器未初始化时静默跳过不影响主流程
- 到期/逾期提醒：`TaskReminderJob`（`task/reminder_job.go`）周期扫描截止时间进入提醒窗口或已逾期的未完成任务，向负责人推送 `task_due`；`bootstrap` 中 `StartTaskReminderJob` 接线
- 提醒参数可配置：`config.yaml` 的 `task_reminder` 段（或环境变量 `METEORX_TASK_REMINDER_HORIZON/INTERVAL/OVERDUE_COOLDOWN/BATCH_LIMIT`）控制提醒窗口、扫描间隔、逾期重复提醒冷却期与单轮上限，缺省回落默认 24h/30m/24h/200
- 任务级提醒提前量：`tasks` 新增 `remind_before` 列（分钟，可为空），创建/更新时以 Go Duration（如 2h/30m）传入；扫描时优先用任务自带 `remind_before`，为空回落全局 `horizon`；修改提前量会重置提醒标记
- 提醒去重：`tasks` 表 `reminder_sent_at` 列（`TaskPO.ReminderSent`）为冷却基准；即将到期首次提醒一次，已逾期则按冷却期（默认 24h）重复提醒；修改截止时间自动重置标记，新截止日可再次触发
- 仓储新增 `FindDueForReminder`/`MarkReminderSent`；`service.SendDueReminders` 为系统级扫描入口；`notify.Manager.NotifyTaskReminder` 封装 WS 定向推送
- 前端：全局 `App.vue` 监听 WS 消息，命中 `task_assigned`/`task_due` 时弹出 `ElNotification`，不再误触发公告刷新

### 16.6 数据完整性与周期任务

- 负责人同租户校验：`UserDirectory` 新增 `IsTenantMember`（按 `id + tenant_id + deleted_at IS NULL` 存活性判定）；`TaskService` 以**类型断言自动检测**该可选能力（无需改构造函数/装配），在创建/更新/批量指派时拦截不存在或跨租户的负责人，非法返回 `400`；未注入时跳过校验，保持向后兼容
- 周期/重复任务：`tasks` 新增 `recurrence` 列；周期任务必须拥有截止时间；仅当任务从非完成**首次**转为 `completed` 时（单条/批量完成、批量改状态、更新置完成均生效），自动复制内容并按周期从原截止日推进生成下一条 `pending` 任务（daily +1天 / weekly +7天 / monthly +1月）；生成失败不阻断本次完成
- 死代码清理：移除仓储层已不被生产代码调用的 `UpdateStatus`（接口/实现/用例），状态变更统一走 `Update`
- 前端：`task.ts` 新增 `TaskRecurrence` 类型与 `recurrence` 字段；`index.vue` 新建/编辑表单加“重复周期”下拉，列表标题列展示“每日/每周/每月”徽章

### 16.7 工时统计与业务时区一致性

- 进度/工时统计：`TaskStatsResp` 新增 `completion_rate`（`completed/total`，handler 计算、保留四位小数）与 `avg_handle_seconds`（已完成任务基于 `started_at`→`completed_at` 的平均处理耗时）；仓储 `Stats` 新增一条 `AVG(TIMESTAMPDIFF(SECOND, ...))` 聚合（仅统计两时间均有值的已完成项，依赖 MySQL）；前端统计卡片由 4 张扩展为 6 张，新增“完成率”与“平均耗时”（`formatDuration` 按天/小时/分钟可读化）
- 业务时区一致性：新增 `database.timezone` 配置（默认 `Asia/Shanghai`）；`bootstrap.InitDB` 在建连接前按该值强制 `time.Local = time.LoadLocation(tz)`，使 `time.Now()`/时间解析/DSN `loc=Local` 与容器 `TZ` 环境变量解耦，开发/生产行为一致；`_ "time/tzdata"` 内嵌 IANA 时区库保证 Windows 无系统时区库也能加载；名称无效时保留系统默认并告警。默认仍为 +08，存量数据含义不变、零迁移

### 16.5 验证状态

- ✅ `go build ./...` / `go vet ./...` 通过
- ✅ `go test ./internal/modules/task/...` service/handler/repository 全部通过（含到期提醒扫描/去重/截止日重置、负责人同租户校验、周期任务生成/去重、平均处理耗时聚合用例）
- ✅ `go test ./internal/config/... ./internal/bootstrap/...` 通过（含 `GetTimezone` 默认/覆盖、`applyTimezone` 生效/无效名称保持用例）
- ✅ 前端 `vitest run`（task 用例 20 个）、`vue-tsc --noEmit` 通过
- ✅ OpenAPI / Apifox 已同步任务模块全部 13 条路由（提醒功能不新增 HTTP 接口，无需同步）
- 📄 接口文档见 `docs/api/task-api.md`
- ℹ️ 按要求暂不接入 AI 能力，代码结构保留后续叠加空间

---

## 十七、Wiki 回收站与删除链路加固

针对 Wiki 模块删除/回收站链路做一致性加固，并补齐过期清理的线上定时任务与测试覆盖。

### 17.1 软删除类型修正

- Wiki 实体（`WikiSpace` / `WikiNode` / `Document` 等）的 `DeletedAt` 由 `*time.Time` 改为 `gorm.DeletedAt`，统一 GORM 软删语义。
- 涉及 `internal/modules/wiki/model/wiki.go`；修正后 `Unscoped` 物理删除与回收站恢复行为正确，避免手动管理时间戳导致软删未生效。

### 17.2 空间删除级联

- 删除空间时级联软删其下全部节点/文档子树并写入回收站，保证回收站恢复与 `GetWikiStats` 统计一致性。
- 修复「空间已删但子树仍被计入统计」的统计偏差（`repository/wiki_repository.go` 的 `SoftDeleteSpaceTree` / `DeleteSpace`）。

### 17.3 回收站过期清理定时任务接线

- 新增 `internal/modules/wiki/expiry_job.go`：`WikiTrashExpiryJob`（结构对齐 `invitation/expiry_job.go`），含 `NewWikiTrashExpiryJob(db)` / `Start(ctx, interval)` / `RunOnce(ctx)`。
- `internal/bootstrap/router.go` 新增 `StartWikiTrashExpiryJob`，`internal/bootstrap/app.go` 启动时拉起（间隔 1 小时，绑定 app context 支持优雅取消）。
- 行为：跨租户全局扫描 `expires_at < now` 的回收站项目，按 `item_type` 物理清除关联实体（`document → PurgeDocument`、`node → PurgeNodeTree`、`space → PurgeSpaceTree`，级联清理版本/附件/子节点/分享链接等）后删除回收站记录；关联实体已缺失时容忍 `ErrDocumentNotFound / ErrWikiNodeNotFound / ErrWikiSpaceNotFound` 并仍清理回收站记录，避免僵尸数据。
- `service/wiki_trash_service.go` 暴露 `ExpireTrashItems(ctx)` 作为定时任务入口。

### 17.4 测试覆盖扩充

- `repository/wiki_repository_coverage_test.go`：真实 SQLite 集成测试覆盖回收站恢复/物理清除、统计（含删除后归零）、搜索（租户隔离）、版本、附件、节点权限、零散查询。
- 并发与过期任务：`TestIntegration_ConcurrentIncrementViewCount`（浏览量原子自增）、`TestIntegration_ConcurrentDocumentUpdate`（乐观锁冲突）、`wiki/expiry_job_test.go` 的 `TestWikiTrashExpiryJob_RunOnce` / `TestWikiTrashExpiryJob_StartStopsWithContext`（任务接线与优雅关闭）。

### 17.5 功能完整性核查

- 核查 `Comment` / `EditLock` / `ReviewComment` / `DocumentTemplate` / `DocumentAccessLog` / `DocumentSubscription` / `Notification` 七个已迁移模型，确认 service + handler + route + 前端四层均完整连通，无「只建表未接功能」的半成品。
- 唯一缺口（已在第十八节补上）：Wiki 通知生成此前仅来自评论 @提及，订阅变更与审核动作不会自动产生通知。现已补全评论/编辑订阅通知与审核流程通知（见十八）。

### 17.6 验证状态

- ✅ `go build ./...` 通过
- ✅ `go vet ./internal/modules/wiki/... ./internal/bootstrap/...` 通过
- ✅ `go test ./internal/modules/wiki/...` 全量通过（repository / service / handler 均 PASS）
- 📄 架构文档见 `docs/architecture/wiki.md`（回收站「定时清理任务」小节）

---

## 十八、Wiki 订阅与审核通知补全

针对 17.5 核查发现的缺口——Wiki 通知此前仅由评论 @提及生成，订阅变更与审核动作不会自动产生通知——补全相关链路。

### 18.1 评论触发订阅通知
- `service/wiki_service_extended.go` 的 `CreateComment`：在发送 @提及通知之外，额外通知该文档的订阅者（订阅类型为 `all` 或 `comment`），排除评论作者本人与已被 @提及者，避免重复打扰。
- 新增辅助方法 `notifyCommentSubscribers`，抽离订阅者筛选与通知逻辑便于测试。

### 18.2 审核流程通知
- `SubmitForReview`：提交审核后，通知空间内具有审核权限的成员（`owner` / `admin` 角色），排除提交者本人。
- `ApproveDocument` / `RejectDocument`：审核结论（通过/驳回）通知文档作者（最后编辑者 `LastEditedBy`），排除审核人本人。
- 新增辅助方法 `notifyReviewers`（按空间成员角色筛选）、`notifyAuthorReviewResult`。

### 18.3 新增通知类型
- `model/wiki.go` 新增常量 `NotificationTypeReview = "review"`，用于审核类通知（此前仅有 `mention` / `comment` / `share` / `document_edit`）。

### 18.4 编辑事件通知
- 新增 `WikiService.NotifyDocumentEdited(ctx, documentID, userID, title)`，在文档被**用户编辑**后通知订阅类型为 `all` / `edit` 的订阅者，排除编辑者本人。
- 入口由 HTTP `UpdateDocument`（`handler/wiki_handler.go`）在编辑成功后显式调用；与 `restoreDocumentFromTrash` 内部重写路径区分，避免恢复操作误触发编辑通知。
- 统一辅助方法 `notifySubscribers`（base `wikiService`）按事件类型与订阅类型过滤，评论（`NotifyTypeComment`）与编辑（`NotifyTypeEdit`）共用，避免重复逻辑。

### 18.5 验证状态
- ✅ `go build ./...` 通过
- ✅ `go vet ./internal/modules/wiki/...` 通过
- ✅ `go test ./internal/modules/wiki/...` 全量通过（新增/更新 `wiki_notification_service_test.go`：`TestNotifyReviewers` / `TestNotifyAuthorReviewResult` / `TestNotifySubscribers` 覆盖审核人、作者、评论/编辑订阅者通知的正向与排除路径）

---

## 十九、回收站过期清理健壮性与可观测性

在第十七节已接线的过期清理任务基础上，补齐「实体缺失时的健壮性」与「运行可观测性 + 运维手动触发」。

### 19.1 Purge* 实体缺失时主动清理回收站记录（健壮性）
- 原 `PurgeDocument` / `PurgeSpaceTree` 在目标实体已缺失（如先被手动永久删除）时返回 `ErrDocumentNotFound` / `ErrWikiSpaceNotFound`，且在此之前不清理回收站记录，可能遗留僵尸记录。
- 现改为**幂等**：实体可能已被先行删除时仍清理其级联关联与回收站记录并返回 nil——`PurgeDocument` 通过 `item_id + item_type` 删除 trash 记录，`PurgeSpaceTree` 删除空间自身 trash 记录。`PurgeNodeTree` 因 `collectNodeIDs` 始终包含请求 id，本就具备该能力，保持不动。
- 效果：任何直接调用 `Purge*` 的路径（含未来新增调用方）都不会遗留僵尸回收站记录；`ExpireTrashItems` 与 `PermanentDeleteTrashItem` 的兜底清理因此更稳。

### 19.2 过期清理计数与失败告警（可观测性）
- `ExpireTrashItems`（repo / service / `WikiService` 接口）由 `error` 改为返回 `(int, error)` 的**清理计数**。
- 定时任务 `WikiTrashExpiryJob.RunOnce(ctx) (int, error)`：成功且清理数 > 0 时记录 `清理 N 个回收站项目, 耗时 ...`；失败时记录错误（作为告警信号）；无清理则不刷屏。`Start` 复用 `RunOnce` 的日志，避免重复。

### 19.3 运维手动触发端点
- 新增 `POST /api/v1/wiki/admin/trash/expire`，由 `WikiHandler.TriggerTrashExpiry` 处理，复用 `service.ExpireTrashItems`（与定时任务同一逻辑，仅入口不同），返回 `{"purged": N}`。
- 路由经 `middleware.RequiresMasterAdmin()` 保护，仅平台超级管理员（`role=superadmin`）可调用，避免任意用户触发跨租户清理。

### 19.4 前端回收站「清理过期」按钮
- `web-admin/src/api/modules/wiki.ts` 新增 `expireTrash()`，调用 `POST /wiki/admin/trash/expire`，返回 `{ purged: number }`。
- `web-admin/src/views/wiki/trash.vue` 头部「刷新」旁新增「清理过期」按钮（`type=warning`），通过 `v-if="isSuperAdmin"` 限定仅 `userStore.isAdmin`（即 `is_master` 超级管理员）可见；点击二次确认后调用，按清理数量反馈（>0 成功提示，否则提示无过期项），并刷新列表。
- 与后端 `RequiresMasterAdmin()` 保护一致：非超级管理员在 UI 上直接看不到该入口。

### 19.5 失败告警接入真实通道
- `expiry_job.go` 的 `RunOnce` 在清理失败时，除保留 `logger.Printf` 日志外，额外通过 `notify.GetGlobalManager().NotifyAlert(...)` 推送**高优先级运维告警**（`ruleName="回收站过期清理失败"`、含耗时与错误详情）。
- 复用平台统一的 `notify.Manager`（email / webhook / WebSocket 渠道），未配置任何渠道时为 no-op，不阻塞定时任务主流程——即「有渠道才真正告警，无渠道仅留日志」。

### 19.6 手动触发写审计/操作日志
- `WikiHandler.TriggerTrashExpiry` 在成功/失败时均通过 `middleware.RecordAuditLog`（全局批量审计处理器，无则直写）写入一条业务审计日志：`module=wiki`、`action=trash_expire`、`resource=wiki_trash`、成功 `RiskLevel=medium` / 失败 `RiskLevel=high`，成功时响应体携带清理数量 `{"purged": N}`、失败时记录 `error_message`。
- 与中间件对原始 HTTP 请求的通用审计互补：通用审计记录「谁、何时、什么路径」，本条目额外记录「业务动作与清理结果」，便于在审计台按 `trash_expire` 动作过滤运维操作。

### 19.7 路由冒烟测试镜像
- `internal/bootstrap/route_check_test.go` 的 `registerWikiRoutes` 镜像补充 `POST /wiki/admin/trash/expire`，使 `TestAllRoutes_DerivePermissionCode` 覆盖该端点（推导权限码 `wiki:create`）。

### 19.8 验证状态
- ✅ `go build ./...` 通过
- ✅ `go vet ./internal/modules/wiki/... ./internal/bootstrap/...` 通过
- ✅ `go test ./internal/modules/wiki/... ./internal/bootstrap/...` 全量通过
  - 新增 `TestIntegration_PurgeDocument_CleansTrashWhenEntityMissing` / `TestIntegration_PurgeSpaceTree_CleansTrashWhenEntityMissing`（#2 幂等清理）
  - 更新 `TestWikiTrashExpiryJob_RunOnce` / `TestWikiTrashExpiryJob_StartStopsWithContext` 断言清理计数
  - 新增 `TestTriggerTrashExpiry`（superadmin 放行 + 非管理员 403 + 计数透传）
  - 新增/更新 `TestNotifySubscribers`（评论/编辑订阅通知）
  - `TestAllRoutes_DerivePermissionCode` 覆盖新增运维端点（权限码 `wiki:create`）
- ✅ 前端 `vue-tsc --noEmit` 对 `wiki.ts` / `trash.vue` 无类型错误

---

## 二十、代码评审问题修复（清理链路与通知装配）

针对 HEAD 提交评审发现的 1 个 Critical + 3 个 Major 问题逐项修复。

### 20.1 文档编辑订阅通知装配修复（Critical）
- 问题：`notifySubscribers` 依赖 `WikiRepositoryExtended` 类型断言，而 `module.go` 主服务注入的是 `NewWikiRepository` 基础实现（不实现扩展接口），生产环境下编辑通知静默失效。
- 修复：`module.go` 统一改用 `NewWikiRepositoryExtended`（继承 `WikiRepository`）同时装配主服务与扩展服务；`notifySubscribers` 断言失败分支补 `log` 告警，同类装配错误不再静默。
- 回归测试：`TestNotifyDocumentEdited_BaseService` 按模块装配方式（`NewWikiService` + 扩展仓储）验证编辑通知真实送达订阅者。

### 20.2 评论 @提及去重实现（Major）
- 问题：`CreateComment` 中 `mentioned` 集合收集后从未被消费，既被 @ 又是订阅者的用户收到 `mention` + `comment` 两条重复通知。
- 修复：`notifySubscribers` 参数由单一 `excludeUserID` 改为排除集合 `map[string]struct{}`；评论事件传入「作者 + 全部被 @ 者」，编辑事件传入「编辑者」。
- 回归测试：`TestNotifySubscribers_MentionDedup`。

### 20.3 Purge* 幂等口径修正（Major）
- 问题：幂等化后 `PurgeDocument` / `PurgeSpaceTree` 丢弃了物理删除的 `res.Error`，将「记录不存在（幂等成立）」与「真实 DB 报错」合并处理；后者发生时仍删除回收站记录并返回 nil，软删实体失去唯一清理入口成为永久残留的孤儿数据，且日志误报成功。
- 修复：`RowsAffected == 0` 视为已清理（保持幂等），但 `res.Error != nil` 必须上抛，交由外层事务回滚保留回收站记录下轮重试。

### 20.4 过期清理事务化 / 分批 / 互斥（Major）
- 仓储层 `ExpireTrashItems(ctx, limit)`：新增单轮上限（按 `expires_at` 升序取批）；每个条目单独开启事务（级联物理清除 + 回收站行原子提交），单条失败仅回滚该条目、不中断其余条目，返回「成功数 + 首个错误」。`db` 包新增 `CtxWithTx` 供仓储内部事务复用上下文传递机制。
- 服务层 `ExpireTrashItems`：循环分批直至排空积压（每批 200 条）；进程内 `sync.Mutex.TryLock` 互斥，定时任务与运维端点并发触发时后到者跳过并记日志；上下文被取消（如 HTTP 客户端超时断开）时安全停止，条目级事务保证不会留下半删数据。
- 集成测试：`TestIntegration_ExpireTrashItems_Limit`（分批上限与顺序）、`TestIntegration_ExpireTrashItems_FailureIsolation`（DROP 版本表模拟真实 DB 错误，验证失败条目保留/成功条目不受影响）。

---

## 二十一、评审 Minor 项收尾与多副本加固

承接第二十节，处理剩余 3 个 Minor 项并补齐运维端点的对外文档与多副本互斥。

### 21.1 前后端超管口径统一（Minor-6）
- 问题：前端以 `is_master` 判定超管（回收站「清理过期」按钮可见性），后端 `RequiresMasterAdmin` 却按 JWT 角色 `superadmin` 拦截；旧逻辑仅在**无任何角色**时兜底注入 superadmin，主账号被显式分配其他角色后点按钮会得到 403。
- 修复：`jwt` 包新增 `EnsureSuperadminRole(isMaster, roleCodes)`，登录/OAuth 登录/刷新三处统一保证主账号 JWT 始终携带 `superadmin`；非主账号不受影响。已登出的旧 token 需重新登录后生效。
- 测试：`TestEnsureSuperadminRole`（4 子用例）+ `TestLoginMasterWithAssignedRolesKeepsSuperadmin` 锁死回归。

### 21.2 回收站头部按钮布局（Minor-5）
- `trash.vue` 头部「清理过期」与「刷新」包进 `.head-actions`（flex + gap），不再被 `space-between` 分散到标题两侧。

### 21.3 仅版本递增时发编辑通知（Minor-7 部分）
- `WikiHandler.UpdateDocument` 的 `NotifyDocumentEdited` 改为仅当 `content`/`format` 有实际提交（与 Service 内版本递增条件一致）时触发，避免仅改摘要等元信息也推送编辑通知。
- 测试：`TestWikiUpdateDocument_Success` 补通知断言；新增 `TestWikiUpdateDocument_NoContentNoNotify`。

### 21.4 过期清理 Redis 分布式锁（多副本加固）
- `service` 新增函数式选项 `WithExpireCacheLock(rdb)`；`ExpireTrashItems` 在进程内 TryLock 基础上增加 Redis `SetNX`（key `wiki:trash:expire:lock`，TTL 30min）：未抢到锁直接跳过并记日志；完成后仅当持有者令牌匹配时释放；Redis 不可用/报错时降级为仅进程内互斥（重复执行安全由条目级事务与幂等 purge 兜底）。
- 装配：`wiki.InitModule` 与 `StartWikiTrashExpiryJob` 增传 `rdb`；定时任务与运维端点共用同一锁。
- 测试：`wiki_expire_lock_test.go` 基于 miniredis 验证获锁/释放、他实例持锁跳过且不误删他人锁、无 Redis 降级三种场景。

### 21.5 运维端点对外文档同步
- `docs/api/wiki-api.md` 新增「6.4 手动触发回收站过期清理（运维）」。
- `scripts/openapi_sync.go` 镜像路由补 `POST /wiki/admin/trash/expire` 并执行 `-apply`：`MeteorX-backend.openapi.json`（手工润色 summary/响应体/403 说明）与 `MeteorX-backend.apifox.json`（知识库分组）均已收录该端点，两个 JSON 通过语法校验。

### 21.6 验证状态
- ✅ `go build ./...` / `go vet` 通过
- ✅ `go test -count=1` jwt、auth/service、wiki 全子包通过
- ⏳ 前端构建与本地运行时验证见提交记录

---

## 二十二、异步数据导出与下载中心（Export）

**后端新增 `internal/modules/export` 模块**
- 路由（登录即可，自动权限推导 `export:*`）：
  - `GET /api/v1/exports` → `export:list` 导出任务列表
  - `POST /api/v1/exports` → `export:create` 创建导出任务（指定资源类型）
  - `GET /api/v1/exports/{id}` → `export:read` 任务详情（仅任务可见可下载）
  - `GET /api/v1/exports/{id}/download` 下载 CSV（`Content-Disposition` 附件 + `http.ServeFile`）
  - `DELETE /api/v1/exports/{id}` → `export:delete` 删除任务与文件
- 新建表 `export_jobs`（AutoMigrate 迁移），任务状态机：pending → running → success/failed
- 异步执行：worker 池周期拉取 pending 任务，按资源类型分派注册表中的 exporter，当前支持 user / task / audit_log / wiki_document 四类；队列上限准入：单租户未完结（pending/running）任务数达 `export.max_active_jobs`（默认 10，可配置）时新建返回 429
- CSV 流式分批写出，默认批大小 1000、worker 数 3、保留 7 天；均可通过 `export.*` 配置段覆盖（dir/retention_days/worker_count/batch_size/max_active_jobs）
- 过期清理 `ExportExpiryJob`：每 6 小时扫描超过 `expires_at` 的任务，单轮上限 200 条，删除 CSV 文件与数据库记录
- 租户隔离：任务仅在本租户范围内可见、可下载、可删除

**前端新增 `web-admin/src/views/export/index.vue`（下载中心）**
- 路由 `/export` + 侧边栏「下载中心」菜单；任务列表（状态/分页）、创建导出、状态轮询、下载与删除

---

## 二十三、租户健康度看板（Dashboard）

**后端扩展 `internal/modules/dashboard` 模块**
- `GET /api/v1/admin/dashboard/tenants-health` → `admin:dashboard:list` 全量租户健康度列表（支持分级/关键字筛选，附总数与 healthy/moderate/at_risk 分级汇总）
- `GET /api/v1/admin/dashboard/tenants-health/{id}` 单租户健康度详情
- 评分模型（0-100）：订阅状态（生效 40 / 到期 15 / 取消或无订阅 0）+ 近 7 天活跃（30 × min(1, 审计数/20)）+ 资源足迹（30 × min(1, (用户+任务+文档)/100)）
- 分级：≥70 healthy、≥40 moderate、<40 at_risk；全部基于既有数据表聚合直查，无新增表

**前端新增 `web-admin/src/views/system/dashboard/tenant-health.vue`**
- 路由 `/system/dashboard/tenant-health`（仅平台管理员可见）；分级分布卡片、健康度表格（评分进度条/分级标签）、单租户详情

---

## 二十四、租户套餐配额与 API 限流（Plan）

**后端**
- `Plan` 模型扩展三类配额：`StorageLimit`（字节）、`ApiRateLimit`（次/分钟）、`WikiSpaceLimit`（个）；-1 表示不限，未显式设置（0）创建时归一化为 -1
- 租户 API 限流中间件 `internal/middleware/tenant_rate_limit.go`：自然分钟窗口 Redis `INCR` + 首次 `EXPIRE 60s`（key `tenant:rate:{id}:{minute}`），本地 5 分钟缓存套餐速率避免查库；超限返回 429 + `Retry-After`；无租户/不限/Redis 不可用/查询失败均 fail-open 放行
- 文件上传存储配额校验：`FileService.Upload` 写入前比对 `SumSizeByTenant` 已用 + 本次大小与配额，超限返回 507（Insufficient Storage）
- Wiki 空间创建配额校验：`CreateSpace` 进入事务前比对 `CountSpacesByTenant` 与配额，超限返回 402（`ErrTenantLimit`）
- 当前套餐用量回填：`GetCurrentPlan` 可选注入 `StorageUsageProvider`/`WikiUsageProvider` 填充 `current_storage`/`current_wiki_spaces`（best-effort，失败保持零值）
- 配额提供者解耦：`plan.NewQuotaService(db)` 返回具体 `*PlanService`，以结构类型同时满足 middleware/file/wiki 各消费方定义的最小配额接口，避免模块间循环依赖

**前端**
- 套餐管理（`system/plan/index.vue`）：列表新增存储配额/API 速率/Wiki 空间三列；新建/编辑对话框支持三项配额配置，存储以 GB 输入并与字节互相换算，-1 统一显示「不限」
- 租户套餐配置弹窗（`system/tenant/index.vue`）：用户/存储/Wiki 空间三条用量进度条，超 100% 显示异常色

---

## 二十五、双因素认证（2FA / TOTP）

为账号登录提供基于时间的一次性口令（RFC 6238 TOTP）二次校验，纯标准库实现，不引入第三方依赖。

**后端**
- 新增算法包 `pkg/totp`：`GenerateSecret`（Base32 密钥）、`Now`/`Validate`（±1 时间窗容错）、`OTPAuthURL`（标准 `otpauth://`  provisioning URI，供 authenticator App 扫码）
- `User` 模型/仓储扩展两字段：`totp_secret`（加密存储）、`totp_enabled`；新增 `UpdateTOTP(ctx, id, secret, enabled)`
- 认证服务（`internal/modules/auth/service`）：
  - `Login` 在租户状态软着陆校验后追加 TOTP 分支——已启用而缺 OTP → 返回 `require_2fa`；OTP 错误 → 计入账户锁定失败次数并返回 `require_2fa`；正确 → 放行签发令牌
  - `Setup2FA`：生成密钥暂存 Redis（`2fa:pending:{userID}`，TTL 15 分钟），返回密钥 + `otpauth://` URL，不落库，避免持久化未验证密钥
  - `Enable2FA`：校验 Redis 暂存密钥对应的验证码后 `UpdateTOTP(enabled=true)` 并清除暂存
  - `Disable2FA`：校验当前密钥验证码后 `UpdateTOTP(secret="", enabled=false)`
- 路由 `RegisterTwoFactorRoutes`：`POST /api/v1/auth/2fa/setup`、`POST /api/v1/auth/2fa/enable`、`POST /api/v1/auth/2fa/disable`（均需登录态，从 `contextx.GetUserID` 取用户）
- 错误映射：Redis/存储不可用 500、密钥过期 400、重复启用/未启用 400

**前端**
- 登录页（`views/login/index.vue`）：后端返回 401 + `require_2fa` 时二段式揭示 6 位验证码输入框，随账号密码一并重提
- 个人中心（`views/profile/index.vue`）：2FA 卡片展示启用状态，开启走「显示密钥 + otpauth 链接 + 复制 + 填验证码确认」三步绑定，关闭需当前验证码
- API（`api/auth.ts`）：登录参数增 `otp`、错误数据增 `require_2fa`、新增 `setup2FA`/`enable2FA`/`disable2FA`

---

## 二十六、慢查询看板（Dashboard）

以 GORM Logger 中间件形式在进程内采集超过阈值的 SQL，无需依赖外部慢查询日志，供平台管理员实时查看。

**后端新增 `internal/dbmonitor`**
- `Logger` 实现 `gorm logger.Interface`，`Trace` 中耗时 ≥ 阈值则记入内存环形缓冲区（默认容量 200，时间倒序），否则仅累计总量；其余方法委托被包装的原始 logger
- `store` 线程安全环形缓冲，累计 total/slow/max/avg 统计；`Snapshot()` 返回统计 + 最近慢查询明细拷贝，`Reset()` 清空
- 配置 `database.slow_query_ms`：`GetSlowQueryThreshold()` 未配置（0）默认 500ms、负值关闭采集
- 装配（`bootstrap/database.go`）：阈值 > 0 时 `dbmonitor.Init` 并用 `dbmonitor.NewLogger` 包装 GORM 日志器
- 端点（`internal/modules/dashboard`）：`GET /api/v1/admin/dashboard/slow-queries` → `admin:dashboard:list`、`POST /api/v1/admin/dashboard/slow-queries/reset` → `admin:dashboard:create`

**前端新增 `web-admin/src/views/system/dashboard/slow-queries.vue`**
- 路由 `/system/dashboard/slow-queries`（仅平台管理员可见）；SQL 总数/慢查询数/最大耗时/平均耗时四指标卡 + 慢查询明细表（耗时着色、SQL、Trace ID、状态标签）+ 5 秒自动刷新 + 重置统计

---

## 二十七、前端暗色主题

**前端**
- 新增 `composables/useTheme.ts`：`localStorage` 持久化 `meteorx_theme`，模块导入即应用，切换 `html.dark` 类与 `data-theme`
- 引入 Element Plus 官方暗色变量 `element-plus/theme-chalk/dark/css-vars.css`，绝大多数组件自动适配暗色
- `App.vue` 定义 `:root` / `html.dark` 品牌级 CSS 变量（`--mx-header-bg` 等），布局 chrome 硬编码色改用变量
- 顶栏（`DefaultLayout.vue`）新增 Sunny/Moon 主题切换按钮

---

## 二十八、前端国际化框架落地（i18n）

**前端**
- `composables/useI18n.ts`：新增 `localStorage` 语言持久化（`meteorx_locale`），初始化读取回退 `zh-CN`，`setLocale` 写入
- `App.vue`：`el-config-provider` 的 `:locale` 改为按当前语言 computed（zh-CN→zhCn / en-US→enUs），Element Plus 内置文案随语言切换
- 顶栏新增中/EN 语言切换入口

---

## 二十九、白标自定义品牌（White-label）

客户端品牌定制：管理员可在界面上修改产品名、Logo 与主题主色，持久化于浏览器并即时应用（服务端级白标见「三十、服务端多租户白标」）。

**前端**
- 新增 `composables/useBrand.ts`：`BrandConfig{name, logo, primaryColor}`，`localStorage` 持久化 `meteorx_brand`；`applyBrand` 设置 `document.title` 并覆盖 `--el-color-primary`/`--mx-brand-primary`
- 布局侧边栏 Logo 区随品牌名/图动态渲染；`App.vue` 启动即 `applyBrand()`
- 顶栏用户下拉新增「品牌设置」入口（仅管理员可见），对话框支持名称/Logo/主色（`el-color-picker`）编辑、保存与恢复默认

---

## 三十、服务端多租户白标（Server-side White-label）

将品牌配置从纯客户端 `localStorage` 升级为服务端按租户下发并全局应用：租户管理员在「租户设置 → 品牌定制」保存的 Logo/主色实时生效，品牌名复用 `Tenant.Name`。采用**服务端优先、本地回退**策略。

**后端**
- `tenant/dto/tenant_settings_dto.go`：`TenantSettingsResp` 新增 `tenant_name` 字段（best-effort 下发）
- `tenant/service/tenant_settings_service.go`：新增 `tenantNameProvider` 抽象与 `WithTenantNameProvider` 函数式选项，`GetSettings`/`UpdateSettings` 经 `fillTenantName` best-effort 填充租户名（provider 为 nil 或查询失败保持空值，不阻断主流程）
- `tenant/module.go`：`NewTenantSettingsHandler` 注入 `TenantRepository`

**前端**
- `stores/brand.ts`（原 `composables/useBrand.ts` 迁入）：`applyServerBrand`（服务端字段优先、空值回退默认并缓存至 localStorage）与 `syncBrandFromServer`（登录后拉取 `GET /tenant-settings` 应用，失败回退本地/默认品牌）
- `App.vue`：登录态变化与已登录挂载时触发 `syncBrandFromServer`
- `views/tenant/settings/index.vue`：保存成功后即时 `applyServerBrand` 使新品牌全局生效
- `api/modules/tenant.ts`：`TenantSettings` 接口补充 `tenant_name`

---

## 三十一、前端全局状态迁移至 Pinia store

将 `useTheme`/`useBrand`/`useI18n` 从“模块级单例 `ref` + localStorage”迁移为 Pinia setup store，与已有 `stores/`（`user`/`app`/`notification`）风格统一；`useTableList`（页面局部列表态）与 `useWebSocket`（携 `onUnmounted` 生命周期、按使用方隔离）仍为合理 composable，不迁为全局 store。

**分层策略**
- 新增 `stores/theme.ts`、`stores/brand.ts`、`stores/i18n.ts`：持有全局响应式状态 + actions（含主题/品牌/语言的 DOM 应用与持久化）
- `composables/useTheme.ts`/`useBrand.ts`/`useI18n.ts` 保留为薄封装（基于 `storeToRefs` 转发 store），对外 API 与 `.value` ref 语义不变，三个消费方（`App.vue`/`DefaultLayout.vue`/`settings`）零改动
- 首屏无闪烁：将原“模块导入即副作用”改为 `main.ts` 挂载前调用纯函数 `applyThemeToDOM/applyBrandToDOM/applyLocaleToDOM`（配合 `readStoredXxx`，不依赖 Pinia），store 实例化时读同一份 localStorage，状态一致

**测试**
- `composables/useI18n.test.ts`：`beforeEach` 新增 `setActivePinia(createPinia())`
- 新增 `stores/theme.test.ts`、`stores/brand.test.ts`（覆盖持久化、服务端优先应用与失败回退）
- 验证：`vue-tsc --noEmit` ✅、`vitest` 118 例全过 ✅、`eslint` ✅、`vite build` ✅

---

## 三十二、i18n 铺开至全局外壳 + 路由平台级门禁修复

**i18n 铺开**
- `layouts/DefaultLayout.vue` 全局外壳（侧边栏菜单/面包屑/顶栏提示/用户下拉/退出确认/品牌对话框）统一改用 `t()`，复用早已存在的 `nav.*` 等 key

---

## 三十三、导出模块多格式支持（CSV / XLSX / JSON / PDF）

原导出模块仅支持 CSV。本次将「数据生产」与「文件格式」解耦，一套数据可导出为四种格式。

### 33.1 解耦设计：RowSink 接口
- `service/exporter/exporter.go`：`Exporter.Stream` 签名由 `*csv.Writer` 改为 `RowSink` 接口（`WriteHeader` / `WriteRows` / `Close`）；新增 `NewSink(format, dst)` 工厂按格式构造对应 sink。
- 四个资源导出器（user / task / audit_log / wiki_document）保持不变的 SQL 与分页逻辑，仅将写 CSV 的 `writeHeader`/`writeRows` 改为调用 sink，零业务逻辑改动。
- 新增四种 sink：
  - `sink_csv.go`：纯 CSV（不含 BOM，由 worker 在文件头写入 UTF-8 BOM，保证 Excel 中文不乱码）。
  - `sink_json.go`：以 `[{列名: 值}, ...]` 数组输出（stdlib `encoding/json`，零依赖）。
  - `sink_xlsx.go`：基于 `github.com/xuri/excelize/v2` 的 `StreamWriter` 流式写出，内存友好。
  - `sink_pdf.go`：基于 `github.com/jung-kurt/gofpdf` 渲染带边框表格；因数据含中文，必须可定位 CJK 字体（见 33.3）。

### 33.2 字段与接线
- `model.ExportJob.Format`、`repository.ExportJobPO.Format` 新增格式字段（GORM AutoMigrate 自动加列）。
- `dto.CreateExportReq.Format`（`omitempty,oneof=csv xlsx json pdf`）、`ExportJobResp.Format`。
- `service.CreateExport` 校验/默认化格式（非法格式回落 `csv`），`worker.runExport` 按格式生成 `.{ext}` 文件名并选择 sink，仅 CSV 写 BOM。
- `handler.Download` 按 `format` 设置 `Content-Type` 与 `filename.{ext}`。
- 前端 `web-admin`：`api/modules/export.ts` 增加 `ExportFormat`、`CreateExportParams.format`、`ExportJobItem.format`；`views/export/index.vue` 新建对话框增加「导出格式」下拉，列表增加格式列，下载文件名按格式扩展名。

### 33.3 PDF 中文字体约束（重要）
- gofpdf 核心字体不支持中文，PDF 必须内嵌/可定位 CJK 字体，否则中文乱码。
- 字体解析顺序：环境变量 `METEORX_EXPORT_PDF_FONT`（可选 `METEORX_EXPORT_PDF_FONT_NAME`）→ 启动配置 `exporter.SetPDFFont(path, name)` → 常见系统路径（Windows `msyh.ttf`、Linux Noto/WQY、macOS 等）。
- 三者皆不可得时，PDF 任务被 worker 标记为 `failed` 并记录「PDF 导出需要 CJK 字体」，**绝不输出乱码**；`.ttc` 集合不被 `AddUTF8Font` 支持，需单一 `.ttf`。

### 33.4 验证状态
- ✅ `go build ./...` 通过；`go mod tidy` 干净（新增 `excelize/v2`、`gofpdf`）
- ✅ `go vet ./internal/modules/export/...` 通过
- ✅ `go test ./internal/modules/export/...` 全量通过
  - 新增 `sink_test.go`：`TestJSONSink` / `TestJSONSink_Empty` / `TestXLSXSink`（excelize 回读校验）/ `TestNewSink`（含 PDF 无字体报错、非法格式报错）
  - 新增 `TestService_CreateExport_Format`（显式格式 / 默认 csv / 非法回落）
  - 更新 `exporter_test.go`、`worker/pool_test.go` 适配 `RowSink`
- ✅ 前端 `vue-tsc --noEmit` 对 `export.ts` / `index.vue` 无类型错误
- 文档：`docs/api/export-api.md` 补充 `format` 字段、按格式 Content-Type 表、4.5 格式支持与 PDF 字体说明
- `stores/i18n.ts` 补齐缺失 key：`nav.task/export/invitation/tenantHealth`、`theme.toLight/toDark`、`header.wsConnected/wsDisconnected`、`brand.*`、`auth.logoutConfirm/loggedOut`、`common.tip`

**路由守卫**
- `router/index.ts`：抽出纯函数 `hasRouteAccess(meta, user)`，`beforeEach` 强制校验此前声明却从未检查的 `meta.requireMaster`（仅平台超管 `is_master` 放行），修复租户管理员持 `admin:*` 权限码越级进入平台页的缺口
- `afterEach` 标题改用当前白标品牌名，修复硬编码 `- MeteorX` 覆盖租户品牌的 bug
- 新增 `router/guard.test.ts`（5 例）覆盖 permission 缺失拦截、requireMaster 对非超管拦截/超管放行
- 验证：`vue-tsc` ✅、`vitest` 123 例全过 ✅、`eslint` ✅、`vite build` ✅

---

## 三十四、OAuth 多 Provider 支持（国内 IdP + 企业 OIDC + 邮箱补全）

原 OAuth 模块仅支持 Google / GitHub，且 provider 以三处 `switch` + 私有函数硬编码。本次重构为**策略 + 注册表**架构，新增 7 类 Provider，并补齐「第三方未返回邮箱」的强制补全闭环。

### 34.1 策略 + 注册表架构（核心重构）
- `service/provider.go`（新增）：
  - `Provider` 接口：`Name()` / `IsEnabled()` / `RedirectURL(state)` / `Exchange(ctx, code)`，将「数据生产」与「身份提供商」解耦。
  - `ProviderRegistry`：`NewProviderRegistry(cfg)` 依据配置构建；`Get(name)` 按名取用；`Enabled()` 返回已启用 provider 的展示元信息（按 ID 排序）。
  - 已知固定 provider（google/github/wechat/feishu/dingtalk/qq/microsoft）**始终登记**，其 `IsEnabled()` 同时要求 `enabled=true` 且已填写 `client_id`；未配置时处于「禁用态」而非「不存在」，保持既有错误语义。
  - 通用 OIDC 仅在配置（填写 `client_id`）时登记，provider 标识为 `oidc_<key>`。
  - 共享 HTTP 辅助：`httpPostForm` / `httpGetJSON` / `standardUserInfo` / `standardOAuthProvider`（标准授权码流程：表单换 token + Bearer 取 userinfo）。
- `service/oauth_service.go`：删除三处 `switch`（`GetRedirectURL` / `Login` / `BindOAuth`）与全部私有 `buildXxx` / `exchangeXxx` / `fetchGitHubPrimaryEmail`，改为 `registry.Get(provider)` + 接口调用；服务持有 `registry` 字段并在 `NewOAuthService` 时构建。

### 34.2 各 Provider 实现
| Provider | 文件 | 关键适配点 |
|----------|------|-----------|
| Google | `provider.go` 内 `standardOAuthProvider` | 标准 OIDC，`openid email profile` |
| GitHub | `service/github.go` | 令牌端点需 `Accept: application/json`；邮箱空时查 `/user/emails` |
| 微信 | `service/wechat.go` | `qrconnect` 扫码授权；ProviderID 优先 `unionid`；**不返回邮箱** |
| 飞书 | `service/feishu.go` | 开放平台 v1 接口，令牌/用户信息均为 JSON 信封（`data` 内层） |
| 钉钉 | `service/dingtalk.go` | 新版 OAuth2.0，`clientId/clientSecret/grantType` 字段名 |
| QQ | `service/qq.go` | 令牌端点返回 urlencoded（非 JSON）；openid 端点返回 JSONP 包裹；**不返回邮箱** |
| Microsoft Entra | `service/microsoft.go` | 目录租户 ID 注入授权/令牌端点，userinfo 走 Graph OIDC |
| 通用 OIDC | `provider.go` 内按 `oidc_<key>` | 完全配置化 `auth/token/userinfo` 端点与 scopes |

### 34.3 邮箱补全闭环（`need_complete_email`）
- 微信 / QQ 不返回邮箱。`Login` 不再拦截空邮箱（移除 `ErrOAuthEmailRequired`），而是签发带 `email_pending` 声明的临时令牌（`jwt.GenerateTokenEx`，`CustomClaims.EmailPending`）。
- 响应新增 `OAuthLoginResponse.NeedCompleteEmail`；前端 `OAuthCallback.vue` 据此暂存 `token/user/permissions` 并跳转 `/oauth/complete-email`。
- 新增公开端点 `POST /api/v1/auth/oauth/complete-email`（`handler.CompleteEmail` / `service.CompleteEmail`）：校验 `email_pending` 声明 → 校验邮箱唯一性 → 写回 `users.email` → 换发正式令牌。
- 新增 `GET /api/v1/auth/oauth/providers`：返回已启用 provider 列表，使前端登录/绑定按钮**与后端解耦**。

### 34.4 配置与前端
- `config.OAuthConfig` 新增 `WeChat/Feishu/DingTalk/QQ/Microsoft` 字段与 `OIDC map[string]OIDCProviderConfig`；新增 `MicrosoftOAuthConfig`（含 `Tenant`）、`OIDCProviderConfig`（含 `AuthURL/TokenURL/UserInfoURL/Scopes/DisplayName`）。
- `config/config.yaml` 增加各 provider 配置示例，**默认全部 `enabled: false`**，不影响现有部署。
- `dto`：移除 `OAuthLoginRequest` / `BindOAuthRequest` / `UnbindOAuthRequest` 中写死的 `oneof=google github`（provider 白名单改由服务层注册表校验）；新增 `CompleteEmailRequest`、`ProviderMeta`。
- 前端：
  - `api/modules/oauth.ts`：新增 `listOAuthProviders`、`completeOAuthEmail`（携带临时令牌于 Authorization 头）。
  - `api/auth.ts`：`OAuthLoginResult` 增加 `need_complete_email`。
  - 新增 `web-admin/src/providers.ts`：provider 品牌色映射（未知/自定义 OIDC 回落中性色）。
  - `views/login/index.vue`、`views/profile/index.vue`：写死的 Google/GitHub 按钮改为 `v-for` 动态渲染（label 取后端、品牌色取前端）。
  - 新增 `views/login/CompleteEmail.vue` 与路由 `/oauth/complete-email`（public）。

### 34.5 验证状态
- ✅ `go build ./...` 通过
- ✅ `go test ./internal/modules/oauth/...`、`./internal/config/...`、`./internal/common/jwt/...`、`./internal/middleware/...` 全绿
  - 新增 `service/providers_test.go`：注册表登记、各 provider 跳转 URL、`extractQQOpenID`（JSONP）、`standardUserInfo`（id/sub 兼容）、`ListProviders` 排序与标签、OIDC 注册、令牌交换失败哨兵错误
  - 新增 `TestListProviders_OnlyEnabled` / `TestNewProviderRegistry_OIDC`
  - 重写 `TestGoogleRedirectURL_*` / `TestGitHubRedirectURL_*` 改用注册表
  - `handler` stub 补齐新接口方法；移除失效的 `TestCallback_EmailRequired`
  - `middleware/auth_test.go` 覆盖 `email_pending` 临时令牌门禁（豁免路径放行 / 其余 403）
- ✅ 前端 `vue-tsc --noEmit`、`eslint`、`vite build` 通过

### 34.6 安全加固（与本功能同步落地）
- **空邮箱串号修复**：`findOrCreateUser` 仅在 `info.Email != ""` 时才按邮箱匹配已有账号，杜绝微信/QQ 等无邮箱 provider 命中任意 `email=''` 用户导致的越权登录。
- **用户名派生兜底**：无邮箱时用户名后缀改用 `ProviderID`，避免生成 `wechat_` 这类退化用户名。
- **统一拒绝空 ProviderID**：`Login` / `BindOAuth` 在交换后校验 `ProviderID` 非空，否则返回 `ErrOAuthUserInfoFailed`。
- **`email_pending` 服务端门禁**：认证中间件对带 `email_pending` 声明的临时令牌仅放行 `complete-email` / `token/refresh` 白名单，其余一律 403（`X-Error-Code: EMAIL_PENDING`），防止仅靠前端拦截被绕过；刷新令牌时保留 pending 标记，不会因此提权。
- **错误脱敏**：各 provider 令牌/用户信息解析失败改为「脱敏日志（`safeBodyPreview`）+ 哨兵错误」，不再向客户端回显上游响应体，避免泄露 IdP 细节。
- **启用态前置校验**：`Login` / `BindOAuth` 在发起上游交换前先校验 provider 已启用。
- **出站请求加固**：统一复用带 10s 超时的共享 HTTP 客户端，并限制响应体大小。
- **前端 Authorization 注入加固**：`request.ts` 仅在调用方未显式设置 `Authorization` 时才注入全局 token，避免补全邮箱等携带临时令牌的请求被登录态 token 覆盖。

---

## 三十五、OAuth 提供商运行时配置（DB 热更新 + 凭据加密 + 管理后台）

第三十四节落地了多 Provider 架构，但配置依赖 `config.yaml`，**每次新增或修改 provider 都要改文件并重启**。本节把 provider 配置下沉到数据库，提供平台超管后台，实现免重启热生效。

### 35.1 数据层：`oauth_providers` 表
- 领域模型 `model.OAuthProvider`（含 `Kind/Code/DisplayName/Enabled/ClientID/ClientSecret/RedirectURL/Tenant/AuthURL/TokenURL/UserInfoURL/Scopes`）。
- PO `oauthProviderPO`（`repository/oauth_provider_repository.go`）：
  - `Code` 唯一索引；`Scopes []string` 使用 `serializer:json`（GORM 原生序列化）。
  - `ClientSecretEnc` 存放密文，**绝不存明文**。
  - 迁移必须用 PO：`repository.AutoMigrateProvider(db)`（领域模型含 `[]string`，GORM 无法直接迁移）。
  - `Update` 采用「先查后 `Save`」：`Save` 会更新全部列（含零值），同时规避手写列名与 GORM 命名策略不一致的问题（`UserInfoURL` → `user_info_url`，URL 是内置缩写）。
- 元数据表 `oauth_provider_meta`（键值对）：记录 `seeded_from_config` 种子化标记，**避免管理员删光配置后被重新种子化**（「删除后复活」问题）。

### 35.2 凭据加密（AES-256-GCM）
- `pkg/crypto/secret_cipher.go`：`SecretCipher` 基于 AES-256-GCM，主密钥为 base64 编码的 32 字节随机值。
  - 密文格式 `enc:v1:<base64(nonce||ciphertext)>`；每次加密使用随机 nonce，同一明文产生不同密文。
  - 无前缀的历史明文**读取时原样返回**（平滑升级），但保存新 secret 时一律加密。
  - `MaskSecret` 脱敏：仅保留末尾 4 位。
- 主密钥来源：`oauth.secret_key`（`METEORX_OAUTH_SECRET_KEY`），生成方式 `openssl rand -base64 32`。
- **未配置主密钥时，后台保存 client_secret 直接返回 422 拒绝**，绝不落明文；启动时告警提示。

### 35.3 配置来源统一为 ProviderSpec
- `service/provider.go` 新增 `ProviderSpec` 作为「配置来源」与「Provider 实现」之间的中介：
  - `ProviderSpecsFromConfig(cfg)`：config.yaml → Spec 列表（供种子化与兜底）。
  - `BuildProvider(spec)`：按 `Kind` 构建具体 Provider（8 类）。
  - `NewProviderRegistryFromSpecs(specs)`：Spec 列表 → 注册表。
- `ProviderRegistry` 改为**可热重载**：`RWMutex` 保护内部快照，`Replace(providers, labels)` 原子替换；`Get`/`Enabled` 走读锁。
- `OAuthService` 持有 `providerRepo` + `cipher`（均可为 nil，nil 时退回 config.yaml，便于测试与降级）。

### 35.4 热重载与种子化
- `service.ReloadProviders(ctx)`：
  1. `Count == 0` 且未标记种子化 → 从 config.yaml 种子化（**只种子化填了 `client_id` 的项**，不造空壳）→ 写入 `seeded_from_config` 标记。
  2. 读取全部行 → 解密 secret → 转 Spec → `Replace` 注册表。
- 触发时机：**写入即时触发**（CRUD 成功后调用）+ **定时 30s 兜底刷新**（`ProviderReloadInterval`），多副本部署下任一副本写入后其余副本在 30s 内收敛。
- 后台协程由 `bootstrap` 传入的 `ctx` 控制，随进程优雅退出。

### 35.5 平台级管理接口
- 路由 `/api/v1/admin/oauth-providers`（挂 `AutoRequirePermission`，权限码由 `singularize` 新增 `oauth-providers → oauth_provider` 映射推导）：
  - `GET /` 列表、`POST /` 新建、`GET /{code}` 详情、`PUT /{code}` 更新、`DELETE /{code}` 删除、`PUT /{code}/status` 启停。
- 权限码 `admin:oauth_provider:{list,create,read,update,delete,status}`，已在 `rbac/permissions.go` 注册。
- 服务层校验（`normalizeProviderReq`）：
  - `Kind` 必须合法；内置类型的 `Code` 必须等于 `Kind`；OIDC 的 `Code` 须匹配 `^oidc_[a-z][a-z0-9_]{2,63}$` 且唯一。
  - `client_id`、`redirect_url` 必填；OIDC 必须填 `auth_url/token_url/userinfo_url` 且为 http(s)。
  - `client_secret` 传空表示「保持原值」（更新场景），支持密钥轮换。
  - 错误分类：`ErrOAuthProviderInvalid`(400) / `ErrOAuthProviderCodeTaken`(409) / `ErrOAuthSecretKeyNotConfigured`(422) / `ErrOAuthProviderNotFound`(404)。
- 响应**只返回脱敏 secret**（`client_secret_mask` + `has_secret`），明文仅在内存中短暂存在。

### 35.6 前端管理页
- 新增 `web-admin/src/api/modules/oauth-provider.ts`（CRUD + 类型选项）。
- 新增 `web-admin/src/views/system/oauth-provider/index.vue`：列表（登录方式/标识/类型/Client ID/脱敏 Secret/回调地址/状态）、新建/编辑对话框（按类型动态显示 Microsoft 目录租户或 OIDC 端点字段）、启停、删除；内置类型自动填充标识与默认回调地址。
- 路由 `/system/oauth-provider`（`permission: admin:oauth_provider:list` + `requireMaster`），侧边栏「平台管理 → 第三方登录」。
- 页面显著提示「配置即时生效，无需重启」与「最多 30 秒同步至全部副本」。

### 35.7 验证状态
- ✅ `go build ./...` 通过
- ✅ 测试全绿：
  - `pkg/crypto`：往返加解密、空明文、nonce 唯一性、错误密钥、历史明文、未配置主密钥、`MaskSecret`
  - `repository`：`TestProviderRepo_*`（sqlite）：增删改查、按创建时间排序、状态切换、计数
  - `service`：`TestReloadProviders_SeedsFromConfig` / `_DBOverridesConfig` / `_NoStoreIsNoop`、`TestCreateProvider_Validation`（6 类非法输入）、`TestCreateProvider_RejectsPlaintextWithoutKey`、`TestCreateProvider_OIDC`、`TestUpdateProvider_KeepsSecretWhenOmitted`（含轮换）、`TestDeleteProvider`、`TestSetProviderStatus`、`TestListProviderConfigs_MasksSecret`
  - `handler`：`TestProviderAdmin_*`（列表/新建/校验失败/非法输入 400/标识冲突 409/缺主密钥 422/更新/删除/启停/详情/404）
  - 回归：`middleware`、`bootstrap`、`rbac`、`config` 全绿
- ✅ 前端 `vue-tsc --noEmit`、`eslint` 通过
- 运维提示：生产环境务必配置 `METEORX_OAUTH_SECRET_KEY`；密钥遗失将无法解密已存储的 client_secret（需重新录入）。

### 35.8 配置方式对照
| 场景 | 做法 |
|------|------|
| 首次升级（已有 config.yaml 配置） | 自动种子化到 DB，原有配置不丢失，后台可见 |
| 日常新增/调整 provider | 后台「第三方登录」页面操作，即时生效，无需重启 |
| 轮换 client_secret | 编辑时填写新值保存；留空表示不变 |
| 彻底改用 config.yaml | 删除 DB 中对应配置即可回落（表非空不会再次种子化） |

## 三十六、任务模块 AI 能力（可选叠加层 + OpenAI 兼容 + 自动降级）

为任务模块叠加四类 AI 辅助能力：**自然语言建任务、任务摘要/周报、优先级/标签建议、描述扩写/拆解**。定位为**可选叠加层、默认关闭**——走 OpenAI 兼容接口，配置缺失或调用失败**自动降级**，任务 CRUD 永不依赖 AI。

### 36.1 通用 AI 抽象层 `internal/ai/`（跨模块可复用）
- `provider.go`：定义 `Message{Role,Content}`、`ChatRequest{Model,Messages,Temperature,MaxTokens,JSONMode}`、`ChatResponse{Content,Model}` 与 `Provider` 接口（`Chat(ctx, ChatRequest) (ChatResponse, error)`）。
- `openai.go`：OpenAI 兼容实现——`POST {base_url}/chat/completions`、`Authorization: Bearer {api_key}`、带超时的 `net/http`；`JSONMode` 时置 `response_format={"type":"json_object"}`；响应体 `io.LimitReader` 限读 2MB；非 2xx 解析上游 `error.message` 包装为错误、**不外泄上游细节**。
- `client.go`：`Client` 实现 `Provider`，含 `Usable()`（Enabled 且 APIKey 非空）、`Enabled()`、`Model()` 与哨兵错误 `ErrDisabled`；`New(cfg config.AIConfig)` 按 getter 回落默认。`Chat` 在 `!Usable()` 时直接返回 `ErrDisabled`（**降级点**）。
- 配置来源为**静态 config.yaml + 环境变量**（非 DB 热更新）：`Provider` 接口保留可插拔性，但无需 §35 式 registry/热重载（配置静态，避免过度设计）。

### 36.2 配置 `internal/config`
- `config.go`：`Config` 增 `AI AIConfig`（`mapstructure:"ai"`）；`AIConfig{Enabled,BaseURL,APIKey,Model,Temperature,MaxTokens,Timeout}`；getter 零值回落默认（`base_url=https://api.openai.com/v1`、`model=gpt-4o-mini`、`max_tokens=512`、`timeout=20s`；`temperature` 仅负值回落 0.2，0 为合法确定性值）；`IsUsable()=Enabled && trim(APIKey)!=""`。
- `config.yaml`：新增默认注释的 `ai:` 段（`enabled: false`）；`.env.example` 追加 `METEORX_AI_ENABLED/BASE_URL/API_KEY/MODEL/TEMPERATURE/MAX_TOKENS/TIMEOUT`。
- `bootstrap/config.go`：`envBindings` 增 7 条 `ai.* → METEORX_AI_*`（因 config.yaml 中该段为注释项，`Unmarshal` 不应用 `AutomaticEnv`，须显式 `BindEnv`）。

### 36.3 装配与路由
- `bootstrap/router.go`：`task.InitModule(r, db, cfg.AI)`。
- `task/module.go`：构造 `ai.New(aiCfg)` → `service.NewTaskAIService(aiClient, repo)` → `handler.NewTaskAIHandler(aiSvc)`；`/ai/*` 挂在既有**受保护** `/tasks` 分组内（需登录，无新 RBAC 权限码），静态段先于 `/{id}` 注册：
  - `GET  /api/v1/tasks/ai/capabilities`、`POST /api/v1/tasks/ai/parse|suggest|expand|summary`。

### 36.4 AI 服务 `service/task_ai_service.go`
- `TaskAIService` 依赖 `aiProvider` 接口（便于 stub 测试）+ `repository.TaskRepository`（摘要只读用户任务）。
- 四类能力统一：`!Usable()` → `ErrAIUnavailable`；构造任务专属 prompt（`ParseDraft` 注入 `now` 以正确解析「下周五/三天后」等相对日期）；`JSONMode` 调用；`json.Unmarshal` + 归一化（非法枚举回落）；解析失败 → `ErrAIRuntime`，**绝不写库/panic**。
- `Summarize` 按 `scope=today/week/overdue/all` + 归属过滤取任务（Page1/PageSize100），无任务时短路返回「暂无任务」，不空跑模型。
- 错误：`ErrAIUnavailable`（未启用/降级）、`ErrAIRuntime`（上游失败/坏响应）。

### 36.5 DTO 与 Handler
- `dto/task_ai_dto.go`：`ParseTaskReq{Text max=500}`、`TaskDraftResp{Title,Priority,DueDate,Tags,Recurrence}`（对齐 `CreateTaskReq` 便于预填，不含 assignee/visibility 敏感项）；`SuggestReq/Resp`、`ExpandReq/Resp`、`SummaryReq{Scope oneof=today week overdue all}/Resp{Text,Highlights,SuggestedFocus,Total}`、`AICapabilityResp{Enabled,Model}`（JSON 字段下划线命名）。
- `handler/task_ai_handler.go`：复用 `currentIdentity(r)` 与 `validator.ValidateJSON`；错误映射 `ErrAIUnavailable`→**503**「AI 能力未启用」、`ErrAIRuntime`→**502**「AI 服务暂不可用，请稍后重试」、校验失败 400、成功 `response.Success`。

### 36.6 前端 `web-admin/`
- 新增 `src/api/modules/task-ai.ts`（复用 `@/api/request`，零新依赖）：`getAICapabilities/parseTaskByNL/suggestTaskFields/expandTaskDescription/summarizeTasks`。
- `src/views/task/index.vue`：新建/编辑对话框接入——①新建态「AI 解析」输入 → 预填表单；②优先级旁「AI 建议」（同时回填标签）；③描述旁「AI 扩写/拆解」；④工具栏「AI 摘要」对话框（范围切换 + 正文/重点/优先处理建议）。所有入口以 `onMounted` 拉 `getAICapabilities` 门控：`enabled=false` 时按钮禁用并 tooltip 说明；503/502 走既有 toast，不阻断手动填写。

### 36.7 验证状态
- ✅ `go build ./...`；`go test ./internal/ai/... ./internal/modules/task/...` 全绿（`openai_test.go`：httptest 正常/非 2xx/`!Usable`→`ErrDisabled`；`task_ai_service_test.go`：stub `Provider` 覆盖四类能力与降级、坏 JSON→`ErrAIRuntime`，`now` 作固定入参不随真实时钟翻转）。
- ✅ 前端 `npm run build`、`npm run lint`、`npm run test`（含 `task-ai.test.ts` 6 例）全绿。
- 手工：`ai.enabled=false` 时 capabilities 返回 disabled、端点 503、前端按钮禁用；配好 key 后四类调用返回合理结果；上游超时/坏响应返回 502 且不影响手动创建任务。

---

## 三十七、用户批量导入（User Import）

新增「上传 CSV / XLSX 一次性建号」能力，覆盖租户端与平台管理端。核心取舍是**全量事务**：先整份校验、再一次性写入，任一行不通过就零写入，避免"导入到一半失败了，得手工清理"。

### 37.1 解析层 `internal/modules/user/importer`
- `Row{LineNo,Username,Nickname,Email,Department,Position,RoleCodes}`；`LineNo` 是文件物理行号（表头为第 1 行），用于把错误精确回传到用户看到的行。
- `Parse(filename, data) ([]Row, []string, error)` 按扩展名分发 `.csv` / `.xlsx`（无扩展名按 CSV 兜底）；CSV 兼容 UTF-8 BOM、CRLF、列数不齐与含引号字段，并用 `csv.Reader.FieldPos` 取真实行号（`encoding/csv` 会静默跳过空行，用记录下标当行号会错位）。
- `headerAliases` 同时接受英文列名与常见中文列名（`用户名` / `登录名` / `账号`、`邮箱` / `电子邮件`、`部门`、`职位` / `岗位`、`角色` / `角色编码`），大小写不敏感；未知列不报错但进 `warnings`，避免用户误以为该列生效。
- 上限：`MaxRows = 500`（bcrypt + 单事务写入的成本随行数线性增长）、`MaxFileSize = 5MB`。文件级哨兵错误 `ErrEmptyFile` / `ErrUnsupportedExt` / `ErrTooManyRows` / `ErrMissingHeader`，handler 据此统一回 400。

### 37.2 密码生成 `pkg/security/password_gen.go`
- `GeneratePassword(length)`：大小写字母 + 数字 + 特殊字符四类字符集，保证每类至少出现一个，再用 `crypto/rand` 洗牌；长度下限校验。初始密码只在导入成功的响应里回显一次。

### 37.3 仓储 `repository/import_repository.go`
- `CheckUsernames(ctx, tenantID, usernames)` 一次查出「现存同名」与「回收站同名」两个集合。回收站那一步必须 `Unscoped()`：软删除的用户仍占着 `(tenant_id, username)` 唯一索引，不提前拦会把冲突留到事务里变成 500。
- `CreateUsersWithRoles(ctx, users, roleIDsByUserID)`：用户 `CreateInBatches` 与 `user_roles` 关联写入在**同一个事务**内，任一步失败整体回滚。

### 37.4 服务层 `service/user_import.go`
- `ImportUsers(ctx, tenantID, rows, defaultRoleIDs)` 三态返回：全部成功 / 行级错误（`[]ImportRowError`，一条未写）/ 整体性错误。顺序固定为：默认角色校验 → **配额整批预检** → 逐行校验（字段规则、文件内重名、角色编码解析）→ 库内与回收站占用 → 有错即返回 → 生成密码并哈希 → 单事务写入 → 组装结果。
- 配额按整批 `len(rows)` 一次校验，而不是逐行累加：否则会在事务末尾才撞上限，前面几百行的哈希白算。
- 行内 `roles` 留空时回落到表单选择的默认角色并产生一条 warning；角色编码按租户作用域解析，解析不到该行直接报错。
- `buildImportUsers` 在 `min(runtime.NumCPU(), 8, n)` 个 worker 里并发生成密码与 bcrypt 哈希，`sync.Mutex` 保护游标与首错，结果下标与文件行序一致。
- 初始密码长度：默认 12；密码策略启用时取 `MinLength+2`，受 `MaxLength` 与 32 封顶、8 兜底。
- `importRepo` / `passwordPolicy` 走可选注入（`SetImportRepository` / `SetPasswordPolicy`），未注入时导入返回「批量导入未启用」，既有的单条 CRUD 调用链与单测构造方式完全不变。

### 37.5 路由与权限
- `POST /api/v1/users/import`（租户端，租户 ID 取自登录态）→ 自动推导权限码 `user:create`；批量导入与逐个新建同门槛，无需新增权限种子。
- `POST /api/v1/admin/tenant-users/import`（管理端，`tenant_id` 取自表单，缺失即 400）→ `admin:tenant_user:create`。
- 均为 `multipart/form-data`：`file` + 可选 `role_ids`（逗号分隔）+ 可选 `tenant_id`。handler 用 `http.MaxBytesReader`（5MB + 表单开销）与 `http.LimitReader` 双重限长，超出直接 400。
- 逐行问题以 `400 {code:"INVALID_PARAM", message:"共 N 处问题…", details:[{row,field,message}]}` 返回，`field` 取 json tag（`username` / `nickname` / …），便于前端映射到表格列。

### 37.6 审计隔离
- 审计中间件对 `multipart/*` **完全不读正文**，只落占位符 `[multipart/form-data 请求体不入审计日志]`——导入文件可能含密码列，既不可读也不应落库；同时避免 5MB 上传在审计层被整体缓冲两次。非 multipart 的请求仍按 64KB 限读并用 `io.MultiReader` 拼回，handler 读到的仍是完整 body。
- `sanitizeBody` 改为按**字段名包含关键字**脱敏（`password` / `token` / `secret` / `authorization` / `credit_card` / `id_card`），覆盖 `initial_password` / `new_password` / `refresh_token` 这类变体；路径含 `import` 的请求额外打 `import` 标签。

### 37.7 前端 `web-admin/`
- `src/api/modules/user.ts` 新增 `importUsers({file, roleIds, tenantId})`：`FormData` 提交，`timeout: 120000`（500 行 + bcrypt 需要时间），传 `tenantId` 时自动走管理端端点。
- 新增 `views/system/user/components/UserImportDialog.vue`：拖拽上传（限制后缀与 5MB）、默认角色多选（`getRoleListForSelect('tenant')`）、CSV 模板下载（带 UTF-8 BOM 防 Excel 乱码）、结果视图（逐行错误表按字段名映射中文列名 / 成功表含初始密码与「复制全部账号密码」）。
- 用户列表页工具栏新增「批量导入」按钮（具备 `user:create` 且不在回收站时显示）。系统管理员的用户列表是跨租户视图，因此弹窗在系统管理员模式下额外提供「目标租户」选择器，并隐藏默认角色选择器。
- 角色 ID 属于具体租户，跨租户复用会把别的租户的角色挂到目标租户用户上：管理端入口因此**忽略表单 `role_ids`**，只认文件里的角色编码（handler 有单测钉住）。

### 37.8 验证状态
- ✅ `go build ./...`；`go test ./...` 全绿。单测覆盖：解析层表头别名/空行/超限/未知列告警、服务层 17 例（全量事务、回收站占用、配额整批、角色回落、行号保持）、handler 12 例（multipart 表单、租户来源、`details` 结构、告警合并、管理端忽略 `role_ids`）、审计中间件 5 例（multipart 跳过、大 body 截断与拼回、密码变体脱敏、`import` 标签）。
- ✅ 前端 `npx vue-tsc --noEmit`、`npm run build` 通过；`vitest run` 22 个文件 225 例全部通过。
- ✅ 真实浏览器目视（Playwright 无头 Chrome + 本地前后端）：系统管理员登录 → 用户管理 → 「批量导入」弹窗可见 → 选目标租户 → 上传 2 行 CSV → 「导入成功，已创建 2 个用户」并回显初始密码；再次导入同名行 + 不存在的角色编码 → 「导入未通过校验，共 2 处问题，未写入任何用户」，逐行表正确显示行号/列名/原因。
- ✅ `go run scripts/openapi_sync.go` 对比后补齐 OpenAPI 与 Apifox 文档（261/261 路由完全匹配）。

---

## 三十八、系统管理员跨租户建号入口

批量导入上线后目视验收时发现：系统管理员在用户管理页点「新增用户」，账号会落到登录态的虚拟租户 `SYSTEM_ROOT` 里——列表里看得见，但没有任何人能登录进去（见 `docs/architecture/tenant.md` 的「虚拟租户 SYSTEM_ROOT」）。

### 38.1 问题成因

系统管理员没有真实租户，认证中间件为其写入占位值 `contextx.SystemTenantID`。`POST /users` 直接取登录态租户写库，于是账号被建在 `SYSTEM_ROOT` 下；而跨租户建号本该走 `POST /admin/tenant-users` 并显式传 `tenant_id`。前端此前也从未在新增弹窗里提供目标租户选择，等于两条路都走不通。

### 38.2 后端

- `CreateUser` 在取到租户后先判 `tenantID == contextx.SystemTenantID`，命中则 400 `系统管理员请使用跨租户接口为指定租户创建用户`，不进入参数校验与写库。
- `ImportUsers`（`POST /users/import`）同理：命中虚拟租户返回 400 `系统管理员请使用跨租户接口为指定租户导入用户`。两处口径一致，避免「建号被拒、导入却能写进虚拟租户」的不对称。
- 单测：`TestCreateUser_SystemTenantRejected_Returns400` 与 `TestImportUsers_SystemTenantRejected` 除断言状态码外，还断言 `GotCreateReq` / `GotImportRows` 为空——锁定「不得落库」而不只是「返回了 400」。

### 38.3 前端

- 新增弹窗在 `userStore.isAdmin` 时多出「目标租户」下拉：打开弹窗时懒加载启用中的租户（`getTenantList({page:1, page_size:200})` 后按 `status === 1` 过滤），未选则本地拦住并提示，不再把请求发出去赌后端报错。
- 提交分支按身份分流：系统管理员走 `createTenantUser(tenantId, payload)`（`POST /admin/tenant-users`，body 带 `tenant_id`），租户管理员仍走 `createUser`。
- 每次 `openCreateDialog` 都重置 `createTenantId`，避免上一次的租户被带到下一次新建。
- 新增弹窗同步补上「角色」多选：`getRoleListForSelect('tenant')` 懒加载（仅首次打开拉取），未选时本地拦截并提示「请至少选择一个角色」，提交时 body 带 `role_ids`。

这条不是顺手优化，而是目视时发现的另一个早就存在的阻塞：`CreateUserReq` 与 `AdminCreateTenantUserReq` 的 `role_ids` 都是 `validate:"required,min=1"`，而弹窗此前从未提交过角色——也就是说无论租户管理员还是系统管理员，点「新增用户」都必定 400。服务层 `AdminCreateTenantUser` 还会 `validateRoleIDs(..., RoleScopeTenant)` 并 `AssignRoles`，所以弹窗只能列出 `scope` 为 `all` / `tenant` 的角色。

### 38.4 验证状态

- ✅ `go build ./...`；`go test ./...` 全绿（`/admin/tenant-users` 路由的权限码推导仍由 `route_check_test` 守护）。
- ✅ `npx vue-tsc --noEmit` 通过；`npx vitest run` 22 个文件 225 例通过；`npm run build` 通过。
- ✅ 真实浏览器目视（系统管理员登录，Playwright 驱动本机 Chrome 跑完整链路，含一次临时管理员 `uicheck_admin2`，验完即删）：
  - 新增弹窗出现「目标租户」与「角色」两个选择器；
  - 未选租户提交 → 提示「请先选择目标租户」，且发往 `/api/v1/users` 的请求次数为 0（本地拦截生效）；
  - 已选租户、未选角色提交 → 提示「请至少选择一个角色」，仍未发出请求；
  - 选租户 + 角色后提交 → `POST /api/v1/admin/tenant-users` 返回 200，响应 `tenant_id=01M52QRSGBV0B243R2ZE6R7RVT`、`role_ids=["role-tenant-admin-001"]`，新账号出现在该租户列表中；
  - 绕过前端直调 `POST /api/v1/users` → 400 `系统管理员请使用跨租户接口为指定租户创建用户`。

## 三十九、用户列表批量操作（Batch Actions）

用户管理页此前只能逐行点「禁用」「删除」。而 `user:batch_status` / `user:batch_delete` 两个权限码早已随权限表注册，`BatchUpdateUserStatusReq` / `BatchDeleteUsersReq` 也已定义，前端 `batchUpdateTenantUserStatus` / `batchDeleteTenantUsers` 还封装了管理端接口——但租户端既没有 handler 也没有路由，前端封装也零调用，属于「权限已发、能力未通」。

### 39.1 后端

- 新增 `PUT /api/v1/users/batch/status` 与 `DELETE /api/v1/users/batch/delete`。权限码由 `middleware.AutoRequirePermission` 按「方法 + 路径」自动推导为 `user:batch_status` / `user:batch_delete`，两个码在 `permissions.go` 里早已定义并随 `SeedPermissions` 幂等注册，因此本次不需要新增权限或迁移。
- 服务层 `BatchUpdateUserStatus` / `BatchDeleteUsers` 先调 `loadUsersInTenant` 逐条校验「存在且属于当前租户」，任一不通过整批拒绝；`BatchDeleteUsers` 要等整体校验通过后才开始解除 `user_roles` 绑定，否则会出现「前面的用户角色已解绑、后面的用户越权报错」的半成品状态。
- 写库复用仓库层的租户作用域方法，SQL 自带 `id IN ? AND tenant_id = ? AND is_master = ?`，就算前端把别的租户的 ID 递进来也越不出去。
- 与三十八节同口径：登录态租户是虚拟租户 `SYSTEM_ROOT` 时直接 400 `系统管理员请使用跨租户接口批量操作用户`，必须改用 `/admin/tenant-users/{tenantID}/batch/...`。
- 错误映射 `failBatchUserAction`：`service.ErrUserNotInTenant` → 400、`gorm.ErrRecordNotFound` → 404、其余 500。
- 单测：handler 10 例（成功透传 tenant/ids/status、空 ids、SYSTEM_ROOT 被拒且不触达服务层、跨租户 400、不存在 404、服务端错误 500）；service 7 例（成功、空 ids 不触达仓库、跨租户整批拒绝且状态与角色绑定未变、用户不存在）。为此把 `mockUserRepo` 的两个租户作用域批量方法从 `return 0, nil` 改成「记录入参 + 按 tenantID 与 `!IsMaster` 真实生效」，才断言得出「越权时仓库一次都没被调用」。

### 39.2 前端

- 列表新增多选列（`selectable` 拦住缺 `tenant_id` 的行）与批量栏：按 `user:batch_status` / `user:batch_delete` 分别显示「批量启用」「批量禁用」「批量删除」，回收站视图不显示。
- 系统管理员模式下列表是跨租户数据，`groupIdsByTenant` 按行上的 `tenant_id` 分桶后逐租户调用管理端接口；租户管理员直接调新增的 `batchUpdateUserStatus` / `batchDeleteUsers`（`PUT /users/batch/status`、`DELETE /users/batch/delete`）。
- `watch(list)` 在翻页/刷新后清空勾选，避免残留 ID 指向已经翻走的数据。

### 39.3 验证状态

- ✅ `go build ./...`；`go test ./...` 全绿（`/users/batch/*` 的权限码推导由 `auto_permission_middleware_test.go` 守护，`route_check_test` 无新增缺口）。
- ✅ `npx vue-tsc --noEmit` 通过；`npx vitest run` 22 个文件 229 例通过；`npm run build` 通过。
- ✅ `go run scripts/openapi_sync.go` 显示 OpenAPI / Apifox 各 263 条路由与代码完全匹配（两个新路由已补上 requestBody：`ids` 与 `status`）。
- ✅ 真实浏览器目视（系统管理员登录，`vite preview` 产线产物 + Playwright 驱动本机 Chrome，用两个临时账号 `uicheck_b1/b2_*` 跑完整链路，验完先软删、再永久删除，库里无残留）：
  - 列表出现多选列，勾选两行后批量栏显示「已选 2 项 / 批量启用 / 批量禁用 / 批量删除 / 取消选择」；
  - 点「批量禁用」→ `PUT /api/v1/admin/tenant-users/01KWMFZNNV7VDNQ4FXD1GJBMKE/batch/status` 返回 200、`{"updated":2}`，两行状态标签变「禁用」；
  - 再点「批量启用」→ 200、`{"updated":2}`，两行恢复「启用」；
  - 点「批量删除」→ `DELETE .../batch/delete` 返回 200、`{"deleted":2}`，列表归零，回收站里能看到这两条。
- ℹ️ 本地 `tenant_admin` 角色只有 5 个 `user:*` 权限（不含 `user:batch_*`），所以租户端 `/users/batch/*` 路径未做浏览器目视，由 handler/service 单测覆盖；若要目视需先给该角色绑定两个权限码。

## 四十、批量分配角色（RBAC Batch Assign）

### 40.1 起因

`POST /api/v1/rbac/user-roles/batch/assign` 与权限码 `rbac:user_role:batch_assign` 早已在权限体系里注册（`permissions.go` 与
`auto_permission_middleware_test.go` 均有覆盖），但前端从未调用过，列表只能一行一行点「分配角色」；同时 service 层只校验用户与角色是否存在，
没有校验目标用户的租户归属和角色的作用域，租户账号理论上可以借这个接口跨租户改角色，或把系统级角色挂到租户用户身上。

### 40.2 后端

- `repository.UserRoleRepository` 新增 `GetUserScope(ctx, userID)`，用一条 `SELECT tenant_id, is_master FROM users WHERE id = ?`
  返回目标用户归属，避免为校验额外拉整个用户实体。
- `RBACService` 新增 `validateRoleAssignScope`：`SYSTEM_ROOT` 直接放行（系统管理员跨租户继续可用）；租户操作者要求目标用户同租户且非主管理员，
  角色必须是 `tenant`/`all` 作用域，且归属 `SYSTEM_ROOT` 或本租户。
- `AssignUserRoles` 与 `BatchAssignUserRoles` 都在写入前调用该校验；批量接口改成先校验全部用户与角色、再逐个写入，任一条越权即整批不写，
  不会出现「前几个用户角色已换、后面才报错」的半成品状态。
- 新增 7 个 service 单测（跨租户、主管理员、系统级角色、他租户角色、系统管理员放行、批量整批拒绝、批量成功），mock 仓库补充 `userScopes`
  与 `seedUser`；`auth`/`invitation`/`user` 三处测试桩补上 `GetUserScope` 以满足接口。

### 40.3 前端

- `role.ts` 新增 `batchAssignUserRoles(assignments)`，对应 `POST /rbac/user-roles/batch/assign`。
- 用户列表批量栏新增「分配角色」：按 `rbac:user_role:batch_assign` 或系统管理员身份显隐，回收站视图不展示；弹窗复用单行分配的角色候选
  （管理员 `scope=all`，租户 `scope=tenant`），带「提交后将覆盖所选用户当前全部角色」的告警，未选角色时前端先拦截（后端 `role_ids` min=1）。
- 提交时把勾选行映射成 `{ user_id, role_ids }` 数组一次提交（接口按用户维度写入，无需按租户分组），成功后清空勾选并刷新列表。

### 40.4 验证状态

- ✅ `go build ./...`、`go test ./...` 全绿（含 rbac / user / auth / invitation 四个包）。
- ✅ `npx vitest run` 22 个文件 230 例通过；`npx vue-tsc --noEmit` 通过；`npm run build` 通过。
- ✅ 真实浏览器目视（系统管理员登录、Playwright 驱动本机 Chrome，用两个临时账号 `uicheck_a1/a2_*`）：勾选两行 → 「分配角色」弹窗列出
  superadmin / system_admin / tenant_admin / user 等角色 → 选 `user` 提交 → `POST /api/v1/rbac/user-roles/batch/assign` 返回
  200、`{"assigned":2}`，随后 `/rbac/user-roles/{id}/roles` 两个账号均只剩 `user`；验完清理临时账号，库里无残留。
- ⚠️ 目视只覆盖系统管理员路径（`SYSTEM_ROOT` 放行）；租户端的越权拒绝由 service 单测覆盖，本地 `tenant_admin` 未绑定
  `rbac:user_role:batch_assign`，无法在界面上触发。
- ℹ️ 已知后续项：该校验只限制「租户 + 角色作用域」，未限制「角色权限集合必须属于操作者」，即持有 `rbac:user_role:assign`
  的普通租户用户仍可给自己授予 `tenant_admin`。（已于第四十二节收口）

## 四十一、用户列表「导出所选用户」（Export Selected Users）

### 41.1 起因

异步导出链路（`/api/v1/exports` + 下载中心）早已可用，但只能按资源类型导全部数据：用户列表没有导出入口，
`userExporter` 也只认 `status` 过滤。上一张卡补齐多选与批量栏后，「勾一批人导出」成为自然诉求，
而所选用户 ID 集合没有任何传递通道，只能先导出全部再人工筛选。

### 41.2 后端

- `userExporter` 支持 `filters.ids`：与租户条件叠加，他租户 ID 与已软删用户不会越界导出；
  新增的 `exporter.filterStrings` 同时兼容 JSON 反序列化后的 `[]any` 与代码内直接构造的 `[]string`。
- 服务层 `normalizeIDFilter` 在创建阶段规范化 `ids`（剔空、去重、去首尾空格）；非字符串数组、剔空后为空、
  超过 `MaxFilterIDs`（200）均返回 400 `ErrInvalidFilters`，避免任务已建却注定失败。
- `CreateExportReq` 新增 `tenant_id`（**仅 `is_master` 生效**）：系统管理员在跨租户用户列表里导出时，
  任务归属被选中的租户而不是虚拟租户 `SYSTEM_ROOT`（否则 exporter 按租户过滤会导出 0 行）；
  普通租户传了同名字段会被忽略，handler 也不会把它混进 `filters`。
- `List` 对 `is_master` 不按租户过滤（仍严格限定 `user_id` 为本人），否则系统管理员在下载中心看不到自己为子租户创建的任务；
  repository 层约定 `tenantID == ""` 表示不限租户，普通租户路径始终带租户条件。
- `permissions.go` 补登记 `export:list/create/read/delete` 四个权限码：此前它们只存在于自动推导测试里，
  权限表中并不存在，任何租户角色都无处勾选；`/exports/{id}/download` 由推导复用 `export:list`，无需单独授权。

### 41.3 前端

- `export.ts` 的 `CreateExportParams` 增加 `tenant_id` 与 `filters.ids` 说明。
- 用户列表批量栏新增「导出所选」（按 `export:create` 或系统管理员身份显隐，回收站视图不展示）：
  把勾选行 ID 作为 `filters.ids` 建任务，成功提示「已创建导出任务，完成后可在下载中心下载」。
- 系统管理员勾选了跨租户的行时前端先拦截并提示先按租户筛选——一个导出任务只能归属一个租户；
  同时把 `tenant_id` 一并提交，避免任务落到虚拟租户上。

### 41.4 验证状态

- ✅ `go build ./...`、`go vet ./internal/modules/export/... ./internal/modules/rbac/...` 通过；
  `go test -count=1 ./internal/modules/export/... ./internal/modules/rbac/...` 全绿（exporter / service / handler / repository 均新增用例）。
- ✅ 单测覆盖：`ids` 规范化后落库为 `{"ids":["u1","u2"]}`；非字符串数组、全空、超过 200 条返回 `ErrInvalidFilters`，恰好 200 条放行；
  他租户 ID 与软删用户不导出；`List` 在 `isMaster=false` 时 0 条、`isMaster=true` 时跨租户可见。
- ✅ `npx vue-tsc --noEmit` 通过；`npx vitest run` 22 个文件 231 例通过（新增 `createExport` 透传 `ids` 与 `tenant_id` 用例）；
  `npm run build` 通过。
- ✅ 真实浏览器目视（系统管理员登录、Playwright 驱动本机 Chrome、本机 MySQL + miniredis）：临时建两个同租户账号
  `uicheck_b1/b2_*` 并搜索出恰好 2 行 → 勾选后批量栏出现「导出所选」→ 点击后 `POST /api/v1/exports` 返回 200，
  请求体为 `{"resource":"user","format":"csv","tenant_id":"<目标租户>","filters":{"ids":[两个临时账号 ID]}}`，
  任务 `tenant_id` 落在目标租户而非 `SYSTEM_ROOT` → 轮询 `GET /api/v1/exports/{id}` 到 `completed` 且 `rows=2` →
  `GET /api/v1/exports?resource=user` 列表可见该任务 → 下载 CSV 为表头 + 2 行且含两个用户名。
  同次目视还用 `GET /api/v1/rbac/permissions?resource=export` 确认 `export:list/create/read/delete` 已入库。
  验完删除任务（文件同步清理）与临时账号，admin 密码 hash 已还原，库中无残留。

## 四十二、角色分配「权限不越出操作者」收口（No Privilege Escalation On Assign）

### 42.1 起因

第四十节为角色分配补齐了「租户 + 角色作用域」校验，但登记了一个已知缺口：`validateRoleAssignScope` 只挡住跨租户与系统级角色，
未限制「被授角色的权限集合必须属于操作者」。后果是：一个仅持有 `rbac:user_role:assign` 的普通租户用户，可以给任意同租户账号
（包括其自己）授予 `tenant_admin` 等更高权限角色，形成水平/垂直提权。

### 42.2 后端

- 新增 `validateRoleAssignPermissionSubset(ctx, roles)`：非系统操作者所分配的**每个角色**，其权限编码集合必须是操作者自身
  有效权限集合（`GetUserPermissionCodes`，合并其全部角色）的子集；出现操作者未持有的编码即拒绝，错误消息点名越权编码。
- 系统操作者（`contextx.GetTenantID == SYSTEM_ROOT`）不受此限制，与 `validateRoleAssignScope` 口径一致；缺少操作者上下文直接拒绝。
- `AssignUserRoles` 与 `BatchAssignUserRoles` 均在 `validateRoleAssignScope` 之后调用它；批量入口整批校验，
  任一角色越权则整批拒绝，避免「前几个已授、后面才报错」的半成品状态。

### 42.3 验证状态

- ✅ `go build ./...`、`go vet ./internal/modules/rbac/...`、`gofmt -l` 通过；`go test -count=1 ./internal/modules/rbac/... ./internal/integration/... ./internal/bootstrap/...` 全绿。
- ✅ 新增 4 条 service 单测：子集放行、越权拒绝且点名 `tenant:delete`、系统操作者绕过子集、批量整批拒绝（不落任何分配）。
- ✅ 变异检验：把 `validateRoleAssignPermissionSubset` 改为恒返回 nil 后，两条「拒绝」用例转红、两条「放行/绕过」用例仍绿，
  证明护栏由拒绝用例真实锚定，还原后全绿。
- ℹ️ 本卡为纯后端安全校验，未改动前端；租户端操作者路径的越权拒绝由 service 单测覆盖（本地 `tenant_admin` 未绑定
  `rbac:user_role:assign`，界面无法触发，与第四十节口径一致）。系统管理员路径经既有目视脚本放行不受影响。
- ℹ️ 边界说明：本卡只收口「分配角色」，未收口「给角色绑定权限」（`BindRolePermissions`）——操作者仍可能先把自身没有的
  权限绑到一个角色上、再分配出去。彻底闭环需同法约束绑定入口（已于第四十三节收口）。

## 四十三、角色绑定权限「权限不越出操作者」收口（No Privilege Escalation On Bind）

### 43.1 起因

第四十二节收口了「分配角色」的权限子集，但登记了一个边界缺口：给角色绑定权限的 `BindRolePermissions` / `BatchBindRolesPermissions`
仅校验角色与权限 ID 是否存在，不校验被绑权限是否属于操作者。于是仍存两条提权路径：

1. **直接自我提权**：持 `rbac:role:bind_perm` 但无高权的租户操作者，把 `tenant:delete` 等自身未持有的权限绑到**自己所属的角色**上，
   无需经过分配环节即刻获得该权限。
2. **先绑后分配绕过**：先把自身没有的权限绑到一个角色，再设法分配出去，绕开第四十二节仅在分配环节设卡的子集校验。

### 43.2 后端

- 新增 `validateBindPermissionSubset(ctx, perms)`：非系统操作者为角色绑定的**每个权限**，其编码必须在操作者自身有效权限集合
  （`GetUserPermissionCodes`）内；出现未持有的编码即拒绝，错误消息点名越权编码。
- 系统操作者（`contextx.GetTenantID == SYSTEM_ROOT`）不受此限制，与 `validateRoleAssignPermissionSubset` 口径一致；
  缺操作者上下文（非系统且无用户 ID）时失败关闭。
- `BindRolePermissions` 与 `BatchBindRolesPermissions` 均在权限存在性校验之后、实际写入之前调用它；批量入口整批校验，
  任一权限越权则整批不写。校验位于写库前，且失败关闭，避免留下部分绑定。

### 43.3 验证状态

- ✅ `go build ./...`、`go vet ./internal/modules/rbac/...`、`gofmt -l` 通过；`go test -count=1 ./internal/modules/rbac/... ./internal/integration/...` 全绿。
- ✅ 新增 5 条 service 单测：子集放行、越权拒绝且点名 `tenant:delete`（不落任何绑定）、系统操作者绕过子集、缺操作者上下文失败关闭、
  批量整批拒绝与批量放行。
- ✅ 变异检验：把 `validateBindPermissionSubset` 改为恒返回 nil 后，三条「拒绝」用例转红、放行/绕过用例仍绿，
  证明护栏由拒绝用例真实锚定，还原后全绿。
- ✅ 既有 `TestBindRolePermissions_Success` 改为携带租户+操作者上下文（操作者持有被绑权限），与新守卫语义一致；
  `RoleNotFound`/`PermissionNotFound` 在到达子集校验前即返回，行为不变。
- ℹ️ 本卡为纯后端安全校验，未改动前端；生产调用方仅两个 HTTP handler（均带真实请求上下文），无种子/后台调用会被误伤。
- ℹ️ 边界说明：本卡与第四十二节共同闭环「租户内分配/绑定提权」主路径。未覆盖项：非系统操作者向**其他租户角色或系统级角色**
  绑定自身已持有权限的跨租户角色改写（属角色归属越权，与提权正交），登记为后续项（已于第四十四节收口）。

## 四十四、角色归属越权收口（绑定/分配只作用于可管理角色）

### 44.1 起因

第四十三节收口了绑定权限的子集，但登记的边界仍未闭：`BindRolePermissions` / `BatchBindRolesPermissions` 从不校验**目标角色的归属**——
一个持 `rbac:role:bind_perm` 的租户操作者可把权限（哪怕自身已持有）绑到其他租户的角色或系统级角色上，形成跨租户/系统角色篡改；
批量绑定入口甚至不校验角色是否存在。而分配入口 `validateRoleAssignScope` 早已拦住“系统级角色 + 他租户角色”，两个入口口径不一致。

### 44.2 后端

- 抽出共享护栏 `validateRolesInOperatorScope(ctx, roles)`：非系统操作者只能对归属本租户或平台（`SYSTEM_ROOT`）且非系统级的
  角色操作；`scope=system` 或归属他租户均拒绝。系统操作者旁路。
- `validateRoleAssignScope` 的角色循环改为委派给该 helper（行为与文案子串不变），分配与绑定从此共用同一护栏，消除口径漂移。
- `BindRolePermissions` 在权限存在性校验后、子集校验前调用它；`BatchBindRolesPermissions` 新增逐角色存在性加载（缺失即 400）并整批校归属。
- 校验顺序：角色/权限存在性 → 角色归属 → 权限子集 → 写入；归属拦截独立于子集（即使只绑自身已持有权限，绑到系统级/他租户角色仍拒）。

### 44.3 验证状态

- ✅ `go build ./...`、`go vet ./internal/modules/rbac/...`、`gofmt -l` 通过；`go test -count=1 ./internal/modules/rbac/... ./internal/integration/... ./internal/bootstrap/...` 全绿。
- ✅ 新增 5 条 service 单测：绑定系统级角色拒绝、绑定他租户角色拒绝且不落绑定、系统操作者绕过归属、批量他租户角色拒绝、批量角色不存在。
- ✅ 变异检验：把 `validateRolesInOperatorScope` 改为恒返回 nil 后，5 条拒绝用例转红（新增 3 绑定 + 因重构而依赖 helper 的 2 分配系统级/他租户用例），
  绕过/放行用例仍绿，证明护栏由拒绝用例真实锚定，还原后全绿。
- ✅ 分配侧重构安全：`TestAssignUserRoles_SystemRoleRejected`/`ForeignRoleRejected` 仅断言子串“系统级角色”/“不属于当前租户”，新文案仍包含；
  `CrossTenantRejected`（用户级）由 `validateRoleAssignScope` 保留的用户循环拦截，不经过 helper。
- ℹ️ 本卡为纯后端安全校验，未改动前端。至此分配与绑定两入口对“角色归属 + 权限子集”形成一致的护栏，租户内跨租户篡改与提权路径均已闭合。

## 四十五、链路追踪关联收口（操作记录/日志/响应头共用真实 TraceID）

### 45.1 起因

项目早已搭好 OpenTelemetry 骨架（`TracingMiddleware` 创建 span、GORM 插件、OTel Collector + Jaeger 部署），
但**追踪与后台操作记录是断开的**：`AuditMiddleware.getTraceID()` 独立于 OTel，入站请求通常不带 `X-Trace-ID`
（那是写在响应上的），于是绝大多数请求它会**生成一个随机 UUID** 落库；即便命中 `Traceparent` 也是存整串 header
（约 55 字符，超过 `trace_id` 列 `size:50`）。结果：审计记录里的 `trace_id` 与 Jaeger 中的真实 trace 对不上，
点不进链路；且审计中间件会把 `TracingMiddleware` 写好的 `X-Trace-ID` 响应头覆盖成自己的 UUID。
`logger.WithTraceID` 定义了却从未被调用，`logger.Ctx()` 也不带 trace_id。审计**覆盖面本身是完整的**
（保护分组内每个请求都自动入库，仅跳过 health 与审计自查端点），缺的是「关联」而非「记录」。

### 45.2 后端

- `TracingMiddleware`：span 创建后，把真实 OTel Trace ID 同时注入 `contextx`（供审计落库）与 `logger`（供结构化日志），
  并统一回填 `X-Trace-ID` 响应头——三处共用同一个值。即使 trace 未被采样，`SpanContext` 仍携带有效的 W3C Trace ID，可正常关联。
- `AuditMiddleware`：抽出 `resolveTraceID(r)`——优先复用 `TracingMiddleware` 注入 context 的真实 Trace ID，
  仅在无追踪上下文（OTel 关闭等）时才回退；回退路径新增 `parseTraceparentTraceID`，只取 `traceparent` 的 32 位 trace-id 段
  （而非整串），避免超列宽且无法检索。
- `logger.Ctx` 自动附加 `trace_id`；全局中间件顺序调整为 `Recovery → Tracing → Logger`，使访问日志 `HTTP Request` 行也带上真实 trace。

### 45.3 验证状态

- ✅ `go build ./...`、`go vet ./internal/middleware/... ./pkg/logger/... ./internal/bootstrap/...`、`gofmt -l` 通过；
  `go test -count=1 ./internal/middleware/... ./pkg/logger/... ./internal/bootstrap/... ./internal/integration/...` 全绿。
- ✅ 新增单测：`tracing_test.go`（注入真实 trace id 且 contextx/logger/响应头三处同源、延续上游 `traceparent`）、
  审计 `resolveTraceID` 优先复用 context、回退取 `traceparent` trace-id 段、`parseTraceparentTraceID` 边界表；
  `logger_test.go` 断言 `Ctx` 输出确含 `trace_id=`。
- ✅ 变异检验：三处还原分别令 `TestTracingMiddleware_*`、`TestResolveTraceID_PrefersContextTrace`、`TestCtxLogger` 精确转红，
  其余仍绿，还原后全绿。
- ✅ 真实启动目视：请求 `GET /health` 返回 `X-Trace-ID=a0e95349f05edac1abde2a75200a2ef4`，同一值出现在访问日志
  `trace_id=` 字段与 OTel stdout 导出 span 的 `"TraceID"` 中，三处逐字节一致，证明关联已贯通（审计经 `contextx` 落同一值）。
- ℹ️ 本卡为纯后端可观测性收口，未改动前端。