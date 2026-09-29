package repository

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/modules/audit/model"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
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