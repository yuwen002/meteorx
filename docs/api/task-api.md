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
- `tenant`（团队协作）：租户内成员共享可见，可将负责人指派给其他成员，负责人也可编辑/完成。

**访问控制**：所有接口均需登录，模块**不使用细粒度 RBAC 权限码**，而是以数据归属（创建人 / 负责人 / 同租户）作为访问边界，任何登录用户开箱可用。

**任务提醒**：支持两类站内提醒（WebSocket 推送），详见第 7 节：
- 指派提醒：创建/更新任务并将负责人指向他人时，实时通知新负责人；
- 到期/逾期提醒：后台定时任务扫描即将到期（24 小时内）或已逾期的未完成任务，提醒负责人，同一任务仅提醒一次。

---

## 2. 接口总览

所有接口均挂载在受保护分组下，需要携带 `Authorization: Bearer <token>`。

| 方法 | 路径 | 功能 |
|------|------|------|
| POST | `/tasks` | 创建任务 |
| GET | `/tasks` | 查询任务列表（分页 + 筛选） |
| GET | `/tasks/stats` | 任务状态统计 |
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

> 路由注册顺序上静态段（`/stats`、`/deleted`、`/batch/*`）必须先于 `/{id}`，避免路径参数吞掉静态段。

---

## 3. 数据结构

### 3.1 TaskResp（任务响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 任务 ID（26 位 ULID） |
| tenant_id | string | 所属租户 ID |
| creator_id | string | 创建人用户 ID |
| assignee_id | string | 负责人用户 ID |
| title | string | 任务标题 |
| description | string | 任务描述 |
| status | string | 状态：`pending` / `in_progress` / `completed` |
| priority | string | 优先级：`low` / `normal` / `high` / `urgent` |
| due_date | string | 截止时间（`2006-01-02 15:04:05`），为空时不返回该字段 |
| tags | []string | 分类标签，无标签时返回 `[]` |
| visibility | string | 可见范围：`personal` / `tenant` |
| completed_at | string | 完成时间，仅 `completed` 状态有值，否则不返回该字段 |
| deleted_at | string | 软删除时间，仅回收站列表返回，否则不返回该字段 |
| created_at | string | 创建时间 |
| updated_at | string | 更新时间 |

### 3.2 CreateTaskReq（创建任务请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| title | string | 是 | max=200 | 任务标题 |
| description | string | 否 | max=2000 | 任务描述 |
| status | string | 否 | oneof pending/in_progress/completed | 状态，默认 `pending` |
| priority | string | 否 | oneof low/normal/high/urgent | 优先级，默认 `normal` |
| due_date | string | 否 | - | 截止时间，支持 RFC3339 / `2006-01-02 15:04:05` / `2006-01-02` |
| tags | []string | 否 | max=10 | 标签列表 |
| visibility | string | 否 | oneof personal/tenant | 可见范围，默认 `personal` |
| assignee_id | string | 否 | max=26 | 负责人；`personal` 任务强制为创建人 |

### 3.3 UpdateTaskReq（更新任务请求）

所有字段均为指针类型，**仅提交需要修改的字段**，未提交（或为 null）表示不改动。

| 字段 | 类型 | 校验规则 | 说明 |
|------|------|----------|------|
| title | *string | max=200 | 标题 |
| description | *string | max=2000 | 描述 |
| status | *string | oneof pending/in_progress/completed | 状态；置为 `completed` 会自动写入 `completed_at`，其它状态清空 |
| priority | *string | oneof low/normal/high/urgent | 优先级 |
| due_date | *string | - | 截止时间；**传空字符串表示清除**截止时间 |
| tags | *[]string | max=10 | 标签 |
| visibility | *string | oneof personal/tenant | 可见范围 |
| assignee_id | *string | max=26 | 负责人；`personal` 任务会强制回写为创建人 |

### 3.4 TaskStatsResp（任务统计响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| pending | int64 | 待办数量 |
| in_progress | int64 | 进行中数量 |
| completed | int64 | 已完成数量 |
| overdue | int64 | 逾期数量（已过截止时间且未完成） |
| total | int64 | 总数量 |

### 3.5 BatchTaskReq（批量操作请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| ids | []string | 是 | min=1, max=100 | 待操作的任务 ID 列表 |

### 3.6 BatchTaskResp（批量操作响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| affected | int | 实际生效的任务数（无权/不存在的项会被跳过，不计入） |

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
  "tags": ["工作", "季度"],
  "visibility": "tenant",
  "assignee_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV"
}
```

**业务规则：**
- `status` 缺省为 `pending`，`priority` 缺省为 `normal`，`visibility` 缺省为 `personal`
- `personal` 任务的负责人强制为创建人本人，忽略传入的 `assignee_id`
- `tenant` 任务可将 `assignee_id` 指派给其他成员，为空时默认为创建人
- 若创建时直接指定 `status=completed`，会写入 `completed_at`
- `due_date` 格式非法返回 `400`
- `tenant` 任务指派给他人（负责人 ≠ 创建人）时，系统会向新负责人发送一条站内指派提醒（见第 7 节）

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "01M2Q3QXR8R1SNP95BASQ1EFCS",
    "tenant_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "creator_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "assignee_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
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
| status | string | 状态筛选 |
| priority | string | 优先级筛选 |
| assignee_id | string | 按负责人筛选 |
| keyword | string | 按标题/描述模糊搜索 |
| sort_by | string | 排序字段（如 `created_at`、`due_date`、`priority`） |
| sort_order | string | 排序方向 `asc` / `desc` |

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
- 仅**创建人**或（`tenant` 任务的）**负责人**可修改，否则返回 `403`
- 跨租户访问返回 `404`
- 将 `status` 改为 `completed` 会写入 `completed_at`，改为其它状态会清空 `completed_at`
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

## 5. 访问控制矩阵

| 操作 | personal 任务 | tenant 任务 |
|------|---------------|-------------|
| 查看 | 创建人 / 负责人 | 同租户任意成员 |
| 编辑 / 完成 / 重开 | 创建人 | 创建人 或 负责人 |
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
| `in_progress` | 进行中，任务正在处理 |
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
| `task_assigned` | 创建/更新任务且负责人指向他人（变更时重新提醒） | 新负责人 | 新任务指派 |
| `task_due` | 定时扫描：截止时间进入 24 小时窗口或已逾期，且状态为 `pending`/`in_progress` | 负责人（为空时创建人） | 任务截止提醒 / 任务逾期提醒 |

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
- 提醒去重：`tasks.reminder_sent_at` 列记录到期提醒发送时间，定时任务只处理该列为空的任务；**修改截止时间会重置该标记**，使新截止日可再次触发提醒。
- 定时任务 `TaskReminderJob` 随应用启动，每 30 分钟扫描一次（启动时先立即执行一轮），单轮最多 200 条；系统级扫描，不受租户/可见范围限制。
- 指派提醒在负责人变更（含首次指派给他人）时触发，自己给自己的任务不提醒。
- 通知不可用（WS 未启用/用户离线）时静默跳过，不影响任务主流程；离线用户不会补发。
- 前端（web-admin）在全局 `App.vue` 中监听，命中提醒类型时弹出 `ElNotification`，不刷新公告列表。

---

## 8. 错误响应约定

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

## 9. 说明

- 本模块**不需要单独配置权限码**：任何已登录用户均可直接使用，靠数据归属实现隔离。
- 任务表 `tasks` 随应用启动自动迁移（非关键迁移项），使用软删除。
- 默认排序依赖 MySQL 的 `FIELD()` 函数，与项目数据库选型一致。
