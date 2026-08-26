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
| manager | 可管理成员和节点 |
| owner | 空间所有者（全部权限） |

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