# OpenAPI 文档同步工具

## 📖 概述

`scripts/openapi_sync.go` 是一个用于同步代码路由与 API 文档的自动化工具。它以代码路由树为唯一事实来源，同一次运行内先后校正两份文档：

| 文件 | 结构 | 用途 |
|------|------|------|
| `docs/apifox/MeteorX-backend.openapi.json` | OpenAPI 3.0（`paths`） | 接口契约 / 第三方工具导入 |
| `docs/apifox/MeteorX-backend.apifox.json` | Apifox 原生（`apiCollection[].items`） | 直接导入 Apifox 的分组集合 |

## 🎯 使用场景

- 新增/删除 API 路由后，自动更新 OpenAPI 与 Apifox 文档
- 检查两份文档是否遗漏或多余路由
- 保持 API 文档与代码实时同步

## 🚀 使用方法

### 1. 检查模式（默认）

仅对比差异，不修改文件：

```bash
go run scripts/openapi_sync.go
```

**输出示例（文档与代码完全一致时）：**
```
======== 代码路由 vs OpenAPI 对比 ========

代码路由总数: 333
OpenAPI 路由总数: 333

✅ OpenAPI 无缺失路由
✅ OpenAPI 无多余路由

🎉 完美匹配！

======== 代码路由 vs Apifox 对比 ========

代码路由总数: 333
Apifox 路由总数: 333

✅ Apifox 无缺失路由
✅ Apifox 无多余路由

🎉 完美匹配！
```

### 2. 自动修复模式

自动补齐缺失路由、删除多余路由：

```bash
go run scripts/openapi_sync.go -apply
```

**输出示例（假设新增了一个会话置顶接口）：**
```
======== 代码路由 vs OpenAPI 对比 ========

代码路由总数: 334
OpenAPI 路由总数: 333

⚠️  OpenAPI 缺失 1 个路由：
  + GET /api/v1/chat/conversations/{id}/pins

✅ OpenAPI 无多余路由

===== 应用修复 =====
+ 添加 GET /api/v1/chat/conversations/{id}/pins

✅ 已修复：+1 / -0

======== 代码路由 vs Apifox 对比 ========

代码路由总数: 334
Apifox 路由总数: 333

⚠️  Apifox 缺失 1 个路由：
  + GET /api/v1/chat/conversations/{id}/pins

✅ Apifox 无多余路由

===== 应用修复 =====
+ 添加 GET /api/v1/chat/conversations/{id}/pins -> 分组 即时消息

✅ 已修复：+1 / -0
```

### 3. 门禁模式（CI）

```bash
go run scripts/openapi_sync.go -check
```

发现差异时以退出码 1 结束（与 `-apply` 同用时不做失败判定）。CI 的 `lint` 任务已接入该命令，路由改了而文档没跟上会直接卡住流水线。日常开发仍用默认模式看差异、`-apply` 修复。

## 📋 工作原理

1. **构建路由树**：`buildRouter()` 直接返回 `internal/routetable.Build()`——由各模块导出的纯注册函数（`RegisterRoutes` 等）拼装的空 handler 路由树，不连接 DB/Redis；它与生产路由是否一致由对齐测试强校验（见「漂移护栏」）
2. **遍历代码路由**：使用 `chi.Walk()` 提取全部路由，归一化为 `METHOD /path`，并把 `:id`、`{id}` 统一成 `{id}`
3. **读取文档**：分别解析 `openapi.json` 的 `paths` 与 `apifox.json` 的 `apiCollection[].items[].api`
4. **对比差异**：
   - 缺失路由：代码中有但文档中没有
   - 多余路由：文档中有但代码中没有
   - 重复条目：同一 `METHOD /path` 在 Apifox 中出现多次（集合用 map 去重，不单独检测会永远静默）
5. **自动修复**（使用 `-apply` 时）：
   - OpenAPI：写入 method 操作对象（默认 summary、tags、权限码 description、200 响应）
   - Apifox：按路径前缀定位中文分组追加条目，分组不存在则新建
   - 两侧同步删除多余路由，Apifox 额外清理重复条目（保留首条）

> 两个文件在一次运行内**各自独立对比、各自写回**：其中一个已同步不会让另一个被跳过。

## 🔧 自动生成的路由信息

当添加缺失路由时，脚本会自动推断以下信息：

| 字段 | 推断规则 |
|------|---------|
| `summary` | 先查 `inferSummary` 内置中文摘要表，未命中则按方法 + 末段路径拼接，如 `GET /users` → "获取users" |
| `tags` | `inferTags` 按路径前缀推断，如 `/api/v1/auth/*` → `["Auth"]` |
| `description` | 自动生成 `权限码: <code>`；命中 `inferDocPath` 时追加 `接口文档: docs/api/xxx.md` |
| `responses` | 默认 200 成功响应 |
| Apifox 分组 | `inferApifoxGroupName` 推断中文分组名，如 `/api/v1/chat/*` → 即时消息 |

### Tags 推断规则（OpenAPI）

| 路径前缀 | Tag |
|---------|-----|
| `/health`, `/metrics`, `/api/v1/csp-report` | System |
| `/api/v1/auth`, `/api/v1/oauth` | Auth |
| `/api/v1/users` | User |
| `/api/v1/tenants`, `/api/v1/tenant` | Tenant |
| `/api/v1/rbac` | RBAC |
| `/api/v1/audit` | Audit |
| `/api/v1/wiki` | Wiki |
| `/api/v1/files` | File |
| `/api/v1/admin/settings`, `/api/v1/settings` | Settings |
| `/api/v1/chat` | Chat |
| `/api/v1/messages` | Message |
| `/api/v1/invitations` | Invitation |
| `/api/v1/dashboard` | Dashboard |
| `/api/v1/admin/oauth` | Auth |
| `/api/v1/admin` | Admin |
| `/api/v1/plan` | Plan |
| `/api/v1/announcements`, `/api/v1/notification` | Notification |
| `/api/v1/tasks` | Task |
| `/api/v1/exports` | Export |
| 其他 | API |

> 匹配顺序即优先级：`/api/v1/admin/settings`、`/api/v1/admin/oauth` 必须排在 `/api/v1/admin` 之前，否则会被 Admin 提前命中。

### 分组推断规则（Apifox）

| 路径前缀 | 分组 |
|---------|------|
| `/health`, `/metrics`, `/api/v1/ws`, `/api/v1/csp-report` | 系统 |
| `/api/v1/auth`, `/api/v1/oauth`, `/api/v1/admin/oauth*` | 认证 |
| `/api/v1/profile` | 个人中心 |
| `/api/v1/tenants`, `/api/v1/admin/tenants*`, `/api/v1/admin/cancel-requests`, `/api/v1/admin/tenants-plan` | 租户管理 |
| `/api/v1/tenant-settings` | 租户独立配置 |
| `/api/v1/admin/tenant-users` | 跨租户用户管理 |
| `/api/v1/admin/users` | 管理员用户管理 |
| `/api/v1/users` | 租户用户管理 |
| `/api/v1/admin/stats`, `/api/v1/admin/dashboard` | 运营看板 |
| `/api/v1/rbac` | 角色与权限 |
| `/api/v1/audit/alert` | 告警管理 |
| `/api/v1/audit/sessions` | 会话分析 |
| `/api/v1/audit` | 审计日志 |
| `/api/v1/files` | 文件管理 |
| `/api/v1/plan`, `/api/v1/admin/plans`, `/api/v1/admin/subscriptions` | 套餐管理 |
| `/api/v1/announcements`, `/api/v1/admin/announcements` | 通知公告 |
| `/api/v1/wiki` | 知识库 |
| `/api/v1/tasks` | 任务管理 |
| `/api/v1/chat` | 即时消息 |
| `/api/v1/messages` | 站内消息 |
| `/api/v1/invitations` | 成员邀请 |
| `/api/v1/settings`, `/api/v1/admin/settings` | 系统配置 |
| `/api/v1/dashboard` | 工作台 |
| `/api/v1/exports` | 数据导出 |

未登记的前缀会打印 `⚠️  未识别路径前缀 ...`，并兜底归入「认证」。看到该告警必须补登记，否则条目会落错分组。

### 接口文档关联（`inferDocPath`）

| 路径前缀 | 关联文档 |
|---------|---------|
| `/api/v1/chat` | `docs/api/chat-api.md` |
| `/api/v1/messages` | `docs/api/message-api.md` |
| `/api/v1/invitations` | `docs/api/invitation-api.md` |
| `/api/v1/settings`, `/api/v1/admin/settings` | `docs/api/settings-api.md` |
| `/api/v1/auth` | `docs/api/auth-api.md` |
| `/api/v1/dashboard` | `docs/api/dashboard-api.md` |

## 🧩 路由表来源与新增模块登记清单

工具不再自带手写镜像：`buildRouter()` 只是 `internal/routetable.Build()` 的薄封装，而 `routetable` 用**各模块自己导出的纯注册函数**（`RegisterRoutes` / `RegisterAdminRoutes` / `RegisterTenantRoutes` 等）拼装路由树，不重复复制一遍 `r.Get(...)`。

| 位置 | 作用 | 漏改后果 |
|------|------|---------|
| `internal/routetable/routetable.go` | 在对应分组内调用新模块的注册函数 | 文档看不到新接口；`TestRouteTable_MatchesProductionRouter` 直接失败 |
| `inferTags()` | OpenAPI tag | 退化为 `API` 兜底 tag |
| `inferApifoxGroupName()` | Apifox 中文分组 | 打印告警并落错「认证」分组 |
| `inferDocPath()` | description 中的 Markdown 文档路径 | 缺少文档跳转 |
| `inferSummary()` | 中文摘要 | 退化为「获取xxx」式拼接 |

唯一的例外是 file 模块：`file.RegisterRoutes` 在注册期就构造存储后端与配额服务，依赖真实 db/cfg，无法在空 handler 场景复用，因此 `routetable` 里保留了 11 条手写镜像（`registerFileRoutes`）。

### 漂移护栏

`internal/bootstrap/route_parity_test.go` 的 `TestRouteTable_MatchesProductionRouter` 会用 sqlite + miniredis 启动**与生产完全一致的** `bootstrap.InitRouter`，逐条比对 `routetable` 的路由集合，分别报告「漏登记」与「多出」。模块改了路由而路由表没跟上，单测阶段就会失败，不会再出现「文档检查假绿」。

```bash
go test ./internal/bootstrap/ -run TestRouteTable_MatchesProductionRouter -count=1
```

新增模块的顺序要求同样适用于 `routetable`：Chi 中静态段必须排在 `{id}` 之前（如 `/chat/conversations/direct` 先于 `/chat/conversations/{id}`），注册顺序与 `internal/bootstrap/router.go` 不一致会被上述对齐测试拦下。

## ⚠️ 注意事项

1. **自动修复仅添加基础结构**：生成的路由只有默认值，需要手动补充详细的请求参数、响应体、业务说明等
2. **先检查后修复**：建议先不加 `-apply` 运行，确认差异后再执行修复
3. **`-apply` 会整体重写文件**：两份 JSON 均由 `json.MarshalIndent` 重新序列化后覆盖写入。OpenAPI 的 `paths` 是 map，diff 通常较小；Apifox 的 `items` 是有序数组，新增条目插在分组末尾，diff 可能很大但语义不变，评审时看 `apiCollection` 结构即可
4. **Apifox 条目 ID 从文档现有最大值递增分配**：`initApifoxIDCounter` 先扫描全文档的 `id`，把计数器抬到现有最大 ID 之上（最低 900000000）。写回前还会自检本次新分配的 ID 是否与既有 ID 冲突，冲突则 `exit(1)` 且不落盘
5. **手动完善文档**：自动修复后，务必补充以下信息：
   - 详细的 `summary` 和 `description`
   - 请求参数（path/query/body）
   - 响应体 schema
   - 错误响应说明
   - 业务规则说明
6. **Git 提交**：修复后提交两个 JSON 的变更
7. **重复条目也要当成不一致**：Apifox 集合按 `METHOD /path` 建 set 对比，天然吞掉重复；脚本现在会单独列出 `⚠️ Apifox 存在 N 条重复条目` 并纳入 `-check` 失败判定，`-apply` 时保留首条、删除其余

## 📝 最佳实践

### 开发流程

```bash
# 1. 开发新功能，添加路由后
go run scripts/openapi_sync.go

# 2. 查看差异，确认无误
go run scripts/openapi_sync.go -apply

# 3. 手动完善生成的 API 文档（两份都要看）
# 编辑 docs/apifox/MeteorX-backend.openapi.json
# 编辑 docs/apifox/MeteorX-backend.apifox.json

# 4. 复跑确认无差异
go run scripts/openapi_sync.go

# 5. 提交变更
git add docs/apifox/MeteorX-backend.openapi.json docs/apifox/MeteorX-backend.apifox.json
git commit -m "docs(api): 同步 OpenAPI/Apifox 文档，新增 xxx 接口"
```

### 定期检查

建议在以下时机运行同步工具：

- ✅ 每次新增/删除 API 路由后
- ✅ 提交代码前
- ✅ 发布版本前
- ✅ CI 自动卡控（`-check`，已接入 `.github/workflows/ci-cd.yml` 的 `lint` 任务）

## 🛠️ 常见问题

### Q: 为什么有些路由没有被自动识别？

A: 先确认该路由已在真实模块的注册函数里注册，并在 `internal/routetable/routetable.go` 的对应分组中调用了该函数。动态拼接路径、`NotFound` / `MethodNotAllowed` 等特殊 handler 不会被统计；`/uploads/*` 等静态资源也不在文档契约范围内。

### Q: 自动生成的 tags / 分组不正确怎么办？

A: 在 `inferTags()` 或 `inferApifoxGroupName()` 中补登记前缀规则（注意顺序），已生成的条目也可以直接在 JSON 中手动修正。

### Q: 脚本报「新增条目 ID 与既有 ID 冲突」怎么办？

A: 说明文档里存在历史遗留的重复 ID（早期版本计数器固定从 900000000 起导致）。本次写回已被中止，需先清理重复 ID 再重跑 `-apply`；不要手工改小现有 ID 来绕过校验。

### Q: 如何自定义生成的路由信息？

A: 修改 `addMissingRoute()` / `buildApifoxAPIItem()` 中的默认值，或者在自动修复后手动编辑 JSON。