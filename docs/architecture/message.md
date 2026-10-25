# 站内消息（收件箱）架构

## 概述

站内消息解决的是**实时推送解决不了的两个问题**：

| 问题 | 实时推送（WebSocket） | 站内消息（收件箱） |
| --- | --- | --- |
| 用户离线时能否收到 | 不能：`hub.clients` 只索引在线连接，离线投递直接 miss | 能：消息已落库，下次登录拉未读数 |
| 是否已读 | 不记录，刷新页面后无从判断 | 记录 `read_at`，按人隔离 |

因此本模块的定位是**补推兜底 + 已读状态**，而不是替代 WebSocket。
在线用户仍会立刻收到角标推送，离线用户则在下次打开页面时通过一次 `GET /messages/unread-count` 补齐。

设计上刻意守住三条边界：

1. **只认用户 ID**：收件箱的收件人必须是 `user_id`，不接受邮箱。邮箱 → WS 的错配在历史上造成过
   「静默 miss 却被记为发送成功」的故障，收件箱层不再给这类值第二次机会。
2. **不参与送达销账**：站内信落库成功不等于通知已送达，`deliverInbox` 的失败只告警，
   不并入 `SendResult` 判定，否则邮件 / WS 的失败会被一条本地写库掩盖。
3. **写放大必须封死**：按人写行意味着一次租户广播 = N 行 INSERT，必须有硬上限。

## 分层与文件

| 层 | 文件 | 职责 |
| --- | --- | --- |
| 入口 | `message/module.go` | 装配服务、注册 9 个路由、导出 `Migrate` |
| 模型 | `message/model/message.go` | 领域模型、分类常量、强制接收判定 |
| 仓储 | `message/repository/user_message_repository.go` | 批量写入、分页查询、未读统计、幂等标记、单条 / 批量删除 |
| 偏好 | `message/repository/user_preference_repository.go` | 偏好读写与「批量筛出关闭接收者」 |
| 目录 | `message/repository/user_directory.go` | 窄接口 `ListActiveUserIDs`（租户成员展开） |
| 服务 | `message/service/message_service.go` | 收件箱读写 + `Deliver` 落库（含偏好过滤）+ 角标推送 |
| DTO | `message/dto/message_dto.go` | `MessageResp` / `UnreadCountResp` / `MarkAllReadReq` / `BatchIDsReq` / `PreferencesResp` / `UpdatePreferenceReq` |
| Handler | `message/handler/message_handler.go` | 参数校验、401/400/404 映射 |
| 定时 | `message/expiry_job.go` | 按保留期物理删除历史消息 |
| 通知侧 | `notify/inbox.go` | `InboxMessage` / `InboxWriter` 接口（避免反向依赖） |

## 数据模型

表 `user_messages`（`AutoMigrate` 由 `bootstrap` 的非关键表清单调度）：

| 列 | 类型 | 说明 |
| --- | --- | --- |
| `id` | varchar(26) PK | ULID |
| `tenant_id` | varchar(26) | 所属租户（隔离与排障） |
| `user_id` | varchar(26) | **收件人**，所有查询的必带条件 |
| `category` | varchar(32) | `system` / `announcement` / `task` / `plan` / `audit` |
| `title` | varchar(200) | 标题 |
| `content` | text | 正文，前端按纯文本渲染（不解析 HTML，规避 XSS） |
| `link` | varchar(255) | 前端跳转路由，空表示不可跳转 |
| `priority` | int | 0 low / 1 normal / 2 high / 3 urgent，默认 1 |
| `source_type` | varchar(32) | 来源类型，便于按业务追溯 |
| `source_id` | varchar(64) | 来源 ID（任务 ID / 剩余天数 / 审批结果） |
| `dedupe_key` | varchar(191) NULL | 去重键，与 `user_id` 组成唯一索引；`NULL` 表示不参与去重 |
| `read_at` | datetime NULL | 已读时间，`NULL` 表示未读 |
| `created_at` | datetime | 创建时间，列表排序与过期清理的依据 |

索引：

| 索引 | 列 | 用途 |
| --- | --- | --- |
| `idx_user_messages_inbox` | `tenant_id` → `user_id` → `created_at` | 收件箱分页主查询（天然覆盖排序） |
| `idx_user_messages_dedupe`（唯一） | `user_id` → `dedupe_key` | 重复投递去重（同一事件只落一条） |
| 单列 | `category` / `source_type` / `source_id` / `read_at` | 分类统计、来源追溯、未读过滤 |

**为什么按「人 × 消息」写行，而不是「消息 + 已读关联表」**：

- 已读状态天然按人隔离，省掉一张关联表与一次 JOIN；
- 删除是收件人自己的动作，独立行避免「一人删除影响他人」；
- 代价是写放大（见下节），用硬上限与「平台级公告不落库」抵消。

偏好表 `user_message_preferences`：

| 列 | 类型 | 说明 |
| --- | --- | --- |
| `id` | varchar(26) PK | ULID |
| `user_id` | varchar(26) | 偏好归属用户 |
| `category` | varchar(32) | 分类 |
| `muted` | bool | `true` 表示关闭接收 |
| `created_at` / `updated_at` | datetime | 审计与排障用 |

索引：`idx_user_message_prefs` 唯一（`user_id` → `category`），既是覆盖写的定位键，也防止同一分类堆行。

**只有关闭接收才需要一行**：缺行即默认接收，因此新分类上线不必给存量用户补行，
也不会出现「无偏好」和「已接收」两种语义并存。`muted=false` 的行是显式覆盖（恢复接收），
读取时只认 `muted=true`。

## 通知链路

```
业务代码
  └─ notify.Manager.NotifyXxx()
       ├─ 渠道投递：Email / Webhook / WebSocket → SendResult（参与送达销账）
       └─ deliverInbox(InboxMessage)          → 失败仅 Warnf，不参与销账
            └─ message.MessageService.Deliver
                 ├─ 收件人为空 + 有 TenantID → ListActiveUserIDs 展开
                 ├─ dedupeIDs 去重剔空 + 二次截断 maxTenantFanout
                 ├─ MutedUserIDs 剔除关闭该类消息的收件人（强制分类跳过）
                 ├─ BatchCreate(200/批) 写 user_messages
                 └─ pushUnreadCounts → ws.MsgTypeUnreadCount 定向推送角标
```

当前落库的通知类型：

| 入口 | 收件人来源 | 分类 | link |
| --- | --- | --- | --- |
| `NotifyTaskReminder` | 单个 `recipientID` | `task` | `/task?task_id={任务ID}`（无任务 ID 时 `/task`） |
| `NotifySubscriptionExpiry` | 租户启用成员用户 ID | `plan` | `/system/plan` |
| `NotifyCancelRequestStatus` | 租户启用成员用户 ID | `system` | 空 |
| `NotifyAlert` | 规则邮箱经 user 仓储反查出的账号 | `audit` | `/system/audit/alert?alert_id={告警ID}` |
| `NotifyAnnouncement`（**仅租户定向**） | `TenantID` 展开 | `announcement` | 空 |

**告警落库的收件人边界**：`NotifyAlert` 的邮件收件人来自规则 `notify_targets`，但收件箱只认用户 ID，
所以由 `AlertService` 注入的窄接口 `AlertRecipientDirectory`（邮箱 → 用户 ID）逐条反查：
解析不到的邮箱（未注册账号的外部值班邮箱）直接跳过，**不展开租户、不广播**——
告警正文含用户行为明细，广播等于向全体在线用户泄露审计信息。
告警的 WS 实时推送仍走 `ws.SendAlert(alert.UserID, …)`，只推给触发用户，与收件箱是两条独立路径；
站内信落库同样不参与 `notified` 回写判定。

装配点在 `bootstrap/router.go`：消息模块的 `InitModule` 创建服务后立即
`notify.GetGlobalManager().SetInbox(msgSvc)`；未装配时 `deliverInbox` 整体跳过，其余渠道不受影响。
`notify` 只依赖 `InboxWriter` 接口，因此不存在 `notify ⇄ message` 循环依赖。

**告警链接带 `alert_id` 的闭环**：告警的 `link` 拼上告警记录 ID，前端 `views/system/audit/alert.vue`
读 `route.query.alert_id` 后调 `GET /audit/alerts/{id}` 直接弹出该条详情；不在页内重新 mount
（只改 query）时靠 `watch` 重读，因此连点多条告警消息都能生效。无记录 ID 的通知（定时任务自检）
不拼 query，避免打开一个空壳弹窗。

**任务提醒带 `task_id` 的同一套闭环**：`NotifyTaskReminder` 的 `link` 拼上 `extra["task_id"]`，
前端 `views/task/index.vue` 读 `route.query.task_id` 后调 `GET /tasks/{id}` 直接打开该任务；
**不在列表里找**——当前筛选、分页甚至列表刷新都可能把任务挡在外面。拿不到任务 ID 时 `link` 仍是 `/task`。
告警再往下钻一层也是同一口径：告警详情里的「审计日志 ID」可点，`views/system/audit/index.vue`
读 `route.query.log_id` 后调 `GET /audit/logs/{id}` 弹出日志详情（日志已被清理时显空态）。
三处的共同点：**业务 ID 进 query，页面自己拉详情**，不把“从长篇列表里找出哪一条”的成本转给用户。

## 角标一致性

角标（顶栏铃铛 / 侧边栏菜单 / 页签）由服务端未读数驱动，但四条路径天然会漂移，由客户端收口：

| 场景 | 为什么会漂移 | 客户端处理 |
| --- | --- | --- |
| 新消息落库 | WS `unread_count` 只带总数（`count`/`total`），分类明细不在 payload 里 | 先覆盖总数保证即时，再合并补拉 `/unread-count` 补齐分类 |
| 本页标记已读 / 删除未读 | 乐观扣减基于本地状态，这条可能已在另一设备读过 | 乐观扣减后合并补拉对账 |
| 全部已读 | 分类明细可能已被 WS 总数覆盖而停在旧值 | 用响应里的 `updated` 扣减，不用本地 `byCategory` |
| 别处已读（另一标签页 / 设备） | 已读不产生任何推送 | 页面重新可见（`visibilitychange`）时拉一次完整读数 |

补拉按 300ms 窗口合并：连续点几条已读、或一次广播带来多条推送，只产生一次请求。
退出登录时清空待执行的补拉，避免新账号登录后带着旧会话的意图打请求。

原则是「服务端是唯一真相」：本地读数只用于把交互做顺，任何写入动作都以服务端的响应或后续补拉覆盖。

## 写放大治理

按人写行意味着一条「发给整个租户」的通知会展开为 N 行 INSERT。治理分两层：

1. **平台级公告不落库**：全平台公告的收件人规模随用户总量线性增长，而公告本身已在公告列表常驻可查，
   因此只有 `targetTenantID != ""` 的定向公告才写收件箱。
2. **`maxTenantFanout = 500` 双重截断**：
   - 第一层：`ListActiveUserIDs(ctx, tenantID, maxTenantFanout)` 传入上限；
   - 第二层：`Deliver` 内再判一次长度，超限 `Warnf` 后切片。
   
   第二层不是冗余——`UserDirectory` 是接口，实现方未必遵守传入上限；
   写放大必须在 `Deliver` 内部封死，不能依赖下游守约。

去重（`dedupeIDs`）在展开之后执行，避免同一用户因数据重复收到多行。

## 重复投递去重

落库是按事件写的，而通知链路本身不保证「恰好一次」：渠道重试、多副本并存、
重启后的补偿扫描都可能把同一条提醒再次投递。收件箱因此在数据库层做幂等：

- 列 `dedupe_key`（varchar(191)）+ 唯一索引 `(user_id, dedupe_key)`；
- 写入使用 `ON CONFLICT DO NOTHING`（GORM `clause.OnConflict{DoNothing: true}`），
  冲突行被静默跳过，接口与角标推送不受影响；
- 键由 `notify` 侧按「业务事件 + 自然日」构造：

| 通知 | 去重键 | 同一键的含义 |
| --- | --- | --- |
| 任务指派 / 到期提醒 | `task:{taskID}:{reminderType}:{YYYYMMDD}` | 同任务同类型当天只一条 |
| 订阅临期 | `subscription:{租户名}:{剩余天数}:{YYYYMMDD}` | 同租户同剩余天数当天只一条 |
| 注销审批结果 | `cancel_request:{租户名}:{状态}:{YYYYMMDD}` | 同租户同状态当天只一条 |
| 审计告警 | `alert:{告警ID}` | 同一条告警只一条（不按天） |
| 租户定向公告 | 不设键 | 每次发布都是新消息 |

`InboxMessage.DedupeKey` 为空时，`Deliver` 回退为 `{SourceType}:{SourceID}`；
两者都定位不到事件（平台公告、无任务 ID 的提醒）则不设键——每次投递都是新行。

**为什么按天而不是按事件永久去重**：逾期任务提醒的语义就是「每天再提醒一次」
（`task_reminder.overdue_cooldown` 24h）。若把键固化为 `task:{id}`，第二天的提醒会被
第一天的行挡掉，用户再也收不到新提醒。按自然日切分后，同日重复（重试 / 多副本）被合并，
跨天提醒照常落库。审计告警是唯一按 ID 永久去重的一类：一条告警本身是一次独立事件，
没有「次日再提醒」的语义，按 `alert:{id}` 反而能准确合并重试。

**为什么 `dedupe_key` 允许 NULL**：唯一索引对 NULL 不生效（MySQL 与 SQLite 一致），
因此 AutoMigrate 给历史行补列后不会因「同一用户多行空键」建索引失败，
而「不参与去重」的消息也不会互相冲突。代码侧用 `nullableKey` 把空串转成 NULL。

## 接收偏好（分类开关）

用户可在消息中心页的「偏好设置」里按分类关闭接收。设计上守三条：

1. **在落库前过滤，不在读时过滤**：关闭后该类消息根本不入库，
   否则「不走库但读时藏」会出现「未读数与列表不一致」——角标算上了却点不进去。
2. **安全告警不可关闭**：`audit` 分类由 `model.IsMandatoryCategory` 判定，
   写入接口返回 400（`ErrMandatoryCategory`）而不是静默忽略；
   即使库里存在历史 `muted=true` 行（人工改库 / 策略上线前写入），`Deliver` 也直接跳过偏好查询。
3. **失败一律开口向上**：偏好查询报错时按默认接收投递并 `Warnf`——
   宁可多打扰，不能因为偏好表的插曲把消息丢掉。

过滤实现是**一次批量 SQL**：`WHERE category = ? AND muted = ? AND user_id IN (...)`，
返回关闭者集合后在内存里剔除。站内信落库本就是批量写，逐人查询会把写放大成 N+1。

过滤位置在 `Deliver` 内、分类推导之后、`BatchCreate` 之前；被过滤的收件人也不进入角标推送，
否则会推一个自己点开看不到的未读数。

隐私与边界：偏好按当前登录用户读写（`user_id` 取自 JWT 上下文），
接口不提供「查别人的偏好」，也不提供跨用户覆盖；不挂 RBAC 权限码，理由与收件箱一致。

## 访问控制

收件箱**不挂 RBAC 权限码**，理由：权限码是「能否操作某类资源」的授权，
而收件箱的边界是「这条数据是不是你的」。因此：

| 动作 | 约束 |
| --- | --- |
| 列表 / 未读数 | `WHERE user_id = ctx.UserID` |
| 标记已读 | `WHERE id = ? AND user_id = ? AND read_at IS NULL` |
| 全部已读 | `WHERE user_id = ? [AND category = ?] AND read_at IS NULL` |
| 删除 | `WHERE id = ? AND user_id = ?` |
| 批量已读 / 批量删除 | `WHERE user_id = ? AND id IN (...)`（已读另加 `read_at IS NULL`） |

**404 而非 403**：他人消息返回 404「消息不存在」，与真正不存在使用同一文案，
避免攻击者通过状态码差异探测消息 ID 是否存在。批量接口是例外（见下节）。

**标记已读的幂等语义**：更新语句带 `read_at IS NULL`，重复标记不会覆写首次阅读时间
（影响行数为 0 时再按 `id + user_id` 查一次，区分「已读」与「不存在」：
前者返回成功，后者返回 `ErrMessageNotFound` → 404）。

## 批量操作

收件箱堆积到几十条时，逐条点「标为已读 / 删除」的时间成本会超过阅读本身，因此提供两个批量接口：

| 接口 | 入参 | 响应 |
| --- | --- | --- |
| `PUT /messages/batch/read` | `{"ids": [...]}` | `{"updated": n}` |
| `DELETE /messages/batch/delete` | `{"ids": [...]}` | `{"deleted": n}` |

**批量语义与单条不同：缺一条不报 404，只不计入结果。** 单条接口的 404 是给「点开一条已经不存在」的
误触兜底；批量选中的集合来自一次列表渲染，其中任一条被其它终端读过或删掉都会变成「脏 ID」，
若整批回滚，用户重试也不可能成功。代价是调用方必须看 `updated` / `deleted` 而不是假定「传多少改多少」。
同一口径也让批量接口天然不可能被用来探测他人消息是否存在——影响行数只反映自己的行。

**上限 100 的双重守卫：**

- handler 层：`validate:"required,min=1,max=100,dive,required"`，空列表 / 含空 ID / 超限均 400；
  `min=1` 必须显式写——validator 对切片的 `required` 只判非 nil，空数组能绕过；
- 服务层：`normalizeIDs` 复用 `dedupeIDs` 去空白去重后按 `maxBatchIDs` 截断，
  内部调用绕过 handler 时也不会拼出超长 `IN` 列表。

**一条 SQL 完成，不产生 N+1**：`WHERE user_id = ? AND id IN ?`，已读操作再加 `read_at IS NULL`
（已读行不重复更新，保留首次阅读时间）。无事务：每行相互独立，不存在「部分成功需要回滚」的业务语义。

**批量操作不推 WS**：角标推送只发生在 `Deliver` 落库后。本人批量操作后由前端按响应重拉一次
`GET /messages/unread-count`——服务端仍是唯一真相，不为了省一次请求在客户端推算分类明细。

前端（`views/message/index.vue`）用 `el-checkbox-group` 绑定**当前页**的 ID：翻页 / 筛选后 `loadList`
会清空选中，避免批量操作带上看不见的消息；批量删除只在「整页被删空」时回退一页。

**交互上的三个一致性约束：**

- 点开带 `link` 的消息先 `markRead` 再跳转：否则从告警页 / 任务页返回时这条仍显示未读，
  用户会重复处理同一条消息；
- 「仅看未读」筛选下把某条标为已读后立即从当前列表移除（`total` 同步减一，整页清空则回退一页）。
  不这么做的话筛选条件与展示自相矛盾——列表里摆着已读消息，看起来像标记没生效。
- 「全部已读」的范围跟随当前页签（`read-all` 带 `category`），禁用条件取**该分类**的未读数而不是全局总数；
  已读不可撤回，执行前弹确认写明范围与条数，成功提示用响应里的 `updated`。

**审计口径**：批量（及单条）写操作**不额外埋点**，由挂在受保护分组的 `AuditMiddleware` 自动记录，
`module=messages`、`action` 按方法归一（`PUT` → `update`、`DELETE` → `delete`），
批量端点的 `tags` 额外含 `batch`——请求体的 ID 列表可能被 2000 字符截断，
标签是「按类型筛出全部批量写」的唯一可靠依据。

## 两个已踩过的坑

**1. typed-nil 接口导致空指针**
`bootstrap` 在 WS 关闭时传入 `nil *ws.Hub`。装箱成 `unreadPusher` 接口后该接口**不等于 nil**，
`if s.hub == nil` 检查失效，`SendToUser` 直接崩溃。现在 `NewMessageService` 显式做类型判空：

```go
if p, ok := hub.(*ws.Hub); ok && p == nil {
    return svc // 不赋值，保持 s.hub 为 nil
}
```

`TestNewMessageService_TypedNilHubIgnored` 锁定该行为。

**2. `json.Unmarshal` 复用 map 保留旧键**
未读数响应是 `map[string]int64`。集成测试里复用同一个结构体解码时，第二次解码不会删除第一次已有的键，
导致「task 分类已清零」的断言假失败。断言前需 `unread = dto.UnreadCountResp{}` 重置。

## WebSocket 推送协议

落库成功后按收件人定向推送（`SendToUser`），推送失败不影响接口与落库：

```json
{ "type": "unread_count", "payload": { "count": 3, "total": 3 }, "time": 1761184800000 }
```

前端处理：`App.vue` 的 WS 消息分派中 `case 'unread_count'` → `messageStore.receiveUnreadPush(count)`。
`count` 与 `total` 当前同值（单连接场景即该用户总未读），保留双字段以便后续接入「分类未读」增量。
因为 payload 不含分类明细，store 会先覆盖总数、再合并补拉一次 `/unread-count`。

## 前端状态同步

| 时机 | 动作 |
| --- | --- |
| 登录（token 出现） | `refreshUnread()` 主动拉一次，补齐离线期间消息 |
| 收到 WS `unread_count` | `receiveUnreadPush(count)` 覆盖总数 + 合并补拉分类明细 |
| 标记单条已读 / 删除未读 | `decrementUnread(category)` 本地扣减，随后 `refreshUnreadSoon()` 对账 |
| 点开带 `link` 的消息 | 先 `markRead`（未读时）再路由跳转，返回后不再显示未读 |
| 全部已读 | `clearUnread(category?, updated)` 按服务端条数扣减 + 合并补拉；入口跟随当前页签 |
| 批量已读 / 批量删除 | 直接 `refreshUnread()` 重拉（不推算分类明细） |
| 页面重新可见 | `visibilitychange` 时 `refreshUnread()`（覆盖其他标签页 / 设备已读） |
| 退出登录 | `reset()` 清空角标并取消排队中的补拉，避免串号 |

顶栏铃铛（`MessageBell.vue`）、侧边栏菜单角标与 `/messages` 页面共用同一个 Pinia store。

## 清理与保留期

`MessageExpiryJob` 每小时执行一次，删除 `created_at < now - retention_days` 的行：

- 未配置（0）→ 默认 90 天；
- 负值 → 关闭清理，连 goroutine 都不启动（`Start` 内提前返回）；
- 走 `created_at` 索引，单轮不设条数上限；物理删除，无软删除语义。

## 扩展指引

**新增一种会落收件箱的通知**：

1. 在 `notify.Manager` 的 `NotifyXxx` 中调用 `m.deliverInbox(ctx, &InboxMessage{...})`，
   `UserIDs` **只能填用户 ID**；
2. 若类型不在 `CategoryForType` 的已知分支内，显式传 `Category`，否则会落到 `system` 兜底；
3. 需要页面跳转时填 `Link`（前端只做路由跳转，不解析参数）；
4. 若收件人来自租户成员，注意 `maxTenantFanout` 截断会丢弃超出部分——面向全员的通知应评估是否
   改为「不落库 + 页面常驻可查」。

## 配置

| 配置 | 默认 | 说明 |
| --- | --- | --- |
| `message.retention_days` | 90 | 消息保留天数；负值关闭清理 |

## 测试

```bash
go test -count=1 ./internal/modules/message/...
go test -count=1 ./internal/notify/...
go test -count=1 ./internal/integration -run Message
cd web-admin && npx vitest run src/api/modules/message.test.ts src/stores/message.test.ts
```

覆盖要点：

- 仓储：幂等标记（两次标记时间相等）、按分类统计、非本人操作返回 not found、非法分页回落；
- 服务：收件人展开与截断（550 → 500）、空收件人不产生行、typed-nil hub 不崩溃；
- 集成：`NotifyTaskReminder` → 落库 → 未读数 / 列表 / 他人 404 / 重复标记幂等 / 删除，
  以及按分类全部已读只清目标分类（列表里断言 `link` 为 `/task?task_id=tk-1`）；
- 清理任务：默认 90 天、负值关闭、只删过期行；
- 去重：同一 `(user_id, dedupe_key)` 只落一条、不同用户各自一条、空键多条共存；
  变异检验移除 `OnConflict` 后用例以 `UNIQUE constraint failed` 失败；
- 告警落库：`notify` 层锁定收件人只含用户 ID、分类 `audit`、去重键 `alert:{id}`、
  链接带 `?alert_id=`（无 ID 时不带 query）、无账号收件人时不写行；
  任务提醒同层锁定链接带 `?task_id=`（无任务 ID 时不带 query）；
  `audit/service` 层锁定「规则邮箱反查成账号才写」与未装配解析能力时不写（nil 安全）；
  集成层经真实路由与 `WireAlertEvaluation` 验证邮箱→账号的装配行（删掉装配时只有集成用例变红）。
- 偏好：仓储层覆盖写幂等（重复提交不堆行）/ 只认 `muted=true` / 按分类批量筛人 / 用户隔离；
  服务层锁定「关闭后不写行、不推角标」「全员关闭零行零推送」「强制分类跳过偏好查询」「查询失败开口向上」；
  handler 锁定 `/preferences` 作为静态段不被 `/{id}` 吞掉、非法分类与关闭告警均 400；
  集成层锁定「关闭 → 不落库 → 重开恢复」与「历史关闭行不阻断安全告警」。
  变异检验：去掉装配行只有集成用例变红；去掉强制分类旁路影响对应服务与集成用例；
  `MutedUserIDs` 去掉分类条件同时打穿仓储与集成用例（跨分类误伤）。
- 角标一致性（`stores/message.test.ts` 5 条）：WS 推送先覆盖总数、到点才补拉；非法值既不覆盖也不触发补拉；
  连续校正（含与推送混用）合并为一次请求；`clearUnread` 按服务端 `updated` 扣减而非过期的分类明细；
  `reset` 取消排队中的补拉。变异检验：删补拉调用 / 去合并守卫 / 忽略 `updated` / 不 `clearTimeout`
  四项各自只打红对应用例（分别表现为 0 次、4 次调用与 `4 to be 2`）。
- 批量操作：仓储层锁定「只改本人的行、已读行不重复更新、他人 ID 不计入、空列表不下发空 IN」；
  服务层锁定去空白去重（`[" m1 ","","m2","m1"]` → `["m1","m2"]`）与按 `maxBatchIDs` 截断、
  仓储错误原样上抛；handler 层锁定空列表 / 含空 ID / 超 100 条均 400、缺失登录态 401、
  `/batch/delete` 不被 `/{id}` 当单条删除；集成层锁定「批量已读少算不存在的 ID」「他人批量删除不生效」
  「重复删除幂等」。变异检验：去掉 `read_at IS NULL` / 去掉 `user_id` 过滤 / 去掉上限截断
  三项各自只打红对应用例（分别表现为 `1 → 2`、`1 → 2`、`100 → 120`）；
  另在开发中真实踩到「切片 `required` 不拦空数组」，补 `min=1` 后空数组用例才转绿。
