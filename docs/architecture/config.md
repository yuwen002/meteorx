# 配置与启动架构

## 概述

通过 YAML 配合环境变量覆盖进行配置管理。Bootstrap 负责应用生命周期：初始化、启动和优雅关闭。

## 配置管理

### 包路径：`internal/config`

### config.yaml 结构

```yaml
server:
  port: 8081
  mode: development

database:
  driver: mysql
  host: localhost
  port: 3306
  user: root
  password: secret
  name: meteorx
  tls: false
  debug: false
  timezone: Asia/Shanghai   # 业务时区：启动时强制 time.Local，与容器 TZ 解耦；留空默认 Asia/Shanghai

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

jwt:
  secret: "your-secret-key"
  expiration: 24h
  issuer: MeteorX

log:
  level: info
  format: json

security:
  login_lockout:
    enabled: true
    max_attempts: 5
    lockout_duration: 30m
    reset_after: 1h
  password_policy:
    enabled: true
    min_length: 8
    max_length: 128
    require_uppercase: true
    require_lowercase: true
    require_digit: true
    require_special: false
  rate_limit:
    enabled: true
    requests: 100
    window: 1m
    burst_size: 200

file:
  upload_path: ./uploads
  upload_url: http://localhost:8081/uploads
  max_file_size: 10485760
  allowed_types:
    - image/jpeg
    - image/png
    - application/pdf
  storage_type: local

email:
  enabled: false
  host: smtp.example.com
  port: 587
  username: noreply@example.com
  password: ""
  from: noreply@example.com
  from_name: MeteorX

client:
  base_url: http://localhost:5173

oauth:
  refresh_token_ttl: 168h    # 刷新令牌有效期，默认 168h（7 天）
  state_ttl: 10m             # CSRF state 有效期，默认 10m
  google:
    enabled: false
    client_id: ""
    client_secret: ""
    redirect_url: "http://localhost:5173/auth/google/callback"
  github:
    enabled: false
    client_id: ""
    client_secret: ""
    redirect_url: "http://localhost:5173/auth/github/callback"

auth:
  api_token_max_ttl: 2160h   # API Token 最长有效期，默认 2160h（90 天）
```

### 环境变量覆盖

```
# 使用 METEORX_ 前缀覆盖任意配置值
METEORX_SERVER_PORT=9090
METEORX_DATABASE_PASSWORD=prod_secret
METEORX_REDIS_PASSWORD=redis_secret
METEORX_JWT_SECRET=production_jwt_secret
METEORX_EMAIL_ENABLED=true
METEORX_OAUTH_REFRESH_TOKEN_TTL=72h
METEORX_AUTH_API_TOKEN_MAX_TTL=4320h
METEORX_OAUTH_STATE_TTL=5m
```

### 加载优先级

```
1. 默认值（代码中）
2. config.yaml 中的值
3. 环境变量（最高优先级）
```

### 业务时区一致性

MySQL `DATETIME` 列与时区无关（存储的是“挂钟”值），而连接串使用 `loc=Local`，因此 `time.Time` 的写入/读取都依赖进程的全局 `time.Local`。若仅靠容器 `TZ` 环境变量隐式决定时区，一旦某环境未设 `TZ`（直接跑二进制、换基础镜像等），`time.Local` 会退化为 UTC，导致存量数据整体偏移、任务提醒/逾期判断错位。

为此在启动连接数据库前（`bootstrap.InitDB`）按 `database.timezone`（默认 `Asia/Shanghai`）强制 `time.Local = time.LoadLocation(tz)`，使时间语义与容器 `TZ` 解耦、开发/生产行为一致；并通过 `_ "time/tzdata"` 内嵌 IANA 时区库，保证 Windows 等无系统时区数据的环境也能加载。名称无效时保留系统默认并告警，不影响启动。默认仍为 +08，与既有生产数据含义一致，无需迁移。

### 配置校验

```go
// 启动时校验配置
func (c *Config) Validate() error {
    v := NewValidator()
    v.Require("server.port", fmt.Sprint(c.Server.Port), "server port is required")
    v.Require("database.driver", c.Database.Driver, "database driver is required")
    v.Require("database.host", c.Database.Host, "database host is required")
    v.Require("database.user", c.Database.User, "database user is required")
    v.Require("database.name", c.Database.Name, "database name is required")
    v.Require("jwt.secret", c.JWT.Secret, "jwt secret is required")
    return v.MustValid()
}
```

### Validator 辅助方法

```go
// 包路径：internal/config
v := config.NewValidator()
v.Require("field", value, "error message")
v.RequireEnv("field", "ENV_VAR", "error message")
err := v.MustValid()
```

### 安全注意事项

```
禁止将包含生产环境密钥的 config.yaml 提交到版本库
使用环境变量存储敏感值
将 config.yaml 加入 .gitignore（改用 config.example.yaml）
```

### JWT密钥安全强化

- 配置文件中的 `jwt.secret` 包含强密钥警告注释
- 提供生成强密钥的方法说明：
  ```bash
  # 生成64字节随机密钥并base64编码
  openssl rand -base64 64
  ```
- 生产环境必须使用随机生成的强密钥（至少32字节）
- 弱密钥（如 "your-secret-key"）仅用于开发环境

## Bootstrap 启动引导

### 包路径：`internal/bootstrap`

### 生命周期

```
main.go
    ↓
1. 加载配置
    ↓
2. 校验配置
    ↓
3. 初始化数据库（连接、迁移）
    ↓
4. 初始化 Redis（可选）
    ↓
5. 初始化 JWT / 认证
    ↓
6. 注册中间件
    ↓
7. 注册路由
    ↓
8. 启动 HTTP Server
    ↓
9. 等待关闭信号
    ↓
10. 优雅关闭
```

### Bootstrap 文件职责

| 文件 | 职责 |
|------|------|
| `app.go` | 主启动入口 |
| `config.go` | 配置加载 |
| `database.go` | 数据库连接、迁移 |
| `middleware.go` | 中间件注册 |
| `router.go` | 路由注册 |
| `migrate.go` | 自动迁移 |

### AppContext

```go
// 集中式应用上下文
type AppContext struct {
    Config    *config.Config
    DB        *gorm.DB
    Redis     *redis.Client  // 如果禁用则为 nil
    JWTManager *jwt.Manager
    Router    chi.Router
    Server    *http.Server
    Workers   []Worker
}
```

### HTTP Server 生命周期

```go
// 在 bootstrap/app.go 中
func StartServer(ctx *AppContext) error {
    // 配置服务器
    server := &http.Server{
        Addr:    fmt.Sprintf(":%d", ctx.Config.Server.Port),
        Handler: ctx.Router,
    }
    ctx.Server = server

    // 在协程中启动
    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()

    return nil
}
```

### 优雅关闭

```go
func Shutdown(ctx *AppContext) error {
    // 1. 停止接受新请求
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // 2. 关闭 HTTP 服务器（等待进行中的请求完成）
    if err := ctx.Server.Shutdown(shutdownCtx); err != nil {
        log.Printf("HTTP server shutdown error: %v", err)
    }

    // 3. 停止后台工作进程
    for _, w := range ctx.Workers {
        w.Stop()
    }

    // 4. 关闭数据库连接
    if ctx.DB != nil {
        sqlDB, _ := ctx.DB.DB()
        sqlDB.Close()
    }

    // 5. 关闭 Redis 连接
    if ctx.Redis != nil {
        ctx.Redis.Close()
    }

    log.Println("Application shutdown complete")
    return nil
}
```

### 信号处理

```go
// 在 main.go 中
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

go func() {
    if err := bootstrap.StartServer(appCtx); err != nil {
        log.Fatal(err)
    }
}()

<-sigChan  // 等待中断信号

// 触发优雅关闭
bootstrap.Shutdown(appCtx)
```

### Redis 可用性

Redis 是可选的 — 没有 Redis 应用也能正常工作：

```go
func InitRedis(cfg *config.Config) (*redis.Client, error) {
    if cfg.Redis.Host == "" {
        log.Println("Redis not configured, skipping")
        return nil, nil  // 返回 nil，不是错误
    }
    // ... 连接 Redis
}
```

| 组件 | 无 Redis | 有 Redis |
|------|----------|----------|
| 接口限流 | 内存（仅单节点） | 分布式（多节点） |
| 权限缓存 | 每次请求查数据库 | Redis 缓存 + TTL |
| 会话存储 | 仅 JWT | JWT + Redis 黑名单 |
| 登录锁定追踪 | 内存 | Redis 共享 |
| OAuth CSRF state | 跳过校验（不安全） | Redis 存储 + TTL（`oauth.state_ttl`） |
| OAuth 刷新令牌 | 不支持 | Redis 存储 + 轮转（`oauth.refresh_token_ttl`） |
| API Token 缓存 | 降级为纯 DB 查询 | Redis 缓存加速（TTL 1h） |

### 数据库启动失败策略

```go
// 数据库是必需的 — 没有数据库应用无法启动
func InitDatabase(cfg *config.Config) (*gorm.DB, error) {
    db, err := connectDB(cfg)
    if err != nil {
        return nil, fmt.Errorf("database connection failed: %w", err)
    }

    // 运行迁移
    if err := migrate(db); err != nil {
        return nil, fmt.Errorf("migration failed: %w", err)
    }

    return db, nil  // 成功
}
```

### 工作进程生命周期

后台工作进程（定时任务、批量处理器）：

```go
type Worker interface {
    Name() string
    Start(ctx context.Context) error
    Stop()
}

// 工作进程注册
func RegisterWorkers(ctx *AppContext) {
    ctx.Workers = append(ctx.Workers,
        audit.NewBatchProcessor(ctx.DB),
        plan.NewExpiryJob(ctx.DB),
    )

    for _, w := range ctx.Workers {
        go func(w Worker) {
            if err := w.Start(context.Background()); err != nil {
                log.Printf("Worker %s failed: %v", w.Name(), err)
            }
        }(w)
    }
}
```

### 中间件链

```go
// 在 bootstrap/middleware.go 中
func RegisterMiddleware(r chi.Router, deps *Dependencies) {
    r.Use(middleware.RequestIDMiddleware)
    r.Use(middleware.GlobalErrorHandler)
    r.Use(middleware.LoggerMiddleware)
    r.Use(middleware.CORSMiddleware())
    r.Use(middleware.RateLimitMiddleware(deps.Redis))

    // 认证（用于受保护路由）
    // 权限（自动推导或显式指定）
    // 审计
}
```

## 目录结构

```
internal/bootstrap/
    app.go          # 入口点，生命周期管理
    config.go       # 配置加载（Viper）
    database.go     # 数据库初始化、连接、连接池
    middleware.go   # 全局中间件注册
    router.go       # 路由注册
    migrate.go      # 自动迁移执行器

internal/config/
    config.go       # 配置结构体定义
    config.yaml     # 默认配置
    validator.go    # 配置校验辅助方法
```

## 最佳实践

1. **禁止**硬编码配置值 — 始终使用 config.yaml 或环境变量
2. **始终**在启动服务之前校验配置
3. **始终**实现优雅关闭（30 秒超时）
4. **将**数据库视为必需，Redis 视为可选
5. **使用**针对不同环境的配置覆盖来管理生产环境
6. **记录**启动时的配置值（但要遮盖密钥！）
7. **保持** bootstrap 代码聚焦于生命周期，而非业务逻辑