package main

import (
	"fmt"
	"os"

	"meteorx/internal/bootstrap"
	"meteorx/pkg/logger"
)

func main() {
	// 支持命令行参数：go run ./cmd/server migrate 执行迁移后退出
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigration()
		return
	}
	bootstrap.StartApp()
}

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