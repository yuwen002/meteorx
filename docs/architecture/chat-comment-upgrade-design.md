# 协作 IM 与评论深化设计（chat / wiki-comment upgrade）

> 状态：已评审通过的设计基线，按批次实现并各自本地提交。
> 前置：1:1 私聊已上线（后端 `e98c02c`、前端 `3c21bf9`）。

## 1. 范围

| 域 | 内容 | 交付批次 |
| --- | --- | --- |
| 私聊加固 | 在线状态、输入中提示、消息撤回 | A（后端）+ B（前端） |
| 协作 IM | 群聊会话：创建、成员管理、群消息广播 | C（后端）+ D（前端） |
| 评论深化 | 表情回应（点赞）、解决/重开、编辑与 @提及 前端补全 | E（后端）+ F（前端） |

不在本轮范围：端到端加密、消息搜索、已读回执逐人列表、频道/话题分离。

## 2. 通用约定

- 多租户隔离沿用 `tenant_id` 列 + `contextx.GetTenantID/GetUserID`；跨租户一律按「不存在或无权」处理，避免存在性泄露。
- 权限模型延续 chat 现状：**仅要求登录 + 会话成员归属**，不挂 RBAC 权限码；评论沿用 wiki 节点权限。
- WS 复用 `internal/ws` Hub 的按用户投递（`SendToUser`），离线用户下次拉 REST 即可补齐，不引入推送补偿队列。
- 响应字段全部 i18n 由前端 locale key 承担，后端只回结构化数据。

## 3. 私聊加固

### 3.1 WS 报文类型

`internal/ws/hub.go` 新增：

| 常量 | 值 | 语义 |
| --- | --- | --- |
| `MsgTypeChatTyping` | `chat_typing` | 输入中提示（瞬态，不落库） |
| `MsgTypeChatRecall` | `chat_recall` | 消息撤回事件 |

### 3.2 在线状态

- Hub 新增 `OnlineUserIDs() []string`（读锁内一次遍历，返回有连接的用户 ID）。
- `GET /api/v1/chat/presence` → `{ online_user_ids: [...] }`：返回**全租户可见的在线用户 ID 集合**（仅 ID，无敏感属性），前端据此本地计算对方/群成员在线点，避免 N 次请求。
- 前端在选中会话与收到 chat 类 WS 事件时刷新 presence（拉取式，不做轮询）。

### 3.3 输入中提示

`POST /api/v1/chat/conversations/{id}/typing`，body `{ typing: bool }`。

- 校验调用者为会话成员；**无状态**：不落库、不存 Redis，仅向会话其他成员转发 `MsgTypeChatTyping`（payload: conversation_id / user_id / typing）。
- 频率控制放在前端：状态翻转必发，持续输入每 3s 最多一次；前端提示 4s 无新事件自动消失。

### 3.4 消息撤回

`chat_messages` 增列（AutoMigrate 自动加列）：

| 列 | 类型 | 说明 |
| --- | --- | --- |
| `deleted` | bool default false | 是否已撤回 |
| `deleted_at` | datetime null | 撤回时间 |

规则：

- `DELETE /api/v1/chat/conversations/{id}/messages/{msgId}`。
- 仅**发送人本人**可撤回；限发送后 **5 分钟内**；已撤回的消息重复撤回返回错误。
- 撤回为**软撤回**：正文置空、置 `deleted` 标记，保留审计可追溯性；列表响应中 content 返回空串并带 `recalled=true`，由前端渲染「撤回了一条消息」。
- 撤回成功后向会话其他成员投递 `MsgTypeChatRecall`（payload: conversation_id / message_id），前端原位替换。

## 4. 协作 IM：群聊

### 4.1 模型

- `Conversation.Type` 新增取值 `group`；group 会话 `conv_key` 用生成的 ULID（天然唯一），`direct` 继续用规范化 `DirectKey`。
- `Conversation` 增 `Name` 字段（群名称），列 `name varchar(100)`（direct 会话留空）。
- `ConversationMember` 增 `Role` 字段：`owner` / `member`，列 `role varchar(20)`。

### 4.2 规则

| 行为 | 规则 |
| --- | --- |
| 创建群 | 任意登录用户；成员须同租户；创建人为 `owner`；初始成员 2–50 |
| 加人 | 任意成员均可邀请（协作优先） |
| 移除成员 | 仅 `owner`；不可移除 owner 自己 |
| 改名 | 仅 `owner`（本期不做头像/公告） |
| 退群 | 任意成员；`owner` 退群 → 群转「无主」，仅禁止后续 owner-only 操作（本期不做转移群主） |
| 末位成员 | 成员数为 0 时会话软删除 |
| 消息 | 复用现表，无结构变更；发送后向全部其他成员逐个 `SendToUser` |

群会话在列表响应中 `peer_name` 承载群名称、`peer_user_id` 为空，前端按 `type` 分支渲染，避免新增字段造成的双轨。

### 4.3 REST

```
POST   /chat/conversations/group                 { name, member_ids[] }
GET    /chat/conversations/{id}/members
PUT    /chat/conversations/{id}                  { name }            # owner
POST   /chat/conversations/{id}/members          { user_ids[] }
DELETE /chat/conversations/{id}/members/{userId} # owner
POST   /chat/conversations/{id}/leave
```

## 5. 评论深化

### 5.1 现状

后端已具备：一级 `parent_id` 回复、@提及通知（`MentionIDs`）、订阅者评论通知、编辑（仅作者）、删除级联。前端 `CommentSection.vue` **仅暴露了发表/一级回复/删除**，编辑与提及均未接入，且表情回应与「解决」缺失（`CommentStatusResolved` 常量已存在但无 API）。

### 5.2 表情回应

新表 `wiki_comment_reactions`：

| 列 | 说明 |
| --- | --- |
| id / tenant_id / comment_id / user_id | 常规隔离列 |
| emoji | varchar(16) |
| created_at | |

`(comment_id, user_id, emoji)` 唯一索引。`POST /wiki/documents/comments/{id}/reactions` body `{emoji}` 为**幂等切换**（已存在则取消，返回 toggle 后状态与聚合计数）。`CommentResp` 增 `reactions []{emoji, count, mine}`。删除评论时物理删除其反应行。

### 5.3 解决 / 重开

`POST /wiki/documents/comments/{id}/resolve` body `{ resolved: bool }`，仅作用于**根评论**：

- 权限：评论作者，或对节点具备 `delete` 权限者（与删除评论同档）。
- `resolved=true` → `status=CommentStatusResolved`；`false` → 回到 `Active`。`CommentResp` 已带 `status`，前端据此渲染徽标。

### 5.4 前端补齐

- 编辑：接入已有 `PUT /comments/{id}`（仅作者可见入口）。
- @提及：输入 `@` 弹出同租户成员选择，提交时填充 `mention_ids`（后端逻辑已就绪）。
- 回复嵌套保持一级（后端 `ListComments` 聚合形状不变，避免破坏既有 API 契约）。

## 6. 数据迁移

全部为**加列/新表**，经 `AutoMigrate` 生效（`chat repository.Models()` 与 `wikirepo.Models()` 单一来源，无需改 bootstrap）：

1. `chat_messages.deleted` / `deleted_at`
2. `chat_conversations.name`
3. `chat_conversation_members.role`
4. 新表 `wiki_comment_reactions`

## 7. 测试与验收

| 层 | 内容 |
| --- | --- |
| 后端单测 | service 层用例（撤回时限与本人校验、群成员权限矩阵、reaction 幂等切换、resolve 权限）；repo 用例沿用现 chat 测试基设 |
| 后端门禁 | `go build ./...` + `go test ./...` |
| 前端门禁 | `vue-tsc --noEmit` + `vitest run` |
| locale | 新增 key 全部 zh/en 对称，`scanrest` HITS=0 |
| 目视 | 复用 `_uicheck` Playwright 脚本验证群聊面板与评论交互中英渲染 |

## 8. 批次

| 批次 | 内容 | 门禁 |
| --- | --- | --- |
| A | 私聊加固后端：presence + typing + recall（含 WS 类型、迁移、单测） | build + test + 提交 |
| B | 私聊加固前端：在线点 / 输入中 / 撤回 UI + api 层 | vue-tsc + vitest + 提交 |
| C | 群聊后端：group 模型 / repo / service / handler / routes + 单测 | build + test + 提交 |
| D | 群聊前端：建群与成员管理对话框、群面板 | vue-tsc + vitest + 提交 |
| E | 评论后端：reactions 表 + resolve API | build + test + 提交 |
| F | 评论前端：编辑 / @提及 / 表情回应 / 解决 | vue-tsc + vitest + 提交 |
