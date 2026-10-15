# 数据导出模块 API 接口说明

> Base URL: `/api/v1`  
> 所属模块：`internal/modules/export`  
> 路由注册：`internal/modules/export/routes.go`  
> Handler：`internal/modules/export/handler/export_handler.go`  
> DTO：`internal/modules/export/dto/export_dto.go`  
> 前端页面：`web-admin/src/views/export/index.vue`（下载中心，路由 `/export`）

异步导出：创建任务后立即返回，worker 池后台执行，完成后到下载中心轮询状态并下载文件。支持 **CSV / Excel(xlsx) / JSON / PDF** 四种格式（由创建请求中的 `format` 字段决定，缺省为 CSV）。

---

## 1. 接口总览

| 方法 | 路径 | 功能 | 权限码（自动推导） |
|------|------|------|------|
| GET | `/exports` | 导出任务列表（当前用户） | `export:list` |
| POST | `/exports` | 创建导出任务 | `export:create` |
| GET | `/exports/{id}` | 导出任务详情 | `export:read` |
| GET | `/exports/{id}/download` | 下载导出文件（格式由任务决定） | `export:list`（下载动作复用列表权限） |
| DELETE | `/exports/{id}` | 删除导出任务及文件 | `export:delete` |

**说明：**
- 全部接口需登录，权限码由 `AutoRequirePermission` 中间件按 Method + 路径自动推导；超级管理员直接放行。
- 任务归属校验：普通用户仅能访问 **本租户 + 本人** 创建的任务；平台管理员（is_master）可跨归属访问。

---

## 2. 数据结构

### 2.1 CreateExportReq（创建导出任务请求）

| 字段 | 类型 | 必填 | 校验规则 | 说明 |
|------|------|------|----------|------|
| resource | string | 是 | oneof | 资源类型：`user` / `task` / `audit_log` / `wiki_document` |
| format | string | 否 | oneof | 导出格式：`csv` / `xlsx` / `json` / `pdf`；不传默认 `csv` |
| filters | object | 否 | 最多 32 个键 | 筛选条件（自由键值对，按资源类型解释，见 4.1） |

### 2.2 ExportJobResp（导出任务响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 任务 ID（ULID） |
| tenant_id | string | 所属租户 ID |
| user_id | string | 创建人 ID |
| resource | string | 资源类型 |
| format | string | 导出格式：`csv` / `xlsx` / `json` / `pdf`（空表示 csv） |
| status | string | 状态：`pending` / `running` / `completed` / `failed` |
| rows | int64 | 导出行数（完成后有值） |
| error_msg | string | 失败原因（仅 failed 时返回） |
| created_at | string | 创建时间（RFC3339） |
| started_at | string | 开始执行时间（可选，进入 running 后返回） |
| completed_at | string | 完成时间（可选，终态后返回） |
| expires_at | string | 文件过期时间（创建时间 + 保留天数） |

### 2.3 任务状态机

```
pending ──worker 拉取──▶ running ──▶ completed
                          │
                          └────────▶ failed（记录 error_msg）
```

---

## 3. 接口详细说明

### 3.1 创建导出任务

`POST /api/v1/exports`

**请求体示例：**
```json
{
  "resource": "task",
  "format": "xlsx",
  "filters": { "status": "done" }
}
```

**成功响应（200）：** ExportJobResp（`status=pending`）

**业务规则：**
- `resource` 不在支持列表内返回 400（`不支持的资源类型`）
- 队列上限准入：本租户未完结（pending/running）任务数 ≥ `export.max_active_jobs`（默认 10）时返回 429；配额计数查询异常时不阻断创建（fail-open）
- 任务落库即返回，不阻塞请求；由 worker 池异步执行
- `expires_at` = 当前时间 + `export.retention_days`（默认 7 天）

### 3.2 导出任务列表

`GET /api/v1/exports`

**Query 参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页条数，默认 20，最大 100 |
| status | string | 按状态筛选：pending/running/completed/failed |
| resource | string | 按资源类型筛选 |

**成功响应（200）：** 分页响应，data 为 ExportJobResp 数组（仅当前用户创建的任务）

### 3.3 导出任务详情

`GET /api/v1/exports/{id}`

**成功响应（200）：** ExportJobResp；前端下载中心按此接口轮询任务状态。

### 3.4 下载导出文件

`GET /api/v1/exports/{id}/download`

**成功响应（200）：** 导出文件流，`Content-Type` 与文件名后缀随任务 `format` 变化：

| format | Content-Type | 文件名后缀 |
|--------|-------------|------------|
| csv（默认） | `text/csv; charset=utf-8` | `.csv`（文件内含 UTF-8 BOM，Excel 直接打开不乱码） |
| xlsx | `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` | `.xlsx` |
| json | `application/json; charset=utf-8` | `.json` |
| pdf | `application/pdf` | `.pdf` |

**错误响应：**

| 状态码 | 场景 |
|--------|------|
| 404 | 任务不存在，或导出文件已过期/被删除 |
| 409 | 任务尚未完成（status ≠ completed），提示「导出任务尚未完成」 |

### 3.5 删除导出任务

`DELETE /api/v1/exports/{id}`

删除任务记录并同步删除磁盘上的 CSV 文件，成功返回 `data=null`。

---

## 4. 附录

### 4.1 各资源类型支持的 filters

| resource | 支持的筛选键 |
|----------|--------------|
| `user` | `status` |
| `task` | `status`、`priority` |
| `audit_log` | `module`、`action`、`result`、`riskLevel`、`startTime`、`endTime` |
| `wiki_document` | `publishStatus` |

未识别的键会被忽略；时间类筛选值格式为 `2006-01-02 15:04:05`。

### 4.2 错误码映射

| 状态码 | 场景 |
|--------|------|
| 400 | 不支持的资源类型 |
| 403 | 无权访问该导出任务（跨归属访问） |
| 404 | 导出任务不存在 |
| 409 | 任务未完成时请求下载 |
| 429 | 导出队列已满：本租户未完结（pending/running）任务数达到上限 `export.max_active_jobs`（默认 10） |

### 4.3 配置项（`internal/config/config.yaml`）

```yaml
export:
  dir: "./data/exports"      # 导出 CSV 文件存储目录
  retention_days: 7          # 文件保留天数，过期由清理任务删除
  worker_count: 3            # 异步导出 worker 池大小
  batch_size: 1000           # CSV 流式分批写出阈值（行）
  max_active_jobs: 10        # 单租户未完结任务上限，达到后新建返回 429（留空/零值回落默认 10）
```

留空或删除配置段即回落代码默认值。

### 4.4 后台任务

| 任务 | 周期 | 职责 |
|------|------|------|
| Export Worker 池 | 每 2 秒轮询 | 拉取 pending 任务，按资源类型分派 exporter 流式写出文件（格式由任务决定） |
| ExportExpiryJob | 每 6 小时 | 删除超过 `expires_at` 的导出文件与任务记录，单轮上限 200 条 |

### 4.5 格式支持与依赖

- **CSV**：零依赖，文件内写入 UTF-8 BOM，Excel / Numbers / WPS 直接打开中文不乱码。
- **XLSX**：基于 `github.com/xuri/excelize/v2` 的 `StreamWriter` 流式写出，内存占用低，适合大数据量。
- **JSON**：以 `[{表头键: 值}, ...]` 数组形式输出，字段名为各资源的导出列名。
- **PDF**：基于 `github.com/jung-kurt/gofpdf` 渲染带边框的表格。**因导出数据含中文，PDF 必须内嵌/可定位 CJK 字体，否则中文会乱码**。服务端按以下顺序解析字体：
  1. 环境变量 `METEORX_EXPORT_PDF_FONT`（指向一个 `.ttf` 字体文件，可选 `METEORX_EXPORT_PDF_FONT_NAME` 指定字体族名）；
  2. 启动配置中通过 `exporter.SetPDFFont(path, name)` 注入的路径；
  3. 常见系统路径（Windows `msyh.ttf`、Linux Noto/WQY、macOS 等）。
  若三者皆不可得，创建 PDF 导出任务后 worker 会将该任务标记为 `failed` 并记录「PDF 导出需要 CJK 字体」，**绝不输出中文乱码**。注意：`.ttc` 字体集合不被 gofpdf 的 `AddUTF8Font` 支持，请使用单一 `.ttf`。
