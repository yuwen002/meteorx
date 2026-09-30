// Command server 启动 MeteorX HTTP 服务；传入 migrate 参数则仅执行数据库迁移后退出。
package main

import (
	"fmt"
	"os"

	"meteorx/internal/bootstrap"
	"meteorx/pkg/logger"
)

// main 程序入口：根据命令行参数选择启动服务或执行迁移。
func main() {
	// 支持命令行参数：go run ./cmd/server migrate 执行迁移后退出
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigration()
		return
	}
	bootstrap.StartApp()
}

// runMigration 初始化日志与配置、连接数据库并执行自动迁移与种子数据。
func runMigration() {
	if err := logger.Init(logger.DefaultConfig()); err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Close()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		logger.Fatalf("Load config failed: %v", err)
	}

	db, err := bootstrap.InitDB(&cfg.Database)
	if err != nil {
		logger.Fatalf("Initialize database failed: %v", err)
	}

	logger.Info("Starting database migration...")
	if err := bootstrap.AutoMigrate(db); err != nil {
		logger.Fatalf("Database migration failed: %v", err)
	}
	if err := bootstrap.SeedDatabase(db); err != nil {
		logger.Warnf("Seed data initialization failed: %v", err)
	}

	logger.Info("Migration completed successfully")
	os.Exit(0)
}
