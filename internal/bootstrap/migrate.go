package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"meteorx/pkg/logger"

	"meteorx/internal/modules/audit/repository"
	"meteorx/internal/modules/file"
	notificationrepo "meteorx/internal/modules/notification/repository"
	planrepo "meteorx/internal/modules/plan/repository"
	rbacrepo "meteorx/internal/modules/rbac/repository"
	tenantrepo "meteorx/internal/modules/tenant/repository"
	authrepo "meteorx/internal/modules/user/repository"
	wikirepo "meteorx/internal/modules/wiki/repository"

	"gorm.io/gorm"
)

// AutoMigrate 执行数据库迁移
// 关键迁移失败会返回错误并阻止应用启动；非关键迁移仅记录警告
func AutoMigrate(db *gorm.DB) error {
	logger.Info("Running database migrations...")

	var criticalErrs []string
	var nonCriticalErrs []string

	// ---- 关键迁移：失败则阻止启动 ----
	critical := []struct {
		name string
		fn   func(db *gorm.DB) error
	}{
		{"users", authrepo.AutoMigrate},
		{"tenants", tenantrepo.AutoMigrate},
		{"tenant_settings", tenantrepo.AutoMigrateTenantSettings},
		{"cancel_requests", tenantrepo.CancelRequestsAutoMigrate},
		{"plans", planrepo.AutoMigrate},
		{"rbac", rbacrepo.AutoMigrate},
	}

	for _, m := range critical {
		if err := m.fn(db); err != nil {
			criticalErrs = append(criticalErrs, fmt.Sprintf("%s: %v", m.name, err))
		}
	}

	// ---- 非关键迁移：失败仅警告，不阻止启动 ----
	nonCritical := []struct {
		name string
		fn   func(db *gorm.DB) error
	}{
		{"audit", repository.AutoMigrate},
		{"files", file.AutoMigrate},
		{"notifications", notificationrepo.AutoMigrate},
		{"wiki", wikirepo.AutoMigrate},
	}

	for _, m := range nonCritical {
		if err := m.fn(db); err != nil {
			nonCriticalErrs = append(nonCriticalErrs, fmt.Sprintf("%s: %v", m.name, err))
		}
	}

	// 处理结果
	if len(nonCriticalErrs) > 0 {
		for _, e := range nonCriticalErrs {
			logger.Warnf("[WARN] Non-critical migration failed: %s", e)
		}
	}

	if len(criticalErrs) > 0 {
		return fmt.Errorf("critical migrations failed: %s", strings.Join(criticalErrs, "; "))
	}

	logger.Info("Migrations completed successfully")
	if len(nonCriticalErrs) > 0 {
		logger.Warnf("Note: %d non-critical migration(s) failed, but the app will continue to run", len(nonCriticalErrs))
	}
	return nil
}

// SeedDatabase 执行种子数据初始化
func SeedDatabase(db *gorm.DB) error {
	logger.Info("Checking seed data...")

	// 检查 roles 表是否已有数据
	var count int64
	db.Raw("SELECT COUNT(*) FROM roles WHERE deleted_at IS NULL").Scan(&count)
	if count > 0 {
		logger.Infof("Seed data already exists (%d roles), skipping...", count)
		return nil
	}

	logger.Info("No roles found, initializing seed data...")

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
		logger.Infof("Trying to read seed file: %s", path)
		sqlBytes, err = os.ReadFile(path)
		if err == nil {
			seedFile = path
			break
		}
	}

	if err != nil {
		logger.Errorf("Failed to read seed file from all possible paths: %v", err)
		return fmt.Errorf("读取种子文件失败: %v", err)
	}

	logger.Infof("Successfully read seed file: %s (size: %d bytes)", seedFile, len(sqlBytes))

	// 解析并逐条执行 SQL 语句
	sqlContent := string(sqlBytes)
	statements := parseSQLStatements(sqlContent)
	logger.Infof("Parsed %d SQL statements", len(statements))

	// 使用事务执行 SQL
	err = db.Transaction(func(tx *gorm.DB) error {
		for i, stmt := range statements {
			if stmt == "" {
				continue
			}
			logger.Infof("Executing statement %d: %s...", i+1, stmt[:min(len(stmt), 50)])
			if err := tx.Exec(stmt).Error; err != nil {
				logger.Errorf("Failed to execute statement %d: %v", i+1, err)
				return fmt.Errorf("执行第 %d 条 SQL 失败: %v", i+1, err)
			}
		}
		return nil
	})

	if err != nil {
		logger.Errorf("Failed to execute seed data: %v", err)
		return fmt.Errorf("执行种子数据失败: %v", err)
	}

	logger.Info("Seed data initialized successfully")
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
