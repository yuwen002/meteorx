# OAuth 第三方登录架构

## 概述

OAuth 模块提供「第三方账号登录 / 注册 / 绑定」能力，并把 provider 配置从静态文件**下沉到数据库**：
平台超管可在后台增删改查、启停、轮换密钥，**免重启热生效**。
凭据以 AES-256-GCM 加密落库，读接口只返回脱敏值。

## 分层与文件

| 层 | 文件 | 职责 |
| --- | --- | --- |
| 入口 | `oauth/module.go` | 装配仓储 / 注册中心 / 服务 / handler，注册三类路由 |
| 模型 | `oauth/model/oauth_account.go`、`oauth_provider.go` | 账号绑定、provider 配置与元数据 |
| 仓储 | `oauth/repository/*` | 账号 CRUD、provider CRUD / 排序 / 启停、模型单一来源 |
| 服务 | `oauth/service/oauth_service.go` | 授权跳转、回调换取用户、绑定 / 解绑、state |
| 服务 | `oauth/service/oauth_provider_service.go` | 配置 CRUD、种子化、热重载、密钥加解密 |
| 服务 | `oauth/service/provider.go` + 各 `*.go` | `Provider` 实现（GitHub / 微信 / 飞书 / 钉钉 / QQ / OIDC …） |
| 接口 | `oauth/handler/*` | 公开登录、账号管理、配置管理三组 handler |

## 数据模型

| 表 | 关键字段 | 说明 |
| --- | --- | --- |
| `oauth_accounts` | `user_id`、`provider`、`provider_user_id`、`union_id` | 用户与第三方账号的绑定关系 |
| `oauth_providers` | `code`（唯一）、`kind`、`client_id`、`client_secret`（密文）、`scopes`、`enabled`、`seeded_from_config` | 运行时配置 |
| `oauth_provider_meta` | `provider_code`、`microsoft_tenant` / OIDC 端点等 | 类型专属扩展参数（避免主表长出一堆空列） |

- `scopes` 使用 GORM `serializer:json` 存储；
- `code` 是运行时标识（固定类型时等于 `kind`；OIDC 由配置 key 派生为 `oidc_<key>`，如 `oidc_acme`）；
- repository 导出 `Models()` 作为 `AutoMigrate` 与「脏列只读巡检」的**单一来源**。

## Provider 抽象与热重载

```
ProviderSpec ──BuildProvider──> Provider(实现)
     ▲                              │
     │                              ▼
config.yaml ─┐              ProviderRegistry
             ├─ Reload  ──>  (RWMutex 快照，Replace 原子替换)
DB 表  ──────┘                      │
                                    ▼
                      OAuthService（按 code 取 Provider 发起授权 / 换取用户）
```

支持的 `kind`：`google` / `github` / `wechat` / `feishu` / `dingtalk` / `qq` / `microsoft` / `oidc`。
未知 kind 在构建时告警并跳过，不影响其他 provider。

**配置优先级与刷新**：

1. 启动时 `ReloadProviders` 从 `config.yaml` 种子化**仅填了 `client_id`** 的项，并写 `seeded_from_config=true`；
2. 管理接口 CRUD 成功后**立即**触发一次刷新；
3. 后台每 30s 兜底刷新，使多副本最终收敛。

种子化标记的意义：管理员删光 DB 配置后，兜底刷新不会让 config 里的旧项「复活」。

## 凭据加密

| 项 | 规则 |
| --- | --- |
| 算法 | AES-256-GCM（`pkg/crypto.SecretCipher`） |
| 密文格式 | `enc:v1:<base64(nonce‖ciphertext)>`，每次加密随机 nonce |
| 主密钥 | `oauth.secret_key`（环境变量 `METEORX_OAUTH_SECRET_KEY`） |
| 未配主密钥 | 保存含 secret 的配置直接返回 **422**，绝不落明文 |
| 读取 | 仅返回 `client_secret_mask` + `has_secret`，明文不出接口 |

密钥轮换：更新配置时若提交了新的 secret 则重新加密，留空表示沿用原密钥。

## 端点

**公开（免登录）**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/auth/oauth/{provider}/redirect` | 取授权跳转地址（同时写入一次性 state） |
| `POST` | `/auth/oauth/callback` | 授权码回调：登录 / 自动注册 / 待补邮箱 |
| `GET` | `/auth/oauth/tenants` | 回调后查询该账号关联的租户列表 |
| `POST` | `/auth/oauth/token/refresh` | 刷新 OAuth 临时令牌 |
| `GET` | `/auth/oauth/providers` | 已启用 provider 列表（登录页按钮数据源） |
| `POST` | `/auth/oauth/complete-email` | 第三方未返回邮箱时补全 |

**需登录（账号管理）**

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/auth/oauth/accounts` | 当前用户已绑定账号 |
| `POST` | `/auth/oauth/bind` | 绑定新账号 |
| `POST` | `/auth/oauth/unbind` | 解绑 |

**平台超管（配置管理，权限码 `admin:oauth_provider:*`）**

| 方法 | 路径 |
| --- | --- |
| `GET` / `POST` | `/admin/oauth-providers` |
| `GET` / `PUT` / `DELETE` | `/admin/oauth-providers/{code}` |
| `PUT` | `/admin/oauth-providers/{code}/status` |

## 安全要点

- **state 一次性**：写入 Redis（TTL 由 `oauth.state_ttl` / `METEORX_OAUTH_STATE_TTL` 控制），
  回调时校验并立即消费，防 CSRF 与重放；
- **待补邮箱**：第三方未返回邮箱时不直接建号，返回 `pending` 令牌（仅存 sessionStorage），
  由 `complete-email` 补齐后才完成登录；
- **回跳地址**：按 `client.base_url` + 固定路径拼装，不信任任意回跳参数；
- **凭据不出接口**：任何读路径都不返回明文 secret；
- **超管角色口径**：`jwt.EnsureSuperadminRole` 保证主账号 JWT 始终携带 `superadmin`，
  避免被显式分配其他角色后丢失运维端点权限。

## 配置

```yaml
oauth:
  secret_key: "..."          # METEORX_OAUTH_SECRET_KEY，AES-256-GCM 主密钥（必须）
  state_ttl: 10m             # METEORX_OAUTH_STATE_TTL
  refresh_token_ttl: 168h    # METEORX_OAUTH_REFRESH_TOKEN_TTL
  providers:                 # 仅作为启动种子，之后以 DB 为准
    github:
      client_id: "..."
      client_secret: "..."
```

## 前端

- 登录页按 `GET /auth/oauth/providers` 动态渲染第三方登录按钮；
- 回调页 `#/auth/{provider}/callback` 处理登录结果，缺邮箱时跳 `#/oauth/complete-email`；
- 个人中心支持绑定 / 解绑第三方账号；
- 平台管理 → 第三方登录（`/system/oauth-provider`，`requireMaster`）：
  列表 / 新建 / 编辑 / 启停 / 删除，按类型动态显示 Microsoft 目录租户或 OIDC 端点字段。

## 测试

```bash
go test -count=1 ./internal/modules/oauth/...
go test -count=1 ./pkg/crypto/...
```

- provider 构建与热重载：`providers_test.go`（未知 kind 跳过、种子化、DB 覆盖 config）；
- 加密与脱敏：`pkg/crypto` 单测覆盖密文格式、随机 nonce、密钥轮换、未配密钥拒绝落库；
- state 与登录：`oauth_service_test.go` 覆盖一次性消费、未知 provider、绑定与刷新失败路径。
