package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"meteorx/internal/modules/audit/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func newAuditTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
		_ = sqlDB.Close()
	})
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("failed to open gorm: %v", err)
	}
	return gormDB, mock
}

func sampleAuditLog() *model.AuditLog {
	now := time.Now()
	return &model.AuditLog{
		ID:         "al-001",
		UserID:     "u-001",
		Username:   "alice",
		TenantID:   "t-001",
		Module:     "auth",
		Action:     "login",
		Resource:   "/api/v1/auth/login",
		Method:     "POST",
		Path:       "/api/v1/auth/login",
		StatusCode: 200,
		Result:     "success",
		ClientIP:   "192.168.1.1",
		RiskLevel:  "low",
		CreatedAt:  now,
	}
}

func TestAuditLogRepo_Create(t *testing.T) {
	gormDB, mock := newAuditTestDB(t)
	repo := NewAuditLogRepository(gormDB)

	mock.ExpectExec("INSERT INTO `audit_logs`").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.Create(context.Background(), sampleAuditLog())
	assert.NoError(t, err)
}

func TestAuditLogRepo_BatchCreate(t *testing.T) {
	gormDB, mock := newAuditTestDB(t)
	repo := NewAuditLogRepository(gormDB)

	mock.ExpectExec("INSERT INTO `audit_logs`").
		WillReturnResult(sqlmock.NewResult(2, 2))

	logs := []*model.AuditLog{sampleAuditLog(), sampleAuditLog()}
	logs[1].ID = "al-002"
	err := repo.BatchCreate(context.Background(), logs)
	assert.NoError(t, err)
}

func TestAuditLogRepo_BatchCreate_Empty(t *testing.T) {
	gormDB, _ := newAuditTestDB(t)
	repo := NewAuditLogRepository(gormDB)

	err := repo.BatchCreate(context.Background(), nil)
	assert.NoError(t, err)
}

func TestAuditLogRepo_GetByID(t *testing.T) {
	gormDB, mock := newAuditTestDB(t)
	repo := NewAuditLogRepository(gormDB)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "user_id", "username", "tenant_id", "module", "action", "resource", "resource_id",
		"method", "path", "request_body", "response_body", "status_code", "result", "error_message",
		"client_ip", "ip_location", "user_agent", "device_info", "duration", "session_id",
		"request_id", "trace_id", "referer", "risk_level", "tags", "created_at",
	}).AddRow(
		"al-001", "u-001", "alice", "t-001", "auth", "login", "/api/v1/auth/login", "",
		"POST", "/api/v1/auth/login", "", "", 200, "success", "",
		"192.168.1.1", "", "", "", 0, "", "", "", "", "low", "", now,
	)
	mock.ExpectQuery("SELECT .+ FROM `audit_logs` WHERE id = \\? .+ LIMIT \\?").
		WithArgs("al-001", 1).
		WillReturnRows(rows)

	log, err := repo.GetByID(context.Background(), "al-001")
	assert.NoError(t, err)
	assert.Equal(t, "al-001", log.ID)
	assert.Equal(t, "alice", log.Username)
	assert.Equal(t, "auth", log.Module)
}

// TestAuditLogRepo_GetAnomalyLogs_ThreeTypes 异常检测应同时产出高频失败、暴力破解、异地登录三类，
// 且各自落到正确的风险等级/详情。以 sqlmock 按序喂三张分组查询的结果行。
func TestAuditLogRepo_GetAnomalyLogs_ThreeTypes(t *testing.T) {
	gormDB, mock := newAuditTestDB(t)
	repo := NewAuditLogRepository(gormDB)
	now := time.Now()

	// 1. 高频失败：alice 8 次失败（未达 5*3，中风险）
	mock.ExpectQuery(regexp.QuoteMeta("as total")).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "username", "count", "total", "min_time", "max_time"}).
			AddRow("u-001", "alice", int64(10), int64(8), now.Add(-5*time.Minute), now))
	// 2. 暴力破解：单 IP 6 次登录失败、命中 3 个账号（未达 5*3，高风险）
	mock.ExpectQuery(regexp.QuoteMeta("as distinct_users")).
		WillReturnRows(sqlmock.NewRows([]string{"client_ip", "failures", "distinct_users", "min_time", "max_time"}).
			AddRow("1.2.3.4", int64(6), int64(3), now.Add(-3*time.Minute), now))
	// 3. 异地登录：bob 窗口内 2 个不同地点成功登录（高风险）
	mock.ExpectQuery(regexp.QuoteMeta("as loc_count")).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "username", "loc_count", "min_time", "max_time"}).
			AddRow("u-002", "bob", int64(2), now.Add(-10*time.Minute), now))

	anomalies, err := repo.GetAnomalyLogs(context.Background(), "", 5, 30)
	require.NoError(t, err)
	require.Len(t, anomalies, 3)

	byType := map[string]*model.AnomalyLog{}
	for _, a := range anomalies {
		byType[a.AnomalyType] = a
	}
	require.Contains(t, byType, model.AnomalyTypeHighFailure)
	require.Contains(t, byType, model.AnomalyTypeBruteForce)
	require.Contains(t, byType, model.AnomalyTypeGeoAnomaly)

	assert.Equal(t, model.RiskMedium, byType[model.AnomalyTypeHighFailure].RiskLevel)
	assert.Equal(t, model.RiskHigh, byType[model.AnomalyTypeBruteForce].RiskLevel)
	assert.Equal(t, int64(3), byType[model.AnomalyTypeBruteForce].TotalCount)
	assert.Contains(t, byType[model.AnomalyTypeBruteForce].Details, "1.2.3.4")
	assert.Equal(t, model.RiskHigh, byType[model.AnomalyTypeGeoAnomaly].RiskLevel)
	assert.NotEmpty(t, byType[model.AnomalyTypeHighFailure].Details)
}

// TestAuditLogRepo_GetAnomalyLogs_TenantScoped 传入 tenantID 时三类查询都必须带 tenant_id 过滤，
// 否则 sqlmock 因“tenant_id”正则不匹配而报错——以此锁定跨租户隔离。
func TestAuditLogRepo_GetAnomalyLogs_TenantScoped(t *testing.T) {
	gormDB, mock := newAuditTestDB(t)
	repo := NewAuditLogRepository(gormDB)

	// 三张分组查询的 SQL 均需含 tenant_id = ?，命中即返回空集。
	for i := 0; i < 3; i++ {
		mock.ExpectQuery("tenant_id = \\?").
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "username", "count", "total", "min_time", "max_time"}))
	}

	anomalies, err := repo.GetAnomalyLogs(context.Background(), "t-001", 5, 30)
	require.NoError(t, err)
	require.Empty(t, anomalies)
}
