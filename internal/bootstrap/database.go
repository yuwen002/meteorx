package bootstrap

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"meteorx/internal/config"
	"meteorx/pkg/logger"
)

// InitDB 返回一个 *gorm.DB 实例，而不是存放在全局变量
func InitDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%v",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.TLS, // 既然是 bool 类型，这里直接用 %v
	)

	// 根据配置决定是否开启 SQL 日志
	gormLog := gormlogger.Default.LogMode(gormlogger.Silent)
	if cfg.Debug {
		gormLog = gormlogger.Default.LogMode(gormlogger.Info)
		logger.Info("GORM SQL debug mode enabled")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                                   gormLog,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

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
