package service

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlertIntegration_EvaluateAndNotify(t *testing.T) {
	alertRepo := repository.NewMockAlertRuleRepository()
	svc := NewAlertService(alertRepo, nil)
	ctx := context.Background()

	rule, err := svc.CreateRule(ctx, dto.CreateAlertRuleReq{
		Name:            "高风险操作告警",
		Description:     "当检测到高风险操作时触发告警",
		Enabled:         true,
		TriggerType:     model.TriggerTypeRiskLevel,
		TriggerValue:    model.RiskHigh,
		NotifyChannels:  []string{model.NotifyChannelEmail},
		NotifyTargets:   []string{"admin@example.com"},
		NotifyTemplate:  "【告警】用户 {{.Username}} 执行了 {{.Action}} 操作，风险等级：{{.RiskLevel}}",
		CooldownMinutes: 30,
	})
	require.NoError(t, err)
	require.NotNil(t, rule)

	highRiskLog := &model.AuditLog{
		ID:         "al-high-001",
		UserID:     "u-001",
		Username:   "alice",
		TenantID:   "t-001",
		Module:     "auth",
		Action:     "delete",
		Resource:   "/api/v1/users/u-002",
		Method:     "DELETE",
		Path:       "/api/v1/users/u-002",
		StatusCode: 200,
		Result:     "success",
		ClientIP:   "192.168.1.100",
		RiskLevel:  model.RiskHigh,
		CreatedAt:  time.Now(),
	}

	err = svc.EvaluateAndAlert(ctx, highRiskLog)
	assert.NoError(t, err)

	alerts, total, err := alertRepo.ListAlerts(ctx, 1, 10, "", "", "")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, int64(1))
	require.NotEmpty(t, alerts)
	first := alerts[0]
	assert.Equal(t, rule.ID, first.RuleID)
	assert.Equal(t, "alice", first.Username)
	assert.Equal(t, model.RiskHigh, first.RiskLevel)
	assert.Contains(t, first.Message, "alice")
	assert.Contains(t, first.Message, "delete")
}

func TestAlertIntegration_LowRiskNoAlert(t *testing.T) {
	alertRepo := repository.NewMockAlertRuleRepository()
	svc := NewAlertService(alertRepo, nil)
	ctx := context.Background()

	_, err := svc.CreateRule(ctx, dto.CreateAlertRuleReq{
		Name:            "仅高风险告警",
		Enabled:         true,
		TriggerType:     model.TriggerTypeRiskLevel,
		TriggerValue:    model.RiskHigh,
		NotifyChannels:  []string{model.NotifyChannelEmail},
		NotifyTargets:   []string{"admin@example.com"},
		CooldownMinutes: 30,
	})
	require.NoError(t, err)

	lowRiskLog := &model.AuditLog{
		ID:        "al-low-001",
		UserID:    "u-001",
		Username:  "bob",
		Module:    "auth",
		Action:    "login",
		Result:    "success",
		RiskLevel: model.RiskLow,
		CreatedAt: time.Now(),
	}

	err = svc.EvaluateAndAlert(ctx, lowRiskLog)
	assert.NoError(t, err)

	_, total, err := alertRepo.ListAlerts(ctx, 1, 10, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(0), total, "低风险操作不应触发高风险告警规则")
}

func TestAlertIntegration_CooldownPreventsDuplicate(t *testing.T) {
	alertRepo := repository.NewMockAlertRuleRepository()
	svc := NewAlertService(alertRepo, nil)
	ctx := context.Background()

	_, err := svc.CreateRule(ctx, dto.CreateAlertRuleReq{
		Name:            "冷却测试规则",
		Enabled:         true,
		TriggerType:     model.TriggerTypeRiskLevel,
		TriggerValue:    model.RiskCritical,
		NotifyChannels:  []string{model.NotifyChannelEmail},
		NotifyTargets:   []string{"admin@example.com"},
		CooldownMinutes: 30,
	})
	require.NoError(t, err)

	criticalLog := &model.AuditLog{
		ID:        "al-crit-001",
		UserID:    "u-001",
		Username:  "eve",
		Module:    "auth",
		Action:    "delete",
		Result:    "success",
		RiskLevel: model.RiskCritical,
		CreatedAt: time.Now(),
	}

	err = svc.EvaluateAndAlert(ctx, criticalLog)
	assert.NoError(t, err)

	_, total1, _ := alertRepo.ListAlerts(ctx, 1, 10, "", "", "")
	assert.GreaterOrEqual(t, total1, int64(1), "首次触发应产生告警")

	err = svc.EvaluateAndAlert(ctx, criticalLog)
	assert.NoError(t, err)

	_, total2, _ := alertRepo.ListAlerts(ctx, 1, 10, "", "", "")
	assert.Equal(t, total1, total2, "冷却期内重复触发不应产生新告警")
}

func TestAlertIntegration_DisabledRuleNoAlert(t *testing.T) {
	alertRepo := repository.NewMockAlertRuleRepository()
	svc := NewAlertService(alertRepo, nil)
	ctx := context.Background()

	_, err := svc.CreateRule(ctx, dto.CreateAlertRuleReq{
		Name:            "已禁用规则",
		Enabled:         false,
		TriggerType:     model.TriggerTypeRiskLevel,
		TriggerValue:    model.RiskHigh,
		NotifyChannels:  []string{model.NotifyChannelEmail},
		NotifyTargets:   []string{"admin@example.com"},
		CooldownMinutes: 30,
	})
	require.NoError(t, err)

	highRiskLog := &model.AuditLog{
		ID:        "al-dis-001",
		UserID:    "u-001",
		Username:  "mallory",
		Module:    "auth",
		Action:    "delete",
		Result:    "success",
		RiskLevel: model.RiskHigh,
		CreatedAt: time.Now(),
	}

	err = svc.EvaluateAndAlert(ctx, highRiskLog)
	assert.NoError(t, err)

	_, total, _ := alertRepo.ListAlerts(ctx, 1, 10, "", "", "")
	assert.Equal(t, int64(0), total, "禁用规则不应触发告警")
}

func TestAlertIntegration_ActionTriggerType(t *testing.T) {
	alertRepo := repository.NewMockAlertRuleRepository()
	svc := NewAlertService(alertRepo, nil)
	ctx := context.Background()

	_, err := svc.CreateRule(ctx, dto.CreateAlertRuleReq{
		Name:            "删除操作告警",
		Enabled:         true,
		TriggerType:     model.TriggerTypeAction,
		TriggerValue:    model.ActionTypeDelete,
		NotifyChannels:  []string{model.NotifyChannelDingTalk},
		NotifyTargets:   []string{"https://oapi.dingtalk.com/robot/send"},
		CooldownMinutes: 10,
	})
	require.NoError(t, err)

	deleteLog := &model.AuditLog{
		ID:        "al-act-001",
		UserID:    "u-001",
		Username:  "charlie",
		Module:    "wiki",
		Action:    model.ActionTypeDelete,
		Result:    "success",
		RiskLevel: model.RiskMedium,
		CreatedAt: time.Now(),
	}

	err = svc.EvaluateAndAlert(ctx, deleteLog)
	assert.NoError(t, err)

	alerts, total, _ := alertRepo.ListAlerts(ctx, 1, 10, "", "", "")
	assert.GreaterOrEqual(t, total, int64(1))
	require.NotEmpty(t, alerts)
	assert.Equal(t, "charlie", alerts[0].Username)
	assert.Equal(t, model.ActionTypeDelete, alerts[0].Action)
}