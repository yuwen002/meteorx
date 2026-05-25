package bootstrap

import (
	"fmt"
	"log"
	"net/http"
)

func StartApp() {
	// 1. 加载配置
	cfg := LoadConfig()

	// 2. 初始化数据库
	db, err := InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Initialize database failed: %v", err)
	}

	// 3. 执行数据库迁移
	if err := AutoMigrate(db); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	// 4. 初始化路由并注入依赖
	// 这样你的路由、中间件、业务模块都能拿到这个 db 实例
	r := InitRouter(db, cfg)

	// 5. 启动服务
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("MeteorX server started on port %d [%s mode]", cfg.Server.Port, cfg.Server.Mode)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
