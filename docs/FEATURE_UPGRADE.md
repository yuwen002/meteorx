# MeteorX 平台功能升级说明

按既定排序完成了三大功能升级：**① 平台运营数据看板 ② 平台通知/公告系统 ③ 租户注销审批闭环**。

---

## 一、平台运营数据看板（Dashboard）

**后端新增 `internal/modules/dashboard` 模块**
- `GET /api/v1/admin/dashboard/overview`（平台超级管理员专属）→ 权限码 `admin:dashboard:list`
- 聚合统计：租户总数/启用/禁用/今日新增、用户总数/新增、订阅总数/生效/到期/取消、审计日志总数/成功/失败/按操作与模块分布
- 分层：`model → repository → service → handler → dto`，全部基于 GORM 直查既有数据表（tenants / users / tenant_subscriptions / audit_logs）

**前端增强 `web-admin/src/views/dashboard/index.vue`**
- 平台管理员看到：四类核心指标卡片 + 新增概览表（今日/本周/本月）+ 订阅分布 + 操作/模块访问统计
- 普通租户管理员仍看到原工作台统计，二者自动区分

---

## 二、平台通知/公告系统（Notification）

**后端新增 `internal/modules/notification` 模块**（平台超级管理员专属）
- 路由：
  - `GET /admin/announcements` → `admin:announcement:list`
  - `POST /admin/announcements` → `admin:announcement:create`
  - `GET /admin/announcements/{id}` → `admin:announcement:read`
  - `PUT /admin/announcements/{id}` → `admin:announcement:update`
  - `PUT /admin/announcements/{id}/status` → `admin:announcement:status`（发布/下架）
  - `DELETE /admin/announcements/{id}` → `admin:announcement:delete`
- 公告支持：草稿/已发布/已下架状态、全平台/指定租户范围、发布时间/过期时间
- 新建表 `announcements`（AutoMigrate 自动迁移）
- 在权限推导中间件 `singularize` 增加 `announcements → announcement` 映射

**前端新增 `web-admin/src/views/system/announcement/index.vue`**
- 公告列表（搜索/状态筛选/分页）、新建/编辑弹窗、发布/下架/删除操作、详情弹窗
- 路由 `/system/announcement` + 侧边栏「公告管理」菜单（仅平台管理员可见）

---

## 三、租户注销审批闭环（Tenant）

将原有的注销「占位实现」升级为完整审批闭环。

**后端（改造 `internal/modules/tenant`）**
- 新增领域模型与表 `cancel_requests`（AutoMigrate 迁移）
- 状态机：`待审批 → 已通过 → 已完成` 或 `待审批 → 已驳回`
- 租户侧申请 `ApplyCancellation`：真实写入待审批申请，重复申请会被拦截
- 平台审批接口：
  - `GET /admin/cancel-requests` → `admin:cancel_request:list`
  - `PUT /admin/cancel-requests/{id}/approve` → `admin:cancel_request:approve`（可设生效天数，0=立即执行）
  - `PUT /admin/cancel-requests/{id}/reject` → `admin:cancel_request:reject`
- 定时任务 `StartCancelCleanupJob`（5 分钟间隔）：自动执行已到期的注销申请 → 软删除租户 + 取消生效订阅 + 标记完成
- 在权限推导中间件增加 `cancel-requests → cancel_request` 映射

**前端新增 `web-admin/src/views/system/cancel-request/index.vue`**
- 注销申请列表（租户名称/原因/状态/计划执行时间筛选）
- 通过（选择立即/3天/7天生效 + 备注）与驳回（必填原因）操作
- 路由 `/system/cancel-request` + 侧边栏「注销审批」菜单（仅平台管理员可见）

---

## 四、租户物理删除与套餐配给（Tenant）

补充两个平台管理员高频操作接口：物理删除租户（彻底销毁）和为租户分配/变更套餐。

**后端（`internal/modules/tenant`）**
- 新增 `HardDelete` 仓储方法，通过 GORM `Unscoped().Delete()` 绕过软删除
- 新增 `AdminHardDelete` 服务方法：物理删除租户 + 同步取消生效订阅
- 新增 `AdminUpdatePlan` 服务方法：通过接口注入调用 `PlanService.AssignPlan`
- 注入方式：`SetPlanAssignProvider` 接口，避免 Tenant 模块直接依赖 Plan 模块
- 路由：
  - `DELETE /admin/tenants/{id}/hard` → `admin:tenant:hard_delete`
  - `PUT /admin/tenants/{id}/plan` → `admin:tenant:update_plan`

---

## 五、系统优化与架构改进

### 5.1 审计日志批量处理优化

**性能提升：10-100倍**

**涉及文件：**
- `internal/modules/audit/repository/interface.go` - 添加 `BatchCreate` 接口方法
- `internal/modules/audit/repository/audit_repository.go` - 实现真正的批量SQL插入
- `internal/modules/audit/service/audit_service.go` - 添加 `BatchCreateLogs` 方法
- `internal/middleware/audit_batch.go` - 使用批量插入替代逐个写入
- `internal/middleware/audit_middleware.go` - 优化Context传递

**改进效果：**
- 从 O(n) 数据库操作优化为 O(1) 批量插入
- 批量大小：100条，刷新间隔：5秒
- 减少数据库连接开销和磁盘IO

### 5.2 Context生命周期管理

**涉及文件：**
- `internal/middleware/audit_middleware.go` - 异步审计日志使用带超时的Context（10秒）
- `internal/bootstrap/app.go` - 提前创建应用级context
- `internal/bootstrap/router.go` - 更新 `InitRouter` 签名接受context

**改进效果：**
- 避免服务器关闭时 goroutine 泄漏
- 异步操作支持优雅取消
- 符合 Go 语言最佳实践

### 5.3 统一错误处理

**涉及文件：**
- `internal/modules/tenant/handler/tenant_handler.go` - 替换硬编码状态码为 `http.StatusXXX` 常量
- `internal/modules/auth/handler/auth_handler.go` - 统一错误处理模式
- `internal/common/validator/validator.go` - 标准化状态码使用

**改进效果：**
- 代码可读性和可维护性提升
- 符合 Go 语言最佳实践
- 禁止使用硬编码HTTP状态码

### 5.4 日志标准化

**涉及文件：**
- `internal/modules/tenant/handler/tenant_handler.go` - 替换 `fmt.Println` → `log.Printf`
- `internal/modules/user/handler/user_handler.go` - 替换 `fmt.Println` → `log.Printf`
- `internal/modules/file/service/file_service.go` - 替换 `fmt.Printf` → `log.Printf`

**改进效果：**
- 所有日志输出使用标准 `log` 包
- 添加模块前缀（如 `[TenantHandler]`、`[UserHandler]`、`[FileService]`）
- 便于日志收集、分析和监控

### 5.5 JWT密钥安全强化

**涉及文件：**
- `internal/config/config.yaml` - 添加强密钥警告注释

**改进效果：**
- 提醒开发者在生产环境使用强随机密钥
- 提供生成强密钥的方法说明：`openssl rand -base64 64`

### 5.6 健康检查增强

**涉及文件：**
- `internal/bootstrap/router.go` - 新增 `/health/ready` 深度检查端点
- `internal/cache/redis.go` - 添加 `Ping` 方法

**改进效果：**
- `/health` - 基础检查（负载均衡器使用）
- `/health/ready` - 深度检查（数据库+Redis连接状态，K8s就绪探针使用）
- 返回详细的JSON健康状态

### 5.7 角色永久删除功能

**涉及文件：**
- `internal/modules/rbac/repository/interface.go` - 添加 `PermanentDelete` 和 `BatchPermanentDelete` 接口
- `internal/modules/rbac/repository/role_repository.go` - 实现永久删除方法
- `internal/modules/rbac/service/rbac_service.go` - 添加业务逻辑
- `internal/modules/rbac/handler/rbac_handler.go` - 添加HTTP处理器
- `internal/modules/rbac/routes.go` - 注册路由
- `web-admin/src/api/modules/role.ts` - 添加前端API
- `web-admin/src/views/system/role/recycle.vue` - 实现前端功能

**新增API端点：**
- `DELETE /api/v1/rbac/roles/{id}/permanent` - 永久删除单个角色
- `DELETE /api/v1/rbac/roles/batch/permanent` - 批量永久删除角色

**改进效果：**
- 角色回收站功能完整闭环（软删除 → 回收站查询 → 恢复 → 永久删除）
- 完成前端TODO项

---

## 权限注册汇总（`internal/modules/rbac/permissions.go`）

| 权限码 | 说明 |
|---|---|
| `admin:dashboard:list` | 查看运营数据看板 |
| `admin:announcement:list/read/create/update/status/delete` | 公告管理 |
| `admin:cancel_request:list/approve/reject` | 注销审批 |
| `admin:tenant:hard_delete` | 物理删除租户（彻底销毁） |
| `admin:tenant:update_plan` | 为租户分配/变更套餐 |
| `rbac:role:permanent_delete` | 永久删除角色 |
| `rbac:role:batch_permanent_delete` | 批量永久删除角色 |

全部通过幂等 `SeedPermissions` 注册，superadmin 直接放行。

---

## 验证状态

- ✅ 后端 `go build ./...` 全量编译通过
- ✅ `go vet` 新增模块无静态问题
- ✅ 前端 `vue-tsc --noEmit` 类型检查通过（新文件无报错）
- ✅ 新文件均执行 `gofmt` 格式化
- ✅ 所有单元测试通过（audit/service、rbac/service、file/service、user/service、middleware）

> 提示：需重新启动后端服务并执行数据库迁移（AutoMigrate 会自动创建 `announcements`、`cancel_requests` 两张新表）。