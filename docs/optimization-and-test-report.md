# MeteorX 代码优化与测试报告

日期：2026-09-08（九轮）
范围：全仓审查（Go 后端 + web-admin 前端概览）、针对性修复、测试补齐、handler 层测试推广（wiki/tenant）+ 基础组件补齐、注册事务边界修复、测试范围收敛与 CI 统一校验入口、handler 层横推 user/plan/audit/rbac/file 收官、文件分享签名密钥缺陷修复（独立密钥 + 平滑轮换）

---

## 一、执行摘要

- 全仓 `go test ./...` 通过（exit 0），`go vet ./internal/... ./pkg/...` 无告警。
- 累计**修复 8 个真实缺陷/残缺**（P0 安全后门 1、功能级 4、事务边界 2、命名语义 1）。
- 第三轮启动、第四轮推广 **handler 层测试**（此前全仓 0 覆盖的最大缺口）：为 `auth` / `notification` / `dashboard` / `wiki` / `tenant` 五个模块 handler 引入**服务接口缝**（生产调用方零改动），累计 **100+ HTTP 用例**，handler 覆盖率：auth 90.3% / notification 83.8% / dashboard 100% / **wiki 69.9%** / **tenant 78.6%**。
- 第四轮同步补齐基础组件：`common/response` 93.3%、`middleware` 42.6%（request_id / 全局 panic 恢复 / master admin / 限流语义）、wiki 服务命名语义修复。
- 新增 **70 个测试文件、691 个测试用例**（后端 `internal`/`pkg`）。
- 第五轮：修复**注册流程角色分配无事务边界**（原 P1 待办）——角色关联并入 `CreateTenantWithAdmin` 同一事务原子落库，并新增 repository 层 sqlmock 回滚用例守护。
- 第六轮：**测试范围收敛 + CI 统一校验入口**——`web-admin` 前端目录声明嵌套 `go.mod`，从根模块隔离（根治本地 `go test ./...` 误扫 `node_modules/flatted/golang`）；新增 `scripts/test.sh`（Linux/CI）与 `scripts/test.ps1`（Windows 本地）作为后端 vet+build+test 的统一入口；CI `backend` job 从脆弱的 `./...` 收敛到后端包范围并复用脚本（仍含 `-race` 与覆盖率上传）。
- 第七轮：**handler 层横推 user / plan / audit（audit+alert+session）**——为三个模块 handler 接入服务接口缝（生产零改动），新增 **166 个 HTTP 用例**：`user`（用户管理/master admin/跨租户/回收站/profile/统计，98 例）、`plan`（套餐 CRUD/下拉/分配/当前套餐，28 例）、`audit`（日志查询/CSV 导出/超管清理权限位/告警规则/告警记录/会话分析，40 例）。注册守卫中 `type XxxService interface` 断言依赖编译期验证；`PathValue` 路由通过标准库 `http.ServeMux` 注入测试。handler 层累计覆盖 **8 个模块、260+ HTTP 用例**。
- 第八轮：**handler 层横推收官 rbac / file**——`rbac`（角色/权限/回收站/绑定解绑/用户-角色/统计，71 例）与 `file`（multipart 上传白名单校验/超限/租户隔离/回收站/流式下载，28 例）接入服务接口缝，新增 **99 个 HTTP 用例**。至此 **10 个模块 handler 全部覆盖，累计 360+ HTTP 用例**，handler 层服务接口缝横推完成（P1 收官）；上传类通过 `multipart.Writer` 在测试中真实构造 MIME 白名单场景。
- 第九轮：**修复"上传访问签名密钥与 JWT 登录密钥耦合"缺陷（原 P2 待办）**——`/uploads` 静态验签、`file` 访问 URL、wiki 内嵌图片与免登录分享重签 **4 处装配点原均复用 `cfg.JWT.Secret`**，更换 JWT secret 会令全部存量短时效链接失效并牵连登录态。已解耦为独立 `file.sign_key`（未配置自动回退 `jwt.secret` 兼容旧部署），并支持**逗号分隔密钥链平滑轮换**：新链接用首项密钥签发、`signedurl.VerifyAny` 与 uploads 中间件按密钥链校验存量链接（宽限期后移除旧密钥即可）。新增 4 组测试函数（signedurl 轮换边界 6 项、uploads 新旧链放行/拒绝 3 场景、local_storage 签名/预签名/无密钥 6 项、config 回退/轮换/纯分隔符 4 类断言）。
- 产物：本报告（`docs/optimization-and-test-report.md`）。

---

## 二、已落地的修复

### 1.【P0 安全】认证调试后门默认开启 → 改为显式开关
`internal/middleware/auth.go`、`bootstrap/router.go`、`config`

固定 Token `123456789` 可免密变身超级管理员，原逻辑仅依赖 `mode != "release"`。新增 `server.test_bypass`（默认 false），仅在**显式开启且非 release** 时放行。配套门控矩阵测试 7 分支（`middleware/auth_test.go`）。

### 2.【P1 功能】注销定时任务未注入订阅仓库 → 到期租户订阅永不取消
`internal/modules/tenant/cancel_cleanup_job.go`

任务构造时 `subRepo` 传 nil，导致 `ExecuteCancellation` 跳过订阅取消分支。已注入订阅仓库并复用测试守护。

### 3.【P1 功能】注销执行吞错 + 无可见性 → 失败可被重试并记录
`internal/modules/tenant/service/tenant_service.go`

原实现 `err == nil && activeSub != nil` 把查询错误静默忽略、`_ = UpdateStatus` 吞掉订阅取消失败，且批量任务 `continue` 不留日志 —— 订阅状态可能永久不一致且无人知晓。修复：
- `ExecuteCancellation`：订阅查询/取消失败即返回带上下文错误；**申请不标记完成**（软删幂等，下轮任务自动重试）；
- `AdminHardDelete`：取消订阅失败不再吞掉，明确提示“租户已删但订阅需人工处理”；
- `ExecuteDueCancellations`：记录 `logger.Errorf` 失败明细。
同步把测试 mock 的 `GetActiveByTenant` 与真实实现对齐（无订阅 → `nil,nil` 而非报错，避免掩盖语义错误），新增 4 个用例：成功取消订阅、取消失败保持待办、查询失败返回、硬删取消失败上报。

### 4.【P1 功能】软删文档/节点物理清除附件 → 恢复后附件永久丢失
`internal/modules/wiki/service/wiki_document_service.go`、`wiki_node_service.go`

`Attachment` 无 `DeletedAt` 字段，`DeleteAttachmentsByDocument` 即物理删除。原先软删文档/节点时同步物理删掉附件行，从回收站恢复文档后附件引用永久丢失。修复：
- 软删**不再触碰附件记录**（文档在回收站期间经文档权限校验不可见，无越权暴露风险）；
- 物理清理统一收敛到回收站“永久删除”的 Purge 级联流程；
- 节点级联删除中文档软删失败从 `_ =` 改为**向上传播并整体回滚**。
新增 `wiki_delete_service_test.go`（4 用例）：文档软删保留附件、DB 错误回滚、节点级联保留附件且错误中止。

### 5.【P2 一致性】回收站永久删除缺事务边界 → 中途失败留半删数据
`internal/modules/wiki/service/wiki_trash_service.go`

`PermanentDeleteTrashItem` 的物理清除（多张关联表）与回收站行兜底清理原来各自独立提交。现统一用 `WithTx` 包裹：任一步失败整体回滚、回收站行不误删。同步更新 3 个永久删除测试用例（成功/实体缺失容忍/真实错误回滚）。

### 6.【P3 语义】`UpdateSpace` 第三参数名 `tenantID` 实为权限校验主体 `userID`
`internal/modules/wiki/service/wiki_service.go`、`wiki_space_service.go`

接口/实现形参名会误导后续调用与维护（handler 实际传入登录用户，服务内按 `userID` 做 `CheckSpacePermission`）。已统一改名为 `userID` 并补注释（纯改名，编译级验证不影响调用方与既有测试）。

### 7.【P1 一致性】租户注册角色分配无事务边界 → 三表同一事务原子落库
`internal/modules/tenant/repository/tenant_repository.go`、`tenant/service/tenant_service.go`

`CreateTenantWithAdmin`（事务内建租户+管理员）与 `AssignRoles` 分属 tenant/rbac 两个 repo、各管各的事务，注册在两步之间失败会遗留"有租户无角色"的脏数据且重试无法自愈（新租户再走注册会被域名/用户名冲突拦截）。修复：
- `CreateTenantWithAdmin` 签名扩展 `roleIDs []string`，在**同一 GORM 事务**内依次写入 `tenants` / `users` / `user_roles`，任一步失败整体回滚；角色写入失败返回带语境的错误。
- `Register` / `AdminCreate` 删除事务外的 `AssignRoles` 二次提交，角色 ID 随创建一次性下发；`TenantService` 不再依赖 `UserRoleRepository`。
- 新增 repository 层 sqlmock 用例 3 个：三表成功提交、**角色写入失败触发回滚**、空角色列表跳过关联表；service 层补充 `AdminCreate` 默认角色下发断言。

---

## 三、待办优化清单（建议，未改）

| 级别 | 位置 | 问题与建议 |
|---|---|---|
| P1 | 各模块 handler 层 | ✅ 完成（第三~八轮收官）：auth/notification/dashboard/wiki/tenant/user/plan/audit/rbac/file **10 模块全部**接入服务接口缝并补 HTTP 测试，累计 360+ 用例；handler 层横推无剩余 |
| P1 | `tenant/service` Register 角色分配 | ✅ 完成（第五轮）：角色关联并入 `CreateTenantWithAdmin` 同一事务原子落库，失败整体回滚，附 sqlmock 回滚用例 |
| P2 | 文件分享 | ✅ 完成（第九轮）：上传访问链接签名从复用 `jwt.secret` 解耦为独立 `file.sign_key`（4 处装配点：/uploads 验签、file 访问 URL、wiki 内嵌图片、免登录分享重签），未配置回退兼容；`signedurl.VerifyAny` + uploads 中间件支持密钥链**平滑轮换**（逗号分隔，首项签发、全链验签），附 signedurl/uploads/storage/config 用例 |
| P2 | wiki 空间/节点删除 | `DeleteSpace` 仅软删空间行，子节点/文档保持活跃、回收站列表不展示子实体；文档级删除已与回收站闭环，空间语义建议文档化 |
| P2 | repository 层 | 仅 wiki 有 sqlmock 轻量测试；建议引入 `glebarez/sqlite` 做内存真实库测试 |
| P2 | 测试工程 | ✅ 完成（第六轮）：`web-admin` 以嵌套 `go.mod` 从 Go 模块隔离（`go list ./...` 已不再含 node_modules）；新增 `scripts/test.sh` / `test.ps1` 统一后端校验入口；CI backend 收敛到 `cmd/internal/pkg` 范围 |
| P3 | `common/auditctx`、`common/response`、`iplocation`、`emailer` | 零覆盖且多为 IO/中间件，成本收益比低，可后续随 handler 层一并覆盖 |

---

## 四、测试现状

### 4.1 测试文件全景（三轮合计）

第一轮（全仓审查报告基线与基建补齐）：`auth_test.go`、`wiki_trash_service_test.go`、`pagination/idgen/ulid/uuid/crypto/apperrors/db`。

第二轮：`contextx_test.go`、`jwt_test.go`、`validator_test.go`、`dashboard_service_test.go`、`announcement_service_test.go`、`wiki_delete_service_test.go` 及 tenant 注销语义用例。

第三轮（handler 层试点）：
| 文件 | 覆盖点 |
|---|---|
| `internal/modules/auth/handler/auth_handler_test.go` | 注册/登录/登出/忘记密码/重置密码的校验 400、成功映射、锁定剩余次数 401、哨兵错误分支、Bearer 头校验 |
| `internal/modules/notification/handler/announcement_handler_test.go` | 创建/更新/改状态/详情/删除/列表的参数校验与转发、分页默认值、上下文发布人/租户注入、错误码 |
| `internal/modules/dashboard/handler/dashboard_handler_test.go` | 总览数据映射、服务错误 500 |
| 接口缝 | `auth`/`notification`/`dashboard` handler 的 service 依赖改为最小接口，`*ConcreteService` 自动满足，module 装配代码零改动 |

第四轮（handler 测试推广 + 基础组件补齐）：

| 文件 | 覆盖点 |
|---|---|
| `internal/modules/wiki/handler/wiki_handler_test.go` | 空间/节点/文档/版本/成员/节点权限/回收站/搜索/附件/预览/统计共 29 用例：URL 上下文传递、参数校验 400（不触达服务层）、分页默认值、哨兵错误到 HTTP 码映射、树/移动/排序路由、导出附件头等 |
| `internal/modules/wiki/handler/wiki_handler_extended_test.go` | 扩展面（标签/评论/分享/模板/通知/订阅/编辑锁/批量/版本对比/导入导出/multipart）22 用例：新增发现——节点创建 body 冗余 required `space_id`（由 validator 先于 handler 注入触发）、批量 action 死代码分支 |
| `internal/modules/tenant/handler/tenant_handler_test.go` | 自助注册/后台管理/回收站/批量/注销审批 46 用例：409/404/400/500 全错误码映射、admin_user 嵌套校验、分页与套餐摘要装配、approver 取自上下文、401（无租户） |
| `internal/modules/tenant/handler/tenant_settings_handler_test.go` | 租户设置读写、401/500 |
| `internal/common/response/response_test.go` | 统一响应契约：状态码与 code 映射表（400/401/403/404/409/429/500）、data 省略、分页结构、结构化错误透传 details/request_id、5 个便捷错误助手 |
| `internal/middleware/middleware_common_test.go` | request_id 注入/透传/自动生成、全局 panic 恢复（含 request_id）、master admin 门控、限流中间件 fail-open 与 disabled 语义 |

### 4.2 覆盖率（2026-09-07 四轮后实测）

- **100%**：`pkg/crypto`、`pkg/idgen`、`pkg/ulid`、`pkg/uuid`、`common/contextx`、`common/signedurl`、`dashboard/service`、`dashboard/handler`
- **97.3%**：`internal/pkg/apperrors`；**93.3%**：`common/response`（第四轮新增）；**91.7%**：`internal/pkg/db`
- **90.9%**：`common/jwt`；**90.3%**：`auth/handler`
- **89.1%**：`pkg/pagination`；**85.7%**：`notification/service`
- **83.8%**：`notification/handler`；**78.6%**：`tenant/handler`（第四轮新增）；**69.9%**：`wiki/handler`（第四轮新增）
- **65.6%**：`common/validator`；**59.2%**：`tenant/service`
- **48.6%**：`common/tenantctx`；**42.6%**：`internal/middleware`（第四轮提升，request_id/panic/admin/限流已覆盖）
- **27.8%**：`wiki/service`（删除/回收站关键路径已覆盖）

### 4.3 运行方式

```bash
go test ./...                                                  # 全仓（exit 0）
go test -cover ./internal/modules/wiki/handler ./internal/modules/tenant/handler
go test ./internal/common/... ./internal/middleware ./internal/modules/auth/... ./internal/modules/notification/... ./internal/modules/dashboard/...
```

说明：全部为单元/HTTP 层测试（sqlmock / 内存 mock / chi httptest / 纯函数），**无需 MySQL/Redis**，可在 CI 直接运行。

---

## 五、测试路线图（建议后续）

1. **handler 层集成测试**（✅ 第八轮收官）：按“service 接口缝 + chi `httptest`”模式已覆盖 **auth / notification / dashboard / wiki / tenant / user / plan / audit / rbac / file 全部 10 个模块**，累计 360+ HTTP 用例，无剩余模块。
2. **repository 真实库**：`glebarez/sqlite` 内存库，替代 sqlmock 覆盖盲区（尤其排序/软删过滤/级联）。
3. **CI 落地**：GitHub Actions 执行 `go vet ./...`、`go test -race`、覆盖率门禁；web-admin 引入 Vitest（当前无前端测试基建）。
