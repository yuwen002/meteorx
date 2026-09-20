# OpenAPI 文档同步工具

## 📖 概述

`scripts/openapi_sync.go` 是一个用于同步代码路由与 OpenAPI 文档的自动化工具。它会对比实际代码中的路由与 `docs/apifox/MeteorX-backend.openapi.json` 中的定义，确保两者保持一致。

## 🎯 使用场景

- 新增/删除 API 路由后，自动更新 OpenAPI 文档
- 检查 OpenAPI 文档是否遗漏或多余路由
- 保持 API 文档与代码实时同步

## 🚀 使用方法

### 1. 检查模式（默认）

仅对比差异，不修改文件：

```bash
go run scripts/openapi_sync.go
```

**输出示例：**
```
======== 代码路由 vs OpenAPI 对比 ========

代码路由总数: 226
OpenAPI 路由总数: 225

⚠️  OpenAPI 缺失 1 个路由：
  + GET /api/v1/auth/oauth/tenants

✅ OpenAPI 无多余路由

💡 加 -apply 参数可自动修复
```

### 2. 自动修复模式

自动添加缺失路由、删除多余路由：

```bash
go run scripts/openapi_sync.go -apply
```

**输出示例：**
```
======== 代码路由 vs OpenAPI 对比 ========

代码路由总数: 226
OpenAPI 路由总数: 225

⚠️  OpenAPI 缺失 1 个路由：
  + GET /api/v1/auth/oauth/tenants

✅ OpenAPI 无多余路由

===== 应用修复 =====
+ 添加 GET /api/v1/auth/oauth/tenants

✅ 已修复：+1 / -0
```

## 📋 工作原理

1. **构建路由树**：通过 `buildRouter()` 函数初始化所有模块的路由
2. **遍历代码路由**：使用 `chi.Walk()` 遍历所有注册的路由
3. **读取 OpenAPI 文件**：解析 `docs/apifox/MeteorX-backend.openapi.json`
4. **对比差异**：
   - 缺失路由：代码中有但 OpenAPI 中没有
   - 多余路由：OpenAPI 中有但代码中没有
5. **自动修复**（使用 `-apply` 时）：
   - 自动添加缺失路由（含默认 summary、tags、responses）
   - 自动删除多余路由

## 🔧 自动生成的路由信息

当添加缺失路由时，脚本会自动推断以下信息：

| 字段 | 推断规则 |
|------|---------|
| `summary` | 根据方法和路径推断，如 `GET /users` → "获取users" |
| `tags` | 根据路径前缀推断，如 `/api/v1/auth/*` → `["Auth"]` |
| `description` | 自动生成权限码，如 `权限码: auth:login` |
| `responses` | 默认 200 成功响应 |

### Tags 推断规则

| 路径前缀 | Tag |
|---------|-----|
| `/health`, `/metrics` | System |
| `/api/v1/auth`, `/api/v1/oauth` | Auth |
| `/api/v1/users` | User |
| `/api/v1/tenants`, `/api/v1/tenant` | Tenant |
| `/api/v1/rbac` | RBAC |
| `/api/v1/audit` | Audit |
| `/api/v1/wiki` | Wiki |
| `/api/v1/files` | File |
| `/api/v1/admin` | Admin |
| `/api/v1/plan` | Plan |
| `/api/v1/announcements`, `/api/v1/notification` | Notification |

## ⚠️ 注意事项

1. **自动修复仅添加基础结构**：生成的路由只有默认值，需要手动补充详细的请求参数、响应体、业务说明等
2. **先检查后修复**：建议先不加 `-apply` 运行，确认差异后再执行修复
3. **手动完善文档**：自动修复后，务必手动补充以下信息：
   - 详细的 `summary` 和 `description`
   - 请求参数（path/query/body）
   - 响应体 schema
   - 错误响应说明
   - 业务规则说明
4. **Git 提交**：修复后记得提交 OpenAPI 文件的变更

## 📝 最佳实践

### 开发流程

```bash
# 1. 开发新功能，添加路由后
go run scripts/openapi_sync.go

# 2. 查看差异，确认无误
go run scripts/openapi_sync.go -apply

# 3. 手动完善生成的 API 文档
# 编辑 docs/apifox/MeteorX-backend.openapi.json

# 4. 提交变更
git add docs/apifox/MeteorX-backend.openapi.json
git commit -m "docs(api): 同步 OpenAPI 文档，新增 xxx 接口"
```

### 定期检查

建议在以下时机运行同步工具：

- ✅ 每次新增/删除 API 路由后
- ✅ 提交代码前
- ✅ 发布版本前

## 🛠️ 常见问题

### Q: 为什么有些路由没有被自动识别？

A: 确保路由在 `buildRouter()` 函数中正确注册。如果使用了动态路由或特殊的路由模式，可能需要手动添加到 OpenAPI 文件。

### Q: 自动生成的 tags 不正确怎么办？

A: 可以手动修改 `inferTags()` 函数添加新的路径前缀规则，或者直接在 OpenAPI 文件中手动修正。

### Q: 如何自定义生成的路由信息？

A: 修改 `addMissingRoute()` 函数中的默认值，或者在自动修复后手动编辑 OpenAPI 文件。