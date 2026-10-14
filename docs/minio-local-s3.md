# 本地用 MinIO 联调云存储（S3 驱动）

> 目的：在**没有真实阿里云 OSS / AWS S3 账号、不产生云费用**的前提下，真实跑通本次落地的云存储代码路径。
> MinIO 是 100% S3 协议兼容的自托管对象存储，直接命中后端的 `S3Storage` 驱动（上传 / 下载 / 删除 / 存在性 / 预签名 URL 全部真实生效）。

所属模块：`internal/modules/file/storage`（`s3_storage.go`）
适用配置：`file.storage_type = s3` + `file.cloud.*`

---

## 1. 前置条件

- 本机已安装 Docker（含 `docker compose`）。
- 后端依赖的 MySQL / Redis 可用（本地联调建议一并由 compose 拉起）。

---

## 2. 启动 MinIO（自动建桶）

`docker-compose.dev.yml` 已内置两个服务：

| 服务 | 作用 |
|------|------|
| `minio` | 对象存储服务，API 端口 `9000`，控制台端口 `9001`，凭据 `minioadmin / minioadmin` |
| `minio-init` | 一次性任务，待 MinIO 健康后自动创建 `meteorx` 桶（已存在则跳过），无需手动建桶 |

```powershell
# 仅拉起存储相关服务（若本地已有 db/redis 可省略它们）
docker compose -f docker-compose.dev.yml up -d minio minio-init
```

启动后可访问控制台 <http://localhost:9001>（`minioadmin / minioadmin`），应能看到桶 `meteorx`。

---

## 3. 让后端指向 MinIO

二选一。**推荐方式 B**：浏览器可直接打开接口返回的预签名 URL（`localhost:9000` 已映射到宿主机）。

### 方式 A：后端也跑在 compose 里

打开 `docker-compose.dev.yml`，取消 `app` 服务中已注释的存储变量（端点走容器网内地址 `http://minio:9000`）：

```yaml
- METEORX_FILE_STORAGE_TYPE=s3
- METEORX_FILE_CLOUD_ENDPOINT=http://minio:9000
- METEORX_FILE_CLOUD_REGION=us-east-1
- METEORX_FILE_CLOUD_BUCKET=meteorx
- METEORX_FILE_CLOUD_ACCESS_KEY=minioadmin
- METEORX_FILE_CLOUD_SECRET_KEY=minioadmin
- METEORX_FILE_CLOUD_FORCE_PATH_STYLE=true
```

然后 `docker compose -f docker-compose.dev.yml up -d app`。
> 注意：此方式下接口返回的预签名 URL 主机为 `minio:9000`，宿主机浏览器无法解析 `minio`，链接需在能访问容器网络的客户端打开；仅验证上传/入库时不受影响。要在浏览器直接点链接，请用方式 B。

### 方式 B：后端在宿主机运行（推荐，便于浏览器目视）

在同一 PowerShell 会话设置环境变量后启动服务（端点用宿主机可访问的 `http://localhost:9000`）：

```powershell
docker compose -f docker-compose.dev.yml up -d db redis   # 若本地无 MySQL/Redis

$env:METEORX_FILE_STORAGE_TYPE            = "s3"
$env:METEORX_FILE_CLOUD_ENDPOINT         = "http://localhost:9000"
$env:METEORX_FILE_CLOUD_REGION           = "us-east-1"
$env:METEORX_FILE_CLOUD_BUCKET           = "meteorx"
$env:METEORX_FILE_CLOUD_ACCESS_KEY       = "minioadmin"
$env:METEORX_FILE_CLOUD_SECRET_KEY       = "minioadmin"
$env:METEORX_FILE_CLOUD_FORCE_PATH_STYLE = "true"

go run ./cmd/server
```

---

## 4. 验证

1. 上传文件：`POST /api/v1/files/upload`（或前端「文件管理」页上传）。
2. 查列表 / 详情：`GET /api/v1/files`，返回的 `url` 应为 `http://localhost:9000/meteorx/<对象键>?X-Amz-...` 形态的**预签名 URL**。
3. 浏览器打开该 URL → 能下载/预览即说明签名与桶策略正常。
4. 控制台 <http://localhost:9001> 进入桶 `meteorx`，应看到对应对象（对象键为「唯一 ID + 原扩展名」，配了 `path_prefix` 时带前缀）。
5. 删除文件后，控制台对象应随之移除。

> `storage_type` 记录在 `file` 表：切换后端后新上传的记录 `storage_type` 会变为 `s3`。

---

## 5. 回退到本地存储

删除上述环境变量（或设 `METEORX_FILE_STORAGE_TYPE=local`）后重启即可，行为与默认完全一致——**云驱动代码随之休眠，不联网、不影响存量本地文件**。

---

## 6. 关于阿里云 OSS 驱动

OSS 驱动（`oss_storage.go`）走的是阿里云私有协议，**没有像 MinIO 这样通用的本地模拟器**，因此本地无法零凭据跑通。如需验证 OSS 路径：
- 使用阿里云 OSS 的真实 `endpoint / bucket / AccessKey`；或
- 用兼容 OSS 签名算法的自建服务（较少见，成本高）。

日常本地联调 **用 MinIO 验证 S3 驱动即可**，两者共用同一套 `Storage` 接口，S3 跑通即代表接口契约、工厂分派、降级与预签名逻辑均正确。

---

## 7. 关停 / 清理

```powershell
docker compose -f docker-compose.dev.yml down          # 停止服务（保留数据卷）
docker volume rm meteorx_minio-data-dev                # 需要清空已上传对象时再删卷
```
