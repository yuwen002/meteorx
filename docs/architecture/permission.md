# 权限与 RBAC 架构

## 概述

基于角色的访问控制，结合路由自动权限推导和特殊业务逻辑的显式权限覆盖。

## 权限模型

```
用户 ──(M:N)──► 角色 ──(M:N)──► 权限
                                    │
                            {命名空间}:{资源}:{操作}
                            例如：admin:user:create
```

## 权限码格式

```
{命名空间}:{资源}:{操作}

命名空间：
  system  → 系统级（跨租户）
  admin   → 管理后台操作
  tenant  → 租户级操作
  （无）  → 默认模块范围

资源：
  user, role, permission, plan, tenant, wiki_space, wiki_node, ...

操作：
  list, create, read, update, delete,
  bind_perm, unbind_perm, batch_delete,
  restore, export, select, reset_password, ...
```

## 自动权限推导

### 包路径：`internal/middleware`

`AutoRequirePermission` 中间件根据 HTTP 方法 + URL 路径自动推导权限码：

```
GET    /api/v1/users                     → user:list
POST   /api/v1/users                     → user:create
GET    /api/v1/users/{id}/detail         → user:read
PUT    /api/v1/users/{id}/update         → user:update
DELETE /api/v1/users/{id}/delete        → user:delete
GET    /api/v1/rbac/roles                → rbac:role:list
PUT    /api/v1/rbac/roles/{id}/permissions → rbac:role:bind_perm
GET    /api/v1/admin/users               → admin:master:list
POST   /api/v1/admin/users               → admin:master:create
```

### 资源命名约定

```
users           → user
roles           → role
permissions     → perm
role-permissions → role_perm
tenant-users    → user（在 admin 命名空间下）
logs            → log（在 audit 命名空间下）
announcements   → announcement
cancel-requests → cancel_request
wiki-spaces     → wiki_space
wiki-nodes      → wiki_node
documents       → document
```

### 特殊操作推导

| 路径模式 | 方法 | 推导操作 |
|----------|------|----------|
| `/{id}/permanent` | DELETE | `permanent_delete` |
| `/reset-password` | PUT/POST | `reset_password` |
| `/export` | GET | `export` |
| `/select` | GET | `list_select` |
| `/batch/status` | PUT | `batch_status` |
| `/batch/delete` | DELETE | `batch_delete` |
| `/{id}/permissions/batch` | PUT | `batch_bind` |
| `/{id}/permissions/batch` | DELETE | `batch_unbind` |
| `/{id}/roles`（1 个参数） | DELETE | `remove_all` |
| `/{id}/roles/{role_id}`（2 个参数） | DELETE | `remove_one` |
| `/deleted` | GET | `list_deleted` |
| `/deleted` | PUT | `restore` |

## 显式权限覆盖

### 何时使用

当自动推导不足时使用显式权限：

- 跨命名空间操作
- 复杂业务规则
- API Key / 服务账号访问
- B2B 渠道 API

### 实现

```go
// 包路径：internal/middleware

// 方式 1：路由级覆盖
r.Route("/custom-endpoint", func(r chi.Router) {
    r.Use(middleware.WithExplicitPermission)
    // 此处需要显式设置的权限
})

// 方式 2：显式权限中间件
r.Use(middleware.RequirePermission(checker, "custom:business:approve"))

// 方式 3：Handler 中的显式权限
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
    // 业务逻辑受显式权限保护
}
```

### 包路径：`internal/middleware`

```go
// 为路由设置显式权限
middleware.WithExplicitPermission(next, "wiki:document:approve")

// 要求特定权限（可指定默认值）
middleware.RequireExplicitPermission(checker, "wiki:document:update")
```

## 超级管理员绕过

拥有 `superadmin` 角色的用户自动跳过**所有**权限检查：

```go
if contextx.HasRole(r.Context(), "superadmin") {
    next.ServeHTTP(w, r)
    return
}
```

## 权限解析流程

```
请求
    ↓
AutoRequirePermission 中间件
    ↓
检查：是否超级管理员？→ 是 → 绕过
    ↓ 否
    ↓
检查：是否有显式权限覆盖？→ 使用它
    ↓ 否
    ↓
根据方法 + 路径推导权限码
    ↓
查询用户的权限码（合并角色后）
    ↓
匹配推导的权限码与用户权限
    ↓
匹配？→ 是 → 允许
    ↓ 否          ↓
    拒绝         允许（如果推导失败）
```

## 权限缓存

### 未来增强

```go
// 基于 Redis 的缓存
// Key: user:{userID}:permissions
// TTL: 5 分钟
// 失效时机：角色变更、权限变更
```

## 中间件配置

```go
// 在 bootstrap/router.go 中
func RegisterRoutes(r chi.Router) {
    r.Use(middleware.AuthMiddleware(jwtManager))
    r.Use(middleware.AutoRequirePermission(permissionChecker))
    // ...
}
```

## 测试矩阵

| 测试场景 | 预期 |
|----------|------|
| 普通用户访问自己的资源 | 允许 |
| 普通用户访问其他租户资源 | 拒绝（租户隔离） |
| 超级管理员访问任意资源 | 允许 |
| 用户缺少所需权限 | 拒绝 |
| 正确设置显式权限 | 允许 |
| 错误设置显式权限 | 拒绝 |
| 缺少认证 | 拒绝（401） |
| 缺少推导（未匹配路由） | 允许（Handler 级检查） |

## 最佳实践

1. **优先使用自动推导** — 仅在复杂场景使用显式权限
2. 将权限码**定义为常量**放在各模块的 `permissions.go` 中
3. **禁止**在 Handler 中硬编码分散的权限字符串
4. **缓存**用户合并后的权限列表以提升性能
5. 通过集成测试**测试**权限边界
6. 为每个模块**文档化**特殊权限码