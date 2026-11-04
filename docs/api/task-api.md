# 任务管理模块 API 接口说明

> Base URL: `/api/v1`  
> 所属模块：`internal/modules/task`  
> 路由注册：`internal/modules/task/module.go`  
> Handler：`internal/modules/task/handler/task_handler.go`  
> Service：`internal/modules/task/service/task_service.go`  
> Repository：`internal/modules/task/repository/task_repository.go`  
> DTO：`internal/modules/task/dto/task_dto.go`

---

## 1. 功能概述

任务管理模块同时支持**个人待办**与**租户团队协作**两种场景，二者共用同一套模型与接口，通过 `visibility` 字段区分：

- `personal`（个人待办）：仅创建人可见、可管理，负责人强制为创建人本人。
- `tenant`（团队协作）：租户内成员共享可见，可将负责人指派给其他成员，负责人也可编辑/完成。支持**多负责人**（`assignees` 集合，主负责人=`assignee_id`保留向后兼容）。

**子任务**：任务可通过 `parent_id` 挂在另一个任务下形成**单层**子任务（父任务自身不能是子任务）；列表/看板可用 `root_only` 仅展示根任务。

**任务依赖（阻塞关系）**：任务可通过 `blocked_by` 声明一组**前置任务**，表示本任务被这些前置阻塞。依赖为**多对多**关系、**单向持久化**（仅存储 `task → blocked_by`），`blocks`（后继集合）为其反向视图、读取时自动派生。依赖是**软约束**：存在未完成前置时任务仍可查看、编辑与完成（不锁死），完成前置时仅向“全部前置均已完成”的后继任务负责人推送一次**解锁提醒**，不自动推进后继状态。写入时拒绝自引用、跨租户/不可见前置与循环依赖（详见 4.1/4.2）。

**访问控制**：所有接口均需登录，模块**不使用细粒度 RBAC 权限码**，而是以数据归属（创建人 / 负责人 / 同租户）作为访问边界，任何登录用户开箱可用。

**任务提醒**：支持两类站内提醒（WebSocket 推送），详见第 7 节：
- 指派提醒：创建/更新任务并将负责人指向他人时，实时通知新负责人；
- 到期/逾期提醒：后台定时任务扫描即将到期或已逾期的未完成任务，提醒负责人，即将到期仅提醒一次、已逾期按冷却期重复提醒。
  提醒窗口/扫描间隔/冷却期/单轮上限均可通过配置 `task_reminder` 调整；任务还可通过 `remind_before` 字段设置**任务级自定义提醒提前量**，未设置则使用全局默认窗口。

---

## 2. 接口总览

所有接口均挂载在受保护分组下，需要携带 `Authorization: Bearer <token>`。

| 方法 | 路径 | 功能 |
|------|------|------|
| POST | `/tasks` | 创建任务 |
| GET | `/tasks` | 查询任务列表（分页 + 筛选） |
| GET | `/tasks/board` | 看板全量加载（不分页，当前筛选范围内上限 500 条） |
| GET | `/tasks/stats` | 任务状态统计 |
| GET | `/tasks/assignable` | 可指派成员列表（仅需登录，供负责人选择器） |
| GET | `/tasks/{id}` | 查询任务详情 |
| PUT | `/tasks/{id}` | 更新任务字段 |
| PUT | `/tasks/{id}/complete` | 标记任务完成 |
| PUT | `/tasks/{id}/reopen` | 重新打开任务 |
| DELETE | `/tasks/{id}` | 删除任务（软删除） |
| GET | `/tasks/deleted` | 回收站列表（已软删除） |
| PUT | `/tasks/{id}/restore` | 从回收站恢复任务 |
| DELETE | `/tasks/{id}/permanent` | 永久删除任务（不可恢复） |
| POST | `/tasks/batch/complete` | 批量完成任务 |
| POST | `/tasks/batch/delete` | 批量删除任务（软删除） |
| POST | `/tasks/batch/status` | 批量修改任务状态 |
| POST | `/tasks/batch/assign` | 批量指派负责人 |
| GET | `/tasks/ai/capabilities` | 探测 AI 能力可用性（可选叠加层，详见第 8 节） |
| POST | `/tasks/ai/parse` | 自然语言解析为任务草稿 |
| POST | `/tasks/ai/suggest` | 优先级/标签建议 |
| POST | `/tasks/ai/expand` | 描述扩写/拆解 |
| POST | `/tasks/ai/summary` | 任务摘要/周报 |

> 路由注册顺序上静态段（`/stats`、`/board`、`/assignable`、`/deleted`、`/batch/*`）必须先于 `/{id}`，避免路径参数吞掉静态段。

---

## 3. 数据结构

### 3.1 TaskResp（任务响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 任务 ID（26 位 ULID） |
| tenant_id | string | 所属租户 ID |
| creator_id | string | 创建人用户 ID |
| creator_name | string | 创建人展示名（昵称优先，回退用户名），无法解析时不返回该字段 |
| assignee_id | string | 主负责人用户 ID（多负责人集合首位，兼容旧字段） |
| assignee_name | string | 主负责人展示名（昵称优先，回退用户名），无法解析时不返回该字段 |
| assignees | []string | 多负责人集合（含主负责人，主负责人排首位），无额外负责人时可为空 |
| assignee_names | []string | 负责人展示名集合（与 `assignees` 同序），无法解析时不返回该字段 |
| parent_id | string | 父任务 ID，为空表示根任务（仅支持单层），根任务不返回该字段 |
| blocked_by | []DependencyBrief | 前置任务集合（本任务被其阻塞），每项含 `id`/`title`/`status`；无前置时不返回该字段 |
| blocks | []DependencyBrief | 后继任务集合（依赖本任务），为 `blocked_by` 的反向视图，每项含 `id`/`title`/`status`；无后继时不返回该字段 |
| title | string | 任务标题 |
| description | string | 任务描述 |
| status | string | 状态：`pending` / `in_progress` / `completed` |
| priority | string | 优先级：`low` / `normal` / `high` / `urgent` |
| due_date | string | 截止时间（`2006-01-02 15:04:05`），为空时不返回该字段 |
| remind_before | string | 任务级提醒提前量（Go Duration 字符串，如 `3h0m0s`），未设置时不返回该字段（表示使用全局默认） |
| tags | []string | 分类标签，无标签时返回 `[]` |
| visibility | string | 可见范围：`personal` / `tenant` |
| recurrence | string | 重复周期：`daily` / `weekly` / `monthly`，不重复时不返回该字段 |
| started_at | string | 开始时间，任务首次进入 `in_progress` 时自动记录，未开始不返回该字段 |
| completed_at | string | 完成时间，仅 `completed` 状态有值，否则不返回该字段 |
| deleted_at | string | 软删除时间，仅回收站列表返回，否则不返回该字段 |
| created_at | string | 创建时间 |
| updated_at | string | 更新时间 |

> `creator_name` / `assignee_name` / `assignee_names` 由服务端基于用户表批量解析后富化（昵称优先、回退用户名），前端无需再单独查询用户；解析失败时对应字段留空。
> `assignee_id` 语义为“主负责人”（= `assignees[0]`），保留以向后兼容既有按单负责人的过滤/提醒/看板；多负责人完整集合见 `assignees`。
> `blocked_by` / `blocks` 由服务端批量富化（携带前置/后继的标题与状态），前端据 `status != completed` 判定“被 N 阻塞”徽标；二者均为只读视图，写入依赖统一通过请求体的 `blocked_by` 完成。

**DependencyBrief（依赖引用项）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 依赖任务 ID |
| title | string | 依赖任务标题 |
| status | string | 依赖任务状态：`pending` / `in_progress` / `completed`（非 `completed` 即视为未完成/阻塞中） |

### 3.2 CreateTaskReq（创建任务请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| title | string | 是 | max=200 | 任务标题 |
| description | string | 否 | max=2000 | 任务描述 |
| status | string | 否 | oneof pending/in_progress/completed | 状态，默认 `pending` |
| priority | string | 否 | oneof low/normal/high/urgent | 优先级，默认 `normal` |
| due_date | string | 否 | - | 截止时间，支持 RFC3339 / `2006-01-02 15:04:05` / `2006-01-02` |
| remind_before | string | 否 | max=16 | 任务级提醒提前量，Go Duration 格式（如 `2h`/`30m`/`1h30m`），仅允许 `[0, 31天]`；留空使用全局默认窗口 |
| tags | []string | 否 | max=10 | 标签列表 |
| visibility | string | 否 | oneof personal/tenant | 可见范围，默认 `personal` |
| assignee_id | string | 否 | max=26 | 主负责人；`personal` 任务强制为创建人 |
| assignees | []string | 否 | max=20，每项 max=26 | 多负责人集合：给出时以集合为准（主负责人=首位），否则回落 `assignee_id`；`personal` 任务强制为创建人 |
| parent_id | string | 否 | max=26 | 父任务 ID：创建子任务时指定，仅允许挂在根任务下（单层） |
| blocked_by | []string | 否 | max=20，每项 max=26 | 前置任务 ID 集合：创建时一并建锁；任一不存在/跨租户/不可见/自引用/成环都会被拒绝 |
| recurrence | string | 否 | oneof daily/weekly/monthly | 重复周期；非空时必须同时设置 `due_date`，完成该任务后自动按周期生成下一条待办 |

### 3.3 UpdateTaskReq（更新任务请求）

所有字段均为指针类型，**仅提交需要修改的字段**，未提交（或为 null）表示不改动。

| 字段 | 类型 | 校验规则 | 说明 |
|------|------|----------|------|
| title | *string | max=200 | 标题 |
| description | *string | max=2000 | 描述 |
| status | *string | oneof pending/in_progress/completed | 状态；置为 `completed` 会自动写入 `completed_at`，其它状态清空 |
| priority | *string | oneof low/normal/high/urgent | 优先级 |
| due_date | *string | - | 截止时间；**传空字符串表示清除**截止时间 |
| remind_before | *string | max=16 | 提醒提前量（Go Duration 格式）；**传空字符串表示恢复使用全局默认** |
| tags | *[]string | max=10 | 标签 |
| visibility | *string | oneof personal/tenant | 可见范围 |
| assignee_id | *string | max=26 | 主负责人；`personal` 任务会强制回写为创建人 |
| assignees | *[]string | max=20，每项 max=26 | 多负责人集合：非 nil 时以集合为准重置负责人（主负责人=首位）；nil 表示不改动 |
| parent_id | *string | max=26 | 父任务：传入非空挂到该根任务下（单层）；传空字符串表示解除父子关系回退为根任务 |
| blocked_by | *[]string | max=20，每项 max=26 | 前置任务集合：**非 nil** 时以其为准重置本任务前置集合（**空数组=清空**）；nil 表示不改动 |
| recurrence | *string | oneof daily/weekly/monthly | 重复周期；**传空字符串表示取消重复**；非空时任务必须拥有截止时间 |

### 3.4 TaskStatsResp（任务统计响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| pending | int64 | 待办数量 |
| in_progress | int64 | 进行中数量 |
| completed | int64 | 已完成数量 |
| overdue | int64 | 逾期数量（已过截止时间且未完成） |
| total | int64 | 总数量 |
| completion_rate | float64 | 完成率 = `completed / total`，取值 0~1（保留四位小数）；总数为 0 时为 0 |
| avg_handle_seconds | int64 | 已完成任务平均处理耗时（秒），基于 `started_at`→`completed_at`，仅统计两者均有值的任务；无可统计项时为 0 |

### 3.5 BatchTaskReq（批量操作请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| ids | []string | 是 | min=1, max=100 | 待操作的任务 ID 列表 |

### 3.6 BatchTaskResp（批量操作响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| affected | int | 实际生效的任务数（无权/不存在的项会被跳过，不计入） |

### 3.7 TaskBoardResp（看板全量加载响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| items | []TaskResp | 当前筛选范围内（按 `created_at` 排序，上限后端 `BoardMax=500`）的任务，不分页 |
| total | int | 不受上限约束的真实匹配总数 |
| truncated | bool | 为真表示 `total` 超过 `items` 长度（即超过 500），看板仅展示其中一部分 |

### 3.8 AssigneeBrief（可指派成员简讯）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 用户 ID |
| name | string | 展示名（昵称优先，回退用户名） |

---

## 4. 接口详细说明

### 4.1 创建任务

`POST /api/v1/tasks`

**请求头：** `Authorization: Bearer <token>`

**请求体：**
```json
{
  "title": "完成季度报告",
  "description": "整理 Q3 数据并输出结论",
  "priority": "high",
  "due_date": "2026-10-15 18:00:00",
  "remind_before": "2h",
  "tags": ["工作", "季度"],
  "visibility": "tenant",
  "assignee_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV"
}
```

**业务规则：**
- `status` 缺省为 `pending`，`priority` 缺省为 `normal`，`visibility` 缺省为 `personal`
- `personal` 任务的负责人强制为创建人本人，忽略传入的 `assignee_id`/`assignees`
- `tenant` 任务可将负责人指派给其他成员：传 `assignees` 时以集合为准（主负责人=首位），否则回落 `assignee_id` 单个；为空时默认为创建人
- 指派负责人时逐个校验其**存在且属于当前租户**（未删除的有效用户），否则返回 `400 负责人不存在或不属于当前租户`；未启用用户目录校验器时跳过该校验
- 多负责人写入关联表 `task_assignees`（与任务同事务替换），`assignee_id` 同步为主负责人（`assignees[0]`）以保证向后兼容
- `parent_id` 创建子任务（仅支持单层）：父任务须存在、同租户、可查看，且父任务自身不能再有父任务，否则返回 `400 父任务不合法`/`404 父任务不存在`；不允许 `parent_id==自身ID`。子任务可见范围继承父任务
- `blocked_by` 声明前置任务集合（多对多阻塞关系）：逐个校验前置任务**存在、同租户、当前用户可查看**，否则返回 `404 前置任务不存在或无权访问`；任一前置为任务自身返回 `400 任务不能作为自己的前置依赖`；若加入后会形成**循环阻塞链**（从候选前置沿现有依赖可达本任务）返回 `400 该前置依赖会形成循环阻塞`。校验在写库前完成，不会产生孤儿任务；通过后依赖写入关联表 `task_dependencies`
- `recurrence` 为重复周期，**设置非空值时必须同时提供 `due_date`**（否则返回 `400 周期性任务必须设置截止时间`）；任务完成后系统会按周期从原截止日推算并自动创建一条新的 `pending` 任务（见 4.16）
- 若创建时直接指定 `status=completed`，会写入 `completed_at`
- `due_date` 格式非法返回 `400`
- `remind_before` 为任务级提醒提前量（仅在设置了 `due_date` 时生效）；无法解析或超出 `[0, 31天]` 范围返回 `400`；留空则使用全局默认提醒窗口
- `tenant` 任务指派给他人（负责人 ≠ 创建人）时，系统会向**全部新负责人**去重发送站内指派提醒（见第 7 节）

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "01M2Q3QXR8R1SNP95BASQ1EFCS",
    "tenant_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "creator_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "creator_name": "张三",
    "assignee_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "assignee_name": "张三",
    "title": "完成季度报告",
    "description": "整理 Q3 数据并输出结论",
    "status": "pending",
    "priority": "high",
    "due_date": "2026-10-15 18:00:00",
    "tags": ["工作", "季度"],
    "visibility": "tenant",
    "created_at": "2026-10-01 09:00:00",
    "updated_at": "2026-10-01 09:00:00"
  }
}
```

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 参数校验失败 / 截止时间格式不正确 |
| 401 | 未登录 |

---

### 4.2 查询任务列表

`GET /api/v1/tasks`

**请求头：** `Authorization: Bearer <token>`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页条数，默认 10 |
| visibility | string | 可见范围筛选：`personal` / `tenant`；不传时返回"租户任务 + 自己相关的个人任务" |
| status | string | 状态筛选：`pending` / `in_progress` / `completed`；**传 `overdue` 为"仅逾期"伪状态**，返回未完成的逾期任务（口径与 `/tasks/stats` 的 `overdue` 一致，供前端逾期卡片下钻） |
| priority | string | 优先级筛选 |
| assignee_id | string | 按负责人筛选（命中主负责人 `assignee_id` **或** `task_assignees` 集合中包含该用户即副负责人也命中） |
| keyword | string | 按标题/描述模糊搜索 |
| tag | string | 按标签精确筛选（任务标签集合包含该标签时命中，基于 MySQL `JSON_CONTAINS`） |
| due_from | string | 截止时间区间下界（含），支持 RFC3339 / `2006-01-02 15:04:05` / `2006-01-02`；格式非法返回 `400` |
| due_to | string | 截止时间区间上界（含）；**仅传日期（`2006-01-02`）时后端自动补足到当天 `23:59:59`**，使按日期区间含当天全部任务；格式非法返回 `400` |
| sort_by | string | 排序字段（如 `created_at`、`due_date`、`priority`） |
| sort_order | string | 排序方向 `asc` / `desc` |
| parent_id | string | 按父任务 ID 过滤，拉取某任务的直接子任务 |
| root_only | bool | 传 `1`/`true`/`on` 仅返回根任务（`parent_id` 为空），列表默认视图用于隐藏子任务 |

**默认排序：** 未完成任务优先（`completed` 靠后）→ 优先级 `urgent > high > normal > low` → 创建时间倒序。

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "01M2Q3QXR8R1SNP95BASQ1EFCS",
        "title": "完成季度报告",
        "status": "pending",
        "priority": "high",
        "visibility": "tenant",
        "...": "其余字段见 TaskResp"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 10,
      "total": 1
    }
  }
}
```

**数据隔离：** 结果始终限制在当前租户内；个人任务只对创建人/负责人可见。

---

### 4.3 任务状态统计

`GET /api/v1/tasks/stats`

**请求头：** `Authorization: Bearer <token>`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| visibility | string | 可选，`personal` / `tenant`，限定统计范围 |

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "pending": 2,
    "in_progress": 1,
    "completed": 5,
    "overdue": 0,
    "total": 8
  }
}
```

---

### 4.4 查询任务详情

`GET /api/v1/tasks/{id}`

**请求头：** `Authorization: Bearer <token>`

**可见性规则：**
- 任务必须属于当前租户
- `tenant` 任务对同租户所有成员可见
- `personal` 任务仅创建人或负责人可见
- 不满足可见性时统一返回 `404`（不泄露任务是否存在）

**成功响应（200）：** 返回 `TaskResp`。

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 参数不完整 |
| 404 | 任务不存在或无权查看 |

---

### 4.5 更新任务

`PUT /api/v1/tasks/{id}`

**请求头：** `Authorization: Bearer <token>`

**请求体（仅提交需修改字段）：**
```json
{
  "title": "新的标题",
  "status": "in_progress"
}
```

**业务规则：**
- 仅**创建人**或（`tenant` 任务的）**任一负责人（主/副）**可修改，否则返回 `403`
- 跨租户访问返回 `404`
- `assignees` 非 nil 时以集合为准重置多负责人（主负责人=首位，同步写回 `assignee_id` 与 `task_assignees`）；逐个校验同租户有效成员
- `parent_id` 传入非空时挂到该根任务下（仅单层，校验同 §4.1）；传空字符串解除父子关系回退为根任务
- 将 `status` 改为 `completed` 会写入 `completed_at`，改为其它状态会清空 `completed_at`
- 将 `status` 改为 `in_progress` 时，若尚未记录过开始时间则自动写入 `started_at`（一旦开始不再清空）
- `due_date` 传空字符串表示清除截止时间

**成功响应（200）：** 返回更新后的 `TaskResp`。

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 参数校验失败 / 截止时间格式不正确 |
| 403 | 无权修改该任务 |
| 404 | 任务不存在 |

---

### 4.6 标记任务完成

`PUT /api/v1/tasks/{id}/complete`

**请求头：** `Authorization: Bearer <token>`

**业务规则：**
- 仅创建人或（`tenant` 任务的）负责人可操作
- 状态置为 `completed`

**成功响应（200）：** 返回更新后的 `TaskResp`（含 `completed_at`）。

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 403 | 无权操作该任务 |
| 404 | 任务不存在 |

---

### 4.7 重新打开任务

`PUT /api/v1/tasks/{id}/reopen`

**请求头：** `Authorization: Bearer <token>`

**请求体（可选）：**
```json
{
  "status": "in_progress"
}
```

**业务规则：**
- 仅创建人或（`tenant` 任务的）负责人可操作
- `status` 传 `in_progress` 回到进行中；其它值或不传均回到 `pending`

**成功响应（200）：** 返回更新后的 `TaskResp`。

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 403 | 无权操作该任务 |
| 404 | 任务不存在 |

---

### 4.8 删除任务

`DELETE /api/v1/tasks/{id}`

**请求头：** `Authorization: Bearer <token>`

**业务规则：**
- **软删除**，写入 `deleted_at`，列表与详情不再返回
- 删除父任务会**级联软删其直接子任务**（单层）；回收站/恢复按单条独立，恢复父不会自动恢复子（已知限制）
- 删除权限更严格：**仅创建人**（或 `superadmin` 角色）可删除，负责人无删除权
- 跨租户访问返回 `404`

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 403 | 无权删除该任务 |
| 404 | 任务不存在 |

---

### 4.9 回收站列表

`GET /api/v1/tasks/deleted`

**请求头：** `Authorization: Bearer <token>`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页条数，默认 10 |
| keyword | string | 按标题/描述模糊搜索 |

**业务规则：**
- 仅返回**当前用户自己创建且已软删除**的任务，避免跨成员泄露
- 按删除时间倒序
- 返回项包含 `deleted_at` 字段

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "01M2Q3QXR8R1SNP95BASQ1EFCS",
        "title": "已删除的任务",
        "status": "pending",
        "visibility": "personal",
        "deleted_at": "2026-10-02 10:00:00",
        "...": "其余字段见 TaskResp"
      }
    ],
    "pagination": { "page": 1, "page_size": 10, "total": 1 }
  }
}
```

---

### 4.10 恢复任务

`PUT /api/v1/tasks/{id}/restore`

**请求头：** `Authorization: Bearer <token>`

**业务规则：**
- 将软删除任务的 `deleted_at` 置空，恢复为正常任务
- 仅**创建人**（或 `superadmin` 角色）可恢复
- 跨租户访问返回 `404`

**成功响应（200）：** 返回恢复后的 `TaskResp`。

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 403 | 无权恢复该任务 |
| 404 | 任务不存在 |

---

### 4.11 永久删除任务

`DELETE /api/v1/tasks/{id}/permanent`

**请求头：** `Authorization: Bearer <token>`

**业务规则：**
- **物理删除**数据库记录，不可恢复
- 仅**创建人**（或 `superadmin` 角色）可操作
- 跨租户访问返回 `404`

**成功响应（200）：**
```json
{ "code": 200, "message": "success", "data": null }
```

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 403 | 无权操作该任务 |
| 404 | 任务不存在 |

---

### 4.12 批量完成任务

`POST /api/v1/tasks/batch/complete`

**请求头：** `Authorization: Bearer <token>`

**请求体：**
```json
{
  "ids": ["01M2Q3QXR8R1SNP95BASQ1EFCS", "01M2Q3QXR8R1SNP95BASQ1EFC2"]
}
```

**业务规则：**
- 逐条按单条完成的归属规则校验（创建人或 `tenant` 任务的负责人）
- 无权或不存在的项**静默跳过**，不影响其它项
- 返回实际生效数 `affected`

**成功响应（200）：**
```json
{ "code": 200, "message": "success", "data": { "affected": 2 } }
```

---

### 4.13 批量删除任务

`POST /api/v1/tasks/batch/delete`

**请求头：** `Authorization: Bearer <token>`

**请求体：**
```json
{
  "ids": ["01M2Q3QXR8R1SNP95BASQ1EFCS"]
}
```

**业务规则：**
- 对每个任务执行**软删除**，规则与单条删除一致（仅创建人或 `superadmin`）
- 无权或不存在的项静默跳过
- 返回实际生效数 `affected`

**成功响应（200）：**
```json
{ "code": 200, "message": "success", "data": { "affected": 1 } }
```

**错误响应（批量接口通用）：**
| 状态码 | 场景 |
|--------|------|
| 400 | 参数校验失败（`ids` 为空或超过 100 项） |
| 401 | 未登录 |

---

### 4.14 批量修改状态

`POST /api/v1/tasks/batch/status`

**请求头：** `Authorization: Bearer <token>`

**请求体：**
```json
{
  "ids": ["01M2Q3QXR8R1SNP95BASQ1EFCS", "01M2Q3QXR8R1SNP95BASQ1EFC2"],
  "status": "in_progress"
}
```

**业务规则：**
- `status` 必填，取值 `pending` / `in_progress` / `completed`，非法值返回 `400`
- 逐条按单条状态变更的归属规则校验（创建人或 `tenant` 任务的负责人），无权/不存在项静默跳过
- 改为 `completed` 写入 `completed_at`；首次改为 `in_progress` 自动写入 `started_at`
- 返回实际生效数 `affected`

**成功响应（200）：**
```json
{ "code": 200, "message": "success", "data": { "affected": 2 } }
```

---

### 4.15 批量指派负责人

`POST /api/v1/tasks/batch/assign`

**请求头：** `Authorization: Bearer <token>`

**请求体：**
```json
{
  "ids": ["01M2Q3QXR8R1SNP95BASQ1EFCS"],
  "assignees": ["01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAV2"]
}
```

**业务规则：**
- `assignees`（集合）与 `assignee_id`（单个）至少传一个；`assignees` 非空时以集合为准（主负责人=首位），否则回落 `assignee_id`
- 逐条复用更新逻辑（含归属校验），无权/不存在项静默跳过
- 会对集合中每个负责人校验是否为**同租户有效成员**；非法则整体返回 `400 负责人不存在或不属于当前租户`（不会逐项吞错）
- `personal` 任务会被强制回写为创建人（仍计入 `affected`）
- `tenant` 任务指派给他人时，逐条向**全部新负责人**去重发送站内指派提醒
- 返回实际生效数 `affected`

**成功响应（200）：**
```json
{ "code": 200, "message": "success", "data": { "affected": 1 } }
```

---

### 4.16 重复任务机制

任务可通过 `recurrence` 字段设定重复周期（`daily` / `weekly` / `monthly`），实现“完成一次、自动生成下一期”的循环待办。

**触发条件：**
- 仅当任务从**非完成状态首次转为 `completed`** 时生成下一实例（重复完成、重开后再完成不会重复生成）。
- 该机制对单条完成、批量完成、批量改状态为 `completed`、以及更新中直接置 `completed` 均生效。

**生成规则：**
- 新任务为全新 ID、状态 `pending`，复制原任务的标题/描述/优先级/负责人/可见范围/标签/提醒提前量与重复周期。
- 新任务截止时间 = 原任务截止时间按周期推进（daily +1 天 / weekly +7 天 / monthly +1 个月，基于原截止日而非完成日，保持节拍对齐）。
- 周期任务必须拥有截止时间；若原任务无 `due_date` 则不生成下一实例。
- 实例生成失败不会阻断本次完成（静默跳过）。

---

### 4.17 看板全量加载

`GET /api/v1/tasks/board`

**请求头：** `Authorization: Bearer <token>`

一次性拉取当前筛选范围内看板分列所需的全部任务，**不分页、忽略 `status`**（看板需同时展示各状态），上限 `BoardMax=500` 条。

**Query 参数：** 与列表接口共享过滤条件（`visibility`、`priority`、`assignee_id`、`keyword`、`tag`、`due_from`、`due_to`、`sort_by`、`sort_order`、`parent_id`、`root_only`），但**不接受 `page`/`page_size`/`status`**。

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [
      { "id": "01M2Q3QXR8R1SNP95BASQ1EFCS", "title": "完成季度报告", "status": "pending", "visibility": "tenant", "...": "其余字段见 TaskResp" }
    ],
    "total": 2,
    "truncated": false
  }
}
```

**业务规则：**
- 可见性过滤与列表一致（租户任务 + 自己相关个人任务，含副负责人命中）。
- 看板默认只展示根任务（前端传 `root_only=1`），保持分列清爽。
- `truncated=true` 仅当匹配总数超过 500（而非当前列卡片少于总数）。

---

### 4.18 可指派成员列表

`GET /api/v1/tasks/assignable`

**请求头：** `Authorization: Bearer <token>`

返回当前租户存活成员（未删除），供前端负责人多选选择器使用。**仅需登录，无需 `user:list` 权限**，解决普通成员无法检索可指派人员的问题。

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": [
    { "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV", "name": "张三" }
  ]
}
```

**业务规则：**
- 仅返回当前用户所属租户内未软删除的成员（按用户名升序）；跨租户自然隔离。

---

## 5. 访问控制矩阵

| 操作 | personal 任务 | tenant 任务 |
|------|---------------|-------------|
| 查看 | 创建人 / 任一负责人（主/副） | 同租户任意成员 |
| 编辑 / 完成 / 重开 | 创建人 | 创建人 或 任一负责人（主/副） |
| 删除（软删除） | 创建人（或 superadmin） | 创建人（或 superadmin） |
| 回收站查看 / 恢复 / 永久删除 | 创建人（或 superadmin） | 创建人（或 superadmin） |

> `personal` 任务的负责人恒等于创建人，因此"负责人"对个人任务与创建人等价。

---

## 6. 状态与优先级

### 6.1 状态流转

```
   创建
    │
    ▼
┌─────────┐   complete   ┌───────────┐
│ pending │ ───────────> │ completed │
└────┬────┘              └─────┬─────┘
     │  reopen(in_progress)    │ reopen
     ▼                         │
┌─────────────┐                │
│ in_progress │ <──────────────┘
└─────────────┘   complete → completed
```

| 状态 | 说明 |
|------|------|
| `pending` | 待办，任务尚未开始 |
| `in_progress` | 进行中，任务正在处理，首次进入时自动记录 `started_at` |
| `completed` | 已完成，`completed_at` 有值 |

### 6.2 优先级

| 值 | 说明 |
|------|------|
| `low` | 低 |
| `normal` | 普通（默认） |
| `high` | 高 |
| `urgent` | 紧急 |

### 6.3 可见范围

| 值 | 说明 |
|------|------|
| `personal` | 个人待办，仅创建人可见可管理 |
| `tenant` | 团队协作，租户内共享可见、可指派负责人 |

---

## 7. 任务提醒机制

提醒不新增 HTTP 接口，通过已有的 WebSocket 站内信通道（`GET /api/v1/ws`）定向推送给接收人，消息以 `type=announcement` 下发，靠 `payload.type` 区分提醒种类：

| payload.type | 触发时机 | 接收人 | 标题 |
|--------------|----------|--------|------|
| `task_assigned` | 创建/更新任务且负责人指向他人（变更时重新提醒） | 全部新负责人（去重） | 新任务指派 |
| `task_due` | 定时扫描：任务进入提醒窗口（任务自定义 `remind_before` 优先，否则全局 `task_reminder.horizon`）或已逾期，且状态为 `pending`/`in_progress`；已逾期的任务按冷却期（默认 24 小时）重复提醒 | 负责人（为空时创建人） | 任务截止提醒 / 任务逾期提醒 |
| `task_unblocked` | 完成一个任务（状态由非 `completed` 变为 `completed`，含批量完成）时，若其后继任务的全部前置均已完成 | 后继任务的负责人（去重、排除空） | 前置任务已完成 |

**消息体示例（WS payload）：**
```json
{
  "type": "announcement",
  "payload": {
    "type": "task_due",
    "title": "任务逾期提醒",
    "content": "您负责的任务「写周报」已逾期（截止：2026-10-01 18:00），请尽快处理。",
    "priority": "normal",
    "time": 1790000000,
    "task_id": "01M2Q3QXR8R1SNP95BASQ1EFCS",
    "tenant_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV"
  }
}
```

**实现要点：**
- 提醒去重：`tasks.reminder_sent_at` 列记录上次到期提醒发送时间。即将到期（未逾期）仅首次提醒一次；**已逾期的任务按冷却期（默认 24 小时）重复提醒**（距上次提醒超过冷却期才会再次推送）；**修改截止时间或提前量会重置该标记**，使新提醒窗口可重新触发。
- 提醒窗口：扫描时优先使用任务自带的 `remind_before`（以 `now + remind_before` 为阈值），未设置（NULL）时回落到全局 `task_reminder.horizon`。
- 定时任务 `TaskReminderJob` 随应用启动，按 `task_reminder.interval`（默认 30 分钟）周期扫描（启动时先立即执行一轮），单轮最多 `task_reminder.batch_limit`（默认 200）条；系统级扫描，不受租户/可见范围限制。
- 指派提醒在负责人变更（含首次指派给他人）时触发，对多负责人会**遍历集合去重后逐个推送**，自己给自己的任务不提醒。
- **解锁提醒**（`task_unblocked`）：当一个任务完成时，若存在依赖它的后继任务且该后继的**全部前置均已完成**，则向该后继负责人推送一次提醒（“前置《X》已完成，《Y》可开始”）。这是**软约束**：仅提醒、不自动推进后继状态；批量完成复用单条完成链路同样生效。
- 通知不可用（WS 未启用/用户离线）时静默跳过，不影响任务主流程；离线用户不会补发。
- 前端（web-admin）在全局 `App.vue` 中监听，命中提醒类型时弹出 `ElNotification`，不刷新公告列表。

---

## 8. AI 辅助接口（可选叠加层）

> Handler：`internal/modules/task/handler/task_ai_handler.go`  
> Service：`internal/modules/task/service/task_ai_service.go`  
> DTO：`internal/modules/task/dto/task_ai_dto.go`  
> 通用 AI 抽象层：`internal/ai/`（OpenAI 兼容）

AI 能力为**默认关闭的可选叠加层**：走 OpenAI 兼容接口，由配置 `ai`（`METEORX_AI_*`）控制；配置缺失或调用失败时**自动降级**，任务 CRUD 永不依赖 AI。所有 AI 端点均为**只读/草稿**能力，本身不写库，解析结果由用户确认后再走原有创建/更新流程。端点同挂受保护 `/tasks` 分组（需登录、无新权限码），静态段 `/ai/*` 先于 `/{id}` 注册。

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/tasks/ai/capabilities` | 探测 AI 可用性与默认模型（供前端门控按钮） |
| POST | `/tasks/ai/parse` | 自然语言解析为任务草稿 |
| POST | `/tasks/ai/suggest` | 优先级/标签建议 |
| POST | `/tasks/ai/expand` | 描述扩写/拆解 |
| POST | `/tasks/ai/summary` | 任务摘要/周报 |

**降级约定**：`ErrAIUnavailable`（未启用/无凭据）返回 `503 AI 能力未启用`；`ErrAIRuntime`（上游失败/坏响应）返回 `502 AI 服务暂不可用，请稍后重试`；校验失败 `400`。

### 8.1 探测能力

`GET /api/v1/tasks/ai/capabilities` — 无需请求体。

**成功响应（200）：**
```json
{ "code": 200, "message": "success", "data": { "enabled": false, "model": "gpt-4o-mini" } }
```

### 8.2 自然语言建任务

`POST /api/v1/tasks/ai/parse`

**请求体：** `{ "text": "下周五前高优先级写完周报并加标签 运营" }`（`text` 必填，max=500）

**成功响应（200）——任务草稿（对齐创建表单，供预填）：**
```json
{
  "code": 200, "message": "success",
  "data": { "title": "写周报", "priority": "high", "due_date": "2026-10-23 18:00:00", "tags": ["运营"], "recurrence": "weekly" }
}
```
> 服务向模型注入当前时间以正确解析「下周五/三天后」等相对日期；`due_date` 无法确定时省略，`recurrence` 未提及为空。

### 8.3 优先级/标签建议

`POST /api/v1/tasks/ai/suggest` — `{ "title": "修复登录报错", "description": "线上故障" }`（`title` 必填 max=200，`description` max=2000）

**响应 `data`：** `{ "priority": "urgent", "tags": ["线上", "故障"], "reason": "线上故障需优先处理" }`

### 8.4 描述扩写/拆解

`POST /api/v1/tasks/ai/expand` — `{ "title": "写周报", "description": "初稿" }`

**响应 `data`：** `{ "description": "扩写后的完整描述", "steps": ["收集本周数据", "撰写结论"] }`

### 8.5 任务摘要/周报

`POST /api/v1/tasks/ai/summary` — `{ "scope": "week" }`（`scope` 可选：`today`/`week`/`overdue`/`all`，缺省 `all`；请求体须为 JSON 对象，`{}` 表示默认）

服务按 `scope` + 数据归属读取当前用户任务后交给模型汇总；无任务时短路返回「暂无任务」，不调用模型。

**响应 `data`：**
```json
{
  "text": "本周共 8 项任务，已完成 5 项……",
  "highlights": ["2 项高优先级待办"],
  "suggested_focus": ["优先处理逾期的「合同复审」"],
  "total": 8
}
```

---

## 9. 错误响应约定

所有错误遵循全局响应结构：

```json
{
  "code": 400,
  "message": "错误描述",
  "data": null
}
```

常见状态码：`400` 参数错误、`401` 未登录、`403` 无权操作、`404` 资源不存在、`500` 服务器内部错误。

---

## 10. 说明

- 本模块**不需要单独配置权限码**：任何已登录用户均可直接使用，靠数据归属实现隔离。
- 任务表 `tasks` 随应用启动自动迁移（非关键迁移项），使用软删除。
- 多负责人关联表 `task_assignees`（复合主键 `task_id`+`user_id`）随启动 `AutoMigrate` 建表，并执行**幂等回填**：将存量 `tasks.assignee_id` 写入 `task_assignees`（已存在则忽略）。
- 任务依赖关联表 `task_dependencies`（复合主键 `task_id`+`blocked_by_id`，`blocked_by_id` 另建索引）随启动 `AutoMigrate` 建表；**单向持久化** `task→blocked_by`，`blocks` 反向视图读取时自动派生，不重复存储；`tasks` 表无需加列。
- 默认排序依赖 MySQL 的 `FIELD()` 函数，与项目数据库选型一致。
