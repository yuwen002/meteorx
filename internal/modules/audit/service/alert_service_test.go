package service

import (
	"context"
	"testing"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"

	"github.com/stretchr/testify/suite"
)

// AlertServiceTestSuite 告警服务测试套件
type AlertServiceTestSuite struct {
	suite.Suite
	svc       *AlertService
	alertRepo *repository.MockAlertRuleRepository
	ctx       context.Context
}

func (s *AlertServiceTestSuite) SetupTest() {
	s.alertRepo = repository.NewMockAlertRuleRepository()
	s.svc = NewAlertService(s.alertRepo, nil)
	s.ctx = context.Background()
}

func TestAlertServiceSuite(t *testing.T) {
	suite.Run(t, new(AlertServiceTestSuite))
}

// TestCreateRule 测试创建告警规则
func (s *AlertServiceTestSuite) TestCreateRule() {
	req := dto.CreateAlertRuleReq{
		Name:            "测试告警规则",
		Description:     "这是一个测试规则",
		Enabled:         true,
		TriggerType:     model.TriggerTypeRiskLevel,
		TriggerValue:    model.RiskHigh,
		NotifyChannels:  []string{model.NotifyChannelEmail},
		NotifyTargets:   []string{"admin@example.com"},
		NotifyTemplate:  "【告警】用户 {{.Username}} 执行了 {{.Action}} 操作",
		CooldownMinutes: 30,
	}

	rule, err := s.svc.CreateRule(s.ctx, req)

	s.NoError(err)
	s.NotNil(rule)
	s.NotEmpty(rule.ID)
	s.Equal(req.Name, rule.Name)
	s.Equal(req.TriggerType, rule.TriggerType)
	s.Equal(req.TriggerValue, rule.TriggerValue)
	s.True(rule.Enabled)
}

// TestUpdateRule 测试更新告警规则
func (s *AlertServiceTestSuite) TestUpdateRule() {
	// 先创建规则
	createReq := dto.CreateAlertRuleReq{
		Name:           "原始规则",
		Enabled:        true,
		TriggerType:    model.TriggerTypeRiskLevel,
		TriggerValue:   model.RiskHigh,
		NotifyChannels: []string{model.NotifyChannelEmail},
		NotifyTargets:  []string{"admin@example.com"},
	}
	rule, _ := s.svc.CreateRule(s.ctx, createReq)

	// 更新规则
	updateReq := dto.UpdateAlertRuleReq{
		Name:            "更新后的规则",
		Description:     "更新描述",
		Enabled:         false,
		TriggerType:     model.TriggerTypeAction,
		TriggerValue:    model.ActionTypeDelete,
		NotifyChannels:  []string{model.NotifyChannelDingTalk},
		NotifyTargets:   []string{"https://oapi.dingtalk.com/robot/send"},
		CooldownMinutes: 60,
	}

	updated, err := s.svc.UpdateRule(s.ctx, rule.ID, updateReq)

	s.NoError(err)
	s.NotNil(updated)
	s.Equal(updateReq.Name, updated.Name)
	s.Equal(updateReq.TriggerType, updated.TriggerType)
	s.False(updated.Enabled)
}

// TestDeleteRule 测试删除告警规则
func (s *AlertServiceTestSuite) TestDeleteRule() {
	// 先创建规则
	createReq := dto.CreateAlertRuleReq{
		Name:           "待删除规则",
		Enabled:        true,
		TriggerType:    model.TriggerTypeRiskLevel,
		TriggerValue:   model.RiskHigh,
		NotifyChannels: []string{model.NotifyChannelEmail},
		NotifyTargets:  []string{"admin@example.com"},
	}
	rule, _ := s.svc.CreateRule(s.ctx, createReq)

	// 删除规则
	err := s.svc.DeleteRule(s.ctx, rule.ID)

	s.NoError(err)

	// 验证规则已被删除
	rules, _ := s.svc.ListRules(s.ctx)
	s.Equal(0, len(rules))
}

// TestListRules 测试获取所有规则
func (s *AlertServiceTestSuite) TestListRules() {
	// 创建多条规则
	for i := 0; i < 3; i++ {
		req := dto.CreateAlertRuleReq{
			Name:           "规则" + string(rune('A'+i)),
			Enabled:        true,
			TriggerType:    model.TriggerTypeRiskLevel,
			TriggerValue:   model.RiskHigh,
			NotifyChannels: []string{model.NotifyChannelEmail},
			NotifyTargets:  []string{"admin@example.com"},
		}
		s.svc.CreateRule(s.ctx, req)
	}

	rules, err := s.svc.ListRules(s.ctx)

	s.NoError(err)
	s.Equal(3, len(rules))
}

// TestMatchRule 测试规则匹配
func (s *AlertServiceTestSuite) TestMatchRule() {
	// 创建按风险等级触发的规则
	rule := &model.AlertRule{
		ID:           "rule-001",
		Name:         "高风险告警",
		TriggerType:  model.TriggerTypeRiskLevel,
		TriggerValue: model.RiskHigh,
	}

	// 测试匹配高风险日志
	highLog := &model.AuditLog{
		RiskLevel: model.RiskHigh,
		Action:    model.ActionTypeCreate,
	}
	s.True(s.svc.matchRule(rule, highLog))

	// 测试不匹配低风险日志
	lowLog := &model.AuditLog{
		RiskLevel: model.RiskLow,
		Action:    model.ActionTypeCreate,
	}
	s.False(s.svc.matchRule(rule, lowLog))

	// 测试按操作类型触发的规则
	actionRule := &model.AlertRule{
		ID:           "rule-002",
		Name:         "删除操作告警",
		TriggerType:  model.TriggerTypeAction,
		TriggerValue: model.ActionTypeDelete,
	}

	deleteLog := &model.AuditLog{
		Action: model.ActionTypeDelete,
	}
	s.True(s.svc.matchRule(actionRule, deleteLog))

	createLog := &model.AuditLog{
		Action: model.ActionTypeCreate,
	}
	s.False(s.svc.matchRule(actionRule, createLog))
}

// TestRenderTemplate 测试模板渲染
func (s *AlertServiceTestSuite) TestRenderTemplate() {
	log := &model.AuditLog{
		Username:  "admin",
		Action:    model.ActionTypeDelete,
		RiskLevel: model.RiskHigh,
	}

	result, err := s.svc.renderTemplate("用户 {{.Username}} 执行了 {{.Action}} 操作，风险等级：{{.RiskLevel}}", log)

	s.NoError(err)
	s.Equal("用户 admin 执行了 delete 操作，风险等级：high", result)
}

// TestGetRiskLevelLabel 测试风险等级标签
func (s *AlertServiceTestSuite) TestGetRiskLevelLabel() {
	s.Equal("严重", s.svc.getRiskLevelLabel("critical"))
	s.Equal("高", s.svc.getRiskLevelLabel("high"))
	s.Equal("中", s.svc.getRiskLevelLabel("medium"))
	s.Equal("低", s.svc.getRiskLevelLabel("low"))
	s.Equal("unknown", s.svc.getRiskLevelLabel("unknown"))
}

// TestGetRiskLevelColor 测试风险等级颜色
func (s *AlertServiceTestSuite) TestGetRiskLevelColor() {
	s.Equal("#ff4d4f", s.svc.getRiskLevelColor("critical"))
	s.Equal("#f56c6c", s.svc.getRiskLevelColor("high"))
	s.Equal("#faad14", s.svc.getRiskLevelColor("medium"))
	s.Equal("#52c41a", s.svc.getRiskLevelColor("low"))
	s.Equal("#8c8c8c", s.svc.getRiskLevelColor("unknown"))
}
