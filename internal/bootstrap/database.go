package bootstrap

import (
	"fmt"
	"time"

	"meteorx/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	fmt.Printf("Database [%s] connected successfully\n", cfg.Name)
	return db, nil
}
