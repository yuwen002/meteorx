package middleware

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/repository"
	"meteorx/internal/modules/audit/service"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

func newAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// 必须用具名 shared-cache 内存库：DSN 为裸 ":memory:" 时，sqlite 连接池每新开一条连接
	// 都会拿到一个全新的空库，AutoMigrate 的表只对第一条连接可见；后台 flush 与轮询断言
	// 并发时就会随机撞出“no such table: audit_logs”。与 testkit 保持同一口径。
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	// 单一连接：内存库的生命系于至少一条存活连接，且避开多连接的可见性差异。
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	require.NoError(t, repository.AutoMigrate(db))
	return db
}

func countAuditRows(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM audit_logs").Scan(&n).Error)
	return n
}

// TestAuditBatchStopFlushesRemainingLogs 锁定优雅关闭时的收尾刷新：
// bootstrap 的关闭顺序是先 cancel() 再 StopAuditBatchProcessor()，处理器持有的是已被取消的
// app context。若最终 flush 沿用该上下文，批量写入会直接失败，缓冲区里最多 bufferSize 条
// 审计日志被静默丢弃。此处断言关闭后缓冲日志必须落库。
func TestAuditBatchStopFlushesRemainingLogs(t *testing.T) {
	db := newAuditTestDB(t)
	svc := service.NewAuditService(repository.NewAuditLogRepository(db))

	// bufferSize 设为 100 且只投 3 条，确保不触发容量 flush，只能靠 Stop 收尾刷出
	ctx, cancel := context.WithCancel(context.Background())
	p := NewAuditBatchProcessor(ctx, svc, 100, time.Hour)
	defer cancel()

	for i := 0; i < 3; i++ {
		p.Add(&dto.CreateAuditLogReq{
			UserID:   "u1",
			TenantID: "t1",
			Module:   "test",
			Action:   "read",
			Method:   "GET",
			Path:     "/api/v1/probe",
			Result:   "success",
		})
	}

	// 复现 bootstrap 的时序：先取消 app ctx，再停止处理器做收尾 flush
	cancel()
	p.Stop()

	require.Equal(t, int64(3), countAuditRows(t, db),
		"Stop 的收尾刷新不能因 app ctx 已取消而丢弃缓冲中的审计日志")
}

// TestAuditBatchFlushOnCapacity 验证正常运行下达到缓冲区容量即异步落库（不依赖 Stop）。
func TestAuditBatchFlushOnCapacity(t *testing.T) {
	db := newAuditTestDB(t)
	svc := service.NewAuditService(repository.NewAuditLogRepository(db))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := NewAuditBatchProcessor(ctx, svc, 2, time.Hour)
	p.Add(&dto.CreateAuditLogReq{UserID: "u1", TenantID: "t1", Path: "/a", Result: "success"})
	p.Add(&dto.CreateAuditLogReq{UserID: "u1", TenantID: "t1", Path: "/b", Result: "success"})

	// 容量触发的是异步 flush，轮询等待落库
	require.Eventually(t, func() bool {
		return countAuditRows(t, db) == 2
	}, 2*time.Second, 10*time.Millisecond, "达到缓冲区容量后应异步落库")

	p.Stop()
}
