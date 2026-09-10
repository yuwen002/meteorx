package main

import (
	"fmt"
	"os"

	"meteorx/internal/bootstrap"
	"meteorx/pkg/logger"
)

func main() {
	// 1. 初始化日志系统
	if err := logger.Init(logger.DefaultConfig()); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Close()

	// 2. 加载配置
	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		logger.Fatalf("Load config failed: %v", err)
	}

	// 3. 初始化数据库
	db, err := bootstrap.InitDB(&cfg.Database)
	if err != nil {
		logger.Fatalf("Initialize database failed: %v", err)
	}

	// 4. 执行数据库迁移
	logger.Info("Starting database migration...")
	if err := bootstrap.AutoMigrate(db); err != nil {
		logger.Fatalf("Database migration failed: %v", err)
	}

	// 5. 初始化种子数据
	if err := bootstrap.SeedDatabase(db); err != nil {
		logger.Warnf("Seed data initialization failed: %v", err)
	}

	logger.Info("Migration completed successfully")
	os.Exit(0)
}