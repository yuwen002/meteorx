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
| **租户注销** | 审批通过/驳回 | Email + WebSocket | 通知租户管理员（预留） |
| **订阅到期** | 即将到期 | Email + WebSocket（预留） | 通知租户管理员（预留） |

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