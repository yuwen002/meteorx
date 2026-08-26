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

## 权限注册汇总（`internal/modules/rbac/permissions.go`）

| 权限码 | 说明 |
|---|---|
| `admin:dashboard:list` | 查看运营数据看板 |
| `admin:announcement:list/read/create/update/status/delete` | 公告管理 |
| `admin:cancel_request:list/approve/reject` | 注销审批 |
| `admin:tenant:hard_delete` | 物理删除租户（彻底销毁） |
| `admin:tenant:update_plan` | 为租户分配/变更套餐 |

全部通过幂等 `SeedPermissions` 注册，superadmin 直接放行。

---

## 验证状态

- ✅ 后端 `go build ./...` 全量编译通过
- ✅ `go vet` 新增模块无静态问题
- ✅ 前端 `vue-tsc --noEmit` 类型检查通过（新文件无报错）
- ✅ 新文件均执行 `gofmt` 格式化

> 提示：需重新启动后端服务并执行数据库迁移（AutoMigrate 会自动创建 `announcements`、`cancel_requests` 两张新表）。