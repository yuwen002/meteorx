package bootstrap

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"meteorx/internal/config"
	"meteorx/pkg/logger"
)

// DBManager 数据库连接管理器（读写分离）
type DBManager struct {
	db      *gorm.DB
	replicas []*gorm.DB
	mu      sync.RWMutex

	// 读写策略
	readWriteMode string
	replicaIndex  int
	replicaLock   sync.Mutex

	// 连接配置
	replicaConfigs []config.ReplicaConfig
}

// NewDBManager 创建数据库连接管理器
func NewDBManager(cfg *config.DatabaseConfig, replicaConfigs []config.ReplicaConfig, readWriteMode string) (*DBManager, error) {
	mgr := &DBManager{
		db:              nil,
		replicas:        make([]*gorm.DB, 0, len(replicaConfigs)),
		readWriteMode:   readWriteMode,
		replicaConfigs:  replicaConfigs,
		replicaIndex:    0,
	}

	// 连接主库
	mgr.db, err := initDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("init primary database failed: %w", err)
	}
	logger.Info("Primary database connected successfully")

	// 连接从库
	for _, replicaCfg := range replicaConfigs {
		replicaDB, err := initReplicaDB(&replicaCfg)
		if err != nil {
			logger.Warnf("Connect to replica database %s:%d failed: %v, will continue with replicas available",
				replicaCfg.Host, replicaCfg.Port, err)
			continue
		}
		mgr.replicas = append(mgr.replicas, replicaDB)
		logger.Infof("Replica database connected: %s:%d (read_only=%v)",
			replicaCfg.Host, replicaCfg.Port, replicaCfg.ReadOnly)
	}

	// 如果没有可用的从库，记录警告
	if len(mgr.replicas) == 0 {
		logger.Warn("No available replica databases, falling back to primary database for all operations")
	}

	return mgr, nil
}

// GetDB 获取数据库实例（根据读写模式）
func (m *DBManager) GetDB() *gorm.DB {
	switch m.readWriteMode {
	case "write":
		return m.db
	case "read":
		if len(m.replicas) > 0 {
			return m.getRandomReplica()
		}
		return m.db
	case "read_write":
		if len(m.replicas) > 0 {
			// 查询操作优先使用从库
			if !m.isWriteOperation() {
				return m.getRandomReplica()
			}
		}
		return m.db
	default:
		return m.db
	}
}

// isWriteOperation 判断是否为写操作（通过 SQL 分析）
// 注意：这是一个简化实现，实际中可能需要更复杂的分析
func (m *DBManager) isWriteOperation() bool {
	// GORM 会自动在写入操作时使用 Master 连接
	// 这里主要是为了调试和日志记录
	// 如果需要更精确的控制，可以通过 hook 实现
	return false
}

// getRandomReplica 获取随机从库（用于负载均衡）
func (m *DBManager) getRandomReplica() *gorm.DB {
	if len(m.replicas) == 0 {
		return m.db
	}

	m.replicaLock.Lock()
	defer m.replicaLock.Unlock()

	// 简单轮询负载均衡
	idx := m.replicaIndex % len(m.replicas)
	m.replicaIndex++

	logger.Debugf("Using replica database index: %d/%d", idx+1, len(m.replicas))
	return m.replicas[idx]
}

// Close 关闭所有数据库连接
func (m *DBManager) Close() error {
	var errs []error

	// 关闭主库
	if m.db != nil {
		sqlDB, err := m.db.DB()
		if err == nil {
			if cerr := sqlDB.Close(); cerr != nil {
				errs = append(errs, cerr)
			} else {
				logger.Info("Primary database connection closed")
			}
		}
	}

	// 关闭从库
	for _, replica := range m.replicas {
		if replica != nil {
			sqlDB, err := replica.DB()
			if err == nil {
				if cerr := sqlDB.Close(); cerr != nil {
					errs = append(errs, cerr)
				}
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("database close errors: %v", errs)
	}
	return nil
}

// initDB 初始化主库连接
func initDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%v",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.TLS)

	// 配置日志
	var gormLog gormlogger.Interface
	if cfg.Debug {
		gormLog = gormlogger.Default.LogMode(gormlogger.Info)
		logger.Info("GORM SQL debug mode enabled")
	} else {
		gormLog = gormlogger.Default.LogMode(gormlogger.Silent)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                                   gormLog,
		DisableForeignKeyConstraintWhenMigrating: true,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to primary database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 主库连接池配置
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)  // 主库连接数较少
	sqlDB.SetConnMaxLifetime(time.Hour)

	logger.Infof("Primary database [%s] connected successfully", cfg.Name)
	return db, nil
}

// initReplicaDB 初始化从库连接
func initReplicaDB(cfg *config.ReplicaConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%v",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, false)

	var gormLog gormlogger.Interface
	// 从库可以开启慢查询日志
	gormLog = gormlogger.Default.LogMode(gormlogger.Silent)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                                   gormLog,
		DisableForeignKeyConstraintWhenMigrating: true,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to replica database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 从库连接池配置（可以比主库更宽松）
	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetMaxOpenConns(100) // 从库可以横向扩展
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}
