# 审计日志增强功能文档

> 本文档描述审计日志模块的增强功能，包括 IP 地理位置解析、告警管理和会话分析。

---

## 1. 功能概览

### 1.1 IP 地理位置解析

自动解析客户端 IP 的地理位置信息，记录到审计日志中。

**实现方式：**
- 使用 HTTP API（ip-api.com）进行在线解析
- 异步解析，不阻塞请求响应
- 支持降级处理（解析失败时不记录地理位置）

**数据结构：**
```go
type IPLocation struct {
    Country   string // 国家
    Region    string // 省份
    City      string // 城市
    FullText  string // 完整文本：国家 省份 城市
}

type IPLocator interface {
    Locate(ctx context.Context, ip string) (*IPLocation, error)
}
```

**使用示例：**
```go
// 初始化
ipLocator := iplocation.NewHTTPLocator("ip-api", 3*time.Second)

// 在中间件中使用
loc, err := ipLocator.Locate(ctx, clientIP)
if err == nil && loc != nil {
    auditLog.IPLocation = loc.FullText
}
```

---

### 1.2 告警管理

基于审计日志的实时告警系统，支持多种触发条件和通知渠道。

#### 1.2.1 告警规则

**触发类型：**
| 类型 | 说明 | 触发值示例 |
|------|------|------------|
| risk_level | 按风险等级触发 | low/medium/high/critical |
| action | 按操作类型触发 | create/update/delete/login |
| user | 按特定用户触发 | user_id |

**通知渠道：**
| 渠道 | 说明 | 目标格式 |
|------|------|----------|
| email | 邮件通知 | 邮箱地址 |
| dingtalk | 钉钉机器人 | Webhook URL |
| wechat | 企业微信机器人 | Webhook URL |
| webhook | 自定义 Webhook | HTTP URL |

**冷却机制：**
- 每个规则可设置冷却时间（默认 30 分钟）
- 冷却期内同一规则不会重复触发告警
- 避免告警风暴

#### 1.2.2 通知模板

支持 Go 模板语法，可用变量：
- `{{.Username}}` - 用户名
- `{{.Action}}` - 操作类型
- `{{.RiskLevel}}` - 风险等级
- `{{.Message}}` - 告警消息
- `{{.CreatedAt}}` - 触发时间

**示例：**
```
【告警】用户 {{.Username}} 执行了 {{.Action}} 操作，风险等级：{{.RiskLevel}}
```

#### 1.2.3 通知内容格式

**Email 通知：**
- HTML 格式邮件
- 包含完整的告警详情表格
- 美观的样式和配色

**DingTalk/WeChat 通知：**
- Markdown 格式消息
- 支持富文本显示
- 风险等级颜色标识

**Webhook 通知：**
- JSON 格式数据
- 包含所有告警字段
- 便于自定义处理

---

### 1.3 会话分析

追踪用户的完整操作会话，提供会话统计和操作时间线分析。

#### 1.3.1 会话追踪

- 每次登录生成新的 session_id
- 同一会话的所有操作自动关联
- 支持按会话 ID 查询完整操作时间线

#### 1.3.2 会话统计

| 指标 | 说明 |
|------|------|
| total_requests | 总请求数 |
| success_count | 成功请求数 |
| failure_count | 失败请求数 |
| avg_duration | 平均耗时（毫秒） |
| duration_minutes | 会话时长（分钟） |

#### 1.3.3 操作时间线

按时间顺序展示会话中的所有操作，包括：
- 操作类型和模块
- HTTP 方法和路径
- 状态码和结果
- 客户端 IP 和地理位置
- 请求耗时
- 错误信息（如有）

---

## 2. 数据库表结构

### 2.1 审计日志表（audit_logs）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string(26) | 日志 ID |
| user_id | string(26) | 操作用户 ID |
| username | string(50) | 操作用户名 |
| tenant_id | string(26) | 租户 ID |
| module | string(50) | 操作模块 |
| action | string(20) | 操作类型 |
| resource | string(100) | 操作资源 |
| resource_id | string(26) | 被操作资源 ID |
| method | string(10) | HTTP 方法 |
| path | string(255) | 请求路径 |
| request_body | text | 请求参数 |
| response_body | text | 响应结果 |
| status_code | int | HTTP 状态码 |
| result | string(10) | 操作结果 |
| error_message | text | 错误信息 |
| client_ip | string(50) | 客户端 IP |
| ip_location | string(100) | IP 地理位置 |
| user_agent | string(255) | 用户代理 |
| device_info | string(255) | 设备信息 |
| duration | int64 | 请求耗时（毫秒） |
| session_id | string(50) | 会话 ID |
| request_id | string(50) | 请求 ID |
| trace_id | string(50) | 链路追踪 ID |
| referer | string(500) | 来源页面 |
| risk_level | string(20) | 风险等级 |
| tags | string(500) | 标签（JSON 数组） |
| created_at | datetime | 创建时间 |

### 2.2 告警规则表（audit_alert_rules）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string(26) | 规则 ID |
| name | string(100) | 规则名称 |
| description | string(500) | 规则描述 |
| enabled | bool | 是否启用 |
| trigger_type | string(20) | 触发类型 |
| trigger_value | string(100) | 触发值 |
| notify_channels | string(500) | 通知渠道（JSON） |
| notify_targets | text | 通知目标（JSON） |
| notify_template | text | 通知模板 |
| cooldown_minutes | int | 冷却时间（分钟） |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

### 2.3 告警记录表（audit_alerts）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string(26) | 告警 ID |
| rule_id | string(26) | 规则 ID |
| rule_name | string(100) | 规则名称 |
| audit_log_id | string(26) | 审计日志 ID |
| user_id | string(26) | 触发用户 ID |
| username | string(50) | 触发用户名 |
| risk_level | string(20) | 风险等级 |
| action | string(20) | 操作类型 |
| message | text | 告警消息 |
| notified | bool | 是否已通知 |
| notify_time | datetime | 通知时间 |
| created_at | datetime | 创建时间 |

---

## 3. API 接口

### 3.1 告警管理

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| POST | `/audit/alert-rules` | 创建告警规则 | `audit:alert-rule:create` |
| GET | `/audit/alert-rules` | 获取所有告警规则 | `audit:alert-rule:list` |
| GET | `/audit/alert-rules/{id}` | 告警规则详情 | `audit:alert-rule:read` |
| PUT | `/audit/alert-rules/{id}` | 更新告警规则 | `audit:alert-rule:update` |
| DELETE | `/audit/alert-rules/{id}` | 删除告警规则 | `audit:alert-rule:delete` |
| GET | `/audit/alerts` | 告警记录列表 | `audit:alert:list` |
| GET | `/audit/alerts/{id}` | 告警记录详情 | `audit:alert:read` |

### 3.2 会话分析

| 方法 | 路径 | 功能 | 权限码 |
|------|------|------|--------|
| GET | `/audit/sessions` | 会话摘要列表 | `audit:session:list` |
| GET | `/audit/sessions/{id}/logs` | 获取会话的所有日志 | `audit:session:read` |

---

## 4. 前端页面

### 4.1 告警管理页面

**路径：** `/system/audit/alert`

**功能：**
- 告警规则 CRUD
- 告警记录查询
- 统计卡片（规则数、已启用、今日告警、已通知）
- 支持按规则、用户、风险等级筛选

### 4.2 会话分析页面

**路径：** `/system/audit/session`

**功能：**
- 会话列表
- 会话详情（操作时间线）
- 支持按用户筛选
- 会话统计展示

---

## 5. 测试用例

### 5.1 告警服务测试

文件：`internal/modules/audit/service/alert_service_test.go`

**测试覆盖：**
- 创建/更新/删除规则
- 规则匹配逻辑
- 模板渲染
- 风险等级标签和颜色

### 5.2 会话服务测试

文件：`internal/modules/audit/service/session_service_test.go`

**测试覆盖：**
- 获取会话日志
- 会话列表查询
- 平均耗时计算
- 空会话处理

---

## 6. 配置说明

### 6.1 IP 地理位置解析

```go
// 使用 ip-api.com（免费，无需 API Key）
ipLocator := iplocation.NewHTTPLocator("ip-api", 3*time.Second)

// 使用 ipinfo.io（需要 API Key）
ipLocator := iplocation.NewHTTPLocator("ipinfo", 3*time.Second)
```

### 6.2 邮件通知

在 `config.yaml` 中配置 SMTP：

```yaml
email:
  host: smtp.example.com
  port: 587
  username: your-email@example.com
  password: your-password
  from: your-email@example.com
  from_name: MeteorX 审计系统
```

### 6.3 钉钉/企业微信通知

在告警规则中配置 Webhook URL：

```
https://oapi.dingtalk.com/robot/send?access_token=xxx
https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx
```

---

## 7. 性能优化

### 7.1 异步处理

- IP 地理位置解析：异步执行，不阻塞请求
- 告警规则匹配：异步执行，不影响主流程
- 通知发送：异步执行，使用 goroutine

### 7.2 批量写入

- 审计日志使用批量处理器（BatchProcessor）
- 减少数据库写入压力
- 支持优雅关闭和超时控制

### 7.3 冷却机制

- 避免短时间内重复告警
- 可配置的冷却时间
- 减少通知渠道的调用频率

---

## 8. 安全考虑

### 8.1 权限控制

- 所有管理接口都需要相应权限码
- 需要超级管理员权限才能访问
- 支持 RBAC 细粒度权限控制

### 8.2 数据脱敏

- 密码字段自动脱敏
- Token 字段自动脱敏
- 敏感字段通过 `AuditIgnore` 标签标记

### 8.3 日志清理

- 支持自动清理过期日志
- 默认保留 90 天
- 可配置的清理策略