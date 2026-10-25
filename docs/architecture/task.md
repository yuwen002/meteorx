# 任务模块架构

## 概述

任务模块把「个人待办」与「租户团队协作」统一到同一张 `tasks` 表，用 `visibility` 区分可见范围；
**不接入细粒度 RBAC**，仅要求登录，访问控制完全由数据归属（创建人 / 负责人 / 同租户）决定。
在 CRUD 之上叠加一层可选的 AI 辅助能力（只读 / 草稿，不直接写库）。

## 分层与文件

| 层 | 文件 | 职责 |
| --- | --- | --- |
| 入口 | `task/module.go` | 装配仓储 / 服务 / handler，注册路由 |
| 模型 | `task/model/task.go` | 领域模型、三组枚举校验、重复周期推算 |
| 仓储 | `task/repository/*` | GORM 实现、可见性过滤、回收站、统计聚合、用户目录 |
| 服务 | `task/service/task_service.go` | 归属校验、状态副作用、周期任务生成、提醒扫描 |
| 服务（AI） | `task/service/task_ai_service.go` | 自然语言解析 / 建议 / 扩写 / 摘要 |
| 接口 | `task/handler/*` | HTTP 处理与 DTO 转换 |
| 定时 | `task/reminder_job.go` | 到期 / 逾期提醒扫描 |

## 数据模型

| 字段 | 说明 |
| --- | --- |
| `id` | ULID 主键 |
| `tenant_id` | 租户隔离键 |
| `creator_id` / `assignee_id` | 创建人 / 负责人（个人任务负责人强制为创建人） |
| `title` / `description` | 标题 / 描述 |
| `status` | `pending` / `in_progress` / `completed` |
| `priority` | `low` / `normal` / `high` / `urgent` |
| `due_date` | 截止时间，可空 |
| `remind_before` | 任务级提醒提前量（纳秒存储，API 以 Duration 传入，单位语义为分钟级配置的覆盖值） |
| `reminder_sent_at` | 最近一次提醒时间，提醒冷却基准 |
| `tags` | JSON 文本存储的字符串数组 |
| `visibility` | `personal` / `tenant` |
| `recurrence` | 空 / `daily` / `weekly` / `monthly` |
| `started_at` | 首次进入 `in_progress` 时写入，之后不清空（用于计算处理耗时） |
| `completed_at` | 仅 `completed` 有值，退回非完成态时清空 |
| `deleted_at` | 领域模型用 `*time.Time` 表达软删，仓储层显式过滤未删除行 |

`creator_name` / `assignee_name` 是**非持久化的富化字段**，读取时由服务层按用户目录批量补全，不参与仓储读写。

## 访问控制

| 场景 | 查看 | 编辑 / 完成 |
| --- | --- | --- |
| 个人任务（`personal`） | 仅创建人 | 仅创建人 |
| 租户任务（`tenant`） | 同租户成员 | 创建人恒可；负责人亦可 |

实现上是两个纯函数：`canView(task, tenantID, userID)` 与 `canEdit(task, userID)`。
所有读写入口先校验 `task.TenantID == tenantID`，不匹配一律返回 `ErrTaskNotFound`（不泄露存在性）。

## 状态与副作用

```
pending ⇄ in_progress → completed
   ↑                      │
   └──────── reopen ──────┘
```

- 进入 `in_progress`：若 `started_at` 为空则写入（只写一次）；
- 进入 `completed`：写 `completed_at`；若该任务配置了 `recurrence` 且是**首次**从非完成态转完成，
  自动按周期推进生成下一条 `pending` 任务（生成失败不阻断本次完成）；
- 退回 `pending` / `in_progress`：清空 `completed_at`，保留 `started_at`。

## 周期任务

`model.NextOccurrence` 负责推算下一次截止时间：

- `daily` / `weekly` 走天数加法（`AddDate(0,0,1)` / `(0,0,7)`）；
- `monthly` 定位到目标月并按其**最后一天钳制**，避免 Go `AddDate` 的溢出规范化
  （1/31 → “2/31” → 3/3），即 1/31 → 2/28（闰年 2/29）、3/31 → 4/30；
- 时分秒原样保留，与「每月最后一天」的主流日历语义一致。

周期任务必须拥有截止时间，否则创建 / 更新时返回校验错误。

## 批量与回收站

| 端点 | 语义 |
| --- | --- |
| `POST /tasks/batch/complete` | 批量完成 |
| `POST /tasks/batch/status` | 批量改状态 |
| `POST /tasks/batch/assign` | 批量指派负责人（含同租户校验） |
| `POST /tasks/batch/delete` | 批量软删除 |

批量操作**逐条**按单条归属规则校验，无权或不存在项静默跳过，返回 `affected` 实际生效数（幂等语义）。
回收站 `GET /tasks/deleted` 只列出当前用户自己删除的任务；`restore` / `permanent` 限创建人或超管。

## 统计口径

`GET /tasks/stats` 返回各状态数量、逾期数、`completion_rate`（完成数 / 总数，保留四位小数）
与 `avg_handle_seconds`（已完成任务 `started_at → completed_at` 的平均耗时）。

聚合依赖 MySQL 方言：`TIMESTAMPDIFF`、`FIELD()`、`JSON_CONTAINS`、`TIMESTAMPADD`。
仓储层单测用 sqlmock 只匹配 SQL 文本，**语义验证需开启方言测试**（见“测试”）。

## 提醒链路

```
TaskReminderJob（默认 30m 扫描）
   └─ FindDueForReminder  ← 截止日窗口 / 逾期 + reminder_sent_at 冷却
        └─ notify.Manager.NotifyTaskReminder（WS 定向推送，收件人是负责人 userID）
             └─ MarkReminderSent 回写冷却基准
```

配置项（`task_reminder` 段 / `METEORX_TASK_REMINDER_*`）：

| 配置 | 默认 | 说明 |
| --- | --- | --- |
| `horizon` | 24h | 全局提醒窗口（任务级 `remind_before` 非空时覆盖） |
| `interval` | 30m | 扫描间隔 |
| `overdue_cooldown` | 24h | 逾期重复提醒冷却 |
| `batch_limit` | 200 | 单轮处理上限 |

**冷却重置规则**：只有当 `due_date` 或 `remind_before` **实质变化**时才清空 `reminder_sent_at`。
前端整体回传任务对象时未变更的截止日同样会带上，若不做值比较，每次保存都会抹掉冷却基准并立即重发提醒。
比较按**秒**精度进行，因为响应以秒级输出 `due_date`，按纳秒比较会把原样回传误判为改期。

已知约束：提醒仅经 WebSocket 推送，收件人离线时消息不落库、不补推（站内消息中心为后续补齐项）。

## 负责人同租户校验

`service` 通过**类型断言**探测仓储是否具备 `IsTenantMember(ctx, tenantID, userID)` 能力
（`repository/user_directory.go` 实现）。具备时在创建 / 更新 / 批量指派中拦截跨租户或不存在的负责人，
返回 400；未注入时跳过校验以保持向后兼容。

## AI 辅助层

| 端点 | 能力 |
| --- | --- |
| `GET /tasks/ai/capabilities` | 能力探测（前端据此门控按钮） |
| `POST /tasks/ai/parse` | 自然语言建任务草稿（注入当前时间以解析“下周五/三天后”） |
| `POST /tasks/ai/suggest` | 优先级 / 标签建议 |
| `POST /tasks/ai/expand` | 描述扩写 / 拆解 |
| `POST /tasks/ai/summary` | 今日 / 本周 / 逾期 / 全部 摘要 |

- 依赖 `internal/ai`（`Provider` 抽象 + OpenAI 兼容客户端），`ai.enabled=false` 时全部返回 503；
- `ErrAIUnavailable` → 503，`ErrAIRuntime` → 502，参数校验失败 → 400；
- 所有能力**不直接写库**，用户确认后走原有创建 / 更新端点；
- 不新增 RBAC 权限码，挂在既有的受保护 `/tasks` 分组内。

## 时区

截止日与提醒窗口统一使用 `database.timezone`（默认 `Asia/Shanghai`）——`bootstrap.InitDB` 在建连接前
强制 `time.Local`，使 `time.Now()`、时间解析与 DSN `loc=Local` 与容器 `TZ` 环境变量解耦。

## 测试

```bash
# service / handler / repository
go test -count=1 ./internal/modules/task/...

# MySQL 方言语义（TIMESTAMPADD / JSON_CONTAINS / FIELD / TIMESTAMPDIFF）
$env:METEORX_MYSQL_DIALECT_TEST='1'; go test -count=1 ./internal/modules/task/repository
```

- 进程内集成测试基于 sqlite，**不支持**上述 MySQL 函数，`internal/integration` 为此绕开了默认排序；
- `task_dialect_mysql_test.go` 默认 `t.Skip`，要求库名以 `_test` 结尾，防止误操作业务库；
- 提醒冷却的“未变更不重置”有正反两条用例守护（未变更保持 / 真改期重置），避免修复削过头。
