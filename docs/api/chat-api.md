# 协作即时消息（IM）API 接口说明

> Base URL: `/api/v1`
> 所属模块：`internal/modules/chat`
> 路由注册：`internal/modules/chat/module.go`
> Handler：`internal/modules/chat/handler/chat_handler.go`
> DTO：`internal/modules/chat/dto/chat_dto.go`
> 前端页面：`web-admin/src/views/chat/index.vue`（路由 `/chat`）+ `stores/chat.ts`

同租户内的实时协作消息：支持 **1:1 私聊**与**群聊**，消息落库后经 `internal/ws` Hub 实时投递；
会话成员各自维护已读水位，用于计算未读数与已读回执。

- 全部接口**需登录**，但**不挂 RBAC 权限码**：访问控制以「会话成员归属」为准，非会话成员一律 404。
- 会话、消息均按 `tenant_id` 隔离，只能与同租户用户会话。
- 静态段 `/conversations/direct`、`/conversations/group`、`/presence` 先于 `/conversations/{id}` 注册。

---

## 1. 接口总览

| 方法 | 路径 | 功能 |
|------|------|------|
| POST | `/chat/conversations/direct` | 打开或复用 1:1 私聊会话 |
| POST | `/chat/conversations/group` | 创建群聊 |
| GET | `/chat/conversations` | 我的会话列表 |
| PUT | `/chat/conversations/{id}` | 群改名（仅群主） |
| GET | `/chat/conversations/{id}/messages` | 历史消息分页 |
| POST | `/chat/conversations/{id}/messages` | 发送消息（文本 / 图片 / 文件） |
| DELETE | `/chat/conversations/{id}/messages/{msgId}` | 撤回自己发送的消息 |
| GET | `/chat/conversations/{id}/members` | 群成员列表 |
| POST | `/chat/conversations/{id}/members` | 添加群成员 |
| DELETE | `/chat/conversations/{id}/members/{userId}` | 移除群成员（仅群主） |
| POST | `/chat/conversations/{id}/leave` | 退出群聊 |
| POST | `/chat/conversations/{id}/read` | 标记会话已读（推进已读水位） |
| GET | `/chat/conversations/{id}/read-status` | 其他成员的已读水位（已读回执） |
| POST | `/chat/conversations/{id}/typing` | 输入中提示（瞬态，不落库） |
| GET | `/chat/presence` | 当前在线用户 ID 集合 |

---

## 2. 数据结构

### 2.1 ConversationResp（会话响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 会话 ID（ULID） |
| `type` | string | `direct` / `group` |
| `peer_user_id` | string | 仅 `direct` 返回对方用户 ID |
| `peer_name` | string | 展示名：`direct` 为对方昵称，`group` 为群名称 |
| `last_message` | string | 最后一条消息正文（媒体消息为元信息 JSON，前端按 `last_message_type` 渲染摘要） |
| `last_message_type` | string | 最后一条消息类型：`text` / `image` / `file`（无消息时省略） |
| `last_at` | string | 最后一条消息时间（`2006-01-02 15:04:05`） |
| `unread_count` | int64 | 当前用户在该会话的未读数 |
| `member_count` | int64 | 成员数（仅群聊有意义） |
| `my_role` | string | 当前用户角色：`owner` / `member` |
| `created_at` | string | 会话创建时间 |

### 2.2 ChatMessageResp（消息响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 消息 ID（ULID） |
| `conversation_id` | string | 所属会话 ID |
| `sender_id` | string | 发送人用户 ID |
| `sender_name` | string | 发送人展示名（富化字段） |
| `content` | string | 消息正文；`image`/`file` 为媒体元信息 JSON；已撤回时为空串 |
| `type` | string | `text` / `image` / `file` |
| `mention_ids` | string | @提及的用户 ID（逗号分隔，已过滤为会话内有效成员） |
| `recalled` | bool | 是否已撤回（软撤回，保留行） |
| `url` | string | 媒体消息的短时效访问 URL（默认 30 分钟；未配置签名时为空） |
| `created_at` | string | 发送时间 |

### 2.3 媒体消息正文（`type=image/file` 时的 `content`）

`content` 是 JSON 字符串，前端需自行 `JSON.parse`：

```json
{ "file": "01M2Q3QXR8R1SNP95BASQ1EFCS.png", "name": "截图.png", "size": 102400 }
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `file` | string | 文件模块返回的存储文件名（`/uploads` 下单段安全字符，禁止路径穿越） |
| `name` | string | 原始文件名（仅展示） |
| `size` | int64 | 字节数（仅展示，非负） |

`file` 为空、含 `/`、`\` 或超长时整条消息被拒（400）。

### 2.4 请求体

**CreateDirectReq**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `participant_id` | string | 是 | 对方用户 ID（≤26 字符），必须与当前用户同租户 |

**CreateGroupReq**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 群名称，1～50 字符 |
| `member_ids` | string[] | 是 | 初始成员（不含创建人），1～49 个 |

**SendMessageReq**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `content` | string | 是 | 正文（≤2000 字符）；`image`/`file` 时为媒体元信息 JSON |
| `mention_ids` | string | 否 | @提及用户 ID，逗号分隔；服务层过滤为会话内有效成员 |
| `type` | string | 否 | `text`（默认）/ `image` / `file` |

**TypingReq**
| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `typing` | bool | 是 | `true` 开始输入，`false` 停止 |

### 2.5 ReadStateResp（已读回执）

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | string | 成员用户 ID |
| `name` | string | 成员展示名 |
| `last_read_at` | string | 该成员已读水位（`2006-01-02 15:04:05`），为空表示从未读 |

### 2.6 MemberResp（成员响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | string | 成员用户 ID |
| `name` | string | 成员展示名 |
| `role` | string | `owner` / `member` |
| `joined_at` | string | 加入时间 |

---

## 3. 接口详细说明

### 3.1 打开或复用单聊会话

`POST /api/v1/chat/conversations/direct`

**请求体：**
```json
{ "participant_id": "01JB8Z3K7Q9X2V5N4M6T8R1S0P" }
```

**业务规则：**
- 双方 ID 规范化排序生成去重键 `direct:{较小的ID}:{较大的ID}`，重复发起复用既有会话；
- 不能与自己私聊（400）；对方不存在或不属于当前租户返回 400。

**成功响应（200）：** `data` 为 ConversationResp。

### 3.2 创建群聊

`POST /api/v1/chat/conversations/group`

**请求体：**
```json
{ "name": "研发协作群", "member_ids": ["01JB8Y...", "01JB8Z..."] }
```

**业务规则：**
- 初始成员去重后 1～49 个，加创建人不超过 **50** 人；
- 创建人角色为 `owner`，其余为 `member`；成员必须同租户。

**成功响应（200）：** `data` 为 ConversationResp。

### 3.3 会话列表

`GET /api/v1/chat/conversations`

**成功响应（200）：** `data` 为 ConversationResp 数组，仅返回当前用户参与的会话，
并富化最后一条消息、未读数、成员数与本人角色。

### 3.4 群改名

`PUT /api/v1/chat/conversations/{id}`

**请求体：** `{ "name": "新群名" }`（1～50 字符）

**业务规则：** 仅群主可操作（403）；非群聊会话返回 400；非成员返回 404。

**成功响应（200）：** `{"data": {"ok": true}}`，并向成员推送 `chat_conversation`。

### 3.5 历史消息分页

`GET /api/v1/chat/conversations/{id}/messages`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码，默认 1 |
| `page_size` | int | 每页条数，默认 20，最大 100 |

**成功响应（200）：** 分页结构，`data` 为 ChatMessageResp 数组。

**分页语义：**
- 取数按 `created_at DESC` 分页（第 1 页为最新消息），**页内再反转为升序**返回；
- 前端向上滚动加载时 `page` 递增即可拼接为连续时间线。

### 3.6 发送消息

`POST /api/v1/chat/conversations/{id}/messages`

**请求体（文本）：**
```json
{ "content": "今天 18:00 评审", "mention_ids": "01JB8Y..." }
```

**请求体（图片/文件）：**
```json
{
  "type": "image",
  "content": "{\"file\":\"01M2Q3QXR8R1SNP95BASQ1EFCS.png\",\"name\":\"截图.png\",\"size\":102400}"
}
```

**业务规则：**
- 发送人必须是会话成员（否则 404）；
- `content` 去空白后为空返回 400；`type` 仅支持 `text`/`image`/`file`；
- `image`/`file` 的正文必须为合法媒体 JSON，否则 400；
- `mention_ids` 中不属于会话成员的用户被静默过滤；
- 落库成功后经 WS 向其他成员推送 `chat` 事件（`hub` 为 nil 时仅落库）。

**成功响应（200）：** `data` 为 ChatMessageResp（媒体消息已带 `url`）。

### 3.7 撤回消息

`DELETE /api/v1/chat/conversations/{id}/messages/{msgId}`

**业务规则：**
- 只能撤回**自己发送**的消息（400）；
- 仅 **5 分钟**内可撤回，超时 400；
- 已撤回的消息重复撤回返回 400；
- 软撤回：`recalled=true`、`content` 置空，行保留；向成员推送 `chat_recall`。

**成功响应（200）：** `{"data": {"ok": true}}`

### 3.8 群成员列表

`GET /api/v1/chat/conversations/{id}/members`

**成功响应（200）：** `data` 为 MemberResp 数组。

### 3.9 添加群成员

`POST /api/v1/chat/conversations/{id}/members`

**请求体：** `{ "user_ids": ["01JB8Y...", "01JB8Z..."] }`（1～49 个）

**业务规则：**
- 仅群聊支持；加完后不得超过 50 人；
- 已在群内的用户自动去重；非本租户用户拒绝；
- 成功后推送 `chat_conversation`。

**成功响应（200）：** `{"data": {"ok": true}}`

### 3.10 移除群成员

`DELETE /api/v1/chat/conversations/{id}/members/{userId}`

**业务规则：** 仅群主可操作（403）；不能移除自己（用退群代替，400）；目标不在群内 400。

**成功响应（200）：** `{"data": {"ok": true}}`，并推送 `chat_conversation`。

### 3.11 退出群聊

`POST /api/v1/chat/conversations/{id}/leave`

**业务规则：** 非群成员 400；最后一名成员离开时会话软删除；成功后推送 `chat_conversation`。

**成功响应（200）：** `{"data": {"ok": true}}`

### 3.12 标记已读

`POST /api/v1/chat/conversations/{id}/read`

**业务规则：** 将当前用户的 `last_read_at` 推进到当前时间，未读数清零；
同时向其他成员推送 `chat_read`（载荷见第 4 节）。

**成功响应（200）：** `{"data": {"ok": true}}`

### 3.13 已读回执

`GET /api/v1/chat/conversations/{id}/read-status`

**成功响应（200）：** `data` 为 ReadStateResp 数组，**仅包含请求者以外**的会话成员；
`last_read_at` 为空表示该成员从未读该会话。

### 3.14 输入中提示

`POST /api/v1/chat/conversations/{id}/typing`

**请求体：** `{ "typing": true }`

**业务规则：** 瞬态事件，不落库、不补偿；仅转发给会话内其他在线成员（WS `chat_typing`）。

**成功响应（200）：** `{"data": {"ok": true}}`

### 3.15 在线状态

`GET /api/v1/chat/presence`

**成功响应（200）：**
```json
{ "data": { "online_user_ids": ["01JB8Y...", "01JB8Z..."] } }
```

在线判定为「当前持有 WS 连接」；集合无序，前端自行比对。

---

## 4. WebSocket 事件

连接 `/api/v1/ws` 后，IM 相关推送类型：

| type | 载荷 | 触发时机 | 前端处理 |
|------|------|----------|----------|
| `chat` | ChatMessageResp | 会话内新消息 | 追加到当前会话、更新列表摘要与未读 |
| `chat_typing` | `{conversation_id, user_id, typing}` | 成员输入中状态变化 | 显示/隐藏「正在输入」 |
| `chat_recall` | `{conversation_id, message_id}` | 消息被撤回 | 原位替换为「已撤回」 |
| `chat_conversation` | 会话元信息（群名/成员数等） | 建群、改名、加减成员、退群、新会话 | 刷新会话列表（保留本地未读，回源校正） |
| `chat_read` | `{conversation_id, user_id, read_at}` | 成员已读水位推进 | 更新对方的「已读至」标记 |

离线期间的事件不补推；重新登录后拉取会话列表与历史消息即可补齐。

---

## 5. 业务规则与限制

| 项 | 取值 | 说明 |
|----|------|------|
| 会话类型 | `direct` / `group` | `direct` 由参与者唯一确定 |
| 群成员上限 | 50（含创建人） | 创建/添加成员均校验 |
| 群初始成员 | 1～49（不含创建人） | 至少 1 人 |
| 消息正文 | ≤2000 字符 | 去空白后不得为空 |
| 撤回时限 | 5 分钟 | 仅本人消息 |
| 历史分页 | 默认 20，最大 100 | 第 1 页最新，页内升序 |
| 媒体签名 URL | 30 分钟 | 与 Wiki 内嵌图片同源签名机制 |
| @提及 | 逗号分隔 | 非会话成员静默过滤 |

---

## 6. 错误响应

| 状态码 | 场景 |
|--------|------|
| 400 | 不能与自己私聊 / 对方不在当前租户 / 消息内容为空 / 只能撤回自己的消息 / 已超撤回时限 / 已撤回 / 群名或成员为空 / 超出群成员上限 / 非群聊会话 / 不能移除自己 / 目标不在群内 / 不支持的消息类型 / 媒体元信息不合法 |
| 401 | 未登录或 Token 失效 |
| 403 | 仅群主可执行（改名、移除成员） |
| 404 | 会话不存在或无权访问 / 消息不存在 |
| 500 | 存储或内部错误 |

---

## 7. 前端入口

| 位置 | 说明 |
|------|------|
| `views/chat/index.vue` | 会话列表、聊天面板、群成员抽屉、在线/输入中/撤回/已读回执 |
| `stores/chat.ts` | 会话与消息状态、WS 事件归并、未读数维护 |
| `api/modules/chat.ts` | REST 封装（分页、媒体消息、群管理） |
