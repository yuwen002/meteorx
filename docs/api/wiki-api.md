# Wiki 知识库模块接口文档

## 模块概述

Wiki 知识库模块提供多租户隔离的知识管理能力，支持空间（Space）、节点（Node，含文件夹/文档）、文档（Document，含 Markdown）、版本历史（Revision）和成员协作（Member）。

所有 Wiki API 统一前缀：`/api/v1/wiki`

---

## 数据模型

```
WikiSpace (空间)
  └── WikiNode (节点：文件夹 / 文档)
        └── Document (文档内容，Markdown)
              └── DocumentRevision (历史版本)

WikiSpaceMember (空间成员：协作权限)
```

### 层级关系

```
Space
├── Node (Folder)
│   ├── Node (Document) → Document → DocumentRevision[]
│   └── Node (Folder)
│       └── ...
├── Node (Document) → Document → DocumentRevision[]
└── Node (Folder)
```

---

## 1. 空间管理 (Space)

### 1.1 获取空间统计

`GET /api/v1/wiki/stats`

**权限码**：`wiki:wiki_space:list`

**描述**：获取当前租户的 Wiki 统计数据（空间数、文档数、节点数）

**响应**：
```json
{
  "spaces_count": 5,
  "documents_count": 42,
  "nodes_count": 128
}
```

---

### 1.2 列出空间

`GET /api/v1/wiki/spaces`

**权限码**：`wiki:wiki_space:list`

**查询参数**：
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20，最大 100 |
| keyword | string | 否 | 按空间名称搜索 |

**响应**：
```json
{
  "data": [
    {
      "id": "01H...",
      "name": "产品文档",
      "description": "产品相关知识库",
      "tenant_id": "01H...",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 5
  }
}
```

---

### 1.3 创建空间

`POST /api/v1/wiki/spaces`

**权限码**：`wiki:wiki_space:create`

**请求体**：
```json
{
  "name": "产品文档",
  "description": "产品相关知识库"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 空间名称（最多 100 字符） |
| description | string | 否 | 空间描述（最多 500 字符） |

**响应**：
```json
{
  "id": "01H...",
  "name": "产品文档",
  "description": "产品相关知识库",
  "tenant_id": "01H...",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

---

### 1.4 获取空间详情

`GET /api/v1/wiki/spaces/{id}`

**权限码**：`wiki:wiki_space:read`

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 空间 ID (ULID) |

**响应**：同创建响应结构

---

### 1.5 更新空间

`PUT /api/v1/wiki/spaces/{id}`

**权限码**：`wiki:wiki_space:update`

**请求体**：
```json
{
  "name": "新名称",
  "description": "新描述"
}
```

---

### 1.6 删除空间

`DELETE /api/v1/wiki/spaces/{id}`

**权限码**：`wiki:wiki_space:delete`

**描述**：软删除空间（可通过数据库恢复）

---

## 2. 节点管理 (Node)

### 2.1 获取节点树

`GET /api/v1/wiki/spaces/{spaceId}/nodes/tree`

**权限码**：`wiki:wiki_node:list`

**描述**：获取指定空间下的完整节点树（文件夹层级结构）

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| spaceId | string | 空间 ID |

**响应**：
```json
[
  {
    "id": "01H...",
    "space_id": "01H...",
    "parent_id": null,
    "type": "folder",
    "title": "技术文档",
    "sort_order": 1,
    "children": [
      {
        "id": "01H...",
        "space_id": "01H...",
        "parent_id": "01H...",
        "type": "document",
        "title": "API 规范",
        "document_id": "01H...",
        "sort_order": 1,
        "children": []
      }
    ]
  }
]
```

---

### 2.2 创建节点

`POST /api/v1/wiki/spaces/{spaceId}/nodes`

**权限码**：`wiki:wiki_node:create`

**请求体**：
```json
{
  "parent_id": "01H...",
  "type": "folder",
  "title": "技术文档",
  "sort_order": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| parent_id | string | 否 | 父节点 ID，为空则在根级 |
| type | string | 是 | 节点类型：`folder` / `document` |
| title | string | 是 | 节点标题（最多 200 字符） |
| sort_order | int | 否 | 排序值，默认 0 |

---

### 2.3 获取节点详情

`GET /api/v1/wiki/spaces/{spaceId}/nodes/{id}`

**权限码**：`wiki:wiki_node:read`

---

### 2.4 更新节点

`PUT /api/v1/wiki/spaces/{spaceId}/nodes/{id}`

**权限码**：`wiki:wiki_node:update`

**请求体**：
```json
{
  "title": "新标题",
  "sort_order": 2
}
```

---

### 2.5 删除节点

`DELETE /api/v1/wiki/spaces/{spaceId}/nodes/{id}`

**权限码**：`wiki:wiki_node:delete`

**描述**：删除节点。文件夹节点会级联删除子节点；文档节点需先解除文档关联。

---

## 3. 文档管理 (Document)

### 3.1 创建文档

`POST /api/v1/wiki/documents/nodes/{nodeId}`

**权限码**：`wiki:document:create`

**描述**：在指定节点下创建文档（自动创建 DocumentRevision v1）

**请求体**：
```json
{
  "title": "新文档",
  "content": "# 标题\n\n内容",
  "sort_order": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| title | string | 是 | 文档标题 |
| content | string | 否 | Markdown 内容 |
| sort_order | int | 否 | 排序值 |

**响应**：
```json
{
  "id": "01H...",
  "node_id": "01H...",
  "space_id": "01H...",
  "title": "新文档",
  "content": "# 标题\n\n内容",
  "version": 1,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

---

### 3.2 获取文档

`GET /api/v1/wiki/documents/nodes/{nodeId}`

**权限码**：`wiki:document:read`

**描述**：获取指定节点下关联的文档（含最新版本内容）

**响应**：同创建响应结构

---

### 3.3 更新文档

`PUT /api/v1/wiki/documents/{id}`

**权限码**：`wiki:document:update`

**请求体**：
```json
{
  "title": "更新后的标题",
  "content": "# 新内容\n\n更新后的正文"
}
```

**描述**：更新文档内容时自动创建新的 DocumentRevision（版本号递增）

---

### 3.4 删除文档

`DELETE /api/v1/wiki/documents/{id}`

**权限码**：`wiki:document:delete`

**描述**：软删除文档（可通过版本历史恢复）

---

### 3.5 Markdown 实时预览

`POST /api/v1/wiki/documents/preview`

**权限码**：登录用户即可（供编辑器分栏预览，不落库、不做空间级权限校验）

**描述**：将 Markdown 内容离线渲染为 HTML 并做安全净化（移除 script/事件属性/危险协议等）。渲染后的内嵌 `/uploads/*` 图片地址会被改写为带时效签名（默认 30 分钟）的完整 URL，用于文档阅读与预览时的防盗链。

**请求体**：
```json
{
  "content": "# 标题\n\n![架构图](/uploads/xxx.png)\n\n- 列表项",
  "format": "markdown"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| content | string | 是 | Markdown 原文 |
| format | string | 否 | 源格式，默认 `markdown` |

**响应**：
```json
{
  "content_html": "<h1>标题</h1>\n\n<p><img src=\"http://host:8081/uploads/xxx.png?e=1788520110&s=75ec...\" alt=\"架构图\"></p>\n\n<ul>\n<li>列表项</li>\n</ul>\n"
}
```

---

## 4. 版本历史 (Revision)

### 4.1 列出版本历史

`GET /api/v1/wiki/documents/{documentId}/revisions`

**权限码**：`wiki:revision:list`

**查询参数**：
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |

**响应**：
```json
{
  "data": [
    {
      "id": "01H...",
      "document_id": "01H...",
      "version": 3,
      "title": "当前版本",
      "content": "# 最新内容",
      "created_by": "01H...",
      "created_at": "2024-01-01T12:00:00Z"
    },
    {
      "id": "01H...",
      "document_id": "01H...",
      "version": 2,
      "title": "历史版本 2",
      "content": "# 旧内容 v2",
      "created_by": "01H...",
      "created_at": "2024-01-01T10:00:00Z"
    }
  ],
  "pagination": { ... }
}
```

---

### 4.2 获取指定版本

`GET /api/v1/wiki/documents/{documentId}/revisions/{version}`

**权限码**：`wiki:revision:read`

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| documentId | string | 文档 ID |
| version | int | 版本号 |

---

### 4.3 恢复到指定版本

`POST /api/v1/wiki/documents/{documentId}/revisions/{version}/restore`

**权限码**：`wiki:revision:restore`

**描述**：将文档恢复到指定版本内容（自动创建新版本作为恢复快照）

**响应**：
```json
{
  "id": "01H...",
  "version": 4,
  "restored_from": 2,
  "created_at": "2024-01-01T14:00:00Z"
}
```

---

## 5. 成员协作 (Member)

### 5.1 列出空间成员

`GET /api/v1/wiki/spaces/{spaceId}/members`

**权限码**：`wiki:wiki_space_member:list`

**响应**：
```json
{
  "data": [
    {
      "id": "01H...",
      "space_id": "01H...",
      "user_id": "01H...",
      "username": "张三",
      "role": "editor",
      "joined_at": "2024-01-01T00:00:00Z"
    }
  ],
  "pagination": { ... }
}
```

**成员角色**：
| 角色 | 说明 |
|------|------|
| viewer | 仅查看 |
| editor | 可编辑文档 |
| admin | 可管理成员和节点 |
| owner | 空间所有者（全部权限；每空间至少保留一名 owner，且 owner 不可被移除/降级） |

---

### 5.2 添加成员

`POST /api/v1/wiki/spaces/{spaceId}/members`

**权限码**：`wiki:wiki_space_member:create`

**请求体**：
```json
{
  "user_id": "01H...",
  "role": "editor"
}
```

---

### 5.3 移除成员

`DELETE /api/v1/wiki/spaces/{spaceId}/members/{userId}`

**权限码**：`wiki:wiki_space_member:delete`

---

## 6. 回收站管理 (Trash)

### 6.1 列出回收站项目

`GET /api/v1/wiki/trash`

**权限码**：`wiki:trash:list`

**描述**：分页列出当前租户的回收站项目，支持按空间和类型过滤

**查询参数**：
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| space_id | string | 否 | 按空间 ID 过滤 |
| item_type | string | 否 | 按类型过滤：`space` / `node` / `document` |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20 |

**响应**：
```json
{
  "items": [
    {
      "id": "01H...",
      "item_type": "node",
      "item_id": "01H...",
      "space_id": "01H...",
      "title": "已删除的文档",
      "deleted_by": "01H...",
      "deleted_at": "2024-01-01T00:00:00Z",
      "expires_at": "2024-01-31T00:00:00Z"
    }
  ],
  "total": 10,
  "page": 1,
  "page_size": 20
}
```

**说明**：回收站项目 30 天后自动过期删除

---

### 6.2 从回收站恢复

`POST /api/v1/wiki/trash/{id}/restore`

**权限码**：`wiki:trash:restore`

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 回收站项目 ID |

**描述**：将回收站中的项目恢复（根据类型调用不同的恢复逻辑）

---

### 6.3 永久删除回收站项目

`DELETE /api/v1/wiki/trash/{id}`

**权限码**：`wiki:trash:delete`

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 回收站项目 ID |

**描述**：永久删除回收站项目（不可恢复）

---

## 7. 搜索 (Search)

### 7.1 搜索 Wiki

`GET /api/v1/wiki/search`

**权限码**：`wiki:wiki_node:list`

**描述**：按标题和内容搜索 Wiki，自动过滤无权限内容，返回带摘要片段的搜索结果

**查询参数**：
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| q | string | 是 | 搜索关键字 |
| space_id | string | 否 | 限定空间范围 |
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20 |

**响应**：
```json
{
  "results": [
    {
      "id": "01H...",
      "type": "document",
      "title": "API 规范",
      "space_id": "01H...",
      "node_id": "01H...",
      "snippet": "...API 接口设计**规范**包括...",
      "highlight": "规范",
      "updated_at": "2024-01-01T00:00:00Z",
      "score": 0.8
    }
  ],
  "total": 25,
  "page": 1,
  "page_size": 20,
  "query": "规范"
}
```

**说明**：标题搜索结果权重高于内容搜索（Score 1.0 vs 0.8），摘要围绕关键字位置生成

---

## 8. 节点权限管理 (Node Permission)

### 8.1 设置节点权限

`POST /api/v1/wiki/spaces/{spaceId}/nodes/{id}/permissions`

**权限码**：`wiki:wiki_node:update`

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| spaceId | string | 空间 ID |
| id | string | 节点 ID |

**请求体**：
```json
{
  "user_id": "01H...",
  "permission": "edit"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| user_id | string | 是 | 目标用户 ID |
| permission | string | 是 | 权限类型：`view` / `edit` / `delete` |

**权限类型说明**：
| 权限 | 能力 |
|------|------|
| view | 仅查看 |
| edit | 查看 + 编辑 |
| delete | 查看 + 编辑 + 删除 |

**响应**：
```json
{
  "id": "01H...",
  "node_id": "01H...",
  "user_id": "01H...",
  "permission": "edit",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**说明**：节点权限是 Space 角色权限的补充，可为特定用户在特定节点授予额外权限

---

### 8.2 获取节点权限列表

`GET /api/v1/wiki/spaces/{spaceId}/nodes/{id}/permissions`

**权限码**：`wiki:wiki_node:read`

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| spaceId | string | 空间 ID |
| id | string | 节点 ID |

**响应**：返回该节点的所有权限配置列表

---

### 8.3 移除节点权限

`DELETE /api/v1/wiki/spaces/{spaceId}/nodes/{id}/permissions/{userId}/{permission}`

**权限码**：`wiki:wiki_node:update`

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| spaceId | string | 空间 ID |
| id | string | 节点 ID |
| userId | string | 用户 ID |
| permission | string | 权限类型 |

---

## 9. 附件管理 (Attachment)

### 9.1 创建附件

`POST /api/v1/wiki/documents/attachments`

**权限码**：`wiki:document:update`

**请求体**：
```json
{
  "document_id": "01H...",
  "file_name": "设计稿.pdf",
  "file_size": 2048576,
  "mime_type": "application/pdf",
  "file_url": "/uploads/tenant1/2024/01/01/xxx.pdf"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| document_id | string | 是 | 关联的文档 ID |
| file_name | string | 是 | 文件名 |
| file_size | int64 | 否 | 文件大小（字节） |
| mime_type | string | 否 | MIME 类型 |
| file_url | string | 是 | 文件访问 URL |

**响应**：
```json
{
  "id": "01H...",
  "document_id": "01H...",
  "file_name": "设计稿.pdf",
  "file_size": 2048576,
  "mime_type": "application/pdf",
  "file_url": "/uploads/tenant1/2024/01/01/xxx.pdf",
  "uploaded_by": "01H...",
  "created_at": "2024-01-01T00:00:00Z"
}
```

---

### 9.2 列出文档附件

`GET /api/v1/wiki/documents/{documentId}/attachments`

**权限码**：`wiki:document:read`

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| documentId | string | 文档 ID |

**响应**：返回文档的所有附件列表

---

### 9.3 删除附件

`DELETE /api/v1/wiki/documents/attachments/{id}`

**权限码**：`wiki:document:update`

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 附件 ID |

---

## 10. 节点移动 (Move)

### 10.1 移动节点

`POST /api/v1/wiki/spaces/{spaceId}/nodes/{id}/move`

**权限码**：`wiki:wiki_node:update`

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| spaceId | string | 空间 ID |
| id | string | 要移动的节点 ID |

**请求体**：
```json
{
  "new_parent_id": "01H..."
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| new_parent_id | string | 是 | 新父节点 ID（为空表示移动到根级） |

**安全校验**：
- 不能移动到自身
- 不能跨 Space 移动
- 目标父节点必须是文件夹类型
- 不能移动到自己的子节点下（防止循环引用）

---

## 11. 节点排序 (Sort)

### 11.1 设置节点排序

`PUT /api/v1/wiki/spaces/{spaceId}/nodes/{id}/sort`

**权限码**：`wiki:wiki_node:update`

**路径参数**：
| 参数 | 类型 | 说明 |
|------|------|------|
| spaceId | string | 空间 ID |
| id | string | 节点 ID |

**请求体**：
```json
{
  "sort": 10
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| sort | int | 是 | 排序值（数值越小越靠前） |

**说明**：节点树按 Sort 字段递归排序，同级节点按 Sort 值升序排列

---

## 12. 扩展功能（标签 / 评论 / 分享 / 模板 / 统计 / 订阅 / 通知 / 编辑锁 / 批量 / 对比 / 导入导出）

> 扩展功能独立注册于 `internal/modules/wiki/handler/wiki_handler_extended.go`。所有端点均在 `/api/v1/wiki` 前缀之下；
> 权限复用空间/节点/文档的 action 级校验（read / update / delete 等），**未新增权限码**。

### 12.1 空间级标签

作用于当前租户内（端点不携带 spaceId）：

| 方法 | 路径 | 说明 | 校验动作 |
|------|------|------|----------|
| POST | `/api/v1/wiki/spaces/tags` | 创建标签，body：`{name, color}` | 租户内空间权限 |
| GET | `/api/v1/wiki/spaces/tags` | 标签列表 | - |
| DELETE | `/api/v1/wiki/spaces/tags/{id}` | 删除标签 | 文档 update 权限 |

### 12.2 文档标签绑定

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/wiki/documents/{id}/tags/{tagId}` | 为文档绑定标签（需文档 update 权限） |
| DELETE | `/api/v1/wiki/documents/{id}/tags/{tagId}` | 移除标签（需文档 update 权限） |
| GET | `/api/v1/wiki/documents/{id}/tags` | 文档标签列表（含标签详情） |

### 12.3 评论系统

支持多级回复与 @提及：

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/wiki/documents/{id}/comments` | 创建评论；body：`{document_id, parent_id?, content, mention_ids?}`；`node_id` 可省略，服务端按文档自动解析 |
| GET | `/api/v1/wiki/documents/{id}/comments` | 评论树（父评论含 `replies`） |
| PUT | `/api/v1/wiki/documents/comments/{id}` | 更新评论，body：`{content}`；仅作者可改 |
| DELETE | `/api/v1/wiki/documents/comments/{id}` | 删除评论；作者本人或具备节点 delete 权限者 |

- `mention_ids` 逗号分隔的用户 ID，被提及者会收到类型为 `mention` 的通知。
- 权限：查看/发表评论要求对应节点 `read` 权限。

### 12.4 分享链接

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/wiki/documents/{id}/share` | 创建分享；body：`{document_id, password?, expire_at?, max_views?, allow_download?}` |
| GET | `/api/v1/wiki/documents/{id}/shares` | 文档分享列表 |
| DELETE | `/api/v1/wiki/documents/shares/{id}` | 删除分享（`id` 为分享记录主键，需文档 update 权限） |
| GET | `/api/v1/wiki/share/{token}?password=` | 公开访问分享文档（校验 token、过期时间、最大次数、密码；成功后自增浏览计数） |

分享地址格式：`/wiki/share/{token}`。访问/创建分享都会写入访问日志（action=`share`）。

### 12.5 文档模板

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/wiki/spaces/templates` | 创建模板，body：`{name, description?, format?, category?, content, is_public?}` |
| GET | `/api/v1/wiki/spaces/templates?category=` | 模板列表（含当前租户 + 公开模板） |
| GET | `/api/v1/wiki/spaces/templates/{id}` | 模板详情 |
| PUT | `/api/v1/wiki/spaces/templates/{id}` | 更新模板 |
| DELETE | `/api/v1/wiki/spaces/templates/{id}` | 删除模板 |

### 12.6 访问统计与日志

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/wiki/documents/{id}/stats` | 文档统计：`total_views / total_edits / total_downloads / total_shares / unique_viewers / last_viewed_at` |
| GET | `/api/v1/wiki/documents/{id}/access-logs?page=&page_size=` | 访问日志（分页响应 `{ data, pagination }`） |

### 12.7 订阅与通知

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/wiki/documents/{id}/subscribe` | 订阅文档（`?notify_type=all` 默认） |
| DELETE | `/api/v1/wiki/documents/{id}/subscribe` | 取消订阅 |
| GET | `/api/v1/wiki/spaces/subscriptions` | 当前用户在空间内的订阅列表 |
| GET | `/api/v1/wiki/spaces/notifications?page=&page_size=` | 通知列表（分页） |
| PUT | `/api/v1/wiki/spaces/notifications/{id}/read` | 标记单条已读 |
| PUT | `/api/v1/wiki/spaces/notifications/read-all` | 全部标记已读 |
| GET | `/api/v1/wiki/spaces/notifications/unread-count` | 未读数：`{count}` |

### 12.8 编辑锁

防止多人同时编辑冲突：

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/wiki/documents/{id}/edit-lock` | 获取编辑锁（被占用时返回冲突信息） |
| PUT | `/api/v1/wiki/documents/{id}/edit-lock` | 刷新编辑锁（延长持有时间） |
| DELETE | `/api/v1/wiki/documents/{id}/edit-lock` | 释放编辑锁 |
| GET | `/api/v1/wiki/documents/{id}/edit-lock` | 查询锁状态（`{document_id, user_id, user_name, locked_at, expires_at, can_edit}`） |

锁默认 30 分钟自动过期，持锁期间可刷新。

### 12.9 批量操作

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/wiki/spaces/nodes/batch` | 批量操作；body：`{action: "delete" \| "move", node_ids: string[], target?: string}` |

- `action=move` 时 `target` 为目标父节点 ID（移动到空间根则传空）；
- 批量操作在事务中执行，逐一校验节点归属与权限。

### 12.10 版本对比

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/wiki/documents/{id}/revisions/compare?version1=&version2=` | 版本行级对比；返回 `{old_version, new_version, diffs[]}`，其中 `diffs` 每项含 `type(added/removed/unchanged)`、`line_num`、`content`、`old_line/new_line` |

### 12.11 导入导出

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/wiki/documents/{id}/export` | 导出；body：`{format: "markdown" \| "pdf" \| "html"}`；响应直接回写文件流（`Content-Disposition: attachment`） |
| POST | `/api/v1/wiki/documents/{id}/import` | 导入；`multipart/form-data` 字段：`file`、`format`（默认 markdown）；将文件内容作为**新修订**写入目标文档（复用乐观锁与版本历史），返回 `{document_id, node_id, filename, size, format}` |

### 12.12 扩展数据模型

| 模型 | 主要字段 |
|------|----------|
| `Tag` | id, tenant_id, name, color, created_by, created_at |
| `DocumentTag` | id, document_id, tag_id, created_at（多对多关联） |
| `Comment` | id, tenant_id, document_id, node_id, parent_id, content, created_by, mention_ids, status, created_at, updated_at |
| `ShareLink` | id, tenant_id, document_id, node_id, token, password, expire_at, max_views, view_count, allow_download, created_by, created_at |
| `DocumentTemplate` | id, tenant_id, name, description, content, format, category, is_public, created_by, created_at, updated_at |
| `DocumentAccessLog` | id, tenant_id, document_id, node_id, user_id, action, ip_address, user_agent, created_at |
| `DocumentSubscription` | id, tenant_id, document_id, node_id, user_id, notify_type, created_at |
| `Notification` | id, tenant_id, user_id, type, title, content, related_id, related_type, is_read, created_at |
| `EditLock` | id, tenant_id, document_id, user_id, locked_at, expires_at |

---

## 权限码汇总

| 权限码 | 说明 | 推导自 |
|--------|------|--------|
| `wiki:wiki_space:list` | 列出空间 | `GET /wiki/spaces` |
| `wiki:wiki_space:create` | 创建空间 | `POST /wiki/spaces` |
| `wiki:wiki_space:read` | 查看空间 | `GET /wiki/spaces/{id}` |
| `wiki:wiki_space:update` | 更新空间 | `PUT /wiki/spaces/{id}` |
| `wiki:wiki_space:delete` | 删除空间 | `DELETE /wiki/spaces/{id}` |
| `wiki:wiki_node:list` | 列出节点 | `GET /wiki/spaces/{spaceId}/nodes/tree` |
| `wiki:wiki_node:create` | 创建节点 | `POST /wiki/spaces/{spaceId}/nodes` |
| `wiki:wiki_node:read` | 查看节点 | `GET /wiki/spaces/{spaceId}/nodes/{id}` |
| `wiki:wiki_node:update` | 更新节点 | `PUT /wiki/spaces/{spaceId}/nodes/{id}` |
| `wiki:wiki_node:delete` | 删除节点 | `DELETE /wiki/spaces/{spaceId}/nodes/{id}` |
| `wiki:wiki_node:move` | 移动节点 | `POST /wiki/spaces/{spaceId}/nodes/{id}/move` |
| `wiki:wiki_node:sort` | 节点排序 | `PUT /wiki/spaces/{spaceId}/nodes/{id}/sort` |
| `wiki:document:create` | 创建文档 | `POST /wiki/documents/nodes/{nodeId}` |
| `wiki:document:read` | 查看文档 | `GET /wiki/documents/nodes/{nodeId}` |
| `wiki:document:update` | 更新文档 | `PUT /wiki/documents/{id}` |
| `wiki:document:delete` | 删除文档 | `DELETE /wiki/documents/{id}` |
| `wiki:revision:list` | 列出版本 | `GET /wiki/documents/{id}/revisions` |
| `wiki:revision:read` | 查看版本 | `GET /wiki/documents/{id}/revisions/{version}` |
| `wiki:revision:restore` | 恢复版本 | `POST /wiki/documents/{id}/revisions/{version}/restore` |
| `wiki:wiki_space_member:list` | 列出成员 | `GET /wiki/spaces/{spaceId}/members` |
| `wiki:wiki_space_member:create` | 添加成员 | `POST /wiki/spaces/{spaceId}/members` |
| `wiki:wiki_space_member:delete` | 移除成员 | `DELETE /wiki/spaces/{spaceId}/members/{userId}` |
| `wiki:wiki_space:stats` | 获取统计 | `GET /wiki/stats` |
| `wiki:trash:list` | 回收站列表 | `GET /wiki/trash` |
| `wiki:trash:restore` | 恢复回收站项目 | `POST /wiki/trash/{id}/restore` |
| `wiki:trash:delete` | 永久删除回收站项目 | `DELETE /wiki/trash/{id}` |
| `wiki:search` | Wiki 搜索 | `GET /wiki/search` |
| `wiki:attachment:create` | 创建附件 | `POST /wiki/documents/attachments` |
| `wiki:attachment:list` | 列出附件 | `GET /wiki/documents/{id}/attachments` |
| `wiki:attachment:delete` | 删除附件 | `DELETE /wiki/documents/attachments/{id}` |

---

## 错误码

| 错误码 | HTTP 状态 | 说明 |
|--------|-----------|------|
| `WIKI_SPACE_NOT_FOUND` | 404 | 空间不存在 |
| `WIKI_NODE_NOT_FOUND` | 404 | 节点不存在 |
| `WIKI_DOCUMENT_NOT_FOUND` | 404 | 文档不存在 |
| `WIKI_REVISION_NOT_FOUND` | 404 | 版本不存在 |
| `WIKI_SPACE_ALREADY_EXISTS` | 409 | 空间名称冲突 |
| `WIKI_NODE_TYPE_INVALID` | 400 | 节点类型无效 |
| `PERMISSION_DENIED` | 403 | 权限不足 |
| `INVALID_PARAM` | 400 | 请求参数错误 |

---

## 租户隔离

所有 Wiki API 自动通过 `tenantctx.FilterQuery()` 进行租户隔离：

- 每个空间、节点、文档、版本、成员均绑定 `tenant_id`
- 普通用户只能访问本租户下的数据
- 系统管理员（`IsMaster=true`）可跨租户查看

---

## 审计事件

| Action 命名 | 触发场景 |
|-------------|----------|
| `WIKI_SPACE_CREATE` | 创建空间 |
| `WIKI_SPACE_UPDATE` | 更新空间 |
| `WIKI_SPACE_DELETE` | 删除空间 |
| `WIKI_NODE_CREATE` | 创建节点 |
| `WIKI_NODE_UPDATE` | 更新节点 |
| `WIKI_NODE_DELETE` | 删除节点 |
| `WIKI_NODE_MOVE` | 移动节点 |
| `WIKI_NODE_SORT` | 节点排序 |
| `WIKI_DOCUMENT_CREATE` | 创建文档 |
| `WIKI_DOCUMENT_UPDATE` | 更新文档 |
| `WIKI_DOCUMENT_DELETE` | 删除文档 |
| `WIKI_REVISION_RESTORE` | 恢复版本 |
| `WIKI_MEMBER_ADD` | 添加成员 |
| `WIKI_MEMBER_REMOVE` | 移除成员 |
| `WIKI_TRASH_RESTORE` | 从回收站恢复 |
| `WIKI_TRASH_PERMANENT_DELETE` | 永久删除回收站项目 |
| `WIKI_PERMISSION_SET` | 设置节点权限 |
| `WIKI_PERMISSION_REMOVE` | 移除节点权限 |
| `WIKI_ATTACHMENT_CREATE` | 创建附件 |
| `WIKI_ATTACHMENT_DELETE` | 删除附件 |