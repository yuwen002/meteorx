# Configuration & Bootstrap Architecture

## Overview

Configuration management via YAML with environment variable overrides. Bootstrap handles application lifecycle: initialization, startup, and graceful shutdown.

## Configuration

### Package: `internal/config`

### config.yaml Structure

```yaml
server:
  port: 8080
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
  upload_url: /files
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
  base_url: http://localhost:8080
```

### Environment Variable Overrides

```
# Override any config value with METEORX_ prefix
METEORX_SERVER_PORT=9090
METEORX_DATABASE_PASSWORD=prod_secret
METEORX_REDIS_PASSWORD=redis_secret
METEORX_JWT_SECRET=production_jwt_secret
METEORX_EMAIL_ENABLED=true
```

### Loading Priority

```
1. Defaults (in code)
2. config.yaml values
3. Environment variables (highest priority)
```

### Validation

```go
// Validate configuration on startup
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

### Validator Helpers

```go
// Package: internal/config
v := config.NewValidator()
v.Require("field", value, "error message")
v.RequireEnv("field", "ENV_VAR", "error message")
err := v.MustValid()
```

### Security Concerns

```
NEVER commit config.yaml with production secrets
Use environment variables for sensitive values
Add config.yaml to .gitignore (use config.example.yaml instead)
```

## Bootstrap

### Package: `internal/bootstrap`

### Lifecycle

```
main.go
    ↓
1. Load Configuration
    ↓
2. Validate Configuration
    ↓
3. Initialize Database (connect, migrate)
    ↓
4. Initialize Redis (optional)
    ↓
5. Initialize JWT / Auth
    ↓
6. Register Middleware
    ↓
7. Register Routes
    ↓
8. Start HTTP Server
    ↓
9. Wait for Shutdown Signal
    ↓
10. Graceful Shutdown
```

### Bootstrap Files

| File | Responsibility |
|------|---------------|
| `app.go` | Main bootstrap entry point |
| `config.go` | Configuration loading |
| `database.go` | DB connection, migration |
| `middleware.go` | Middleware registration |
| `router.go` | Route registration |
| `migrate.go` | Auto-migration |

### AppContext

```go
// Centralized application context
type AppContext struct {
    Config    *config.Config
    DB        *gorm.DB
    Redis     *redis.Client  // nil if disabled
    JWTManager *jwt.Manager
    Router    chi.Router
    Server    *http.Server
    Workers   []Worker
}
```

### HTTP Server Lifecycle

```go
// In bootstrap/app.go
func StartServer(ctx *AppContext) error {
    // Configure server
    server := &http.Server{
        Addr:    fmt.Sprintf(":%d", ctx.Config.Server.Port),
        Handler: ctx.Router,
    }
    ctx.Server = server

    // Start in goroutine
    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()

    return nil
}
```

### Graceful Shutdown

```go
func Shutdown(ctx *AppContext) error {
    // 1. Stop accepting new requests
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // 2. Shutdown HTTP server (wait for in-flight requests)
    if err := ctx.Server.Shutdown(shutdownCtx); err != nil {
        log.Printf("HTTP server shutdown error: %v", err)
    }

    // 3. Stop background workers
    for _, w := range ctx.Workers {
        w.Stop()
    }

    // 4. Close DB connection
    if ctx.DB != nil {
        sqlDB, _ := ctx.DB.DB()
        sqlDB.Close()
    }

    // 5. Close Redis connection
    if ctx.Redis != nil {
        ctx.Redis.Close()
    }

    log.Println("Application shutdown complete")
    return nil
}
```

### Signal Handling

```go
// In main.go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

go func() {
    if err := bootstrap.StartServer(appCtx); err != nil {
        log.Fatal(err)
    }
}()

<-sigChan  // Wait for interrupt

// Trigger graceful shutdown
bootstrap.Shutdown(appCtx)
```

### Redis Availability

Redis is optional — the application works without it:

```go
func InitRedis(cfg *config.Config) (*redis.Client, error) {
    if cfg.Redis.Host == "" {
        log.Println("Redis not configured, skipping")
        return nil, nil  // Return nil, not error
    }
    // ... connect to Redis
}
```

| Component | Without Redis | With Redis |
|-----------|--------------|------------|
| Rate limiting | In-memory (single-node only) | Distributed (multi-node) |
| Permission cache | DB query per request | Redis cache with TTL |
| Session store | JWT-only | JWT + Redis blacklist |
| Lockout tracking | In-memory | Redis-based (shared) |

### DB Startup Failure Strategy

```go
// Database is REQUIRED — application cannot start without it
func InitDatabase(cfg *config.Config) (*gorm.DB, error) {
    db, err := connectDB(cfg)
    if err != nil {
        return nil, fmt.Errorf("database connection failed: %w", err)
    }

    // Run migrations
    if err := migrate(db); err != nil {
        return nil, fmt.Errorf("migration failed: %w", err)
    }

    return db, nil  // Success
}
```

### Worker Lifecycle

Background workers (cron jobs, batch processors):

```go
type Worker interface {
    Name() string
    Start(ctx context.Context) error
    Stop()
}

// Worker registration
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

### Middleware Chain

```go
// In bootstrap/middleware.go
func RegisterMiddleware(r chi.Router, deps *Dependencies) {
    r.Use(middleware.RequestIDMiddleware)
    r.Use(middleware.GlobalErrorHandler)
    r.Use(middleware.LoggerMiddleware)
    r.Use(middleware.CORSMiddleware())
    r.Use(middleware.RateLimitMiddleware(deps.Redis))

    // Auth (for protected routes)
    // Permission (auto-derive or explicit)
    // Audit
}
```

## Directory Structure

```
internal/bootstrap/
    app.go          # Entry point, lifecycle management
    config.go       # Config loading (Viper)
    database.go     # DB init, connection, pool
    middleware.go   # Global middleware registration
    router.go       # Route registration
    migrate.go      # Auto-migration runner

internal/config/
    config.go       # Config struct definitions
    config.yaml     # Default configuration
    validator.go    # Config validation helpers
```

## Best Practices

1. **NEVER** hard-code config values — always use config.yaml or env vars
2. **Always** validate config before starting services
3. **Always** implement graceful shutdown (30s timeout)
4. **Treat** Database as required, Redis as optional
5. **Use** environment-specific config overrides for production
6. **Log** config values at startup (but mask secrets!)
7. **Keep** bootstrap code focused on lifecycle, not business logic