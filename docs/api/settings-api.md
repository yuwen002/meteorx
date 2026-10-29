# 平台运行时配置中心 API

平台级运行时配置（注册开关、维护模式等）的管理与公开读取接口。配置存储于 `system_settings` 表并叠加 Redis 整表缓存（TTL 5 分钟），**通过管理接口保存后即时失效缓存生效，无需重启服务**；直接改库路径最迟 TTL 内同步。

- 管理路由前缀：`/api/v1/admin/settings`（仅平台超级管理员，`requireMaster` + 权限码）
- 公开路由：`/api/v1/settings/public`（免鉴权，登录/注册页消费）
- 白名单模式：仅 `model/registry.go` 登记过的键允许读写，未登记键的更新请求直接拒绝（400），杜绝「可编辑却无效」的死配置

## 1. 接口总览

| 方法 | 路径 | 说明 | 权限码 |
|------|------|------|--------|
| GET | `/admin/settings` | 配置列表（含元数据与来源） | `admin:settings:list` |
| PUT | `/admin/settings` | 批量更新配置 | `admin:settings:update` |
| GET | `/settings/public` | 公开配置（免鉴权） | - |

## 2. 已登记配置键

| 键 | 类型 | 默认值 | 公开 | 说明 |
|----|------|--------|------|------|
| `registration_enabled` | bool | `true` | ✅ | 是否开放自助注册；关闭后 `/auth/register` 与 `/tenants/register` 返回 403，前端隐藏注册入口 |
| `maintenance_mode` | bool | `false` | ✅ | 站点维护模式；开启后已登录普通用户的业务接口返回 503（平台超管豁免，保证能进后台关闭维护） |
| `maintenance_message` | string | `系统维护中，请稍后再试` | ✅ | 维护模式提示文案，随 503 响应与登录页公告展示 |

## 3. 查询配置列表

`GET /api/v1/admin/settings`

**成功响应（200）：**
```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "key": "registration_enabled",
      "value": "true",
      "value_type": "bool",
      "default_value": "true",
      "is_public": true,
      "source": "db",
      "description": "是否开放用户自助注册（关闭后注册接口返回 403，前端隐藏注册入口）",
      "updated_at": "2026-10-28 09:00:00"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `value` | string | 当前生效值（字符串形态，按 `value_type` 解析） |
| `source` | string | `db`=已自定义 / `default`=回落注册表默认值（DB 无行） |
| `is_public` | bool | 是否随公开接口暴露给未登录前端 |

## 4. 批量更新配置

`PUT /api/v1/admin/settings`

**请求体：**
```json
{
  "settings": [
    { "key": "registration_enabled", "value": "false" },
    { "key": "maintenance_message", "value": "今晚 22:00 升级" }
  ]
}
```

**业务规则：**
- 整批校验：任一键未登记或值与类型不符（如 bool 键传 `"yes"`）则**整批拒绝**，不产生半写状态
- 值按类型归一化后落库（bool 统一存 `"true"`/`"false"`）
- 成功后自动失效整表缓存并返回更新后的**全量列表**（前端以此重建基线）

**错误响应：**

| 状态码 | 场景 |
|--------|------|
| 400 | 键未登记（`unknown setting key`）/ 值类型非法（`invalid setting value`）/ 更新项为空 |
| 403 | 非平台超管或缺少权限码 |

## 5. 公开配置

`GET /api/v1/settings/public`

免鉴权、且不在维护模式拦截范围内（登录页需要读取公告展示）。仅返回 `is_public=true` 的键，值按类型还原为原生 JSON 类型：

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "registration_enabled": true,
    "maintenance_mode": false,
    "maintenance_message": "系统维护中，请稍后再试"
  }
}
```

## 6. 消费点与生效行为

| 消费点 | 位置 | 行为 |
|--------|------|------|
| 用户自助注册 | `POST /auth/register` | 关闭时 403「自助注册已关闭，请联系管理员开通」 |
| 租户自助开户 | `POST /tenants/register` | 同上；后台 `AdminCreate` 建租户**不受开关限制** |
| 维护模式闸门 | `middleware.MaintenanceGate`（Protected 组） | 开启时非超管业务请求 503 + 可配文案（空文案回落默认） |
| 登录页 | `web-admin/src/views/login/index.vue` | 公告 el-alert 渲染、注册入口隐藏；配置接口失败时 fail-open 保持默认展示 |
| 注册页 | `register/index.vue`、`register-user/index.vue` | 直达 URL 时读取公开配置，关闭则提示并跳回登录页 |

**读取回退链**：DB 行 → 注册表默认值 →（读取本身失败时）消费者各自按「失败开放」口径处理（注册开关回退默认值、维护模式按未开启处理），配置中心故障不阻断主链路。

## 7. 新增配置键流程

1. `internal/modules/settings/model/registry.go` 登记键（常量 + SettingDef）
2. 实装真实消费者（service gate / 中间件），**先接消费者再登记**，避免死配置
3. `SeedDefaults` 启动播种自动覆盖新键，无需手写迁移
4. 前端配置页零改动（按列表动态渲染，编辑器按 `value_type` 选择）
