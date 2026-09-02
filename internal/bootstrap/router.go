package bootstrap

import (
	"context"
	"encoding/json"
	"meteorx/internal/modules/audit"
	"meteorx/internal/modules/dashboard"
	"meteorx/internal/modules/notification"
	"net/http"
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
	"meteorx/internal/modules/file"
	"meteorx/internal/modules/plan"
	planRepo "meteorx/internal/modules/plan/repository"
	planSvc "meteorx/internal/modules/plan/service"
	"meteorx/internal/modules/rbac"
	"meteorx/internal/modules/tenant"
	"meteorx/internal/modules/user"
	userRepo "meteorx/internal/modules/user/repository"
	"meteorx/internal/modules/wiki"
	dbpkg "meteorx/internal/pkg/db"
	"meteorx/pkg/security"
)

func InitRouter(ctx context.Context, db *gorm.DB, cfg *config.Config, rdb *cache.Redis) *chi.Mux {
	r := chi.NewRouter()
	SetupMiddleware(r)

	tokenHelper := jwt.NewTokenHelper(cfg.JWT)

	// 初始化事务管理器
	txManager := dbpkg.NewTxManager(db)

	// 初始化限流器
	rateLimiter := security.NewRateLimiter(rdb, cfg.Security.RateLimit)

	// 初始化默认套餐（幂等）
	initPlans(db)

	// 健康检查（基础）
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

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
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(health)
	})

	// 静态文件服务 - 提供上传文件的访问
	fileServer := http.FileServer(http.Dir(cfg.File.UploadPath))
	r.Handle("/uploads/*", http.StripPrefix("/uploads", fileServer))

	r.Route("/api/v1", func(r chi.Router) {

		// --- 分组一：公开接口 (Public) ---
		r.Group(func(r chi.Router) {
			// 全局接口限流
			r.Use(middleware.RateLimitMiddleware(rateLimiter))

			// 1. 认证模块（登录、签发 Token）
			auth.InitModule(r, db, *cfg, rdb)

			// 2. 租户公开接口（仅限注册）
			tenant.InitPublicModule(r, db)
		})

		// --- 分组二：受保护接口 (Protected) ---
		r.Group(func(r chi.Router) {
			// 【第一层防线】挂载认证中间件，解析 Token 并注入 UserID, TenantID, Role
			// 同时检查 token 是否在黑名单中（已登出的 token）
			blacklistChecker := &middleware.RedisBlacklistChecker{Redis: rdb}
			r.Use(middleware.Auth(tokenHelper, blacklistChecker))

			// 审计日志中间件：自动记录所有请求（挂载在认证之后，确保能获取用户信息）
			repo := auditRepo.NewAuditLogRepository(db)
			auditService := auditSvc.NewAuditService(repo)

			// 初始化批量处理器（性能优化，使用传入的context支持优雅取消）
			middleware.InitAuditBatchProcessor(ctx, auditService)

			r.Use(middleware.AuditMiddleware(auditService))

			// 3. 租户私有接口（租户管理员登录后：管理本公司信息、查看套餐等）
			tenant.InitPrivateModule(r, db)

			// 3.1 租户设置接口
			tenantSettingsHandler := tenant.NewTenantSettingsHandler(db)
			tenant.RegisterTenantSettingsRoutes(r, tenantSettingsHandler)

			// 4. 用户/业务接口
			user.InitModule(r, db)

			// 4.1 当前用户个人信息接口
			user.InitProfileModule(r, db)

			// 4.2 租户侧当前套餐查询
			plan.InitPrivateModule(r, db)

			// 4.3 文件管理接口
			file.RegisterRoutes(r, db, cfg)

			// 4.4 Wiki 知识库接口
			wiki.InitModule(r, db, txManager)

			// 4.5 租户端公告查看接口
			notification.InitTenantModule(r, db)

			// ========================================================
			// 🔥 新增分组三：MaaS 平台运营后台特权接口 (Platform Admin Only)
			// ========================================================
			r.Group(func(r chi.Router) {
				// 【第二层防线】剥洋葱：在已登录的基础上，必须是平台超级管理员 (role == "superadmin")
				r.Use(middleware.RequiresMasterAdmin())

				// 5. 租户管理特权接口（后台手动新建租户、禁用/启用、软删除等）
				tenant.InitAdminModule(r, db)

				// 6. 系统管理员管理接口
				user.InitAdminModule(r, db)

				// 7. RBAC 角色权限管理接口（仅后台管理员可操作）
				rbac.InitModule(r, db)

				// 8. 审计日志管理接口（仅后台管理员可操作）
				audit.InitModule(r, db)

				// 9. 套餐管理接口（仅平台超级管理员可操作）
				plan.InitAdminModule(r, db)

				// 10. 数据看板接口（仅平台超级管理员可操作）
				dashboard.InitModule(r, db)

				// 11. 通知公告接口（仅平台超级管理员可操作）
				notification.InitModule(r, db)
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