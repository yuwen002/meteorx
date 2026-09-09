package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"meteorx/internal/cache"
	"meteorx/internal/middleware"
	"meteorx/pkg/logger"
)

func StartApp() {
	// 1. 初始化日志系统
	if err := logger.Init(logger.DefaultConfig()); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	// 2. 加载配置
	cfg, err := LoadConfig()
	if err != nil {
		logger.Fatalf("Load config failed: %v", err)
	}

	// 2. 初始化数据库
	db, err := InitDB(&cfg.Database)
	if err != nil {
		logger.Fatalf("Initialize database failed: %v", err)
	}

	// 3. 执行数据库迁移
	if err := AutoMigrate(db); err != nil {
		logger.Fatalf("Database migration failed: %v", err)
	}

	// 4. 初始化种子数据（如果不存在）
	if err := SeedDatabase(db); err != nil {
		logger.Warnf("Seed data initialization failed: %v", err)
	}

	// 5. 初始化 Redis（允许降级：连接失败时继续运行，仅禁用缓存能力）
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
	rdb, redisErr := cache.NewRedis(redisAddr, cfg.Redis.Password, cfg.Redis.DB)
	if redisErr != nil {
		logger.Warnf("Redis connection failed, running without cache: %v", redisErr)
	}

	// 6. 创建应用级 context（支持优雅取消）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 7. 初始化路由并注入依赖
	r := InitRouter(ctx, db, cfg, rdb)

	// 7.1 启动订阅到期自动禁用租户的定时任务（绑定 app 主 context，支持优雅取消）
	StartPlanExpiryJob(ctx, db)

	// 7.2 启动租户注销定时执行任务（执行已到期的注销申请）
	StartCancelCleanupJob(ctx, db)

	// 7. 构造 http.Server 以支持优雅关闭
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// 监听系统信号实现优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Infof("MeteorX server started on port %d [%s mode]", cfg.Server.Port, cfg.Server.Mode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("HTTP server error: %v", err)
		}
	}()

	sig := <-quit
	logger.Infof("Received signal %v, shutting down gracefully...", sig)

	// 按依赖方向逆序关闭：
	// 1) 停止周期性任务（让正在执行的任务自行完成）
	cancel()

	// 2) 停止批量审计日志处理器（刷新缓冲中剩余的日志）
	middleware.StopAuditBatchProcessor()

	// 3) 关闭 HTTP 服务器（等待在途请求完成）
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Errorf("HTTP server shutdown error: %v", err)
	}

	// 4) 关闭 Redis 连接
	if rdb != nil {
		if err := rdb.Close(); err != nil {
			logger.Errorf("Redis close error: %v", err)
		} else {
			logger.Info("Redis connection closed")
		}
	}

	// 5) 关闭数据库连接池
	sqlDB, err := db.DB()
	if err == nil {
		if cerr := sqlDB.Close(); cerr != nil {
			logger.Errorf("Database close error: %v", cerr)
		} else {
			logger.Info("Database connection pool closed")
		}
	}

	logger.Info("MeteorX server stopped")

	// 6) 关闭日志系统（刷新异步日志缓冲区，放在最后一步）
	logger.Close()
}