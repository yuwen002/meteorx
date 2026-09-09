# 通知公告 API 文档

通知公告模块（`internal/modules/notification`）提供平台公告的完整生命周期管理：创建、编辑、发布、下架、删除。公告可面向全平台推送，也可指定单个租户定向推送。

**权限：** 需超级管理员角色，且具备对应权限码（`RequiresMasterAdmin` + `AutoRequirePermission`）。

**基础路径：** `/api/v1`

---

## 1. 接口列表

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/admin/announcements` | 公告列表（分页） | `admin:announcement:list` |
| POST | `/admin/announcements` | 创建公告 | `admin:announcement:create` |
| GET | `/admin/announcements/{id}` | 公告详情 | `admin:announcement:read` |
| PUT | `/admin/announcements/{id}` | 更新公告 | `admin:announcement:update` |
| PUT | `/admin/announcements/{id}/status` | 发布 / 下架公告 | `admin:announcement:status` |
| DELETE | `/admin/announcements/{id}` | 删除公告 | `admin:announcement:delete` |

### 1.2 租户侧接口（需登录）

| 方法 | 路径 | 功能 | 认证 |
|------|------|------|------|
| GET | `/announcements` | 获取当前租户可见公告列表（分页） | 需登录 |

---

## 2. 数据结构

### 2.1 公告状态与范围常量

| 枚举 | 值 | 说明 |
|------|-----|------|
| status 草稿 | 0 | 未发布 |
| status 已发布 | 1 | 已对外可见 |
| status 已下架 | 2 | 下架后不可见 |
| scope 全平台 | all | 面向所有租户 |
| scope 指定租户 | tenant | 仅面向指定租户（target_tenant_id） |

### 2.2 CreateAnnouncementReq（创建公告请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 是 | 公告标题（max=200） |
| content | string | 是 | 公告内容 |
| scope | string | 是 | 范围：all / tenant |
| target_tenant_id | string | 否 | scope=tenant 时必填 |
| status | int | 否 | 初始状态：0 草稿 / 1 已发布 / 2 已下架 |
| publish_at | string | 否 | 发布时间（`2006-01-02 15:04:05`） |
| expire_at | string | 否 | 过期时间（`2006-01-02 15:04:05`） |

### 2.3 UpdateAnnouncementReq（更新公告请求）

字段结构与 CreateAnnouncementReq 相同。

### 2.4 UpdateAnnouncementStatusReq（发布/下架请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| status | int | 是 | 1 发布 / 2 下架 |

### 2.5 AnnouncementResp（公告响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 公告 ID |
| title | string | 标题 |
| content | string | 内容 |
| scope | string | 范围 |
| target_tenant_id | string | 目标租户 ID |
| status | int | 状态：0 草稿 / 1 已发布 / 2 已下架 |
| status_text | string | 状态文案 |
| publisher_id | string | 发布人（平台管理员） |
| publish_at | string | 发布时间 |
| expire_at | string | 过期时间 |
| created_at | string | 创建时间 |
| updated_at | string | 更新时间 |

### 2.6 AnnouncementListResp（公告列表响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| items | array\<AnnouncementResp\> | 公告列表 |
| total | int64 | 总数 |

---

## 3. 接口详细说明

### 3.1 公告列表

`GET /api/v1/admin/announcements`

**Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20 |
| keyword | string | 否 | 标题关键字模糊搜索 |
| status | int | 否 | 状态筛选：0/1/2 |
| scope | string | 否 | 范围筛选：all / tenant |

**成功响应（200）：** AnnouncementListResp
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "items": [
      {
        "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
        "title": "平台维护通知",
        "content": "本周六凌晨进行系统升级...",
        "scope": "all",
        "target_tenant_id": "",
        "status": 1,
        "status_text": "已发布",
        "publisher_id": "01ARZ3NDEKTSV4RRFFQ69G5FAW",
        "publish_at": "2026-08-20 10:00:00",
        "expire_at": "2026-08-27 10:00:00",
        "created_at": "2026-08-19 09:00:00",
        "updated_at": "2026-08-20 10:00:00"
      }
    ],
    "total": 1
  }
}
```

### 3.2 创建公告

`POST /api/v1/admin/announcements`

**请求体：** CreateAnnouncementReq
```json
{
  "title": "平台维护通知",
  "content": "本周六凌晨进行系统升级，期间服务可能出现短暂不可用。",
  "scope": "all",
  "status": 0,
  "publish_at": "2026-08-20 10:00:00",
  "expire_at": "2026-08-27 10:00:00"
}
```

**业务规则：** 若 `scope=tenant` 则必须提供 `target_tenant_id`。

### 3.3 公告详情

`GET /api/v1/admin/announcements/{id}`

**成功响应（200）：** AnnouncementResp

### 3.4 更新公告

`PUT /api/v1/admin/announcements/{id}`

**请求体：** UpdateAnnouncementReq

### 3.5 发布 / 下架公告

`PUT /api/v1/admin/announcements/{id}/status`

**请求体：** UpdateAnnouncementStatusReq
```json
{
  "status": 1
}
```

**业务规则：** 发布（status=1）与下架（status=2）互斥切换，下架后公告不再对外可见。

### 3.6 删除公告

`DELETE /api/v1/admin/announcements/{id}`

### 3.7 租户侧公告列表

`GET /api/v1/announcements`

**Query 参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20 |

**业务规则：**
- 仅返回当前租户可见的公告：全平台公告（scope=all）+ 定向推送给当前租户的公告（scope=tenant, target_tenant_id=当前租户ID）
- 仅返回已发布状态（status=1）且在有效期内（publish_at <= now <= expire_at，若 expire_at 为空则不限制）
- 无需细粒度权限码，仅需登录

**成功响应（200）：** AnnouncementListResp

---

## 4. 实时推送（WebSocket）

公告模块支持通过 WebSocket 实时推送新公告通知到所有已连接的前端客户端。

### 4.1 WebSocket 连接

**端点：** `GET /api/v1/ws?token={jwt_token}`

**认证方式：** URL 查询参数 `token` 或 `Authorization: Bearer {token}` 请求头

### 4.2 消息类型

| 类型 | 方向 | 说明 |
|------|------|------|
| `ping` | Server → Client | 心跳请求（服务端每30秒发送） |
| `pong` | Client → Server | 心跳响应 |
| `announcement` | Server → Client | 新公告通知 |
| `alert` | Server → Client | 告警通知 |
| `unread_count` | Server → Client | 未读数量更新 |

### 4.3 公告推送消息格式

```json
{
  "type": "announcement",
  "payload": {
    "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "title": "平台维护通知",
    "content": "本周六凌晨进行系统升级..."
  },
  "time": 1724158800
}
```

### 4.4 前端处理流程

```
WebSocket 连接建立
        │
        ▼
  收到 announcement 消息
        │
        ▼
  刷新通知中心公告列表
        │
        ▼
  更新通知铃铛未读角标
        │
        ▼
  用户点击铃铛 → 弹出通知面板
        │
        ▼
  点击公告 → 跳转公告列表页
```

---

## 5. 权限码列表

| 权限码 | 说明 |
|--------|------|
| `admin:announcement:list` | 查看公告列表 |
| `admin:announcement:create` | 创建公告 |
| `admin:announcement:read` | 查看公告详情 |
| `admin:announcement:update` | 更新公告 |
| `admin:announcement:status` | 发布 / 下架公告 |
| `admin:announcement:delete` | 删除公告 |