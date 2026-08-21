# 多租户隔离架构

## 概述

基于 `tenant_id` 行级隔离的多租户 SaaS 架构。所有操作租户数据的查询都会被框架自动限定租户范围。

## 架构流程

```
HTTP 请求
    ↓
JWT / 域名解析
    ↓
AuthMiddleware → contextx.SetVars(ctx, tenantID, userID, roles)
    ↓
tenantctx.From(ctx) — 统一租户上下文访问
    ↓
Service / Repository — 自动租户范围限定
    ↓
数据库查询 — 始终包含 tenant_id
```

## 租户上下文

### 包路径：`internal/common/tenantctx`

```go
// 从上下文获取当前租户信息
info, err := tenantctx.From(ctx)
// info.ID       — 租户 ID
// info.IsMaster — 超级管理员/系统操作为 true

// 获取租户上下文（缺失时 panic）
info := tenantctx.RequireTenant(ctx)

// 检查用户是否可以访问目标租户
ok := tenantctx.CanAccessTenant(ctx, targetTenantID)

// 使用 tenant_id 自动限定 GORM 查询
query := tenantctx.FilterQuery(ctx, db, "tenant_id")
```

### 核心规则

| 场景 | 行为 |
|------|------|
| 普通用户 | 查询自动按 `tenant_id` 过滤 |
| 系统管理员 | `IsMaster=true`，查询**不**过滤（可查看全部） |
| 跨租户访问 | `CanAccessTenant()` 返回 `false` |
| 缺少租户上下文 | `FilterQuery()` 返回 `WHERE 1=0`（防止数据泄露） |

## 租户所有模型

| 模型 | 表名 | tenant_id | 说明 |
|------|------|-----------|------|
| User | users | 是 | 按租户分区 |
| Role | roles | 是 | 租户级角色 |
| AuditLog | audit_logs | 是 | 所有操作 |
| TenantSettings | tenant_settings | 是 | 租户独立配置 |
| WikiSpace | wiki_spaces | 是 | Wiki 模块 |
| WikiNode | wiki_nodes | 是 | Wiki 模块 |
| Document | documents | 是 | Wiki 模块 |
| WikiSpaceMember | wiki_space_members | 是 | Wiki 模块 |
| Subscription | subscriptions | 是 | 租户订阅 |
| Announcement | announcements | 是 | 租户公告 |

## 实现示例

```go
// Repository：始终使用 FilterQuery
func (r *wikiRepository) ListSpaces(ctx context.Context, tenantID string, page, pageSize int) ([]*model.WikiSpace, int64, error) {
    var spaces []*model.WikiSpace
    var total int64

    query := tenantctx.FilterQuery(ctx, r.db.WithContext(ctx), "tenant_id")
    query.Where("tenant_id = ?", tenantID)

    if err := query.Count(&total).Error; err != nil {
        return nil, 0, err
    }

    offset := (page - 1) * pageSize
    if err := query.Offset(offset).Limit(pageSize).Find(&spaces).Error; err != nil {
        return nil, 0, err
    }
    return spaces, total, nil
}
```

## 测试

测试必须验证跨租户隔离：

```go
func TestTenantIsolation(t *testing.T) {
    // 租户 A 的用户不能访问租户 B 的数据
    // 系统管理员可以访问全部
}
```

参考：`internal/common/tenantctx/tenantctx_test.go`

## 中间件链

```
RequestIDMiddleware
    ↓
AuthMiddleware → 解析 JWT，设置租户上下文
    ↓
RoleMiddleware → 检查角色
    ↓
AutoRequirePermission / ExplicitPermission
    ↓
RateLimitMiddleware
    ↓
GlobalErrorHandler
```

## 最佳实践

1. **禁止**跳过 `FilterQuery()` 对租户所有查询的过滤
2. Repository 层**必须**强制租户范围限定，不能只在 Service 层做
3. 系统管理员操作应在上下文中设置 `IsMaster=true`
4. 使用 `FilterQuery()` 时，WHERE 子句中的 `tenant_id` 是多余的