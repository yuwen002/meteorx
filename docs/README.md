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

### 架构设计文档
- [多租户隔离架构](architecture/tenant.md)
- [RBAC 权限架构](architecture/permission.md)
- [错误/响应规范](architecture/error.md)
- [数据库/事务架构](architecture/database.md)
- [审计架构](architecture/audit.md)
- [审计增强功能](architecture/audit-enhanced.md)
- [分页/排序/过滤架构](architecture/pagination.md)
- [配置/启动架构](architecture/config.md)
- [ID/ULID 统一架构](architecture/idgen.md)

### 开发文档
- [功能完成记录](development/features-complete.md)

### 其他文档
- [项目结构说明](project-structure.md)
- [IP 数据库下载说明](ip2region-download.md)
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
│   └── announcement-api.md
├── architecture/                # 架构设计文档
│   ├── tenant.md
│   ├── permission.md
│   ├── error.md
│   ├── database.md
│   ├── audit.md
│   ├── audit-enhanced.md
│   ├── pagination.md
│   ├── config.md
│   └── idgen.md
├── development/                 # 开发相关文档
│   └── features-complete.md
├── apifox/                      # Apifox API 配置
│   ├── MeteorX-backend.apifox.json
│   └── MeteorX-backend.openapi.json
├── project-structure.md         # 项目结构说明
├── ip2region-download.md        # IP 数据库下载说明
├── DESIGN.md                    # 设计文档
└── FEATURE_UPGRADE.md           # 功能升级记录
```