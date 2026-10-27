# 审计架构

## 概述

框架级审计能力，用于追踪所有模块的操作。审计日志记录每个重要操作的操作者、内容、地点、时间和结果。

## 审计日志模型

### 包路径：`internal/modules/audit/model`

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 日志 ID（ULID） |
| user_id | string | 操作者用户 ID |
| username | string | 操作者用户名（反规范化） |
| tenant_id | string | 租户 ID（用于隔离） |
| module | string | 模块名（user、tenant、auth、wiki 等） |
| action | string | 操作类型（create、update、delete、login 等） |
| resource | string | 资源路径（如 /api/v1/users） |
| resource_id | string | 资源 ID（如用户 ID） |
| method | string | HTTP 方法（GET/POST/PUT/DELETE） |
| path | string | 请求路径 |
| request_body | string | 请求载荷（JSON，敏感信息已脱敏；multipart 请求只记占位符） |
| response_body | string | 响应数据（JSON，敏感信息已脱敏） |
| status_code | int | HTTP 状态码 |
| result | string | 操作结果（成功/失败） |
| error_message | string | 失败时的错误详情 |
| client_ip | string | 客户端 IP |
| user_agent | string | 浏览器 UA |
| duration | int64 | 请求耗时（毫秒） |
| record_hash | string | 行级完整性摘要（HMAC-SHA256 hex，防篡改；未启用完整性时为空） |
| created_at | time.Time | 时间戳 |

### 索引

```
idx_audit_logs_tenant_id  (tenant_id)
idx_audit_logs_user_id    (user_id)
idx_audit_logs_module     (module)
idx_audit_logs_action     (action)
idx_audit_logs_result     (result)
idx_audit_logs_created_at (created_at)
```

## 审计上下文

### 包路径：`internal/common/auditctx`

审计上下文允许服务层通过 before/after 值丰富审计条目：

```go
// 创建审计操作
action := auditctx.NewAction("wiki", "UPDATE_DOCUMENT", "/documents/"+docID, docID)

// 设置用户上下文
action.WithUser(userID, username, tenantID)

// 捕获 before/after
action.WithBefore(oldDoc).WithAfter(newDoc)

// 设置请求元数据
action.WithRequest(clientIP, userAgent, statusCode)

// 标记结果
action.Succeeded()
// 或
action.Failed(err)

// 通过上下文传递
ctx := auditctx.WithAction(r.Context(), action)
```

### 操作命名约定

```
{模块}_{操作}

模块：
  AUTH, USER, ROLE, PERMISSION, TENANT, PLAN,
  WIKI_SPACE, WIKI_NODE, WIKI_DOCUMENT, WIKI_REVISION,
  ANNOUNCEMENT, FILE, SYSTEM

操作：
  CREATE, UPDATE, DELETE, RESTORE, PERMANENT_DELETE,
  LOGIN, LOGOUT, EXPORT, IMPORT, APPROVE, REJECT,
  BATCH_CREATE, BATCH_UPDATE, BATCH_DELETE
```

示例：
```
WIKI_DOCUMENT_CREATE
WIKI_DOCUMENT_UPDATE
WIKI_DOCUMENT_DELETE
WIKI_DOCUMENT_RESTORE
WIKI_REVISION_ROLLBACK
USER_LOGIN
USER_LOGOUT
TENANT_PLAN_ASSIGN
```

## 两种审计模式

### 1. 自动审计（中间件）

`AuditMiddleware` 自动记录所有 HTTP 请求：

```go
// 在中间件链中
r.Use(middleware.AuditMiddleware(auditLogger))

// 自动捕获：
// - 谁（user_id, username, tenant_id）
// - 什么（method, path, request_body）
// - 在哪里（client_ip, user_agent）
// - 何时（duration, created_at）
// - 结果（status_code, success/failure, error_message）
// - 关联（request_id, session_id, trace_id）
// - 标签（tags：模块 / 动作 / 方法 / 结果，以及 upload、batch、download 等特征）
```

**批量写操作的审计**：路径含 `batch` 的请求额外打 `batch` 标签（如 `PUT /messages/batch/read`、
`DELETE /messages/batch/delete`）。批量接口的请求体是 ID 数组，可能因 2000 字符截断而不完整，
标签是「按类型筛出全部批量写」的唯一可靠依据；风险等级仍按方法归一（批量删除 = delete = critical）。
业务模块不需要为此单独埋点，避免一次操作落两条审计。

**导入操作的审计**：路径含 `import` 的请求额外打 `import` 标签（如 `POST /api/v1/users/import`、
`POST /api/v1/admin/tenant-users/import`），用于把「批量建号」与「逐个新建」区分开。

**请求体采集规则**（`readRequestBodyForAudit`）：

| 请求类型 | 采集行为 | 落库内容 |
|------|------|------|
| `multipart/form-data` | **完全不读、不缓存**，直接返回占位符，handler 照常流式解析 | `[multipart/form-data 请求体不入审计日志]` |
| 其他（JSON 等） | 用 `io.LimitReader` 最多读 64 KB，未读完的部分用 `io.MultiReader` 拼回 | 前 64 KB，脱敏 + 2000 字符截断 |

文件导入的正文是二进制与用户明文数据（CSV/XLSX 可能含密码列），既不可读也不应落库；
同时限制采集长度可避免上传请求在审计层被整体缓冲两次。因此**新增上传类接口无需任何审计埋点**，
只要沿用 `multipart/form-data` 就自动隔离。

**链路追踪关联（trace_id）**：审计记录的 `trace_id` 直接复用 `TracingMiddleware` 注入的真实 OTel Trace ID（经 `resolveTraceID` 读取
请求上下文），因此它与结构化日志的 `trace_id` 字段、`X-Trace-ID` 响应头、Jaeger 中的 span `TraceID` **逐字节一致**——
从一条操作记录可直接跳查完整调用链。仅当无追踪上下文（OTel 关闭等）时才回退：优先取 W3C `traceparent` 的 32 位
trace-id 段（而非整串 header，避免超出 `trace_id` 列 `size:50`），最后才生成占位 UUID。详见 `docs/DESIGN.md` 与 FEATURE_UPGRADE 第四十五节。

### 2. 手动审计（服务层）

使用 `auditctx` 丰富 before/after 值：

```go
func (s *WikiService) UpdateDocument(ctx context.Context, id, userID string, req *dto.UpdateDocumentReq) (*model.Document, error) {
    oldDoc, _ := s.repo.GetDocumentByID(ctx, id)

    // ... 业务逻辑 ...

    // 通过 before/after 丰富审计
    action := auditctx.NewAction("wiki", "UPDATE_DOCUMENT", "/documents/"+id, id)
    action.WithUser(userID, username, tenantID)
    action.WithBefore(oldDoc).WithAfter(newDoc)
    ctx = auditctx.WithAction(ctx, action)

    return newDoc, nil
}
```

### 3. 公开端点的手动审计（handler 层）

登录 / 注册 / 忘记密码 / 重置密码 / OAuth 回调等**公开端点不在受保护分组**，不经过
`AuditMiddleware`，因此必须在 handler 内手动记录，否则这些凭据类高危操作（账号枚举、
暴力破解、令牌接管）将完全静默。

```go
auditReq := middleware.NewAuditLogReq(r) // 预填 IP/UA/设备/Referer + trace/request/session
auditReq.Module = "auth"
auditReq.Action = auditmodel.ActionTypeLogin
auditReq.Result = auditmodel.ResultFailure // 成功/失败都记录
auditReq.RiskLevel = auditmodel.RiskHigh
middleware.RecordAuditLog(r.Context(), &auditReq)
```

- `NewAuditLogReq` 与自动审计**同口径**携带 `trace_id`/`request_id`/`session_id`（取自
  `contextx` 与 `getSessionID`），手动记录也能从操作记录跳查 Jaeger 链路；无追踪上下文时
  `trace_id` 留空而非编造占位值。
- 手动记录**不采集请求体**，密码 / 重置令牌 / OAuth `code`/`state` 等凭据天然不入库。

### 4. 审计自身操作的留痕（handler 层）

审计管理路由里，`shouldSkipAudit` 用前缀跳过 `/api/v1/audit/logs`（避免审计自查页面自我刷屏），
但**连带**跳过了两个高危端点：`DELETE /api/v1/audit/logs/cleanup`（按天删除审计历史）与
`GET /api/v1/audit/logs/export`（一次导出最多 1 万条）。二者恰好是「最能掩盖痕迹」的操作，若
不补审计则平台超管删空审计库 / 批量拖库完全零痕迹。因此在 handler 内手动补记：

```go
// CleanupLogs / ExportLogs 的成功与失败各路径均调用：
h.recordSelfOpAudit(r, start, action, risk, reqBody, respBody, statusCode, result, errMsg)
```

- 清理记为 `module=audit / action=delete / risk=critical`；导出记为 `module=audit / action=export / risk=high`。
- **成功与失败都入库**：非超管越权（403）、服务失败（500）、非法格式（400）均落 `failure` 并带 `error_message`。
- 只记**参数摘要与结果摘要**（如 `{days:60}`、`{deleted_count:88}`、导出筛选条件与 `{exported_count:N}`），
  不落任何正文。
- 经由 `middleware.RecordAuditLog`，**不经过 `shouldSkipAudit`**，故不会被跳过；而自动中间件对这两
  路径本就跳过，因此**不会双记**。`recordSelfOpAudit` 基于 `NewAuditLogReq(r)`，与自动/手动审计
  同口径携带 `trace_id`/`request_id`/`session_id`。

## 审计流程

```
HTTP 请求
    ↓
AuditMiddleware（自动捕获）
    ↓
服务层（可选：通过 auditctx 丰富）
    ↓
Handler 响应
    ↓
AuditBatchProcessor（异步写入）
    ↓
数据库（audit_logs 表）
```

### 批量处理

```go
// AuditBatchProcessor
间隔：5 秒
批量大小：100 条
超时：10 秒（context.WithTimeout）
```

日志在内存中缓冲，定期批量写入数据库（批量INSERT），性能提升10-100倍。

**告警评估衔接**：批量 flush **落库成功后**整批交给告警评估链（每轮只查一次启用规则），因此告警触发延迟上限即本节的 flush 间隔（默认 5 秒，缓冲写满 100 条会提前 flush）；无批量处理器时的直写路径落库后立即评估。该延迟属最终一致的设计取舍，不含邮件/Webhook 投递耗时，详见 `docs/api/audit-api.md` 7.4.1。

**Context传递优化**：
- 异步审计日志使用带超时的Context（10秒），避免goroutine泄漏
- 服务器关闭时自动取消未完成的批量写入
- 通过 `context.WithTimeout(context.Background(), 10*time.Second)` 实现

**批量插入实现**：
- `AuditService.BatchCreateLogs()` 方法支持批量创建
- `AuditRepository.BatchCreate()` 使用单条SQL批量插入
- 从O(n)数据库操作优化为O(1)

## 查询审计日志

### 包路径：`internal/modules/audit/service`

```go
// 按条件过滤
query := &model.AuditLogQuery{
    UserID:    "user_001",
    Module:    "wiki",
    Action:    "UPDATE",
    Result:    "failure",
    StartTime: "2024-01-01T00:00:00Z",
    EndTime:   "2024-12-31T23:59:59Z",
    Page:      1,
    PageSize:  20,
}

// Dashboard 统计
dashboard, err := auditService.GetDashboard(ctx, tenantID)
// 返回：总数、今日数量、操作统计、模块统计、趋势、热门模块

// 趋势分析
trend, err := auditService.GetTrend(ctx, tenantID, 30)
// 返回最近 30 天的每日数量
```

## 租户隔离

审计日志自动按 `tenant_id` 过滤：

```go
// 在仓储中
func (r *auditRepository) ListLogs(ctx context.Context, query *model.AuditLogQuery) ([]*model.AuditLog, int64, error) {
    db := tenantctx.FilterQuery(ctx, r.db.WithContext(ctx), "tenant_id")
    // ... 应用查询条件 ...
}
```

### 系统管理员例外

系统管理员可以查看跨租户的审计日志以满足合规和调试需求：

```go
// 当 IsMaster=true 时，FilterQuery() 不添加 tenant_id 过滤
// 允许查看所有租户日志
```

## 数据保留

日志可按配置的天数自动清理，详见下节「数据保留策略（自动化）」。手动清理端点
`DELETE /api/v1/audit/logs/cleanup` 始终可用（见「审计自身操作的留痕」）。

## 数据保留策略（自动化）

配置项 `audit.retention_days`（env `METEORX_AUDIT_RETENTION_DAYS`）控制自动保留窗口，**默认 0 = 关闭**
（向后兼容，不做任何自动删除）。开启后由 `AuditRetentionJob`（`internal/modules/audit/retention_job.go`）
每小时执行一轮 `svc.CleanupLogs(ctx, days)`，删除早于保留期的日志行。

- `Start(ctx, time.Hour)` 在 `days <= 0` 时**立即返回**，不起后台协程、零开销。
- 清理直连 service（不经 handler 的 `recordSelfOpAudit`），故任务在删除成功后**主动补记一条自指审计**
  （经 `middleware.RecordAuditLog`）：`module=audit / action=delete / risk=critical / result=success`，
  触发者记为 `system`，只落 `{days}` 与 `{deleted_count}` 摘要，不落任何正文。定时清理从此不再零痕迹。

## 防篡改完整性校验

因审计走**异步批量 + 多副本并发 flush**，严格逐行线性哈希链会强制写串行化、不可行。故采用**行级摘要 +
周期锚点链**双层方案，两者均可独立检出不同篡改，全部逻辑默认关闭、配 pepper 才生效。

配置项（`audit.integrity.*`）：

| 配置 | env | 默认 | 说明 |
|------|-----|------|------|
| `enabled` | `METEORX_AUDIT_INTEGRITY_ENABLED` | `false` | 总开关 |
| `secret_key` | `METEORX_AUDIT_INTEGRITY_SECRET_KEY` | 空 | HMAC pepper，**不落库**；未配置则整卡降级关闭 |
| `previous_secret_key` | `METEORX_AUDIT_INTEGRITY_PREVIOUS_SECRET_KEY` | 空 | 轮换过渡期的旧 pepper；仅在换 key 迁移期临时配置，迁移完成后清空即回单密钥常态 |
| `anchor_interval` | `METEORX_AUDIT_INTEGRITY_ANCHOR_INTERVAL` | `1h` | 锚点封口周期 |

**行级摘要**：每行落库前计算 `record_hash = hex(HMAC-SHA256(pepper, canonical(log)))`。`canonical`（
`service/integrity.go`）按固定顺序对全部被覆盖字段做**长度前缀稳定序列化**，含 `created_at`、`status_code`、
`duration`，**排除 `record_hash` 自身**（否则无法自证）。任一字段被改、`pepper` 不同都会使重算摘要不符。
未启用或无 pepper 时 `Compute` 返回空串——**既不写摘要也不阻断写入**（可用性优先）；`record_hash` 为空的历史行
视为 legacy，校验时**不计入失败**。

> ⚠️ router 侧（供审计中间件/批量）与 `audit.InitModule` 侧两个 `AuditService` 实例**必须共享同一
> `IntegrityGuard`**，否则两条写入路径算出的摘要不一致、校验必坏。`bootstrap/router.go` 只构造一次 guard 并同时注入两处。

**周期锚点链**：`IntegrityAnchorJob`（`integrity_job.go`）按周期把「窗口内全部行摘要」按 `(created_at, id)`
有序折叠为一个 `WindowDigest`，追加进 append-only 表 `audit_integrity_anchors`，并以 `PrevAnchorHash → AnchorHash`
链向上一条锚点（`AnchorHash` 同时纳入窗口摘要、行数与窗口边界）。窗口右端取 `now - 1min`（安全边界，规避迟到行），
**空窗口不封口**（无新行不产噪声），创世窗口无下界、落库存最早行的 `created_at`。

**校验接口**：`POST /api/v1/audit/integrity/verify`（权限码 `audit:integrity:verify`，仅平台超管）。只读重放、
不改任何数据，返回 `verdict`：

- `verified`：行级重算全部相符且锚点链完整；
- `tampered`：检出篡改行（改内容不改摘要）或窗口异常（删行/插行/重序/断链），并给出篡改行样本与异常锚点 ID；
- `pending_rehash`：存在仅在旧 pepper 下可验证的历史行（密钥轮换所致，**非篡改**），可执行 rehash 迁移；
- `not_configured`：未配 pepper，无法校验。

行级重算可发现**内容篡改**（攻击者无 pepper 无法同步 `record_hash`）；锚点链可发现**删行/插行/重序/断链**
（删除整行时行级重算无从发现，唯有窗口行数与累加摘要会失配）。

### 密钥轮换与 rehash

`secret_key`（pepper）一旦更换，旧密钥下写入的历史行与锚点在新 key 下重算必然不符——若只持一把 key，
`Verify` 会把它们一律误判为 `tampered`，无法区分「轮换」与「真篡改」。为此引入**双密钥过渡 + 显式 rehash**：

- **双密钥分类**：过渡期同时配置 `secret_key`（新）与 `previous_secret_key`（旧）。`ClassifyRow` 对每行同时
  验两把 key：命中当前 → `RowCurrent`（已验证）；仅命中旧 → `RowPrevious`（待重算，合法历史）；两把都不命中
  → `RowNone`（真篡改）；摘要为空 → `RowLegacy`。锚点折叠同理，先用新 key 复验、不命中再用旧 key 复验。
  据此 `Verify` 给出三态优先级：有真篡改 → `tampered`；否则有待重算 → `pending_rehash`；否则 `verified`。
- **rehash 迁移**：`POST /api/v1/audit/integrity/rehash`（权限码 `audit:integrity:rehash`，仅平台超管）。前置须
  `enabled` 且配了 `previous_secret_key`，否则 400。它**只重写 `RowPrevious` 行**（有旧 key 的密码学自证，绝不
  盲重包，`RowNone` 真篡改行不被洗白），把行摘要迁到新 key；随后因行摘要值改变会使旧锚点必然失效，
  **从创世按新 key 全量重建整条锚点链**（复用 `anchor_interval` 时间桶口径）。
- **重建原子性**：重建走 `ReplaceAnchors`——在**单个事务**内先删旧链再插新链，二者不可分割。避免“旧链已删、
  新链未建”中途失败导致锚点表被清空却无新链，`Verify` 反而因“无锚点可查”静默判 `verified`、丢历史防护。
- **封口边界**：重建窗口右端一律 **clamp 到 `now - grace`**（与周期任务同口径，grace 取 `IntegrityGuard.AnchorGrace`），
  绝不产出以**未来时刻**收尾的窗口；否则其后新写入的行会落进已封窗口，令校验重放时窗口内容漂移而误报。
  落在 `now - grace` 之后的新行不纳入本次重建，留待周期任务下次封口（它们仍以新 key 写入，行级校验照常通过）。
- **并发串行**：rehash 与周期任务 `IntegrityAnchorJob` 共享 `integrityAnchorMu`，重建期间不被追加锚点，避免链错乱。
- **留痕**：rehash 是挂在受保护分组下的 HTTP POST，已被审计中间件自动记录（含真实操作者、状态码、响应体计数），
  不再在 handler 内重复埋点。
- **收尾**：迁移完成、`Verify` 回到 `verified` 后，清空 `previous_secret_key` 即回到单密钥常态。

## 安全

- 请求体敏感数据脱敏：解析 JSON 后按**字段名包含关键字**匹配，命中即替换为 `***`。用包含匹配而非全等匹配，
  是为了覆盖 `old_password` / `new_password` / `initial_password` / `refresh_token` 这类变体；关键字表包含
  `password` / `passwd` / `pwd` / `token` / `secret` / `authorization` / `api_key` / `apikey` / `access_key` /
  `private_key` / `secret_key` / `credit_card` / `card_no` / `cvv` / `id_card` / `idcard` / `totp` / `otp` / `mfa`
- **递归脱敏**：脱敏不仅作用于 JSON 顶层 key，`sanitizeValue` 会**递归下钻嵌套对象与数组元素**（顶层是
  数组的批量请求同样被处理），因此 `config.db_password`、`users[].api_key` 这类嵌套明文也会被掩码
- 响应体走同一套脱敏规则（初始化密码、重置密码结果同样被掩码）
- multipart 请求体不采集（见上节），上传文件内容永远不会进入审计日志
- 审计**自身高危操作**（清理 `DELETE /logs/cleanup`、导出 `GET /logs/export`）虽被自动中间件前缀跳过，
  但在 handler 内手动补记（见「审计自身操作的留痕」），删历史 / 拖库不再零痕迹
- 记录 IP 和 User Agent 用于安全分析
- 所有写入不可变（只追加）
- **防篡改完整性校验**：行级 HMAC 摘要（`record_hash`）+ 周期锚点链（`audit_integrity_anchors`）双层覆盖，
  内容篡改与删/插/重序/断链均可检出；pepper 不落库，未配置时整卡关闭、不阻断写入（见「防篡改完整性校验」）
- **自动保留清理留痕**：`audit.retention_days` 开启后，`AuditRetentionJob` 每次清理主动补记一条
  `audit/delete/critical` 自指审计（触发者 `system`），删历史不再零痕迹（见「数据保留策略（自动化）」）

## 实现清单

| 功能 | 状态 |
|------|------|
| 自动审计中间件 | ✅ 已完成 |
| 手动审计上下文（auditctx） | ✅ 已完成 |
| Before/after 值追踪 | ✅ 已完成 |
| 操作命名约定 | ✅ 已完成 |
| 基于模块的操作 | ✅ 已完成 |
| 批量异步处理 | ✅ 已完成 |
| 查询时租户隔离 | ✅ 已完成 |
| Dashboard 统计 | ✅ 已完成 |
| 趋势分析 | ✅ 已完成 |
| 热门模块 | ✅ 已完成 |
| 数据清理 | ✅ 已完成 |
| 请求/响应脱敏 | ✅ 已完成 |
| 嵌套对象/数组递归脱敏（`sanitizeValue`） | ✅ 已完成 |
| 审计自身操作留痕（cleanup 严重 / export 高风险） | ✅ 已完成 |
| multipart 正文隔离（导入/附件） | ✅ 已完成 |
| trace_id 与 OTel/Jaeger 链路同源关联 | ✅ 已完成 |
| 公开端点手动审计覆盖（登录/注册/重置/OAuth 回调） | ✅ 已完成 |
| 手动审计携带 trace/request/session 关联 | ✅ 已完成 |
| 保留策略 | ✅ 已完成（`audit.retention_days`，默认关闭） |
| 防篡改行级摘要（record_hash） | ✅ 已完成（默认关闭，需配 pepper） |
| 防篡改周期锚点链（audit_integrity_anchors） | ✅ 已完成（默认关闭，需配 pepper） |
| 完整性校验接口（verify） | ✅ 已完成（`audit:integrity:verify`） |
| 密钥轮换双密钥分类 + rehash 迁移 | ✅ 已完成（`audit:integrity:rehash`，仅轮换过渡期可用） |

## 最佳实践

1. 在服务层对重要操作**始终**使用 `auditctx`
2. **禁止**在审计日志中包含敏感数据（密码、Token）
3. 跨模块**使用**一致的操作名称
4. 为可追溯性**包含** `resource_id`
5. 对变更操作**捕获** before/after 以支持回滚分析
6. 查询审计日志时**遵守**租户隔离
7. **监控**审计日志量以防止存储问题