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
| PUT | `/messages/batch/read` | 批量标记已读（≤100 条） | 需登录 |
| DELETE | `/messages/{id}` | 删除单条消息 | 需登录 |
| DELETE | `/messages/batch/delete` | 批量删除（≤100 条） | 需登录 |
| GET | `/messages/preferences` | 查询分类接收偏好 | 需登录 |
| PUT | `/messages/preferences` | 修改单个分类的接收开关 | 需登录 |

**说明：**
- 全部接口需登录，但**不挂 RBAC 权限码**：收件箱按「数据归属」鉴权，普通成员也应能看自己的消息。
- 所有查询与写操作的 `WHERE` 都带 `user_id = 当前用户`，他人的消息既不可见、也不可代为标记已读或删除。
- 静态段（`/preferences`、`/unread-count`、`/read-all`、`/batch/*`）先于 `/{id}` 注册，避免被路径参数吞掉。
- 单条与批量接口的错误语义不同：单条缺一条回 404，批量缺一条静默跳过并只反映在 `updated` / `deleted` 里。

---

## 2. 数据结构

### 2.1 MessageResp（消息响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 消息 ID（ULID） |
| category | string | 分类：`system` / `announcement` / `task` / `plan` / `audit` |
| title | string | 标题 |
| content | string | 正文（纯文本，前端按纯文本渲染） |
| link | string | 前端跳转路由，空表示不可跳转；任务提醒为 `/task?task_id={任务ID}`，告警为 `/system/audit/alert?alert_id={告警ID}`，订阅临期为 `/system/tenant?tenant_id={租户ID}` |
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

### 2.4 PreferenceItem（单个分类的偏好）

| 字段 | 类型 | 说明 |
|------|------|------|
| category | string | 分类：`system` / `announcement` / `task` / `plan` / `audit` |
| muted | bool | `true` 表示关闭接收：该类消息**不再落库**（也不推角标） |
| mandatory | bool | `true` 表示强制接收（当前仅 `audit`），前端应禁用开关 |

### 2.5 PreferencesResp（偏好响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| items | PreferenceItem[] | **全部分类**按固定顺序返回，未设置过的分类也在其中（`muted=false`） |

### 2.6 UpdatePreferenceReq（修改偏好请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| category | string | 是 | 合法分类；未知分类返回 400 |
| muted | bool | 否 | 缺省为 `false`，即恢复接收 |

一次只改一个分类：开关即时生效，不存在「半提交」，也就不需要批量 + 事务。

### 2.7 BatchIDsReq（批量操作请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | string[] | 是 | 消息 ID 列表，**1 ~ 100** 条；空数组、含空字符串、超 100 条均返回 400 |

服务端会再去一次首尾空白并去重，超限截断（正常请求不会触发）。

```json
{ "ids": ["01JB8Z3K7Q9X2V5N4M6T8R1S0P", "01JB8Z3K7Q9X2V5N4M6T8R1S0Q"] }
```

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
      "link": "/task?task_id=01JB8Y...",
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

### 3.6 查询消息偏好

`GET /api/v1/messages/preferences`

**成功响应（200）：**
```json
{
  "data": {
    "items": [
      { "category": "system", "muted": false, "mandatory": false },
      { "category": "announcement", "muted": true, "mandatory": false },
      { "category": "task", "muted": false, "mandatory": false },
      { "category": "plan", "muted": false, "mandatory": false },
      { "category": "audit", "muted": false, "mandatory": true }
    ]
  }
}
```

**业务规则：**
- 只返回当前用户的偏好，无需传用户 ID；
- **未设置过 = 已接收**：新分类上线不会给存量用户补行，也不会出现「未设置」这种第三态；
- 分类顺序由后端固定（`model.AllCategories`），前端不必自行排序。

### 3.7 修改消息偏好

`PUT /api/v1/messages/preferences`

**请求体：**
```json
{ "category": "task", "muted": true }
```

**成功响应（200）：** 同 3.6 的完整偏好列表（修改后的最新视图），前端可直接用于刷新开关。

**业务规则：**
- 覆盖写且幂等：同一 `(user_id, category)` 只保留一行，重复提交不会堆行；
- `muted=true` 后该类消息**不再落库**，既无收件箱记录也无未读角标；改回 `false` 立即恢复；
- **安全告警（`audit`）不可关闭**：`muted=true` 返回 400 而不是静默忽略——静默会让用户以为关掉了，实际仍会收到；
- 未知分类返回 400。

### 3.8 批量标记已读

`PUT /api/v1/messages/batch/read`

**请求体：** BatchIDsReq

**成功响应（200）：**
```json
{ "data": { "updated": 2 } }
```

**业务规则：**
- 只影响 `user_id = 当前用户` 的行，且只改未读行（`read_at IS NULL`），已读行不重复更新；
- `updated` 是**实际更新条数**：已读的、已被其他终端删除的、不属于本人的、不存在的 ID 都不计入，
  也不报 404——批量选中的集合来自一次列表渲染，其中一条失效不应让整批回滚；
- 一条 `UPDATE ... WHERE id IN (...)` 完成，不逐条发请求；
- 操作后不推 WS，前端按响应重拉一次 `GET /messages/unread-count` 刷新角标。

### 3.9 批量删除

`DELETE /api/v1/messages/batch/delete`

**请求体：** BatchIDsReq（DELETE 也走请求体，与其他模块的 `batch/delete` 一致）

**成功响应（200）：**
```json
{ "data": { "deleted": 2 } }
```

**业务规则：**
- 物理删除，语义同单条删除；`deleted` 为实际删除条数；
- 他人 ID 不计入删除结果，也不报错（不泄露消息是否存在）；
- 重复提交幂等：已删掉的 ID 再次提交返回 `{"deleted": 0}`。

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

**告警消息的跳转闭环**：`link` 为 `/system/audit/alert?alert_id={告警ID}`，`source_id` 与该 ID 相同。
前端告警页读取 `alert_id` 后调 `GET /audit/alerts/{id}` 直接弹出这条告警的详情，
而不是只跳到告警列表首页让用户自己去找；告警无记录 ID 时（如定时任务自检）`link` 不带 query。

**任务提醒的跳转闭环**：`link` 为 `/task?task_id={任务ID}`，`source_id` 即任务 ID。
前端任务页读取 `task_id` 后调 `GET /tasks/{id}` 直接打开这条任务（不依赖它是否在当前筛选 / 分页里）；
提醒无法定位任务时（如手工触发、无任务 ID）`link` 为 `/task`，不带 query。

**订阅临期的跳转闭环**：`link` 为 `/system/tenant?tenant_id={租户ID}`，不带空 query 时回落 `/system/tenant`。
前端租户管理页读取 `tenant_id` 后调 `GET /admin/tenants/{id}/detail` 再打开该租户的「套餐配置」弹窗，
直接看到当前套餐与到期时间。**没有指向 `/system/plan`**：套餐管理页只有套餐定义（名称/额度/价格），
「这个租户什么时候到期」只存在于租户侧的订阅记录里，跳过去看不到到期日，等于把定位成本又推回给用户。
**query 只带租户 ID、不带剩余天数**：天数由 `GET /admin/tenants/{id}/plan` 实时返回（`effective_days`），
写进 query 的只是一个会在几天后过期的快照。
以上三类消息同一套口径：**业务 ID 进 query，页面自己去拉详情，不把定位成本转给用户**。

### 4.2 错误码映射

| 状态码 | 场景 |
|--------|------|
| 400 | 非法消息分类（列表筛选 / 全部已读 / 偏好提交）；消息 ID 或分类为空；批量 ID 列表为空 / 含空值 / 超 100 条；试图关闭强制接收的安全告警 |
| 401 | 未登录或 Token 失效（上下文缺失用户 ID） |
| 404 | 消息不存在，或不属于当前用户（仅单条标记已读 / 删除返回；批量接口不返回 404） |
| 500 | 列表 / 未读数 / 标记 / 删除 / 批量操作 / 偏好的存储异常 |

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

前端在 `App.vue` 收到该消息后调用 `messageStore.receiveUnreadPush(count)` 覆盖角标。
离线用户没有推送，靠 3.2 的主动拉取补齐。

> `payload` 只带未读总数，**不含 `by_category`**：store 会先覆盖总数保证即时，
> 再合并补拉一次 `GET /messages/unread-count` 补齐分类明细，
> 否则会出现「顶栏总数已涨、页签角标不动」的不一致（见 4.9）。

### 4.6 前端入口

| 位置 | 文件 | 行为 |
|------|------|------|
| 顶栏铃铛 | `components/MessageBell.vue` | 最近 6 条 + 全部已读 + 查看全部；点击条目按 `link` 跳转并标记已读；「查看全部消息」先清掉记住的分类再进收件箱 |
| 侧边栏菜单 | `layouts/DefaultLayout.vue` | 「消息中心」菜单项带未读数角标 |
| 消息中心页 | `views/message/index.vue` | 分类页签（带角标、跨会话记住）+ 仅看未读 + 关键字 + 分页 + 标记已读 / 删除 + 全选本页 / 批量已读 / 批量删除 + 「本分类全部已读」 + 「偏好设置」弹窗 |
| 告警定位 | `views/system/audit/alert.vue` | 读 `?alert_id=` 直接打开该条告警详情；告警记录行也可手动点「详情」；详情里的「审计日志 ID」可点，跳到审计日志页打开该条日志 |
| 任务定位 | `views/task/index.vue` | 读 `?task_id=` 调 `GET /tasks/{id}` 直接打开该任务（编辑弹窗），不要求它已在当前页 |
| 审计日志定位 | `views/system/audit/index.vue` | 读 `?log_id=` 调 `GET /audit/logs/{id}` 直接弹出该条日志详情；日志已被清理时提示空态 |
| 用户时间线定位 | `views/system/audit/timeline.vue` | 读 `?user_id=` 直接查这个人的操作时间线（无 query 时退回当前登录用户）；告警详情里的「触发用户」即跳到这里 |
| 订阅临期定位 | `views/system/tenant/index.vue` | 读 `?tenant_id=` 调 `GET /admin/tenants/{id}/detail` 后打开该租户的「套餐配置」弹窗，不要求该租户行在当前页 |
| 状态 | `stores/message.ts` | `total` / `byCategory` / `refreshUnread` / `refreshUnreadSoon` / `receiveUnreadPush` / `setUnreadTotal` / `decrementUnread` / `clearUnread` / `reset` |

退出登录时 `token` watcher 会调用 `messageStore.reset()`，避免下一个账号看到上一个账号的角标。

**列表与已读状态的一致性**：点开带 `link` 的消息会先标记已读再跳转（跳回消息中心时不再显示为未读）；
在「仅看未读」筛选下把某条标为已读，该条会立即从当前页移除、总数同步减一，整页清空时自动回退一页
——不靠刷新才能看到筛选生效。批量已读 / 批量删除后统一按响应重拉列表与未读数。

**「全部已读」的范围跟随页签**：按钮文案与请求体都取当前分类（“全部标为已读” / “本分类全部已读”），
禁用条件是**该分类**未读数为 0（而不是全局总数）——总数为 0 但切到某分类、或反过来，都不会出现可点却无事发生的按钮。
已读不可撤回，执行前弹一次确认并写明范围与条数；成功后用响应里的 `updated` 提示实际条数。

**分类页签跨会话记住**：页签值存 `localStorage`（键 `meteorx_message_category`，见 `utils/messageCategory.ts`），
刷新或重新进入消息中心仍停在用户上次看的分类；只有页签值持久化，关键字与「仅看未读」属于一次性筛选。
两条边界：存进去的值不在分类枚举里（旧版本残留）时按「全部」处理；
铃铛里的「查看全部消息」会先清掉这个键——名字叫「全部」却落在被筛过的列表上，是另一种误导。

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

### 4.8 接收偏好的生效边界

偏好表 `user_message_preferences` 按「用户 × 分类」存，**只有关闭接收才建行**（`muted=false` 也是显式覆盖，等价于恢复接收）：

| 环节 | 行为 |
|------|------|
| 落库前（`Deliver`） | 一次 `SELECT user_id ... WHERE category=? AND muted=true AND user_id IN (...)` 批量过滤，被过滤者既不写行也不推角标 |
| 强制分类（`audit`） | 跳过偏好查询；即使库里存在历史关闭行，告警仍强制落库 |
| 偏好查询失败 | 按默认接收投递并记告警日志——宁可多打扰，不可漏消息 |
| 关闭后的存量消息 | 不受影响，直到保留期清理或用户手动删除 |

**为什么只在落库前过滤：** 实时推送（`ws.SendAlert` / 任务提醒的 WS 通道）与收件箱是两条独立链路，
偏好只作用于「持久化」这一层；它解决的是「不想在收件箱里看到这类消息」，不是「拦截告警本身」。

### 4.9 未读角标的一致性

角标是三个位置（顶栏铃铛 / 侧边栏菜单 / 页签）共用的同一个读数，凡是会让它与服务端产生偏差的路径都已收口：

| 场景 | 偏差来源 | 前端处理 |
|------|----------|----------|
| 收到 WS `unread_count` | 推送只带总数，分类明细仍是旧值 | 先覆盖总数，再合并补拉一次完整读数 |
| 本页标记已读 / 删除未读 | 乐观扣减基于本地状态，这条可能已被别处读过 | 乐观扣减后合并补拉对账 |
| 全部已读 | 分类明细可能过期，按它扣减会残留 | 用响应里的 `updated` 扣减（`clearUnread(category, updated)`）；带分类时只清该分类 |
| 其他标签页 / 设备已读 | 已读不产生任何推送 | 页面重新可见（`visibilitychange`）时拉一次 |

补拉在 300ms 窗口内合并：连续点几条已读、或一次广播带来多条推送，只产生一次 `GET /messages/unread-count`。
退出登录时 `reset()` 会取消排队中的补拉，避免新账号登录后带着旧会话的意图发请求。

原则不变：**服务端是唯一真相**，本地读数只用来把交互做顺，任何写入动作都以响应或后续补拉覆盖。

### 4.10 写操作的审计口径

收件箱的写接口不额外埋审计点，**由 `AuditMiddleware` 自动记录**（消息路由挂载在受保护分组内，所有请求都经过它）：

| 接口 | module | action | 风险等级 | 备注 |
|------|--------|--------|----------|------|
| `PUT /messages/{id}/read` | `messages` | `update` | medium | `resource` 即请求路径 |
| `PUT /messages/read-all` | `messages` | `update` | medium | 请求体带 `category` |
| `PUT /messages/batch/read` | `messages` | `update` | medium | `tags` 含 `batch` |
| `DELETE /messages/{id}` | `messages` | `delete` | critical | |
| `DELETE /messages/batch/delete` | `messages` | `delete` | critical | `tags` 含 `batch` |

批量接口的请求体是 ID 数组，超过 2000 字符会被审计截断，因此额外打 `batch` 标签：
按标签即可筛出全部批量写操作，不必依赖请求体是否完整。实际影响条数（`updated` / `deleted`）
在响应体里返回，但**审计不落响应体**，需要精确条数时应按请求 ID 回查业务数据。
