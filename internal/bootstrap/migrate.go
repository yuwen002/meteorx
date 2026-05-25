package bootstrap

import (
	"fmt"
	"log"

	authrepo "meteorx/internal/modules/auth/repository"
	tenantrepo "meteorx/internal/modules/tenant/repository"

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

	fmt.Println("Migrations completed successfully")
	return nil
}
