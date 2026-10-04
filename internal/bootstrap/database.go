package bootstrap

import (
	"fmt"
	"time"
	_ "time/tzdata" // 内嵌 IANA 时区库，保证 Windows 等无系统时区数据的环境也能 LoadLocation

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
	otelgorm "gorm.io/plugin/opentelemetry/tracing"

	"meteorx/internal/config"
	"meteorx/pkg/logger"
)

// InitDB 返回一个 *gorm.DB 实例。
// 配置了 replicas 时自动启用读写分离（SELECT → 从库，INSERT/UPDATE/DELETE → 主库）。
func InitDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	// 先按配置强制全局 time.Local，使后续 DSN loc=Local、time.Now()、时间解析与业务时区一致，
	// 与容器 TZ 环境变量解耦（失败则保留系统默认并告警）。
	applyTimezone(cfg.GetTimezone())

	primaryDSN := dsn(cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.TLS)

	// 根据配置决定是否开启 SQL 日志
	gormLog := gormlogger.Default.LogMode(gormlogger.Silent)
	if cfg.Debug {
		gormLog = gormlogger.Default.LogMode(gormlogger.Info)
		logger.Info("GORM SQL debug mode enabled")
	}

	db, err := gorm.Open(mysql.Open(primaryDSN), &gorm.Config{
		Logger:                                   gormLog,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to primary database: %w", err)
	}

	// 注册 DBResolver 读写分离插件（仅在配置了从库时生效）
	if len(cfg.Replicas) > 0 {
		resolverCfg := dbresolver.Config{}
		var replicaDialectors []gorm.Dialector

		for i, replica := range cfg.Replicas {
			rDSN := dsn(replica.User, replica.Password, replica.Host, replica.Port, replica.Name, replica.TLS)
			dialector := mysql.Open(rDSN)
			replicaDialectors = append(replicaDialectors, dialector)

			weight := replica.Weight
			if weight <= 0 {
				weight = 1 // 默认权重 1
			}
			logger.Infof("Registering replica [%d] %s:%d (weight=%d)", i+1, replica.Host, replica.Port, weight)
		}

		resolverCfg.Replicas = replicaDialectors

		resolver := dbresolver.Register(resolverCfg).
			SetMaxIdleConns(20).
			SetMaxOpenConns(100).
			SetConnMaxLifetime(time.Hour)

		if err := db.Use(resolver); err != nil {
			logger.Warnf("Failed to register DBResolver plugin, running without read-write splitting: %v", err)
		} else {
			logger.Infof("DBResolver registered with %d replica(s), read-write splitting enabled", len(cfg.Replicas))
		}
	} else {
		logger.Info("No replicas configured, database running in single-instance mode")
	}

	// 注册 OTel 追踪插件（如果全局 TracerProvider 已初始化，会自动关联 Trace）
	if err := db.Use(otelgorm.NewPlugin()); err != nil {
		logger.Warnf("Failed to register GORM OTel plugin, DB tracing disabled: %v", err)
	} else {
		logger.Info("GORM OTel tracing plugin registered")
	}

	// 主库连接池配置
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logger.Infof("Database [%s] connected successfully", cfg.Name)
	return db, nil
}

// dsn 构建 MySQL 连接字符串
func dsn(user, password, host string, port int, name string, tls bool) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%v",
		user, password, host, port, name, tls)
}

// applyTimezone 将全局 time.Local 强制设为指定名称时区。
// 名称无效时保持系统默认不变并告警，避免因配置错误导致进程无法启动。
func applyTimezone(name string) {
	if name == "" {
		return
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		logger.Warnf("无效的业务时区 %q，沿用系统默认时区（Local=%s）: %v", name, time.Local, err)
		return
	}
	time.Local = loc
	logger.Infof("Business timezone set to %s (now=%s)", name, time.Now().Format("2006-01-02 15:04:05 MST"))
}
