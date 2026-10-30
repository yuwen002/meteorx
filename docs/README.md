# MeteorX 文档

## 📚 文档目录

### API 接口文档
- [认证 API](api/auth-api.md) - 注册、登录、Token 管理
- [用户 API](api/user-api.md) - 用户 CRUD、个人中心
- [租户 API](api/tenant-api.md) - 租户管理、注销审批
- [RBAC API](api/rbac-api.md) - 角色权限管理
- [文件 API](api/file-module-api.md) - 文件上传下载
- [套餐 API](api/plan-api.md) - 套餐管理
- [审计 API](api/audit-api.md) - 审计日志、告警管理、会话分析
- [Wiki API](api/wiki-api.md) - 知识库管理
- [运营看板 API](api/dashboard-api.md) - 数据统计
- [公告 API](api/announcement-api.md) - 通知公告
- [邀请 API](api/invitation-api.md) - 成员邀请与接受注册
- [任务 API](api/task-api.md) - 个人待办与团队协作任务
- [导出 API](api/export-api.md) - 异步数据导出与下载中心
- [站内消息 API](api/message-api.md) - 收件箱、未读角标、离线补推、已读与批量已读/删除、重复投递去重与接收偏好
- [OAuth 提供商 API](api/oauth-provider-api.md) - 第三方登录提供商配置管理
- [系统配置 API](api/settings-api.md) - 平台运行时配置中心（注册开关/维护模式）
- [协作即时消息 API](api/chat-api.md) - 私聊/群聊、历史分页、已读回执、@提及、图片/文件消息
- [系统与运维 API](api/system-api.md) - 健康检查、Prometheus 指标、CSP 违规上报

### 架构设计文档
- [多租户隔离架构](architecture/tenant.md)
- [RBAC 权限架构](architecture/permission.md)
- [错误/响应规范](architecture/error.md)
- [数据库/事务架构](architecture/database.md)
- [审计架构](architecture/audit.md)
- [Wiki 架构](architecture/wiki.md)
- [任务管理架构](architecture/task.md) - 访问控制、状态副作用、重复周期与提醒
- [数据导出架构](architecture/export.md) - 执行租约、多格式输出与过期清理
- [第三方登录架构](architecture/oauth.md) - 配置下沉 DB、凭据加密与热重载
- [运营看板架构](architecture/dashboard.md) - 趋势分桶、健康度评分与慢查询
- [站内消息架构](architecture/message.md) - 按人写行、写放大封顶、已读幂等、按天事件去重与接收偏好
- [分页/排序/过滤架构](architecture/pagination.md)
- [配置/启动架构](architecture/config.md)
- [ID/ULID 统一架构](architecture/idgen.md)

### 开发与测试
- [测试手册（分层矩阵 / 一键命令 / 故障速查）](development/testing.md) - 改动前后该跑什么、怎么跑、看什么
- [E2E 运行手册](../e2e/README.md) - Playwright + 真实 MySQL/Redis 的浏览器冒烟

### 其他文档
- [项目结构说明](project-structure.md)
- [本地 MinIO 联调云存储（S3）](minio-local-s3.md)
- [IP 数据库下载说明](ip2region-download.md)
- [OpenAPI 同步工具说明](openapi-sync.md)
- [设计文档](DESIGN.md)
- [功能升级记录](FEATURE_UPGRADE.md)

## 📁 目录结构

```
docs/
├── README.md                    # 本文档索引
├── api/                         # API 接口文档
│   ├── auth-api.md
│   ├── user-api.md
│   ├── tenant-api.md
│   ├── rbac-api.md
│   ├── file-module-api.md
│   ├── plan-api.md
│   ├── audit-api.md
│   ├── wiki-api.md
│   ├── dashboard-api.md
│   ├── invitation-api.md
│   ├── task-api.md
│   ├── export-api.md
│   ├── message-api.md
│   ├── oauth-provider-api.md
│   ├── settings-api.md
│   ├── chat-api.md
│   ├── system-api.md
│   └── announcement-api.md
├── architecture/                # 架构设计文档
│   ├── tenant.md
│   ├── permission.md
│   ├── error.md
│   ├── database.md
│   ├── audit.md
│   ├── wiki.md
│   ├── task.md
│   ├── export.md
│   ├── oauth.md
│   ├── dashboard.md
│   ├── message.md
│   ├── pagination.md
│   ├── config.md
│   └── idgen.md
├── apifox/                      # Apifox API 配置
│   ├── MeteorX-backend.apifox.json
│   └── MeteorX-backend.openapi.json
├── development/                 # 开发与测试
│   └── testing.md               # 测试手册
├── project-structure.md         # 项目结构说明
├── ip2region-download.md        # IP 数据库下载说明
├── DESIGN.md                    # 设计文档
└── FEATURE_UPGRADE.md           # 功能升级记录
```