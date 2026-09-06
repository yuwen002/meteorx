# 文件管理模块 API 接口说明

> Base URL: `/api/v1`  
> 认证方式：所有接口需携带 `Authorization: Bearer <token>`  
> 所属模块：`internal/modules/file`  
> 路由注册：`internal/modules/file/routes.go`  
> 配置文件：`internal/config/config.yaml` → `file` 节点

---

## 1. 接口总览

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| POST | `/files/upload` | 上传文件 | `file:upload` |
| GET | `/files` | 租户文件列表 | `file:list` |
| GET | `/files/my` | 当前用户文件列表 | `file:list` |
| GET | `/files/{id}` | 文件详情 | `file:read` |
| GET | `/files/{id}/download` | 文件下载 | `file:download` |
| PUT | `/files/{id}` | 更新文件信息（重命名） | `file:update` |
| DELETE | `/files/{id}` | 删除文件（软删除） | `file:delete` |
| POST | `/files/batch/delete` | 批量删除文件 | `file:batch_delete` |
| GET | `/files/deleted` | 回收站列表 | `file:list_deleted` |
| PUT | `/files/{id}/restore` | 恢复已删除文件 | `file:restore` |
| DELETE | `/files/{id}/permanent` | 永久删除文件（物理删除） | `file:permanent_delete` |

---

## 2. 数据结构

### 2.1 File（文件实体）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string (ULID) | 文件唯一标识 |
| tenant_id | string | 租户 ID（租户隔离） |
| user_id | string | 上传用户 ID |
| file_name | string | 存储文件名（不含路径） |
| original_name | string | 原始文件名（用户上传时的名称） |
| file_path | string | 存储相对路径 |
| file_size | int64 | 文件大小（字节） |
| mime_type | string | MIME 类型 |
| file_type | string | 文件分类：`image` / `document` / `video` / `audio` / `other` |
| storage_type | string | 存储类型：`local` / `oss` / `s3`（预留） |
| md5 | string | 文件 MD5 哈希（用于去重） |
| status | int | 状态：`1` 正常，`0` 已删除 |
| created_at | string (RFC3339) | 创建时间 |
| updated_at | string (RFC3339) | 更新时间 |
| deleted_at | string (RFC3339) | 软删除时间（仅回收站有值） |

### 2.2 UploadFileResp（上传响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 文件 ID |
| file_name | string | 存储文件名 |
| original_name | string | 原始文件名 |
| file_size | int64 | 文件大小（字节） |
| mime_type | string | MIME 类型 |
| file_type | string | 文件分类 |
| url | string | 文件访问 URL |

### 2.3 FileListReq（列表查询参数）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页条数，默认 10，最大 100 |
| file_type | string | 否 | 按文件类型筛选 |
| keyword | string | 否 | 按文件名关键词搜索 |

### 2.4 FileUpdateReq（更新请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file_name | string | 是 | 新文件名，最大 255 字符 |

### 2.5 BatchDeleteReq（批量删除请求）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | []string | 是 | 文件 ID 列表，至少 1 个 |

### 2.6 BatchDeleteResp（批量删除响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| success_count | int | 成功删除数量 |
| failed_count | int | 失败数量 |
| failed_ids | []string | 失败的文件 ID 列表 |

### 2.7 分页响应结构

```json
{
  "data": [ /* File 数组 */ ],
  "pagination": {
    "page": 1,
    "page_size": 10,
    "total": 100
  }
}
```

---

## 3. 接口详细说明

### 3.1 上传文件

`POST /api/v1/files/upload`

**请求格式：** `multipart/form-data`

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | File | 是 | 上传的文件 |

**业务规则：**
- 大小限制：由 `config.yaml` → `file.max_file_size` 决定（默认 10MB）
- 类型限制：由 `config.yaml` → `file.allowed_types` 决定
- 自动去重：相同租户内 MD5 相同的文件直接返回已有记录，不重复上传
- 自动检测文件类型：`image` / `document` / `video` / `audio` / `other`

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "file_name": "01ARZ3NDEKTSV4RRFFQ69G5FAV.pdf",
    "original_name": "产品说明书.pdf",
    "file_size": 2048576,
    "mime_type": "application/pdf",
    "file_type": "document",
    "url": "/uploads/2024/01/01/01ARZ3NDEKTSV4RRFFQ69G5FAV.pdf"
  }
}
```

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 400 | 文件过大 / 不支持的类型 / 未找到文件 |
| 401 | 未登录或 Token 无效 |
| 403 | 无 `file:upload` 权限 |
| 500 | 存储失败 / 数据库写入失败 |

---

### 3.2 获取租户文件列表

`GET /api/v1/files`

**请求参数（Query）：** 参见 2.3 FileListReq

**成功响应（200）：** 分页响应结构，data 为 File 数组

---

### 3.3 获取当前用户文件列表

`GET /api/v1/files/my`

与 3.2 相同，但仅返回当前登录用户上传的文件

---

### 3.4 获取文件详情

`GET /api/v1/files/{id}`

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 文件 ID (ULID) |

**成功响应（200）：** File 对象

**错误响应：**
| 状态码 | 场景 |
|--------|------|
| 401 | 未授权 |
| 403 | 无 `file:read` 权限或文件不属于当前租户 |
| 404 | 文件不存在 |

---

### 3.5 文件下载

`GET /api/v1/files/{id}/download`

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| id | string | 文件 ID |

**成功响应：** 二进制文件流，Header 包含：
```
Content-Disposition: attachment; filename="original_name.pdf"
Content-Type: application/octet-stream
```

---

### 3.6 更新文件信息（重命名）

`PUT /api/v1/files/{id}`

**请求体：**
```json
{
  "file_name": "新文件名.pdf"
}
```

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

---

### 3.7 删除文件（软删除）

`DELETE /api/v1/files/{id}`

软删除：数据库记录标记 `deleted_at`，物理文件同步删除。可从回收站恢复。

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

---

### 3.8 批量删除文件

`POST /api/v1/files/batch/delete`

**请求体：**
```json
{
  "ids": ["id1", "id2", "id3"]
}
```

**成功响应（200）：**
```json
{
  "code": 200,
  "data": {
    "success_count": 2,
    "failed_count": 1,
    "failed_ids": ["id3"]
  }
}
```

---

### 3.9 回收站列表

`GET /api/v1/files/deleted`

**请求参数（Query）：** 参见 2.3 FileListReq

**说明：** 返回当前租户已软删除的文件，`deleted_at` 字段有值

---

### 3.10 恢复已删除文件

`PUT /api/v1/files/{id}/restore`

将回收站中的文件恢复为正常状态。

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

---

### 3.11 永久删除文件

`DELETE /api/v1/files/{id}/permanent`

**说明：** 物理删除数据库记录和物理文件，不可恢复。仅用于回收站中的文件。

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
| 400 | 文件 ID 为空 |
| 404 | 文件不存在或不属于当前租户 |

---

## 4. 权限码列表

| 权限码 | 说明 | 对应接口 |
|--------|------|----------|
| `file:list` | 查询文件列表 | GET `/files`, GET `/files/my` |
| `file:upload` | 上传文件 | POST `/files/upload` |
| `file:read` | 查看文件详情 | GET `/files/{id}` |
| `file:download` | 下载文件 | GET `/files/{id}/download` |
| `file:update` | 更新文件信息 | PUT `/files/{id}` |
| `file:delete` | 删除文件（软删除） | DELETE `/files/{id}` |
| `file:batch_delete` | 批量删除文件 | POST `/files/batch/delete` |
| `file:list_deleted` | 查看回收站 | GET `/files/deleted` |
| `file:restore` | 恢复已删除文件 | PUT `/files/{id}/restore` |
| `file:permanent_delete` | 永久删除文件 | DELETE `/files/{id}/permanent` |

**权限分配：** superadmin 角色自动获取全部权限（RBAC seed 机制）

---

## 5. 租户隔离机制

所有文件操作均在 Service 层校验 `tenant_id`：

- **上传时**：`tenant_id` 从 JWT Token 解析，自动注入文件记录
- **查询时**：仅返回当前租户的文件
- **详情/下载/更新/删除时**：校验文件 `tenant_id` 是否与当前用户租户一致
- 若跨租户访问，返回 `404 文件不存在`（不暴露文件存在信息，防止 IDOR）

---

## 6. 文件存储说明

### 6.1 本地存储（当前实现）

- 存储路径：`config.yaml` → `file.upload_path`（默认 `./uploads`）
- 访问 URL：`/uploads/*`（`router.go` 中 `middleware.SignedUploadsHandler` 提供）
  - **签名保护**：接口返回的 `url` 字段已附带短时效 HMAC 签名（默认 30 分钟，密钥派生自 `jwt.secret`）；
    `/uploads` 静态服务会校验签名与过期时间，并拒绝目录列举、子路径与路径穿越请求
  - 推荐走 `GET /files/{id}/download`（携带 Bearer Token，无有效期问题）获取文件内容
- 文件组织结构：上传文件名为 `{ulid}.{ext}`（平铺于 `upload_path` 单层目录）

### 6.2 云存储（预留）

通过 `Storage` 接口可扩展实现：

```go
type Storage interface {
    Upload(ctx context.Context, reader io.Reader, originalName string) (string, error)
    Download(ctx context.Context, path string) (io.ReadCloser, error)
    Delete(ctx context.Context, path string) error
    Exists(ctx context.Context, path string) (bool, error)
    GetURL(path string) string
    PresignURL(ctx context.Context, path string, expiration int64) (string, error)
    Type() string
}
```

`NewStorage()` 工厂函数根据 `config.yaml → file.storage_type` 创建对应实现，支持 `local` / `oss` / `s3`。

---

## 7. 配置说明（config.yaml）

```yaml
file:
  upload_path: "./uploads"           # 文件存储根目录
  upload_url: "/uploads"             # 访问 URL 前缀
  max_file_size: 10485760           # 最大文件大小（字节），10MB
  allowed_types:                     # 允许的 MIME 类型（空则使用默认白名单）
    - "image/jpeg"
    - "image/png"
    - "image/gif"
    - "image/webp"
    - "application/pdf"
    - "application/msword"
    - "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
    - "application/vnd.ms-excel"
    - "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
    - "text/plain"
  storage_type: "local"              # 存储类型：local / oss / s3
  cloud:                             # 云存储配置（storage_type 为 oss/s3 时使用）
    endpoint: ""
    access_key: ""
    secret_key: ""
    bucket: ""
    region: ""
```

---

## 8. 前端对接说明

### 8.1 API 文件

路径：`web-admin/src/api/modules/file.ts`

导出接口列表：

| 函数 | 返回类型 | 说明 |
|------|----------|------|
| `getFileList(params)` | `PaginatedResult<FileItem>` | 租户文件列表 |
| `getMyFileList(params)` | `PaginatedResult<FileItem>` | 当前用户文件列表 |
| `getFileDetail(id)` | `FileItem` | 文件详情 |
| `uploadFile(file)` | `UploadFileResp` | 上传文件 |
| `updateFile(id, data)` | `null` | 更新文件信息 |
| `deleteFile(id)` | `null` | 删除文件 |
| `batchDeleteFile(data)` | `BatchDeleteResp` | 批量删除 |
| `getDeletedFileList(params)` | `PaginatedResult<FileItem>` | 回收站列表 |
| `restoreFile(id)` | `null` | 恢复文件 |
| `permanentDeleteFile(id)` | `null` | 永久删除 |
| `downloadFile(id)` | `Blob` | 下载文件 |

### 8.2 路由

路径：`web-admin/src/router/index.ts`

```typescript
{
  path: 'system/file',
  name: 'File',
  component: () => import('@/views/system/file/index.vue'),
  meta: { title: '文件管理', icon: 'Folder', permission: 'file:list' }
}
```

### 8.3 侧边栏菜单

路径：`web-admin/src/layouts/DefaultLayout.vue`

```html
<el-menu-item 
  index="/system/file" 
  v-if="userStore.hasPermission('file:list') || userStore.isAdmin">
  <el-icon><Folder /></el-icon>
  <template #title>文件管理</template>
</el-menu-item>
```