# MeteorX 项目目录结构

```
meteorx/
├── .env                           # 环境变量配置文件
├── Dockerfile                     # Docker 镜像构建文件
├── LICENSE                        # 开源许可证文件
├── PROJECT_STRUCTURE.md           # 项目结构说明
├── README.md                      # 项目说明文档
├── docker-compose.yml             # Docker Compose 编排文件
├── go.mod                         # Go 模块依赖文件
├── go.sum                         # Go 模块依赖校验文件
├── meteorx.exe                    # 编译后的可执行文件
│
├── cmd/                           # 应用程序入口
│   └── server/
│       └── main.go               # 服务器启动入口
│
├── internal/                      # 内部包（不对外暴露）
│   ├── bootstrap/                # 应用启动引导（核心）
│   │   ├── app.go               # 应用初始化流程
│   │   ├── config.go            # 配置加载
│   │   ├── database.go          # 数据库连接初始化
│   │   ├── middleware.go        # 全局中间件注册
│   │   ├── migrate.go           # 数据库自动迁移
│   │   └── router.go            # 路由注册（Chi）
│   │
│   ├── cache/                    # 缓存相关
│   │   └── redis.go             # Redis 缓存实现
│   │
│   ├── common/                   # 通用组件
│   │   ├── contextx/            # 上下文扩展
│   │   │   ├── constants.go     # 上下文常量定义
│   │   │   └── contextx.go      # 上下文工具方法
│   │   ├── jwt/                 # JWT 工具封装
│   │   │   └── jwt.go
│   │   ├── response/            # 统一响应格式化
│   │   │   └── response.go
│   │   └── validator/           # 数据验证
│   │       └── validator.go
│   │
│   ├── config/                   # 配置管理
│   │   ├── config.go            # 配置结构体定义
│   │   └── config.yaml          # YAML 配置文件
│   │
│   ├── middleware/               # HTTP 中间件
│   │   ├── admin_middleware.go  # 平台管理员权限校验
│   │   ├── auth.go              # JWT 认证中间件
│   │   └── logger.go            # 请求日志中间件
│   │
│   └── modules/                  # 业务模块
│       ├── audit/               # 审计模块（待完善）
│       │   └── routes.go
│       ├── auth/                # 认证模块（已实现）
│       │   ├── dto/
│       │   │   └── auth_dto.go
│       │   ├── handler/
│       │   │   └── auth_handler.go
│       │   ├── service/
│       │   │   └── auth_service.go
│       │   ├── module.go
│       │   └── routes.go
│       ├── rbac/                # 权限控制模块（待完善）
│       │   └── routes.go
│       ├── tenant/              # 租户管理模块（已实现）
│       │   ├── dto/
│       │   │   ├── admin_tenant_dto.go
│       │   │   ├── tenant_converter.go
│       │   │   └── tenant_dto.go
│       │   ├── handler/
│       │   │   └── tenant_handler.go
│       │   ├── model/
│       │   │   └── tenant.go
│       │   ├── repository/
│       │   │   ├── interface.go
│       │   │   └── tenant_repository.go
│       │   ├── service/
│       │   │   └── tenant_service.go
│       │   ├── module.go
│       │   └── routes.go
│       └── user/                # 用户管理模块（部分实现）
│           ├── dto/
│           │   ├── user_converter.go
│           │   └── user_dto.go
│           ├── model/
│           │   └── user.go
│           ├── repository/
│           │   ├── interface.go
│           │   └── user_repository.go
│           ├── module.go
│           └── routes.go
│
├── pkg/                          # 公共包（可对外暴露）
│   ├── crypto/                  # 加密工具
│   │   └── crypto.go            # 密码哈希等
│   ├── logger/                  # 日志工具
│   │   └── logger.go
│   ├── pagination/              # 分页工具
│   │   └── pagination.go
│   └── uuid/                    # UUID 生成工具
│       └── uuid.go
│
└── scripts/                      # 脚本文件
    └── sql/
        ├── init.sql             # 初始化 SQL
        └── seed.sql             # 数据种子 SQL
```

## 架构说明

### 分层架构
- **Handler Layer** (handler/) - HTTP 请求处理，参数验证，响应封装
- **Service Layer** (service/) - 业务逻辑处理，事务管理，DTO 转换
- **Repository Layer** (repository/) - 数据访问抽象，数据库操作，事务执行
- **Model Layer** (model/) - 业务实体定义，数据库映射
- **DTO Layer** (dto/) - 数据传输对象，请求/响应结构

### 设计模式
- **Repository Pattern** - 数据访问抽象层
- **Dependency Injection** - 依赖注入
- **Clean Architecture** - 清洁架构原则
- **Multi-tenancy** - 多租户架构支持
- **Onion Architecture** - 洋葱架构（依赖方向向内）

### 技术栈
| 分类 | 技术 | 版本 |
|------|------|------|
| Web Framework | Chi | v5.2.5 |
| ORM | GORM | v1.31.1 |
| Database | MySQL | - |
| Cache | Redis | v9.19.0 |
| Authentication | JWT | v5.3.1 |
| Validation | Validator | v10.30.2 |
| Configuration | Viper | v1.21.0 |
| Deployment | Docker | - |

### 模块说明
- **tenant** - 多租户核心模块（已实现完整 CRUD）
- **auth** - 用户认证授权（已实现注册、登录）
- **user** - 用户管理（已实现 Model、Repository，待实现 Handler、Service）
- **rbac** - 基于角色的访问控制（待完善）
- **audit** - 操作审计日志（待完善）

### 公共包说明
- **crypto** - 加密解密工具（bcrypt 密码哈希）
- **logger** - 结构化日志记录工具
- **pagination** - 分页查询工具
- **uuid** - UUID 生成工具

### 中间件说明
- **auth** - JWT Token 解析，注入 UserID/TenantID/Role 到上下文
- **admin_middleware** - 平台超级管理员权限校验
- **logger** - HTTP 请求日志记录

### 配置管理
- 环境变量配置 (.env)
- YAML 配置文件 (internal/config/config.yaml)
- Viper 多配置源合并

### 数据库设计
- GORM 自动迁移
- 多租户数据隔离（TenantID 字段）
- 软删除支持
- 事务支持（租户创建等场景）

### API 端点
- `GET /health` - 健康检查
- `POST /api/v1/auth/register` - 用户注册
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/tenants/register` - 租户注册
- `GET /api/v1/tenants` - 获取租户列表（需认证）
- `GET /api/v1/tenants/:id` - 获取租户详情（需认证）
- `PUT /api/v1/tenants/:id` - 更新租户信息（需认证）
- `DELETE /api/v1/tenants/:id` - 删除租户（需平台管理员）