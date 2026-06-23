package bootstrap

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	rbacrepo "meteorx/internal/modules/rbac/repository"
	tenantrepo "meteorx/internal/modules/tenant/repository"
	authrepo "meteorx/internal/modules/user/repository"

	"gorm.io/gorm"
)

// AutoMigrate 执行数据库迁移
func AutoMigrate(db *gorm.DB) error {
	fmt.Println("Running database migrations...")

	err := authrepo.AutoMigrate(db)
	if err != nil {
		log.Printf("Migration failed: %v", err)
		return err
	}

	err = tenantrepo.AutoMigrate(db)
	if err != nil {
		log.Printf("Migration failed: %v", err)
		return err
	}

	err = rbacrepo.AutoMigrate(db)
	if err != nil {
		log.Printf("RBAC migration failed: %v", err)
		return err
	}

	fmt.Println("Migrations completed successfully")
	return nil
}

// SeedDatabase 执行种子数据初始化
func SeedDatabase(db *gorm.DB) error {
	fmt.Println("Checking seed data...")

	// 检查 roles 表是否已有数据
	var count int64
	db.Raw("SELECT COUNT(*) FROM roles WHERE deleted_at IS NULL").Scan(&count)
	if count > 0 {
		fmt.Printf("Seed data already exists (%d roles), skipping...\n", count)
		return nil
	}

	fmt.Println("No roles found, initializing seed data...")

	// 读取 seed.sql 文件
	seedFile := filepath.Join("scripts", "sql", "seed.sql")
	fmt.Printf("Reading seed file: %s\n", seedFile)
	
	sqlBytes, err := os.ReadFile(seedFile)
	if err != nil {
		log.Printf("Failed to read seed file: %v", err)
		return fmt.Errorf("读取种子文件失败: %v", err)
	}

	fmt.Printf("Seed file size: %d bytes\n", len(sqlBytes))

	// 执行 SQL（逐行执行，跳过注释）
	sqlContent := string(sqlBytes)
	if err := db.Exec(sqlContent).Error; err != nil {
		log.Printf("Failed to execute seed data: %v", err)
		return fmt.Errorf("执行种子数据失败: %v", err)
	}

	fmt.Println("Seed data initialized successfully")
	return nil
}
