package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"
	"meteorx/pkg/idgen"
)

type AlertService struct {
	alertRepo repository.AlertRuleRepository
}

func NewAlertService(alertRepo repository.AlertRuleRepository) *AlertService {
	return &AlertService{
		alertRepo: alertRepo,
	}
}

// CreateRule 创建告警规则
func (s *AlertService) CreateRule(ctx context.Context, req dto.CreateAlertRuleReq) (*model.AlertRule, error) {
	channelsJSON, _ := json.Marshal(req.NotifyChannels)
	targetsJSON, _ := json.Marshal(req.NotifyTargets)

	rule := &model.AlertRule{
		ID:              idgen.New(),
		Name:            req.Name,
		Description:     req.Description,
		Enabled:         req.Enabled,
		TriggerType:     req.TriggerType,
		TriggerValue:    req.TriggerValue,
		NotifyChannels:  string(channelsJSON),
		NotifyTargets:   string(targetsJSON),
		NotifyTemplate:  req.NotifyTemplate,
		CooldownMinutes: req.CooldownMinutes,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if rule.CooldownMinutes == 0 {
		rule.CooldownMinutes = 30
	}

	if rule.NotifyTemplate == "" {
		rule.NotifyTemplate = "【告警】用户 {{.Username}} 执行了 {{.Action}} 操作，风险等级：{{.RiskLevel}}"
	}

	if err := s.alertRepo.Create(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

// UpdateRule 更新告警规则
func (s *AlertService) UpdateRule(ctx context.Context, id string, req dto.UpdateAlertRuleReq) (*model.AlertRule, error) {
	rule, err := s.alertRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		rule.Name = req.Name
	}
	if req.Description != "" {
		rule.Description = req.Description
	}
	rule.Enabled = req.Enabled
	if req.TriggerType != "" {
		rule.TriggerType = req.TriggerType
	}
	if req.TriggerValue != "" {
		rule.TriggerValue = req.TriggerValue
	}
	if len(req.NotifyChannels) > 0 {
		channelsJSON, _ := json.Marshal(req.NotifyChannels)
		rule.NotifyChannels = string(channelsJSON)
	}
	if len(req.NotifyTargets) > 0 {
		targetsJSON, _ := json.Marshal(req.NotifyTargets)
		rule.NotifyTargets = string(targetsJSON)
	}
	if req.NotifyTemplate != "" {
		rule.NotifyTemplate = req.NotifyTemplate
	}
	if req.CooldownMinutes > 0 {
		rule.CooldownMinutes = req.CooldownMinutes
	}
	rule.UpdatedAt = time.Now()

	if err := s.alertRepo.Update(ctx, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

// DeleteRule 删除告警规则
func (s *AlertService) DeleteRule(ctx context.Context, id string) error {
	return s.alertRepo.Delete(ctx, id)
}

// ListRules 获取所有告警规则
func (s *AlertService) ListRules(ctx context.Context) ([]*model.AlertRule, error) {
	return s.alertRepo.List(ctx)
}

// ListAlerts 分页查询告警记录
func (s *AlertService) ListAlerts(ctx context.Context, page, pageSize int, ruleID, userID, riskLevel string) ([]*model.AuditAlert, int64, error) {
	return s.alertRepo.ListAlerts(ctx, page, pageSize, ruleID, userID, riskLevel)
}

// EvaluateAndAlert 评估审计日志并触发告警
func (s *AlertService) EvaluateAndAlert(ctx context.Context, log *model.AuditLog) error {
	rules, err := s.alertRepo.GetEnabledRules(ctx)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if !s.matchRule(rule, log) {
			continue
		}

		inCooldown, err := s.alertRepo.IsInCooldown(ctx, rule.ID, rule.CooldownMinutes)
		if err != nil {
			continue
		}
		if inCooldown {
			continue
		}

		message, err := s.renderTemplate(rule.NotifyTemplate, log)
		if err != nil {
			message = fmt.Sprintf("用户 %s 执行了 %s 操作", log.Username, log.Action)
		}

		alert := &model.AuditAlert{
			ID:         idgen.New(),
			RuleID:     rule.ID,
			RuleName:   rule.Name,
			AuditLogID: log.ID,
			UserID:     log.UserID,
			Username:   log.Username,
			RiskLevel:  log.RiskLevel,
			Action:     log.Action,
			Message:    message,
			Notified:   false,
			CreatedAt:  time.Now(),
		}

		if err := s.alertRepo.CreateAlert(ctx, alert); err != nil {
			continue
		}

		s.sendNotifications(ctx, rule, alert)
	}

	return nil
}

func (s *AlertService) matchRule(rule *model.AlertRule, log *model.AuditLog) bool {
	switch rule.TriggerType {
	case model.TriggerTypeRiskLevel:
		return strings.EqualFold(log.RiskLevel, rule.TriggerValue)
	case model.TriggerTypeAction:
		return strings.EqualFold(log.Action, rule.TriggerValue)
	case model.TriggerTypeUser:
		return strings.EqualFold(log.UserID, rule.TriggerValue)
	default:
		return false
	}
}

func (s *AlertService) renderTemplate(tmplStr string, log *model.AuditLog) (string, error) {
	tmpl, err := template.New("alert").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, log); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (s *AlertService) sendNotifications(ctx context.Context, rule *model.AlertRule, alert *model.AuditAlert) {
	var channels []string
	if err := json.Unmarshal([]byte(rule.NotifyChannels), &channels); err != nil {
		return
	}

	var targets []string
	if err := json.Unmarshal([]byte(rule.NotifyTargets), &targets); err != nil {
		return
	}

	for _, channel := range channels {
		switch channel {
		case model.NotifyChannelEmail:
			s.sendEmailNotifications(targets, alert)
		case model.NotifyChannelDingTalk:
			s.sendDingTalkNotification(targets, alert)
		case model.NotifyChannelWeChat:
			s.sendWeChatNotification(targets, alert)
		case model.NotifyChannelWebhook:
			s.sendWebhookNotification(targets, alert)
		}
	}

	alert.Notified = true
	alert.NotifyTime = time.Now()
	_ = s.alertRepo.CreateAlert(ctx, alert)
}

func (s *AlertService) sendEmailNotification(targets []string, alert *model.AuditAlert) {
	for _, target := range targets {
		if !strings.Contains(target, "@") {
			continue
		}
		fmt.Printf("[Email Alert] To: %s, Subject: 审计告警, Body: %s\n", target, alert.Message)
	}
}

func (s *AlertService) sendDingTalkNotification(targets []string, alert *model.AuditAlert) {
	for _, target := range targets {
		if !strings.HasPrefix(target, "http") {
			continue
		}
		fmt.Printf("[DingTalk Alert] Webhook: %s, Message: %s\n", target, alert.Message)
	}
}

func (s *AlertService) sendWeChatNotification(targets []string, alert *model.AuditAlert) {
	for _, target := range targets {
		if !strings.HasPrefix(target, "http") {
			continue
		}
		fmt.Printf("[WeChat Alert] Webhook: %s, Message: %s\n", target, alert.Message)
	}
}

func (s *AlertService) sendWebhookNotification(targets []string, alert *model.AuditAlert) {
	for _, target := range targets {
		if !strings.HasPrefix(target, "http") {
			continue
		}
		fmt.Printf("[Webhook Alert] URL: %s, Payload: %+v\n", target, alert)
	}
}