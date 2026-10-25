package service

import (
	"context"
	"errors"
	"testing"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"

	"github.com/stretchr/testify/require"
)

// errWriteFail 注入仓储写入失败的哨兵错误。
var errWriteFail = errors.New("audit write failed")

// stubEvaluator 记录审计写入路径回调进来的日志，用于断言告警评估链的触发时机。
type stubEvaluator struct {
	batches [][]*model.AuditLog
}

func (e *stubEvaluator) EvaluateAuditLogs(_ context.Context, logs []*model.AuditLog) {
	e.batches = append(e.batches, logs)
}

func highRiskReq() dto.CreateAuditLogReq {
	return dto.CreateAuditLogReq{
		UserID: "u-001", Username: "alice", Module: "auth",
		Action: model.ActionTypeDelete, Result: model.ResultSuccess,
		RiskLevel: model.RiskHigh,
	}
}

// TestAuditService_CreateLog_TriggersEvaluation 单条直写成功后必须把落库的日志
// 交给告警评估链——接入前 EvaluateAndAlert 在生产里没有任何调用方，整条告警链
// 是永远不会触发的死代码。
func TestAuditService_CreateLog_TriggersEvaluation(t *testing.T) {
	repo := repository.NewMockAuditLogRepository()
	svc := NewAuditService(repo)
	eval := &stubEvaluator{}
	svc.SetAlertEvaluator(eval)

	created, err := svc.CreateLog(context.Background(), highRiskReq())
	require.NoError(t, err)

	require.Len(t, eval.batches, 1, "直写成功必须触发一次评估")
	require.Len(t, eval.batches[0], 1)
	require.Equal(t, created.ID, eval.batches[0][0].ID, "评估的必须是刚落库的那条日志")
}

// TestAuditService_BatchCreateLogs_TriggersEvaluation 批量路径（审计中间件主通道）
// 整批成功后一次性交给评估链。
func TestAuditService_BatchCreateLogs_TriggersEvaluation(t *testing.T) {
	repo := repository.NewMockAuditLogRepository()
	svc := NewAuditService(repo)
	eval := &stubEvaluator{}
	svc.SetAlertEvaluator(eval)

	require.NoError(t, svc.BatchCreateLogs(context.Background(),
		[]dto.CreateAuditLogReq{highRiskReq(), highRiskReq()}))

	require.Len(t, eval.batches, 1, "一批日志只触发一次评估（规则查询 amortize）")
	require.Len(t, eval.batches[0], 2)
}

// TestAuditService_WriteFailureDoesNotEvaluate 落库失败时不得评估：日志行并不存在，
// 基于它触发的告警会指向一条查无此记录的 audit_log_id。
func TestAuditService_WriteFailureDoesNotEvaluate(t *testing.T) {
	repo := repository.NewMockAuditLogRepository()
	repo.SetBatchCreateErr(errWriteFail)
	svc := NewAuditService(repo)
	eval := &stubEvaluator{}
	svc.SetAlertEvaluator(eval)

	require.Error(t, svc.BatchCreateLogs(context.Background(), []dto.CreateAuditLogReq{highRiskReq()}))
	require.Empty(t, eval.batches, "写入失败不得触发评估")

	repo2 := repository.NewMockAuditLogRepository()
	repo2.SetCreateErr(errWriteFail)
	svc2 := NewAuditService(repo2)
	eval2 := &stubEvaluator{}
	svc2.SetAlertEvaluator(eval2)
	_, err := svc2.CreateLog(context.Background(), highRiskReq())
	require.Error(t, err)
	require.Empty(t, eval2.batches)
}

// TestAuditService_NoEvaluatorNoPanic 未接线（如测试环境、模块未启用告警）时
// 写入路径必须保持原行为。
func TestAuditService_NoEvaluatorNoPanic(t *testing.T) {
	svc := NewAuditService(repository.NewMockAuditLogRepository())
	_, err := svc.CreateLog(context.Background(), highRiskReq())
	require.NoError(t, err)
	require.NoError(t, svc.BatchCreateLogs(context.Background(), []dto.CreateAuditLogReq{highRiskReq()}))
}

// TestEvaluateAuditLogs_SharedRuleFetch 批量评估只查一次启用规则（整批共用），
// 但冷却判定逐条生效：第一条 high 产出告警后，同规则第二条 high 落在冷却窗口内
// 被拦住（与单条路径同口径），low 不匹配不产出。
func TestEvaluateAuditLogs_SharedRuleFetch(t *testing.T) {
	repo := repository.NewMockAlertRuleRepository()
	svc := NewAlertService(repo, nil)
	_, err := svc.CreateRule(context.Background(), dto.CreateAlertRuleReq{
		Name: "高风险告警", Enabled: true,
		TriggerType: model.TriggerTypeRiskLevel, TriggerValue: model.RiskHigh,
		NotifyChannels: []string{model.NotifyChannelEmail},
		NotifyTargets:  []string{"admin@example.com"},
	})
	require.NoError(t, err)

	logs := []*model.AuditLog{
		{ID: "l1", UserID: "u1", Username: "alice", Action: "delete", RiskLevel: model.RiskHigh},
		{ID: "l2", UserID: "u2", Username: "bob", Action: "delete", RiskLevel: model.RiskHigh},
		{ID: "l3", UserID: "u3", Username: "carol", Action: "query", RiskLevel: model.RiskLow},
	}
	require.NotPanics(t, func() { svc.EvaluateAuditLogs(context.Background(), logs) })

	require.Equal(t, 1, repo.RuleFetches, "整批只允许查询一次启用规则")
	alerts, total, err := repo.ListAlerts(context.Background(), 1, 10, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total, "同规则第二条 high 被冷却拦住，low 不产出")
	require.Equal(t, "l1", alerts[0].AuditLogID)
}
