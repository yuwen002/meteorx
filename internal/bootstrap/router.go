package bootstrap

import (
	"context"
	"encoding/json"
	"meteorx/internal/modules/audit"
	"meteorx/internal/modules/dashboard"
	"meteorx/internal/modules/export"
	"meteorx/internal/modules/notification"
	"meteorx/internal/pkg/emailer"
	"meteorx/pkg/iplocation"
	"net/http"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"meteorx/internal/cache"
	"meteorx/internal/common/jwt"
	"meteorx/internal/config"
	"meteorx/internal/middleware"
	auditRepo "meteorx/internal/modules/audit/repository"
	auditSvc "meteorx/internal/modules/audit/service"
	"meteorx/internal/modules/auth"
	authHandler "meteorx/internal/modules/auth/handler"
	authRepo "meteorx/internal/modules/auth/repository"
	authSvc "meteorx/internal/modules/auth/service"
	"meteorx/internal/modules/file"
	"meteorx/internal/modules/invitation"
	"meteorx/internal/modules/message"
	"meteorx/internal/modules/oauth"
	"meteorx/internal/modules/plan"
	planRepo "meteorx/internal/modules/plan/repository"
	planSvc "meteorx/internal/modules/plan/service"
	"meteorx/internal/modules/rbac"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	"meteorx/internal/modules/task"
	"meteorx/internal/modules/tenant"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	"meteorx/internal/modules/user"
	userRepo "meteorx/internal/modules/user/repository"
	"meteorx/internal/modules/wiki"
	wikiSvc "meteorx/internal/modules/wiki/service"
	"meteorx/internal/notify"
	dbpkg "meteorx/internal/pkg/db"
	"meteorx/internal/ws"
	"meteorx/pkg/logger"
	"meteorx/pkg/security"
)

// InitRouter 构建 chi 路由，装配全局中间件并将各业务模块按公开/受保护/平台管理员三个分组注册。
func InitRouter(ctx context.Context, db *gorm.DB, cfg *config.Config, rdb *cache.Redis) *chi.Mux {
	r := chi.NewRouter()
	SetupMiddleware(r, BuildAllowedOrigins(cfg.Client), cfg.Server.TrustedProxies)

	tokenHelper := jwt.NewTokenHelper(cfg.JWT)

	// 初始化事务管理器
	txManager := dbpkg.NewTxManager(db)

	// 初始化限流器
	rateLimiter := security.NewRateLimiter(rdb, cfg.Security.RateLimit)

	// 初始化指标收集器
	metricsCollector := middleware.NewMetricsCollector()
	promMetrics := middleware.NewPrometheusMetrics()

	// 全局中间件：Recovery -> Tracing -> Logger -> Metrics -> CORS
	// Tracing 置于 Logger 之前，使访问日志与结构化日志都能带上真实 OTel Trace ID
	r.Use(middleware.Recovery)
	r.Use(middleware.TracingMiddleware)
	r.Use(middleware.Logger)
	r.Use(metricsCollector.Metrics)
	r.Use(promMetrics.Middleware)

	// 初始化默认套餐（幂等）
	initPlans(db)

	// 健康检查（基础）
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// 指标查询接口（Prometheus 格式，含 Go 运行时指标）
	r.Handle("/metrics", promMetrics.Handler())

	// 深度健康检查（检查数据库和Redis连接状态）
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		health := map[string]interface{}{
			"status": "ok",
			"checks": map[string]string{},
		}

		// 检查数据库连接
		sqlDB, err := db.DB()
		if err != nil {
			health["status"] = "error"
			health["checks"].(map[string]string)["database"] = "error: " + err.Error()
		} else if err := sqlDB.Ping(); err != nil {
			health["status"] = "error"
			health["checks"].(map[string]string)["database"] = "error: " + err.Error()
		} else {
			health["checks"].(map[string]string)["database"] = "ok"
		}

		// 检查Redis连接
		if rdb != nil && rdb.IsAvailable() {
			if err := rdb.Ping(r.Context()); err != nil {
				health["status"] = "degraded"
				health["checks"].(map[string]string)["redis"] = "error: " + err.Error()
			} else {
				health["checks"].(map[string]string)["redis"] = "ok"
			}
		} else if rdb == nil {
			health["status"] = "degraded"
			health["checks"].(map[string]string)["redis"] = "not_configured"
		} else {
			health["checks"].(map[string]string)["redis"] = "unavailable"
		}

		statusCode := http.StatusOK
		if health["status"] == "error" {
			statusCode = http.StatusServiceUnavailable
		}

		// 运行时与数据库连接池明细：辅助容量规划与泄漏/连接耗尽排查。均为进程内本地读数，
		// 开销极小；但 runtime.ReadMemStats 会短暂锁堆，若健康检查被高频拉取可改走 runtime/metrics。
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		health["runtime"] = map[string]interface{}{
			"goroutines":    runtime.NumGoroutine(),
			"heap_alloc_mb": ms.HeapAlloc / 1024 / 1024,
			"sys_mb":        ms.Sys / 1024 / 1024,
			"num_gc":        ms.NumGC,
		}
		if sqlDB != nil {
			pool := sqlDB.Stats()
			health["db_pool"] = map[string]interface{}{
				"open_connections": pool.OpenConnections,
				"in_use":           pool.InUse,
				"idle":             pool.Idle,
				"wait_count":       pool.WaitCount,
				"wait_duration_ms": pool.WaitDuration.Milliseconds(),
			}
		}

		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(health)
	})

	// 上传文件静态服务：带短时效 HMAC 签名校验，杜绝目录列举与未授权访问。
	// 使用独立签名密钥链（首项签发、全链校验），与 jwt.secret 解耦以便单独轮换。
	r.Handle("/uploads/*", middleware.SignedUploadsHandler(cfg.File.UploadPath, cfg.File.UploadSignKeys(cfg.JWT.Secret)))

	r.Route("/api/v1", func(r chi.Router) {

		var authH *authHandler.AuthHandler

		// --- 分组一：公开接口 (Public) ---
		r.Group(func(r chi.Router) {
			// 全局接口限流
			r.Use(middleware.RateLimitMiddleware(rateLimiter))

			// CSP 违规上报（配合 nginx 强制执行的 Content-Security-Policy）：
			// 浏览器直接 POST、无法携带 Authorization，故置于公开分组、仅落服务端日志，
			// 由上行 IP 限流防止被刷爆日志。
			r.Post("/csp-report", middleware.CSPReport())

			// 1. 认证模块（登录、签发 Token）
			authH = auth.InitModule(r, db, *cfg, rdb)

			// 2. 租户公开接口（仅限注册）
			tenant.InitPublicModule(r, db)

			// 3. OAuth2 第三方登录（免登录，公开接口）
			oauth.InitModule(r, db, *cfg, tokenHelper, rdb)

			// 3.1 邀请接受公开接口（免登录，凭 token 接受邀请）
			invitation.InitPublicModule(r, db, *cfg)

			// 4. Wiki 文档公开分享（免登录，凭 token/密码访问）
			wiki.RegisterPublicShareRoute(r, db, txManager, cfg)
		})

		// --- 分组二：受保护接口 (Protected) ---
		r.Group(func(r chi.Router) {
			// 【第一层防线】挂载认证中间件，解析 Token 并注入 UserID, TenantID, Role
			// 同时检查 token 是否在黑名单中（已登出的 token）
			// JWT 解析失败时 fallback 查 API Token（mxat_ 前缀）
			blacklistChecker := &middleware.RedisBlacklistChecker{Redis: rdb}
			apiTokenRepo := authRepo.NewAPITokenRepository(db)
			apiTokenValidator := authSvc.NewAPITokenService(apiTokenRepo, userRepo.NewUserRepository(db), rbacRepo.NewUserRoleRepository(db), rbacRepo.NewRolePermissionRepository(db), rdb, cfg.Auth)
			r.Use(middleware.Auth(tokenHelper, blacklistChecker, cfg.Server.Mode, cfg.Server.TestBypass, apiTokenValidator))

			// 【租户状态门禁】被平台暂停/禁用的租户，其成员所有业务请求返回 403（租户不存在/查询失败 fail-open 放行）
			r.Use(middleware.TenantStatusGate(tenantRepo.NewTenantRepository(db)))

			// 【租户级限流】依据套餐 API 速率配额（每分钟请求数）限流，Redis 不可用/未设置配额时 fail-open 放行
			r.Use(middleware.TenantRateLimit(plan.NewQuotaRateProvider(db), rdb))

			// 审计日志中间件：自动记录所有请求（挂载在认证之后，确保能获取用户信息）
			repo := auditRepo.NewAuditLogRepository(db)
			ipLocator := initIPLocator(cfg)
			auditService := auditSvc.NewAuditService(repo)

			// 邮件配置提前构造：告警评估链接线与审计管理接口（下方 admin 分组）共用
			emailCfg := emailer.NewEmailer(cfg.Email.Host, cfg.Email.Port, cfg.Email.Username, cfg.Email.Password, cfg.Email.From, cfg.Email.FromName)

			// 告警评估链接入审计写入路径：日志落库成功即评估启用规则、命中产出告警
			// 并按渠道类型送达——接入前整条告警链在生产里无触发者（死代码）
			audit.WireAlertEvaluation(auditService, db, emailCfg)

			// 初始化批量处理器（性能优化，使用传入的context支持优雅取消）
			middleware.InitAuditBatchProcessor(ctx, auditService, ipLocator)

			r.Use(middleware.AuditMiddleware(auditService, ipLocator))

			// 2.1 WebSocket 实时推送（需认证）
			if cfg.WS.Enabled {
				wsHub := ws.GetHub()
				wsHandler := ws.NewHandler(wsHub, tokenHelper)
				r.Get("/ws", wsHandler.ServeWS)
			}

			// 3. 租户私有接口（租户管理员登录后：管理本公司信息、查看套餐等）
			tenant.InitPrivateModule(r, db)

			// 3.1 租户设置接口
			tenantSettingsHandler := tenant.NewTenantSettingsHandler(db)
			tenant.RegisterTenantSettingsRoutes(r, tenantSettingsHandler)

			// 4. 用户/业务接口
			user.InitModule(r, db, cfg.Security.PasswordPolicy)

			// 4.1 当前用户个人信息接口
			user.InitProfileModule(r, db)

			// 4.2 OAuth2 账号管理（需登录：查看绑定、解绑、绑定新账号）
			oauth.InitProtectedModule(r, db, *cfg, tokenHelper, rdb)

			// 4.3 API Token 管理（需登录：创建、查看、撤销长期令牌）
			auth.InitAPITokenModule(r, db, *cfg, rdb)

			// 4.3.1 邮箱验证（需登录：发送验证链接、验证邮箱）
			auth.RegisterEmailVerificationRoutes(r, authH)

			// 4.3.2 双因素认证管理（需登录：开启、确认启用、关闭）
			auth.RegisterTwoFactorRoutes(r, authH)

			// 4.4 租户侧当前套餐查询
			plan.InitPrivateModule(r, db)

			// 4.5 文件管理接口
			file.RegisterRoutes(r, db, cfg)

			// 4.6 Wiki 知识库接口（rdb 用于回收站过期清理的跨副本分布式锁，可为 nil）
			wiki.InitModule(r, db, txManager, cfg, rdb)

			// 4.7 租户端公告查看接口
			notification.InitTenantModule(r, db)

			// 4.8 邀请管理接口（租户管理员邀请成员）
			invitation.InitModule(r, db, *cfg)

			// 4.9 任务管理接口（个人待办 + 租户团队协作，仅需登录）；AI 辅助为可选叠加层，由 cfg.AI 控制
			task.InitModule(r, db, cfg.AI)

			// 4.10 数据导出接口（下载中心，异步 CSV 导出）
			export.InitModule(r, db, *cfg)

			// 4.11 站内消息收件箱（按收件人隔离，访问控制以数据归属为准，不挂 RBAC 权限码）。
			// 同时作为 notify 的 InboxWriter：实时推送只解决在线可达，落库才能让离线用户
			// 重新登录后补看到漏掉的消息（公告/注销审批/订阅到期/任务提醒）。
			var msgHub *ws.Hub
			if cfg.WS.Enabled {
				msgHub = ws.GetHub()
			}
			msgSvc := message.InitModule(r, db, msgHub)
			if m := notify.GetGlobalManager(); m != nil {
				m.SetInbox(msgSvc)
			}

			// ========================================================
			// 🔥 新增分组三：MaaS 平台运营后台特权接口 (Platform Admin Only)
			// ========================================================
			r.Group(func(r chi.Router) {
				// 【第二层防线】剥洋葱：在已登录的基础上，必须是平台超级管理员 (role == "superadmin")
				r.Use(middleware.RequiresMasterAdmin())

				// 5. 租户管理特权接口（后台手动新建租户、禁用/启用、软删除等）
				tenant.InitAdminModule(r, db)

				// 6. 系统管理员管理接口
				user.InitAdminModule(r, db, cfg.Security.PasswordPolicy)

				// 7. RBAC 角色权限管理接口（仅后台管理员可操作）
				rbac.InitModule(r, db, rdb, repo)

				// 8. 审计日志管理接口（仅后台管理员可操作）
				audit.InitModule(r, db, emailCfg)

				// 9. 套餐管理接口（仅平台超级管理员可操作）
				plan.InitAdminModule(r, db)

				// 10. 数据看板接口（仅平台超级管理员可操作）
				dashboard.InitModule(r, db)

				// 11. 通知公告接口（仅平台超级管理员可操作）
				notification.InitModule(r, db)

				// 12. 第三方登录提供商配置管理（仅平台超管）+ 注册表定时刷新（免重启生效）
				oauth.InitAdminModule(ctx, r, db, *cfg, tokenHelper, rdb)
			})
		})
	})

	return r
}

// initPlans 初始化默认套餐（幂等）
func initPlans(db *gorm.DB) {
	pRepo := planRepo.NewPlanRepository(db)
	subRepo := planRepo.NewSubscriptionRepository(db)
	uRepo := userRepo.NewUserRepository(db)
	svc := planSvc.NewPlanService(pRepo, subRepo, uRepo)
	plan.SeedPlans(context.Background(), svc)
}

// StartPlanExpiryJob 启动订阅到期自动禁用租户的定时任务
// 传入的 ctx 用于在应用关闭时取消后台 goroutine
func StartPlanExpiryJob(ctx context.Context, db *gorm.DB) {
	job := plan.NewExpiryJob(db)
	job.Start(ctx, 5*time.Minute)
}

// StartCancelCleanupJob 启动租户注销定时执行任务
// 将已通过审批且到期的注销申请真正执行（软删除租户、取消订阅）
func StartCancelCleanupJob(ctx context.Context, db *gorm.DB) {
	job := tenant.NewCancelCleanupJob(db)
	job.Start(ctx, 5*time.Minute)
}

// StartInvitationExpiryJob 启动邀请过期清理定时任务
// 将超过有效期仍为 pending 的邀请自动置为 expired
func StartInvitationExpiryJob(ctx context.Context, db *gorm.DB) {
	job := invitation.NewInvitationExpiryJob(db)
	job.Start(ctx, time.Hour)
}

// StartWikiTrashExpiryJob 启动 Wiki 回收站过期清理定时任务
// 周期性物理清除已过 expires_at 的回收站项目及其关联实体（文档/节点树/空间树）
// rdb 可为 nil：非空时启用跨副本分布式锁，多副本部署下仅一个实例执行清理
func StartWikiTrashExpiryJob(ctx context.Context, db *gorm.DB, rdb *cache.Redis) {
	job := wiki.NewWikiTrashExpiryJob(db, wikiSvc.WithExpireCacheLock(rdb))
	job.Start(ctx, time.Hour)
}

// StartTaskReminderJob 启动任务到期提醒定时任务
// 周期扫描即将到期/已逾期且未提醒过的任务，向负责人推送站内提醒
// 提醒窗口/扫描间隔/逾期冷却期/单轮上限均由配置注入
func StartTaskReminderJob(ctx context.Context, db *gorm.DB, cfg *config.Config) {
	job := task.NewTaskReminderJob(db, task.ReminderJobOptions{
		Horizon:         cfg.TaskReminder.GetHorizon(),
		Interval:        cfg.TaskReminder.GetInterval(),
		OverdueCooldown: cfg.TaskReminder.GetOverdueCooldown(),
		BatchLimit:      cfg.TaskReminder.GetBatchLimit(),
	})
	job.Start(ctx)
}

// StartExportWorkerJob 启动数据导出 worker 池
// 周期性拉取 pending 导出任务，按资源类型分派到对应 exporter 异步流式写出 CSV
func StartExportWorkerJob(ctx context.Context, db *gorm.DB, cfg *config.Config) {
	export.StartExportWorker(ctx, db, *cfg)
}

// StartExportExpiryJob 启动导出文件过期清理定时任务
// 周期性删除超过保留期的导出 CSV 文件与对应任务记录
func StartExportExpiryJob(ctx context.Context, db *gorm.DB, cfg *config.Config) {
	job := export.NewExportExpiryJob(db, *cfg)
	job.Start(ctx, 6*time.Hour)
}

// StartMessageExpiryJob 启动站内消息过期清理定时任务
// 周期性物理删除超过 message.retention_days 的收件箱消息（负值表示不清理）
func StartMessageExpiryJob(ctx context.Context, db *gorm.DB, cfg *config.Config) {
	job := message.NewMessageExpiryJob(db, *cfg)
	job.Start(ctx, time.Hour)
}

// StartAuditRetentionJob 启动审计日志保留清理定时任务
// 周期性物理删除超过 audit.retention_days 的审计日志（≤0 表示关闭，默认不启动）
func StartAuditRetentionJob(ctx context.Context, db *gorm.DB, cfg *config.Config) {
	job := audit.NewAuditRetentionJob(db, *cfg)
	job.Start(ctx, time.Hour)
}

// initIPLocator 根据配置初始化 IP 地理位置解析器
func initIPLocator(cfg *config.Config) iplocation.IPLocator {
	provider := cfg.IPLocation.Provider
	if provider == "" {
		provider = "http-api" // 默认使用 HTTP API
	}

	switch provider {
	case "ip2region":
		// 使用本地 ip2region 数据库（离线解析）
		dbPath := cfg.IPLocation.DBPath
		if dbPath == "" {
			dbPath = "./data/ip2region.xdb"
		}

		locator, err := iplocation.NewLocalLocator(dbPath)
		if err != nil {
			// 如果本地数据库加载失败，降级为 HTTP API
			logger.Warnf("Failed to load ip2region database, falling back to HTTP API: %v", err)
			return iplocation.NewHTTPLocator("ip-api", 3*time.Second)
		}

		logger.Info("IP location provider: ip2region (offline)")
		return locator

	default:
		// 使用 HTTP API（在线解析）
		timeout := time.Duration(cfg.IPLocation.Timeout) * time.Second
		if timeout == 0 {
			timeout = 3 * time.Second
		}

		logger.Info("IP location provider: http-api (online)")
		return iplocation.NewHTTPLocator("ip-api", timeout)
	}
}
