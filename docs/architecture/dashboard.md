# 运营看板架构

## 概述

运营看板面向**平台超级管理员**，把租户 / 用户 / 订阅 / 审计等分散数据聚合成可决策的视图，
并额外提供「租户健康度」与「慢查询」两个运维视角。全部端点只读（除慢查询缓冲重置），
均基于既有数据表直查，不引入新的汇总表。

## 分层与文件

| 层 | 文件 | 职责 |
| --- | --- | --- |
| 入口 | `dashboard/module.go` | 装配 handler 与路由 |
| 路由 | `dashboard/routes.go` | 6 个端点，统一挂 `AutoRequirePermission` |
| 模型 | `dashboard/model/dashboard.go` | 概览、趋势、健康度查询模型 |
| 仓储 | `dashboard/repository/dashboard_repository.go` | 计数 / 分组聚合 |
| 仓储 | `dashboard/repository/dashboard_health_repository.go` | 健康度批量聚合与评分 |
| 服务 | `dashboard/service/dashboard_service.go` | 编排与 DTO 转换 |
| 接口 | `dashboard/handler/dashboard_handler.go` | 参数校验与响应 |

## 端点与权限

| 方法 | 路径 | 权限码 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/admin/dashboard/overview` | `admin:dashboard:list` | 平台运营总览 |
| `GET` | `/admin/dashboard/trend?days=N` | `admin:dashboard:list` | 增长趋势（默认 30，上限 365） |
| `GET` | `/admin/dashboard/tenants-health` | `admin:dashboard:list` | 租户健康度列表 |
| `GET` | `/admin/dashboard/tenants-health/{id}` | `admin:dashboard:list` | 单租户健康详情 |
| `GET` | `/admin/dashboard/slow-queries` | `admin:dashboard:list` | 慢查询快照 |
| `POST` | `/admin/dashboard/slow-queries/reset` | `admin:dashboard:create` | 清空慢查询缓冲 |

## 总览（overview）

聚合四类数据：

- **租户**：总数 / 启用 / 禁用 / 今日新增；
- **用户**：总数 / 今日新增；
- **订阅**：总数 / 生效 / 到期 / 取消；
- **审计**：日志总数 / 成功 / 失败 / 按操作与模块分布。

容错策略：单表查询失败只记日志并置零，不让一个坏查询拖垮整页；
软删除行不计入统计（沿用各表 `deleted_at IS NULL` 口径）。

## 增长趋势（trend）

- 按**业务时区**（`database.timezone`）的自然日分桶，统计新增租户 / 新增用户 / 审计日志数；
- 缺失日期补零，输出连续序列（前端可直接画柱状图）；
- `days` 默认 30，非法值或超出 1–365 返回 400；
- 软删除记录不计入增长。

测试注意：用例不得写死绝对日期，必须用与实现相同的口径（`timeutil.StartOfDay(time.Now())`）
生成相对日期键，否则只在写入当天或非 UTC runner 上通过。

## 租户健康度

评分模型（0–100）：

| 维度 | 满分 | 口径 |
| --- | --- | --- |
| 订阅状态 | 40 | 生效 40 / 到期 15 / 取消或无订阅 0 |
| 近 7 天活跃 | 30 | `30 × min(1, 近 7 天审计数 / 20)` |
| 资源足迹 | 30 | `30 × min(1, (用户数 + 任务数 + 文档数) / 100)` |

分级：`≥70 healthy`、`≥40 moderate`、`<40 at_risk`。

列表接口支持分级 / 关键字筛选与分页（默认页大小 20），并返回 healthy / moderate / at_risk 的分级汇总。
实现上用 `IN + GROUP BY` 批量聚合，避免逐租户查询造成 N+1。

## 慢查询看板

`internal/dbmonitor` 以 GORM `logger.Interface` 包装器在**进程内**采集超过阈值的 SQL 到内存环形缓冲
（容量 200），无需开启外部慢查询日志。

| 配置 | 默认 | 说明 |
| --- | --- | --- |
| `database.slow_query_ms` | 500 | 超过该值（毫秒）的 SQL 记入缓冲；`0` 回落默认，负值关闭采集 |

- 采集的是进程内本地读数，重启即清空，不适合作为长期审计数据源；
- 与 OTel 的 SQL span、`/health/ready` 的 `db_pool` 读数互补：前者看单条语句慢，后者看连接池是否耗尽；
- `POST /slow-queries/reset` 用于排障后清零，改动走 `admin:dashboard:create` 权限。

## 前端页面

| 页面 | 路由 | 说明 |
| --- | --- | --- |
| 工作台 / 运营看板 | `/dashboard` | 平台管理员看总览 + 趋势（零依赖内联 SVG 柱状图，支持 7/30/90 天切换） |
| 租户健康度 | `/system/dashboard/tenant-health` | 分级筛选、评分明细 |
| 慢查询 | `/system/dashboard/slow-queries` | 四指标卡 + 明细表 + 5 秒自动刷新 + 重置 |

所有页面均声明 `requireMaster`，路由守卫会强制校验（仅 `is_master` 放行），
避免被授予 `admin:*` 权限码的租户管理员越级进入平台页。

## 测试

```bash
go test -count=1 ./internal/modules/dashboard/...
go test -count=1 ./internal/dbmonitor/...
```

- 仓储层用 sqlmock 验证 SQL 形状（group-by-day、`IN + GROUP BY`、软删过滤）；
- 健康度评分有纯函数用例覆盖订阅三态、活跃度与足迹的边界（0 / 恰好达标 / 超上限）；
- 慢查询缓冲覆盖环形容量、阈值关闭、重置三条路径。
