// Command migrate 一次性执行数据库自动迁移与种子数据初始化。
// 传入 -audit-columns 则切换为只读脏列巡检：报告库中残留、但当前模型已删除的孤儿列。
package main

import (
	"flag"
	"fmt"
	"os"

	"meteorx/internal/bootstrap"
	"meteorx/pkg/logger"
)

// main 初始化日志与配置、连接数据库，执行迁移与种子数据后退出。
func main() {
	auditColumns := flag.Bool("audit-columns", false, "只读巡检：报告库中残留但模型已删除的孤儿列，不执行迁移")
	flag.Parse()

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

	// 只读巡检模式：仅比对模型定义列与库实际列，报出孤儿列后退出，不做任何 DDL/写操作。
	if *auditColumns {
		reports, err := bootstrap.AuditOrphanColumns(db)
		if err != nil {
			logger.Fatalf("Column audit failed: %v", err)
		}
		if len(reports) == 0 {
			logger.Info("列巡检：未发现孤儿列，表结构与当前模型一致")
			os.Exit(0)
		}
		for _, r := range reports {
			logger.Warnf("表 %s 存在 %d 个孤儿列（模型已无、AutoMigrate 不会删除，请人工确认后清理）: %v",
				r.Table, len(r.OrphanColumns), r.OrphanColumns)
		}
		os.Exit(0)
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
