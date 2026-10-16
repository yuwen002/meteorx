# 第三方登录提供商配置管理 API

平台级第三方登录（OAuth）提供商配置管理接口。配置存储于 `oauth_providers` 表，**保存后即时生效，无需重启服务**（多副本部署最多 30 秒内同步至全部副本）。

- 路由前缀：`/api/v1/admin/oauth-providers`
- 权限要求：全部接口需平台超级管理员（`is_master`）且持有 `admin:oauth_provider:*` 权限码
- 安全说明：`client_secret` 以 AES-256-GCM 加密后存储，接口**只返回脱敏值**，任何情况下不返回明文

## 1. 接口总览

| 方法 | 路径 | 说明 | 权限码 |
|------|------|------|--------|
| GET | `/admin/oauth-providers` | 查询全部配置 | `admin:oauth_provider:list` |
| POST | `/admin/oauth-providers` | 新建配置 | `admin:oauth_provider:create` |
| GET | `/admin/oauth-providers/{code}` | 查询单个配置 | `admin:oauth_provider:read` |
| PUT | `/admin/oauth-providers/{code}` | 更新配置 | `admin:oauth_provider:update` |
| DELETE | `/admin/oauth-providers/{code}` | 删除配置 | `admin:oauth_provider:delete` |
| PUT | `/admin/oauth-providers/{code}/status` | 切换启用状态 | `admin:oauth_provider:status` |

## 2. 提供商类型（kind）

| kind | 标识(code) | 说明 | 额外字段 |
|-----|-----------|------|---------|
| `google` | 固定 `google` | Google 登录 | - |
| `github` | 固定 `github` | GitHub 登录 | - |
| `wechat` | 固定 `wechat` | 微信扫码登录（**不返回邮箱**，需补全） | - |
| `feishu` | 固定 `feishu` | 飞书登录 | - |
| `dingtalk` | 固定 `dingtalk` | 钉钉登录 | - |
| `qq` | 固定 `qq` | QQ 登录（**不返回邮箱**，需补全） | - |
| `microsoft` | 固定 `microsoft` | Microsoft Entra ID | `tenant` |
| `oidc` | 自定义 `oidc_<key>` | 通用企业 OIDC | `auth_url` / `token_url` / `userinfo_url` / `scopes` |

内置类型的 `code` 必须等于 `kind` 且不可修改；`oidc` 的 `code` 须匹配 `^oidc_[a-z][a-z0-9_]{2,63}$`。

## 3. 查询配置列表

`GET /api/v1/admin/oauth-providers`

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "providers": [
      {
        "code": "wechat",
        "kind": "wechat",
        "display_name": "微信",
        "enabled": true,
        "client_id": "wx1234567890abcdef",
        "client_secret_mask": "************cret",
        "has_secret": true,
        "redirect_url": "http://localhost:5173/auth/wechat/callback",
        "editable_endpoint": false,
        "created_at": "2026-10-16T10:00:00Z",
        "updated_at": "2026-10-16T10:00:00Z"
      }
    ]
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `client_secret_mask` | string | 脱敏后的 secret（仅保留末尾 4 位），未设置时为空串 |
| `has_secret` | bool | 是否已设置 secret |
| `editable_endpoint` | bool | 是否可自行配置端点（仅 `oidc` 为 true） |

## 4. 新建配置

`POST /api/v1/admin/oauth-providers`

**请求体（以微信为例）：**
```json
{
  "kind": "wechat",
  "code": "wechat",
  "display_name": "微信",
  "enabled": true,
  "client_id": "wx1234567890abcdef",
  "client_secret": "your-app-secret",
  "redirect_url": "http://localhost:5173/auth/wechat/callback"
}
```

**请求体（通用 OIDC）：**
```json
{
  "kind": "oidc",
  "code": "oidc_acme",
  "display_name": "Acme SSO",
  "enabled": true,
  "client_id": "acme-client-id",
  "client_secret": "acme-secret",
  "redirect_url": "http://localhost:5173/auth/oidc_acme/callback",
  "auth_url": "https://idp.example.com/oauth2/authorize",
  "token_url": "https://idp.example.com/oauth2/token",
  "userinfo_url": "https://idp.example.com/oauth2/userinfo",
  "scopes": ["openid", "email", "profile"]
}
```

**请求体（Microsoft Entra ID，多一个 tenant）：**
```json
{
  "kind": "microsoft",
  "code": "microsoft",
  "enabled": true,
  "client_id": "azure-app-id",
  "client_secret": "azure-secret",
  "tenant": "common",
  "redirect_url": "http://localhost:5173/auth/microsoft/callback"
}
```

**业务规则：**
- `kind` / `code` / `client_id` / `redirect_url` 必填；`oidc` 额外要求 `auth_url` / `token_url` / `userinfo_url` 且必须为 http(s) 地址
- `client_secret` 新建时建议填写；服务端未配置 `oauth.secret_key` 时返回 **422**，拒绝保存明文
- 保存成功后立即热生效，无需重启

**错误响应：**

| 状态码 | 场景 |
|--------|------|
| 400 | 参数非法（类型不支持 / 标识不匹配 / 缺少必填项 / OIDC 端点缺失或非 http） |
| 409 | 该 `code` 已存在 |
| 422 | 服务端未配置加密主密钥，拒绝保存明文 secret |

## 5. 更新配置

`PUT /api/v1/admin/oauth-providers/{code}`

**业务规则：**
- 请求体与新建一致，`code` 不可修改（路径参数即目标）
- **`client_secret` 留空表示保持原值不变**；填写新值则替换（支持密钥轮换）
- 其余字段全量覆盖，保存后立即热生效

## 6. 删除配置

`DELETE /api/v1/admin/oauth-providers/{code}`

**业务规则：**
- 删除后该 provider 立即从登录页消失，已绑定用户无法再通过它登录
- 配置被删光后**不会**自动从 config.yaml 恢复（种子化仅执行一次）

## 7. 切换启用状态

`PUT /api/v1/admin/oauth-providers/{code}/status`

**请求体：**
```json
{ "enabled": false }
```

**业务规则：**
- `enabled` 为布尔值且**必填**（传布尔字面量，缺省字段视为非法请求）
- 停用后登录页不再展示该按钮，但已绑定用户仍可正常登录
- 保存后立即热生效

## 8. 与 config.yaml 的关系

| 阶段 | 行为 |
|------|------|
| 首次启动（表为空） | 自动把 config.yaml 中**已填写 `client_id`** 的 provider 种子化到 DB（secret 加密存储），并写入种子化标记 |
| 之后 | DB 为唯一配置来源；config.yaml 不再被读取用于 provider 构建 |
| 管理员删除配置 | 不会再次种子化（避免「删除后复活」） |
| 未配置主密钥升级 | 已有明文 secret 可正常读取（无前缀视为历史明文），但无法保存新 secret |

## 9. 登录页联动

登录页按钮由 `GET /api/v1/auth/oauth/providers` 驱动（见 [auth-api.md](./auth-api.md)），该接口仅返回**已启用**的 provider。因此在后台启用/停用后，登录页会在 30 秒内自动同步，无需刷新令牌或重启。
