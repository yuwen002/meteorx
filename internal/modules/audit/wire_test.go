// Package audit 的接线测试：告警评估链必须真实挂到审计日志写入路径上。
// 接入前 EvaluateAndAlert 在生产里没有任何调用方——规则、告警记录、通知渠道
// 全部就绪，唯独没有触发者，整条链是死代码。
package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"
	"meteorx/internal/modules/audit/service"
	"meteorx/internal/notify"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// newWireDB 具名 shared-cache 内存库（与 service 层真库用例同口径）
func newWireDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"),
		&gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, repository.AutoMigrate(db))
	return db
}

// TestWireAlertEvaluation_EndToEnd 经生产接线函数 WireAlertEvaluation 装配后，
// 走真实审计批量写入路径必须自动产出并送达告警（httptest webhook 直发渠道）：
// 告警记录落库、notified 经 UPDATE 真实回写。接线被删时 BatchCreateLogs 只写日志，
// hookHits 与告警数都为 0。
func TestWireAlertEvaluation_EndToEnd(t *testing.T) {
	prev := notify.GetGlobalManager()
	notify.SetGlobalManager(nil) // 只验证直发渠道与落库回写
	t.Cleanup(func() { notify.SetGlobalManager(prev) })

	var hookHits int32
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hookHits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer hook.Close()

	db := newWireDB(t)
	alertRepo := repository.NewAlertRuleRepository(db)
	alertSvc := service.NewAlertService(alertRepo, nil)
	_, err := alertSvc.CreateRule(context.Background(), dto.CreateAlertRuleReq{
		Name:            "高风险操作告警",
		Enabled:         true,
		TriggerType:     model.TriggerTypeRiskLevel,
		TriggerValue:    model.RiskHigh,
		NotifyChannels:  []string{model.NotifyChannelWebhook},
		NotifyTargets:   []string{hook.URL},
		CooldownMinutes: 30,
	})
	require.NoError(t, err)

	auditSvc := service.NewAuditService(repository.NewAuditLogRepository(db))
	WireAlertEvaluation(auditSvc, db, nil)

	require.NoError(t, auditSvc.BatchCreateLogs(context.Background(),
		[]dto.CreateAuditLogReq{{
			UserID: "u-001", Username: "alice", Module: "auth",
			Action: model.ActionTypeDelete, Result: model.ResultSuccess,
			RiskLevel: model.RiskHigh,
		}}))

	require.Equal(t, int32(1), hookHits, "批量写入路径必须自动触发告警送达")

	alerts, total, err := alertRepo.ListAlerts(context.Background(), 1, 10, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total, "告警记录须由写入路径自动产出")
	fresh, err := alertRepo.GetAlertByID(context.Background(), alerts[0].ID)
	require.NoError(t, err)
	require.True(t, fresh.Notified, "送达后 notified 必须真实落库")
	require.False(t, fresh.NotifyTime.IsZero())
}

// TestWireAlertEvaluation_UnmatchedLogNoAlert 低风险日志经同一写入路径落库时
// 不得产出告警——防止接线被改成「无条件告警」。
func TestWireAlertEvaluation_UnmatchedLogNoAlert(t *testing.T) {
	db := newWireDB(t)
	alertRepo := repository.NewAlertRuleRepository(db)
	alertSvc := service.NewAlertService(alertRepo, nil)
	_, err := alertSvc.CreateRule(context.Background(), dto.CreateAlertRuleReq{
		Name:           "仅高风险",
		Enabled:        true,
		TriggerType:    model.TriggerTypeRiskLevel,
		TriggerValue:   model.RiskHigh,
		NotifyChannels: []string{model.NotifyChannelWebhook},
		NotifyTargets:  []string{"http://127.0.0.1:0/never"},
	})
	require.NoError(t, err)

	auditSvc := service.NewAuditService(repository.NewAuditLogRepository(db))
	WireAlertEvaluation(auditSvc, db, nil)

	require.NoError(t, auditSvc.BatchCreateLogs(context.Background(),
		[]dto.CreateAuditLogReq{{
			UserID: "u-002", Username: "bob", Module: "user",
			Action: model.ActionTypeQuery, Result: model.ResultSuccess,
			RiskLevel: model.RiskLow,
		}}))

	_, total, err := alertRepo.ListAlerts(context.Background(), 1, 10, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(0), total, "不匹配规则的日志不得产出告警")

	// 日志本身照常落库
	logs, logTotal, err := repository.NewAuditLogRepository(db).List(context.Background(),
		&repository.AuditLogQuery{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Equal(t, int64(1), logTotal)
	require.WithinDuration(t, time.Now(), logs[0].CreatedAt, time.Minute)
}
