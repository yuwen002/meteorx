---
alwaysApply: true
scene: git_message
---

## 提交信息规范

### 格式要求

提交信息必须严格遵循 `type(scope): 描述` 格式，**scope（模块名）必填**，不允许裸 type。

### 类型（type）

- `feat` — 新功能
- `fix` — 修复 Bug
- `refactor` — 重构（不改变功能、不修复 Bug）
- `docs` — 文档变更
- `style` — 代码风格调整（空格、缩进等，不影响逻辑）
- `perf` — 性能优化
- `test` — 增加或修改测试
- `chore` — 构建/工具/依赖变更
- `ci` — CI 配置变更
- `revert` — 回退提交

### 范围（scope）

scope 用于标识本次提交涉及的模块，从以下值中选择：

| scope | 说明 |
|-------|------|
| `auth` | 认证模块 |
| `user` | 用户管理 |
| `tenant` | 租户管理 |
| `rbac` | 权限/角色 |
| `audit` | 审计日志 |
| `wiki` | Wiki 模块 |
| `file` | 文件模块 |
| `dashboard` | 仪表盘 |
| `plan` | 套餐/订阅 |
| `notification` | 通知/公告 |
| `oauth` | 第三方登录 |
| `db` | 数据库/迁移 |
| `middleware` | 中间件 |
| `config` | 配置 |
| `deploy` | 部署/运维 |
| `ui` | 前端页面/组件 |
| `api` | 接口/路由 |
| `other` | 跨模块或不明确的变更 |

### 描述

- 冒号后空一格，然后用中文简要描述本次提交做了什么
- 尽量控制在 50 字以内，说清楚改动的目的和内容

### 示例

✅ **正确**：
- `feat(wiki): 新增页面评论功能`
- `fix(auth): 修复 OAuth 回调参数丢失问题`
- `refactor(rbac): 抽取权限校验为独立中间件`
- `feat(db): 增加租户设置表迁移脚本`
- `fix(user): 修复重置密码链接过期判断`
- `chore(deploy): 更新 Docker Compose 镜像版本`

❌ **错误**：
- `feat: 新增功能`（缺少 scope）
- `fix: bugfix`（缺少 scope 且描述不清晰）
- `feat(MAAS): 新功能`（scope 不是约定值）
