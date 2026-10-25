# 测试手册（Test Playbook）

> 目标：任何一次改动，都能在 1 分钟内确定「该跑什么、怎么跑、看什么」。
> 基线数据截至 2026-10-24，结构性变化（新增层级 / 门槛调整）时同步更新本文。

## 1. 分层矩阵

| 层级 | 覆盖内容 | 入口 | 依赖 | 本机耗时 |
| --- | --- | --- | --- | --- |
| Go 单测 | 183 个 `*_test.go`，handler/service/repository/middleware | `scripts/test.ps1` | 无 | 2~5 min |
| Go 竞态 | 同上的 `-race` 版本 | CI `test` job | Linux + CGO | 仅 CI |
| 进程内集成 | `testkit` 全量真实路由（sqlite + miniredis） | `go test ./internal/integration/...` | 无 | 1~2 min |
| MySQL 方言 | `TIMESTAMPADD`/`JSON_CONTAINS`/`FIELD()` 等真库语义 | `METEORX_MYSQL_DIALECT_TEST=1` | MySQL（库名以 `_test` 结尾） | < 30 s |
| 迁移 | AutoMigrate + Seed 在空库跑通 | `go run ./cmd/server migrate` 或 CI `migrate` job | MySQL | < 30 s |
| 前端单测 | 20 个 `*.test.ts`，api/stores/utils/composables 层 | `npm run test:coverage` | Node | 20~40 s |
| 前端静态 | ESLint + `vue-tsc --noEmit` + 生产构建 | `npm run lint` / `npx vue-tsc` / `npm run build` | Node | 1~2 min |
| 浏览器 E2E | 真实 Chrome → `vite preview` → 后端 → MySQL/Redis | `e2e/`（见 [README](../../e2e/README.md)） | MySQL/Redis/后端/Chrome | 30~60 s |
| 安全 | govulncheck + Trivy | CI `security` job | 网络 | 仅 CI |

## 2. 变更类型 → 最小必跑集

| 改动位置 | 必跑 |
| --- | --- |
| Go handler/service/repository | `scripts/test.ps1` |
| 涉及 MySQL 专属 SQL（JSON/日期函数/排序） | 上一行 + `METEORX_MYSQL_DIALECT_TEST=1` 方言测试 |
| 路由 / 中间件 / RBAC 权限 | `scripts/test.ps1`（integration 已覆盖）+ 相关 E2E 冒烟 |
| 数据库模型 / 迁移 | `scripts/test.ps1` + 本地 `go run ./cmd/server migrate` |
| 前端视图 / 组件 / store | `npm run lint` + `npx vue-tsc --noEmit` + `npm run test:coverage` + `npm run build` |
| 前端 api/utils/composables | 同上（覆盖率门槛会拦截） |
| 登录 / 套餐 / 租户等关键 UI 流程 | 上述前端四项 + `npx playwright test` |
| 配置 / 启动 / 依赖注入 | `scripts/test.ps1` + 本地启动冒烟（`/health` 200） |
| CI workflow / 脚本 | 本地执行等价命令 + YAML 结构解析 |
| 纯文档 / 注释 | 不需要跑测试 |

## 3. 一键命令

### 本地全量（Windows，推荐）

```powershell
# 后端 + 前端（不含 E2E）
powershell -File scripts\test-all.ps1

# 追加浏览器 E2E：需先启动 MySQL/Redis/后端（8081），并已构建 web-admin/dist
powershell -File scripts\test-all.ps1 -WithE2E

# 只跑后端
powershell -File scripts\test-all.ps1 -SkipFrontend
```

### 本地全量（Linux / macOS，与 CI 同口径）

```bash
bash scripts/test.sh          # vet + build + go test -race -count=1 + coverage.out
cd web-admin && npm ci && npm run lint && npx vue-tsc --noEmit && npm run test:coverage && npm run build
```

### 单项命令

```powershell
# Go 全量（Windows 无 -race）
go test -count=1 ./cmd/... ./internal/... ./pkg/...

# 只跑某个包
go test -count=1 -v ./internal/modules/plan/...

# MySQL 方言（库名必须以 _test 结尾）
$env:METEORX_MYSQL_DIALECT_TEST='1'; go test -count=1 ./internal/modules/task/repository

# 前端
cd web-admin
npm run lint; npx vue-tsc --noEmit; npm run test:coverage; npm run build

# 浏览器 E2E
cd e2e
$env:E2E_CHANNEL='chrome'; npx playwright test
```

## 4. E2E 前置检查清单

- [ ] MySQL `3306` 在监听
- [ ] Redis `6379` 在监听且密码与 `.env` 一致（当前为空密码）
- [ ] 后端 `8081` 已启动：`go run ./cmd/server`，`/health` 返回 200
- [ ] `web-admin/dist` 已构建（`npm run build`）
- [ ] `e2e/node_modules` 已安装（`npm ci`）
- [ ] 本机有 Chrome，或已执行 `npx playwright install chromium`
- [ ] 不需要知道 admin 密码：E2E 用临时超管夹具，测试结束自动删除

## 5. 发布前验收清单

- [ ] `scripts/test.ps1` 通过（CI 等价物带 `-race`）
- [ ] 前端 lint / type-check / coverage / build 全部通过
- [ ] 涉及 UI 或接口契约：E2E 冒烟通过（4 条）
- [ ] 涉及模型：`migrate` job 通过
- [ ] Go 覆盖率 ≥ 35%，前端覆盖率不低于 85/70/95/85（lines/functions/branches/statements）
- [ ] `CHANGELOG.md` 已记录
- [ ] 工作区无测试残留：`e2e/.auth/`、临时账号、测试套餐
- [ ] `git status` 只包含预期改动

## 6. 故障速查

| 现象 | 原因 | 处理 |
| --- | --- | --- |
| 登录接口 30s+ 超时 | Redis 未启动（限流/锁定走缓存） | 启动 Redis；本地无 Redis 时可用 miniredis 顶替 |
| 后端日志停写、`/health` 挂起 | OTel stdout + GORM SQL 经 PowerShell 管道反压 | `OTEL_SDK_DISABLED=true`、`METEORX_DB_DEBUG=false`，用 `*>` 重定向后台启动 |
| Windows 上 `-race` 不可用 | 缺少 gcc/CGO | 正常，race 只在 CI 跑；本地以 `test.ps1` 为准 |
| `compile: version "goX" does not match go tool version "goY"` | 本机 Go 版本低于 `go.mod` 的 `go` 指令，`GOTOOLCHAIN=auto` 已切到新工具链，但覆盖率收集时嵌套的 `go tool covdata` 从 PATH 取到了旧版 go（只影响 `-coverprofile`） | `scripts/test.ps1` 已自动把 `go env GOROOT\bin` 前置到 PATH；若手工跑，先执行同样前置，或直接安装与 `go.mod` 一致的 Go |
| E2E 打开页面但接口全 404 | `vite preview` 未代理 `/api` | 确认 `web-admin/vite.config.ts` 的 `preview.proxy` |
| Playwright 报 strict mode violation | 选择器匹配多个元素（如 `.el-menu` 嵌套） | 用 `.first()` 或更精确的选择器 |
| Chromium 未安装 | 未执行 `playwright install` | 本机设 `E2E_CHANNEL=chrome`，或安装 chromium |
| MySQL 方言用例全部 skipped | 未设开关或库名不以 `_test` 结尾 | `METEORX_MYSQL_DIALECT_TEST=1` + 测试库 |
| E2E 结束库里仍有 `e2e_admin_*` | globalTeardown 未执行（进程被强杀） | 手动 `go run ./e2e/fixtures/fixture delete-admin <用户名>` |
| PowerShell 脚本乱码/解析失败 | 5.1 把无 BOM UTF-8 当 ANSI | `.ps1` 保持 ASCII-only |
| 端口被占用 | 残留 go/node 进程 | 检查 `3306/6379/8081/4173` 的监听 PID 后结束 |

## 7. 当前基线（2026-10-25）

- Go 测试文件：**183**（全绿）；集成套件：`internal/integration/` 21 个文件
- Go 总覆盖率：**50.6%**（`go tool cover "-func=coverage.out"` 实测）
- 前端测试文件：**20**；覆盖率门槛仅统计 `api/utils/composables`（实测 99.6 / 97.2 / 99.25 / 99.6）
- 浏览器 E2E：**4 条**冒烟（认证 3 + 套餐 1）
- CI jobs：**10**（lint / test / security / migrate / build / frontend / e2e / docker / deploy / release）
- 覆盖率门槛：Go **35%**（实际已 50.6%）；前端 **85 / 70 / 95 / 85**

已知短板：前端页面组件（`views/`）不在覆盖率统计范围内，改动页面时需靠 E2E 与手工验证兑付。

## 8. 相关文档

- [E2E 运行手册](../../e2e/README.md)
- [CI 流水线](../../.github/workflows/ci-cd.yml)
- [配置与启动架构](../architecture/config.md)
