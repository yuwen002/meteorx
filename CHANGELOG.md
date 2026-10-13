# 更新日志（Changelog）

本项目的所有重要变更都记录在此文件中。

格式遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### 新增
- 导出队列上限可配置：`export.max_active_jobs`（默认 10）覆盖原硬编码常量，未完结任务达上限新建返回 429

### 变更
- OpenTelemetry `service_version` 默认值与发布版本对齐（1.0.0 → 1.1.0），发版时随本文件一并更新

### 修复
- 套餐配额遗留数据归一化：配额列引入前创建的套餐行其 `storage_limit`/`api_rate_limit`/`wiki_space_limit`/`user_limit` 数据库默认值为 0，此前读路径原样返回 0，导致前端展示为「0 B / 0 次/分 / 0 个」，且消费端以 `limit < 0` 判不限时会把 0 误当作「上限为 0」而拦截全部上传/请求/建空间。现于 repository `toDomain` 读路径统一将 0 归一为 -1（不限），与写路径 `CreatePlan` 归一化口径对齐；补 `plan_repository_test.go` 覆盖遗留 0 值归一与显式值保留两条路径

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
