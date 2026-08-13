package bootstrap

import (
	"fmt"
	"log"
	"meteorx/internal/modules/audit/repository"
	"os"
	"path/filepath"
	"strings"

	planrepo "meteorx/internal/modules/plan/repository"
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

	err = planrepo.AutoMigrate(db)
	if err != nil {
		log.Printf("Plan migration failed: %v", err)
		return err
	}

	err = rbacrepo.AutoMigrate(db)
	if err != nil {
		log.Printf("RBAC migration failed: %v", err)
		return err
	}

	err = repository.AutoMigrate(db)
	if err != nil {
		log.Printf("Audit migration failed: %v", err)
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

	// 读取 seed.sql 文件 - 尝试多个可能的路径
	possiblePaths := []string{
		filepath.Join("scripts", "sql", "seed.sql"),
		filepath.Join("..", "scripts", "sql", "seed.sql"),
		filepath.Join(".", "scripts", "sql", "seed.sql"),
	}

	var sqlBytes []byte
	var err error
	var seedFile string

	for _, path := range possiblePaths {
		fmt.Printf("Trying to read seed file: %s\n", path)
		sqlBytes, err = os.ReadFile(path)
		if err == nil {
			seedFile = path
			break
		}
	}

	if err != nil {
		log.Printf("Failed to read seed file from all possible paths: %v", err)
		return fmt.Errorf("读取种子文件失败: %v", err)
	}

	fmt.Printf("Successfully read seed file: %s (size: %d bytes)\n", seedFile, len(sqlBytes))

	// 解析并逐条执行 SQL 语句
	sqlContent := string(sqlBytes)
	statements := parseSQLStatements(sqlContent)
	fmt.Printf("Parsed %d SQL statements\n", len(statements))

	// 使用事务执行 SQL
	err = db.Transaction(func(tx *gorm.DB) error {
		for i, stmt := range statements {
			if stmt == "" {
				continue
			}
			fmt.Printf("Executing statement %d: %s...\n", i+1, stmt[:min(len(stmt), 50)])
			if err := tx.Exec(stmt).Error; err != nil {
				log.Printf("Failed to execute statement %d: %v", i+1, err)
				return fmt.Errorf("执行第 %d 条 SQL 失败: %v", i+1, err)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Failed to execute seed data: %v", err)
		return fmt.Errorf("执行种子数据失败: %v", err)
	}

	fmt.Println("Seed data initialized successfully")
	return nil
}

// parseSQLStatements 解析 SQL 文件，将多条语句分割成数组
func parseSQLStatements(sqlContent string) []string {
	var statements []string
	var currentStmt strings.Builder
	lines := strings.Split(sqlContent, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过空行和注释
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		currentStmt.WriteString(line)
		currentStmt.WriteString("\n")
		// 以分号结尾表示一条语句结束
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(currentStmt.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			currentStmt.Reset()
		}
	}

	return statements
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
