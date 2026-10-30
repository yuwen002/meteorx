# 系统与运维 API 接口说明

> Base URL：根路径（健康检查与指标不在 `/api/v1` 之下）
> 路由注册：`internal/bootstrap/router.go`
> 处理器：健康检查（router.go 内联）、CSP 上报 `internal/middleware/csp_report.go`

---

## 1. 接口总览

| 方法 | 路径 | 功能 | 鉴权 |
|------|------|------|------|
| GET | `/health` | 基础健康检查（存活探针） | 无 |
| GET | `/health/ready` | 深度就绪检查（数据库 / Redis / 运行时 / 连接池） | 无 |
| GET | `/metrics` | Prometheus 指标 | 无（建议在网关层限制来源） |
| POST | `/api/v1/csp-report` | 浏览器 CSP 违规上报 | 无（受公开分组 IP 限流） |
| GET | `/api/v1/ws` | WebSocket 实时推送 | 需登录，协议见 [协作即时消息 API](chat-api.md) |

隔离边界：

- `/health`、`/health/ready`、`/metrics`、`/uploads/*` 注册在根路径，**不经过** `/api/v1` 的认证、维护模式、租户状态门禁、审计与租户限流。
- `POST /api/v1/csp-report` 虽位于 `/api/v1`，但注册在**公开分组**：无认证、不受租户隔离，仅受公开分组 IP 限流保护。

---

## 2. 健康检查

### 2.1 基础健康检查

`GET /health`

**描述**：进程存活探针，不访问任何外部依赖，适合容器 `livenessProbe`。

**响应（200）**：
```json
{"status":"ok"}
```

> 该端点直接写裸 JSON，不使用统一响应封装（无 `data` 包装）。

### 2.2 深度就绪检查

`GET /health/ready`

**描述**：检查数据库与 Redis 连通性，并附带运行时与连接池读数，适合容器 `readinessProbe` 与容量排查。

**响应（200）**：
```json
{
  "status": "ok",
  "checks": {
    "database": "ok",
    "redis": "ok"
  },
  "runtime": {
    "goroutines": 42,
    "heap_alloc_mb": 12,
    "sys_mb": 30,
    "num_gc": 7
  },
  "db_pool": {
    "open_connections": 5,
    "in_use": 1,
    "idle": 4,
    "wait_count": 0,
    "wait_duration_ms": 0
  }
}
```

**status 取值**：

| status | HTTP | 含义 |
|--------|------|------|
| `ok` | 200 | 数据库与 Redis 均正常 |
| `degraded` | 200 | Redis 未配置（`not_configured`）或不可用（`unavailable`），业务仍可继续 |
| `error` | 503 | 数据库连接不可用 |

**checks 取值**：

| 键 | 取值 | 说明 |
|----|------|------|
| `database` | `ok` / `error: <原因>` | 获取连接池或 `Ping` 失败时记 error |
| `redis` | `ok` / `unavailable` / `not_configured` / `error: <原因>` | Redis 关闭时整体降级但不返回 503 |

---

## 3. Prometheus 指标

`GET /metrics`

**描述**：以 Prometheus 文本格式输出指标，由全局中间件 `promMetrics.Middleware` 采集 HTTP 请求维度，并包含 Go 运行时指标（GC、堆、协程等）。

**响应**：`Content-Type: text/plain`，Prometheus 文本格式（非 JSON）。

**部署建议**：

- 仅允许内网/采集器访问（nginx 层限制来源 IP 或另开管理端口）；
- 该端点对全部 HTTP 方法开放（`r.Handle`），实际只应使用 `GET`。

---

## 4. CSP 违规上报

`POST /api/v1/csp-report`

**鉴权**：无。浏览器按 CSP 规范自动发送，无法携带 `Authorization`。

**来源**：nginx 下发的 `Content-Security-Policy` 头中 `report-uri /api/v1/csp-report`（见 `deploy/nginx/nginx.conf`）。

**请求体**（浏览器生成，字段名由 CSP 规范约定）：
```json
{
  "csp-report": {
    "document-uri": "https://app.example.com/wiki/01H...",
    "referrer": "",
    "blocked-uri": "inline",
    "violated-directive": "script-src-elem",
    "effective-directive": "script-src-elem",
    "status-code": 200,
    "sample": ""
  }
}
```

**响应**：始终 `204 No Content`，无响应体。

**处理约定**：

| 场景 | 行为 |
|------|------|
| 请求体超过 8 KB | 按 8 KB 截断后解析 |
| JSON 非法 / 被截断 | 静默丢弃，仍返回 204 |
| `blocked-uri` 与 `violated-directive` 均为空 | 视为无效上报，跳过 |
| 有效上报 | 以 `[csp-violation]` 前缀写 Warn 日志（`sample` 截断至 120 字节） |

**设计要点**：

- 只落日志、不落库、不写审计、不做租户隔离（无用户身份）；
- 静默失败：任何解析错误都不回吐给浏览器，避免信息泄露并确保页面渲染不受影响；
- 不记录 `original-policy`（每次重复且体积大）。

---

## 5. 其他根路径端点

| 路径 | 说明 |
|------|------|
| `/uploads/*` | 上传文件静态服务，需 URL 携带短时效 HMAC 签名（`middleware.SignedUploadsHandler`），禁止目录列举；签名密钥链与 `jwt.secret` 解耦，可单独轮换 |
| `/api/v1/ws` | WebSocket 实时推送入口，需登录；连接与消息协议见 [协作即时消息 API](chat-api.md) |
