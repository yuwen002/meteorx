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

func newAlertTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func sampleAlertRule() *model.AlertRule {
	now := time.Now()
	return &model.AlertRule{
		ID:              "ar-001",
		Name:            "critical-risk-alert",
		Description:     "alert on critical risk",
		Enabled:         true,
		TriggerType:     model.TriggerTypeRiskLevel,
		TriggerValue:    model.RiskCritical,
		NotifyChannels:  `["email","dingtalk"]`,
		NotifyTargets:   `["admin@example.com"]`,
		NotifyTemplate:  "alert template",
		CooldownMinutes: 15,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func sampleAuditAlert() *model.AuditAlert {
	now := time.Now()
	return &model.AuditAlert{
		ID:         "aa-001",
		RuleID:     "ar-001",
		RuleName:   "critical-risk-alert",
		AuditLogID: "al-001",
		UserID:     "u-001",
		Username:   "alice",
		RiskLevel:  "critical",
		Action:     "delete",
		Message:    "user alice deleted resource",
		Notified:   false,
		CreatedAt:  now,
	}
}

func TestAlertRuleRepo_Create(t *testing.T) {
	gormDB, mock := newAlertTestDB(t)
	repo := NewAlertRuleRepository(gormDB)
	mock.ExpectExec("INSERT INTO `audit_alert_rules`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	err := repo.Create(context.Background(), sampleAlertRule())
	assert.NoError(t, err)
}

func TestAlertRuleRepo_GetByID(t *testing.T) {
	gormDB, mock := newAlertTestDB(t)
	repo := NewAlertRuleRepository(gormDB)
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "name", "description", "enabled", "trigger_type", "trigger_value",
		"notify_channels", "notify_targets", "notify_template", "cooldown_minutes",
		"created_at", "updated_at",
	}).AddRow(
		"ar-001", "critical-risk-alert", "alert on critical risk",
		true, "risk_level", "critical",
		`["email"]`, `["admin@example.com"]`, "template", 15, now, now,
	)
	mock.ExpectQuery("SELECT .+ FROM `audit_alert_rules` WHERE id = \\? .+ LIMIT \\?").
		WithArgs("ar-001", 1).
		WillReturnRows(rows)
	rule, err := repo.GetByID(context.Background(), "ar-001")
	assert.NoError(t, err)
	assert.Equal(t, "ar-001", rule.ID)
	assert.Equal(t, "critical-risk-alert", rule.Name)
	assert.True(t, rule.Enabled)
}

func TestAlertRuleRepo_Delete(t *testing.T) {
	gormDB, mock := newAlertTestDB(t)
	repo := NewAlertRuleRepository(gormDB)
	mock.ExpectExec("DELETE FROM `audit_alert_rules` WHERE id = \\?").
		WithArgs("ar-001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	err := repo.Delete(context.Background(), "ar-001")
	assert.NoError(t, err)
}

func TestAlertRuleRepo_CreateAlert(t *testing.T) {
	gormDB, mock := newAlertTestDB(t)
	repo := NewAlertRuleRepository(gormDB)
	mock.ExpectExec("INSERT INTO `audit_alerts`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	err := repo.CreateAlert(context.Background(), sampleAuditAlert())
	assert.NoError(t, err)
}

func TestAlertRuleRepo_GetAlertByID(t *testing.T) {
	gormDB, mock := newAlertTestDB(t)
	repo := NewAlertRuleRepository(gormDB)
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "rule_id", "rule_name", "audit_log_id", "user_id", "username",
		"risk_level", "action", "message", "notified", "notify_time", "created_at",
	}).AddRow(
		"aa-001", "ar-001", "critical-risk-alert", "al-001", "u-001", "alice",
		"critical", "delete", "alert msg", false, time.Time{}, now,
	)
	mock.ExpectQuery("SELECT .+ FROM `audit_alerts` WHERE id = \\? .+ LIMIT \\?").
		WithArgs("aa-001", 1).
		WillReturnRows(rows)
	alert, err := repo.GetAlertByID(context.Background(), "aa-001")
	assert.NoError(t, err)
	assert.Equal(t, "aa-001", alert.ID)
	assert.Equal(t, "critical", alert.RiskLevel)
}

func TestAlertRuleRepo_ListAlerts(t *testing.T) {
	gormDB, mock := newAlertTestDB(t)
	repo := NewAlertRuleRepository(gormDB)
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `audit_alerts`").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT .+ FROM `audit_alerts` ORDER BY created_at DESC LIMIT \\?").
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "rule_id", "rule_name", "audit_log_id", "user_id", "username",
			"risk_level", "action", "message", "notified", "notify_time", "created_at",
		}))
	alerts, total, err := repo.ListAlerts(context.Background(), 1, 10, "", "", "")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, alerts, 0)
}

func TestAlertRuleRepo_IsInCooldown(t *testing.T) {
	gormDB, mock := newAlertTestDB(t)
	repo := NewAlertRuleRepository(gormDB)
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `audit_alerts` WHERE rule_id = \\? AND created_at > \\?").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	inCooldown, err := repo.IsInCooldown(context.Background(), "ar-001", 30)
	assert.NoError(t, err)
	assert.False(t, inCooldown)
}

func TestAlertRuleRepo_IsInCooldown_Active(t *testing.T) {
	gormDB, mock := newAlertTestDB(t)
	repo := NewAlertRuleRepository(gormDB)
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `audit_alerts` WHERE rule_id = \\? AND created_at > \\?").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	inCooldown, err := repo.IsInCooldown(context.Background(), "ar-001", 30)
	assert.NoError(t, err)
	assert.True(t, inCooldown)
}