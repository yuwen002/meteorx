# 数据导出架构

## 概述

导出模块以「异步任务 + 下载中心」的方式提供批量数据导出，避免大结果集阻塞 HTTP 请求。
核心抽象是 **Exporter（数据生产）× RowSink（文件格式）** 两维解耦：一套数据可导出为多种格式，
新增资源或新增格式互不影响。

- 资源：`user` / `task` / `audit_log` / `wiki_document`
- 格式：`csv`（默认）/ `xlsx` / `json` / `pdf`
- 生命周期：`pending → running → completed / failed`

## 分层与文件

| 层 | 文件 | 职责 |
| --- | --- | --- |
| 入口 | `export/module.go` | 装配、注册导出器、启动 worker / 清理任务 |
| 路由 | `export/routes.go` | 5 个受保护端点，全部走自动权限推导 |
| 模型 | `export/model/export_job.go` | 任务状态、资源与格式常量、扩展名 / Content-Type |
| 仓储 | `export/repository/*` | 状态机条件更新、租约心跳、回收、过期清理 |
| 服务 | `export/service/export_service.go` | 创建准入、删除编排、过期清理 |
| 执行器 | `export/service/exporter/*` | `Exporter` 接口、`RowSink` 四种实现、资源导出器 |
| worker | `export/worker/pool.go` | 轮询、抢占、心跳、回收、优雅关闭 |
| 定时 | `export/expiry_job.go` | 文件过期清理 |

## 状态机与执行租约

```
pending ──抢占──> running ──成功──> completed
   ↑                  │
   │                  ├──失败──> failed
   └──租约过期回收────┘（attempts < 3）
```

抢占是一次**条件更新** `pending → running`，同时写入：

| 字段 | 含义 |
| --- | --- |
| `lease_expires_at` | 租约到期时间（TTL 60s），执行期间每 TTL/3（20s）心跳续租 |
| `lease_token` | 本次执行的归属令牌（随机），终态与续租的 `WHERE` 都带它 |
| `attempts` | 累计抢占次数，用于封顶重驱 |

**为什么需要令牌**：若回写只按 `id` 匹配，一次心跳断供超过 TTL 的「僵尸执行」（笔记本睡眠、DB 超时）
会把新持有者的租约拉到未来、使其逃离回收，并覆写新执行的状态与产出文件。带令牌后，旧执行的迟到回写
因令牌不匹配而自然空转。

**回收**：`ReclaimExpiredLeases` 在服务启动时与每轮 poll 之前执行（先回收再拉取，回队任务本轮即可重驱），
只处理租约已过期的行，因此多副本安全；`attempts` 达到 3 次直接置 `failed`，避免「必然撞死进程的任务」
形成崩溃循环。历史行租约为 `NULL` 时降级按 `started_at` 是否早于一个 TTL 之前判定。

## 队列准入

单租户未完结任务（`pending + running`）数量达到 `export.max_active_jobs`（默认 10）时，
新建任务返回 **429**。计数直接复用「活跃任务」口径，因此租约回收后的回队任务会重新占用额度。

## 文件与过期

- 产物落盘在 `export.dir`（默认 `data/exports`），文件名带任务 ID 与格式扩展名；
- 任务行记录 `expires_at`（创建时间 + `retention_days`，默认 7 天）；
- `ExportExpiryJob` 每 6 小时扫描一次，单轮上限 200 条，删除文件后再删除记录。

**删除顺序：先删文件、再删记录**。记录是文件的唯一引用（清理只扫 `expires_at` 命中的行），
记录一删，文件就永远无法被扫到，只能人工上盘清理。因此：

| 场景 | 行为 |
| --- | --- |
| 文件已不存在（`ErrNotExist`） | 照常删记录（避免记录堆积占满单轮窗口） |
| 文件被占用（Windows 句柄） | 返回 `ErrExportFileInUse`（HTTP 409），**保留记录**供下一轮重试 |
| 用户主动删除 | 同样「先文件、后记录」，删除职责在 `service.Delete`，handler 不自行删文件 |

已知权衡：永久删不掉的文件会每 6 小时重试一次并留一行告警（未引入失败计数）。

## 优雅关闭

终态回写（`MarkCompleted` / `markFailed`）派生自 `context.WithoutCancel(ctx)` + 5s 超时，
不使用随关闭被取消的应用级 ctx——否则导出中途遇到优雅关闭时回写失败，任务会永久停在 `running`，
并因 `CountActiveByTenant` 把它计入额度而**泄漏一个租户队列槽位**。
`context.Canceled` / `DeadlineExceeded` 会映射为用户可读文案，不回显内部错误字符串。

## 接口与权限

| 方法 | 路径 | 权限码（自动推导） |
| --- | --- | --- |
| `GET` | `/exports` | `export:list` |
| `POST` | `/exports` | `export:create` |
| `GET` | `/exports/{id}` | `export:read` |
| `GET` | `/exports/{id}/download` | `export:read` |
| `DELETE` | `/exports/{id}` | `export:delete` |

租户隔离：任务与文件均只在创建者所属租户内可见 / 可下载 / 可删除。下载响应按格式设置
`Content-Type`，中文文件名使用 RFC 5987 `filename*=UTF-8''<percent-encoded>` 并提供 ASCII 兜底名。

## 扩展指引

**新增可导出资源**：实现 `Exporter` 接口（`Resource()` + `Stream(ctx, tenantID, filters, sink)`），
在 `export/exporters.go` 的 `registerExporters` 中 `Register`，并在 `model.ValidResourceType` 放行。

**新增格式**：实现 `RowSink`（`WriteHeader` / `WriteRows` / `Close`），在 `exporter.NewSink` 分派，
并在 `model` 中补格式常量、扩展名与 Content-Type。PDF 依赖 CJK 字体，缺失时构造 sink 直接报错。

## 配置

| 配置 | 默认 | 说明 |
| --- | --- | --- |
| `export.dir` | `data/exports` | 产物目录 |
| `export.retention_days` | 7 | 文件保留天数 |
| `export.worker_count` | 3 | 并发 worker 数 |
| `export.batch_size` | 1000 | 单批读库行数（流式写出） |
| `export.max_active_jobs` | 10 | 单租户未完结任务上限 |

## 测试

```bash
go test -count=1 ./internal/modules/export/...
go test -count=1 ./internal/integration -run Export
```

- 租约 / 心跳 / 回收的用例把 TTL、心跳周期、轮询间隔缩到毫秒级（`Pool` 中这些是带默认值的字段而非常量），
  用阻塞型导出器稳定复现「执行中」中间态，而非靠 `sleep` 撞时序；
- 内存库统一使用同名 shared-cache DSN（`file:<name>?mode=memory&cache=shared`）并钉死单连接，
  裸 `:memory:` 在心跳协程与主线程并发时会随机撞出 `no such table`；
- `ZombieWritesIgnoredAfterReclaim` 锁住「回收后旧执行的续租 / 完成 / 失败三种回写全部空转」。
