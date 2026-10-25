# MeteorX 浏览器 E2E

真实浏览器（Playwright）→ 前端生产构建（`vite preview`）→ 真实后端 → 真实 MySQL / Redis。
与 `internal/integration`（sqlite + miniredis 的进程内集成）互补：这里验证的是**部署形态**。

## 覆盖范围（8 条冒烟）

| 用例 | 断言 |
| --- | --- |
| 未登录访问后台 | 重定向到登录页 |
| 密码错误 | 页面给出错误提示且停留在登录页 |
| 临时超管登录 | 跳转工作台、侧边栏可见 |
| 新增套餐 | 接口 200、列表可见、**直查 MySQL** 校验字段、用例结束自动物理删除 |
| 消息中心分类页签 | 切到「任务」→ 刷新仍停在该分类，且请求带 `category=task` |
| 分类记忆清除 | 清键后回到「全部」，与铃铛「查看全部消息」同一口径 |
| 时间线深链（`?user_id=`） | 用非登录用户验证 query 被真正消费，并回填查询框；无 query 时回退登录用户 |
| 租户套餐深链（`?tenant_id=`） | 按 ID 取详情后直接开「套餐配置」弹窗，不要求该行在当前页 |

## 前置条件

1. MySQL 与 Redis 已运行（配置见根目录 `.env` / `config.yaml`）
2. 后端已启动并监听 `8081`：
   ```powershell
   go run ./cmd/server
   ```
3. 前端已构建（`web-admin/dist` 由 Playwright 的 `webServer` 用 `vite preview` 托管）：
   ```powershell
   cd web-admin
   npm ci
   npm run build
   ```

## 首次安装

```powershell
cd e2e
npm ci
```

## 运行

本机使用系统 Chrome，无需下载 Playwright 浏览器：

```powershell
$env:E2E_CHANNEL='chrome'
npx playwright test
```

只跑冒烟：

```powershell
$env:E2E_CHANNEL='chrome'
npx playwright test --grep @smoke
```

CI 不设置 `E2E_CHANNEL`，使用 `npx playwright install --with-deps chromium` 安装的自带浏览器。

## 数据隔离与清理

- `globalSetup` 直接向 `users` 表插入**临时超管**（`is_master=true`），不改动任何既有账号、不依赖 admin 密码；
- 深链用例通过夹具 `list-tenants` 取一个未删除的租户 ID（只读，不新建也不修改租户）；
- 用例数据带 `Date.now()` 唯一后缀，`finally` 中通过夹具物理删除；
- `globalTeardown` 删除临时超管并清空 `.auth/`，即使测试失败也会执行；
- `.auth/`、`node_modules/`、`test-results/`、`playwright-report/` 均已 gitignore。

## 可配置环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `E2E_API_URL` | `http://127.0.0.1:8081` | 后端地址（globalSetup 检查 `/health`） |
| `E2E_WEB_PORT` | `4173` | 前端 preview 端口 |
| `E2E_BASE_URL` | `http://127.0.0.1:4173` | 浏览器访问地址 |
| `E2E_CHANNEL` | 空（chromium） | 设为 `chrome` 使用系统 Chrome |

## 故障排查

- **globalSetup 报"需要后端运行"**：先启动后端，或检查 `E2E_API_URL`；
- **preview 启动超时**：确认 `web-admin/dist` 存在且 `npm run build` 成功；
- **登录失败**：确认 Redis 在线（登录限流/锁定依赖 Redis），并查看后端日志；
- **失败现场**：`e2e/test-results/`（trace + 截图），或 `npx playwright show-report`。
