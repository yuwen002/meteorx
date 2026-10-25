# 站内消息中心 API 接口说明

> Base URL: `/api/v1`  
> 所属模块：`internal/modules/message`  
> 路由注册：`internal/modules/message/module.go`  
> Handler：`internal/modules/message/handler/message_handler.go`  
> DTO：`internal/modules/message/dto/message_dto.go`  
> 前端页面：`web-admin/src/views/message/index.vue`（消息中心，路由 `/messages`）+ `web-admin/src/components/MessageBell.vue`（顶栏铃铛）

站内消息是**持久化的收件箱**：业务通知在发送时落库为「人 × 消息」一行，
在线用户额外收到 WebSocket 角标推送；离线用户下次打开页面拉未读数即可补齐。
因此它解决的是实时推送解决不了的两件事——**离线可达**与**已读状态**。

> 与「通知公告」的区别：公告是平台/租户维度的**广播内容**（`/announcements`，有列表页），
> 收件箱是**按人**的消息投递记录（同一条公告若定向到租户，会给该租户成员各写一行）。
> 与顶栏「通知中心」的区别：后者展示公告与审计告警的**实时快照**，不记录已读。

---

## 1. 接口总览

| 方法 | 路径 | 功能 | 权限 |
|------|------|------|------|
| GET | `/messages` | 分页查询我的收件箱 | 需登录 |
| GET | `/messages/unread-count` | 未读总数 + 分类明细 | 需登录 |
| PUT | `/messages/{id}/read` | 标记单条已读（幂等） | 需登录 |
| PUT | `/messages/read-all` | 全部已读（可按分类） | 需登录 |
| DELETE | `/messages/{id}` | 删除单条消息 | 需登录 |

**说明：**
- 全部接口需登录，但**不挂 RBAC 权限码**：收件箱按「数据归属」鉴权，普通成员也应能看自己的消息。
- 所有查询与写操作的 `WHERE` 都带 `user_id = 当前用户`，他人的消息既不可见、也不可代为标记已读或删除。
- 静态段（`/unread-count`、`/read-all`）先于 `/{id}` 注册，避免被路径参数吞掉。

---

## 2. 数据结构

### 2.1 MessageResp（消息响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 消息 ID（ULID） |
| category | string | 分类：`system` / `announcement` / `task` / `plan` / `audit` |
| title | string | 标题 |
| content | string | 正文（纯文本，前端按纯文本渲染） |
| link | string | 前端跳转路由（如 `/task`），空表示不可跳转 |
| priority | int | 优先级：`0` low / `1` normal / `2` high / `3` urgent |
| source_type | string | 来源类型（`task` / `announcement` / `subscription` / `tenant_cancel_request`） |
| source_id | string | 来源 ID（如任务 ID、剩余天数、审批结果），便于按业务追溯 |
| read_at | string \| null | 已读时间（RFC3339），`null` 表示未读 |
| created_at | string | 创建时间（RFC3339） |

### 2.2 UnreadCountResp（未读数响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| total | int64 | 未读总数（顶栏铃铛角标） |
| by_category | object | 分类 → 未读数，供页签角标使用；无未读的分类不出现 |

```json
{
  "total": 5,
  "by_category": { "task": 3, "plan": 2 }
}
```

### 2.3 MarkAllReadReq（全部已读请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| category | string | 否 | 合法分类时只清该分类；为空或省略表示全部已读 |

请求体可为空（`Content-Length: 0` 时按「全部已读」处理）。

---

## 3. 接口详细说明

### 3.1 我的收件箱列表

`GET /api/v1/messages`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页条数，默认 20，最大 100 |
| category | string | 按分类筛选；非法分类返回 400（不静默当作全部） |
| unread | bool | `true` / `1` 时只看未读 |
| keyword | string | 标题模糊匹配（`LIKE %keyword%`） |

**成功响应（200）：** 分页响应，`data` 为 MessageResp 数组，按 `created_at DESC` 排序。

```json
{
  "data": [
    {
      "id": "01JB8Z3K7Q9X2V5N4M6T8R1S0P",
      "category": "task",
      "title": "任务即将到期",
      "content": "任务「季度复盘」将在 2 小时后到期",
      "link": "/task",
      "priority": 1,
      "source_type": "task",
      "source_id": "01JB8Y...",
      "read_at": null,
      "created_at": "2026-10-23T10:00:00+08:00"
    }
  ],
  "pagination": { "page": 1, "page_size": 20, "total": 1 },
  "request_id": "req_..."
}
```

**业务规则：**
- 查询固定附加 `user_id = 当前用户`，跨租户/跨用户数据天然不可见。
- `category` 非法时返回 400，避免前端页签拼错得到「看起来正常但内容不对」的列表。

### 3.2 未读数

`GET /api/v1/messages/unread-count`

**成功响应（200）：** UnreadCountResp。

**使用方式：**
- 顶栏铃铛与侧边栏菜单角标读 `total`；
- 页面分类页签读 `by_category[分类]`；
- 登录后主动拉一次（补齐离线期间的消息），之后由 WebSocket `unread_count` 推送增量覆盖。

### 3.3 标记单条已读

`PUT /api/v1/messages/{id}/read`

**成功响应（200）：** `{"data": {"id": "..."}}`

**业务规则：**
- **幂等**：已读消息重复标记仍返回 200，且不覆写首次阅读时间
  （更新语句带 `read_at IS NULL`，影响行数为 0 时再按归属查一次，区分「已读」与「不存在」）。
- 他人消息返回 **404**（与「不存在」同一文案），不暴露消息存在性。

### 3.4 全部已读

`PUT /api/v1/messages/read-all`

**请求体（可选）：**
```json
{ "category": "task" }
```

**成功响应（200）：**
```json
{ "data": { "updated": 3 } }
```

**业务规则：**
- 不传 `category` / 传空串 → 清除当前用户全部未读；
- 传合法分类 → 只清该分类，其它分类未读保留；
- 非法分类 → 400。

### 3.5 删除消息

`DELETE /api/v1/messages/{id}`

**成功响应（200）：** `{"data": {"id": "..."}}`

**业务规则：**
- 物理删除（收件箱是投递记录，无回收站语义）；
- 他人消息返回 404。

---

## 4. 附录

### 4.1 消息来源 → 分类映射

`Deliver` 未显式指定 `Category` 时，按通知业务类型（`Type`）映射：

| 通知类型 | 分类 | 触发点 |
|----------|------|--------|
| `task_assigned` / `task_due`（`task` 前缀） | `task` | 任务指派、任务到期提醒 |
| `announcement` | `announcement` | **租户定向**公告发布（平台级公告不落库） |
| `subscription_expiry` | `plan` | 套餐/订阅临期提醒 |
| `cancel_request` | `system` | 租户注销审批通过/驳回 |
| `alert` | `audit` | 审计告警（收件人为规则邮箱反查出的平台账号） |
| 其它未知类型 | `system` | 兜底：宁可分类不准，也不让消息丢失 |

`audit` 分类由 `NotifyAlert` 写入：规则 `notify_targets` 里的邮箱经 user 仓储反查得到账号后
才落收件箱，未注册账号的外部邮箱不产生记录；告警正文含用户行为明细，**不按租户或全平台展开**。
告警的 WebSocket 实时推送走 `ws.SendAlert`（仅推给触发用户），与收件箱是两条独立路径。

### 4.2 错误码映射

| 状态码 | 场景 |
|--------|------|
| 400 | 非法消息分类（列表筛选 / 全部已读）；消息 ID 为空 |
| 401 | 未登录或 Token 失效（上下文缺失用户 ID） |
| 404 | 消息不存在，或不属于当前用户（含标记已读 / 删除） |
| 500 | 列表 / 未读数 / 标记 / 删除的存储异常 |

### 4.3 配置项（`internal/config/config.yaml`）

```yaml
message:
  retention_days: 90    # 收件箱消息保留天数；未配置(0)回落 90，负值表示不清理
```

### 4.4 后台任务

| 任务 | 周期 | 职责 |
|------|------|------|
| MessageExpiryJob | 每 1 小时 | 物理删除 `created_at` 早于「当前时间 - retention_days」的消息，单轮不设条数上限（走 `created_at` 索引）；`retention_days` 为负值时整体关闭，连 goroutine 都不启动 |

启动编排：`bootstrap/app.go` 在导出清理任务之后调用 `StartMessageExpiryJob(ctx, db, cfg)`，
随应用级 ctx 退出而停止。

### 4.5 WebSocket 推送协议

落库成功后，服务端向每个收件人定向推送未读角标（推送失败只告警，不影响落库与接口结果）：

```json
{
  "type": "unread_count",
  "payload": { "count": 3, "total": 3 },
  "time": 1761184800000
}
```

前端在 `App.vue` 收到该消息后调用 `messageStore.setUnreadTotal(count)` 覆盖角标。
离线用户没有推送，靠 3.2 的主动拉取补齐。

### 4.6 前端入口

| 位置 | 文件 | 行为 |
|------|------|------|
| 顶栏铃铛 | `components/MessageBell.vue` | 最近 6 条 + 全部已读 + 查看全部；点击条目按 `link` 跳转并标记已读 |
| 侧边栏菜单 | `layouts/DefaultLayout.vue` | 「消息中心」菜单项带未读数角标 |
| 消息中心页 | `views/message/index.vue` | 分类页签（带角标）+ 仅看未读 + 关键字 + 分页 + 标记已读 / 删除 |
| 状态 | `stores/message.ts` | `total` / `byCategory` / `refreshUnread` / `setUnreadTotal` / `decrementUnread` / `clearUnread` / `reset` |

退出登录时 `token` watcher 会调用 `messageStore.reset()`，避免下一个账号看到上一个账号的角标。

### 4.7 重复投递去重

收件箱按事件落库，而通知链路并不保证「恰好一次」（渠道重试、多副本并存、重启后的补偿扫描都会重投）。去重在**数据库层**完成，对调用方与前端透明：

| 通知 | 去重键 | 同一键的含义 |
|------|--------|--------------|
| 任务指派 / 到期提醒 | `task:{taskID}:{reminderType}:{YYYYMMDD}` | 同任务同提醒类型当天只落一条 |
| 订阅临期 | `subscription:{租户名}:{剩余天数}:{YYYYMMDD}` | 同租户同剩余天数当天只落一条 |
| 注销审批结果 | `cancel_request:{租户名}:{状态}:{YYYYMMDD}` | 同租户同状态当天只落一条 |
| 审计告警 | `alert:{告警ID}` | 同一条告警只落一条（不按天） |
| 租户定向公告 | 不设去重键 | 每次发布都是一条新消息 |

**实现口径：**
- 列 `dedupe_key`（varchar(191)）+ 唯一索引 `(user_id, dedupe_key)`，写入用 `ON CONFLICT DO NOTHING`，冲突行静默跳过；
- `InboxMessage.DedupeKey` 为空时 `Deliver` 回退为 `{source_type}:{source_id}`；两者都定位不到事件（平台公告、无任务 ID 的提醒）则不设键；
- 键允许 `NULL`：唯一索引对 `NULL` 不生效，历史行由 AutoMigrate 补列后不会因「同用户多行空键」建索引失败。

**为什么不按事件永久去重：** 逾期任务提醒的语义就是「每天再提醒一次」（`task_reminder.overdue_cooldown`），键里带自然日才能做到「同日重试合并、跨天照常落库」。
**为什么审计告警按 ID 而非自然日：** 一条告警本身是一次独立事件，没有「第二天再提醒一次」的语义；
按 `alert:{告警ID}` 永久去重足以合并渠道重试与多副本的重复落库，又不会把不同告警误合并。
**对接口的影响：** 被去重的事件不产生新行，列表条数与未读数不增加；落库后的角标推送会按 `CountUnreadByUsers` 重新查库计数，因此也不会虚增。
