package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"meteorx/internal/modules/audit/dto"
	"meteorx/internal/modules/audit/model"
	"meteorx/internal/modules/audit/repository"
	"meteorx/internal/pkg/emailer"
	"meteorx/pkg/idgen"
)

type AlertService struct {
	alertRepo repository.AlertRuleRepository
	emailer   *emailer.Emailer
}

func NewAlertService(alertRepo repository.AlertRuleRepository, emailer *emailer.Emailer) *AlertService {
	return &AlertService{
		alertRepo: alertRepo,
		emailer:   emailer,
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
			s.sendEmailNotification(targets, alert)
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
	if s.emailer == nil {
		return
	}
	
	for _, target := range targets {
		if !strings.Contains(target, "@") {
			continue
		}
		
		subject := "【审计告警】" + alert.RuleName
		body := fmt.Sprintf(`
			<div style="max-width: 600px; margin: 0 auto; padding: 20px; font-family: Arial, sans-serif;">
				<div style="background: linear-gradient(135deg, #f56c6c 0%%, #e6a23c 100%%); padding: 30px; border-radius: 10px 10px 0 0;">
					<h1 style="color: white; margin: 0; text-align: center;">审计告警通知</h1>
				</div>
				<div style="background: #f8f9fa; padding: 30px; border-radius: 0 0 10px 10px; border: 1px solid #e9ecef; border-top: none;">
					<h2 style="color: #333; margin-top: 0;">告警详情</h2>
					<table style="width: 100%%; border-collapse: collapse;">
						<tr style="background: #fff;">
							<td style="padding: 12px; border: 1px solid #e9ecef; font-weight: bold; width: 120px;">规则名称</td>
							<td style="padding: 12px; border: 1px solid #e9ecef;">%s</td>
						</tr>
						<tr style="background: #f8f9fa;">
							<td style="padding: 12px; border: 1px solid #e9ecef; font-weight: bold;">触发用户</td>
							<td style="padding: 12px; border: 1px solid #e9ecef;">%s</td>
						</tr>
						<tr style="background: #fff;">
							<td style="padding: 12px; border: 1px solid #e9ecef; font-weight: bold;">操作类型</td>
							<td style="padding: 12px; border: 1px solid #e9ecef;">%s</td>
						</tr>
						<tr style="background: #f8f9fa;">
							<td style="padding: 12px; border: 1px solid #e9ecef; font-weight: bold;">风险等级</td>
							<td style="padding: 12px; border: 1px solid #e9ecef;"><span style="color: %s; font-weight: bold;">%s</span></td>
						</tr>
						<tr style="background: #fff;">
							<td style="padding: 12px; border: 1px solid #e9ecef; font-weight: bold;">告警消息</td>
							<td style="padding: 12px; border: 1px solid #e9ecef;">%s</td>
						</tr>
						<tr style="background: #f8f9fa;">
							<td style="padding: 12px; border: 1px solid #e9ecef; font-weight: bold;">触发时间</td>
							<td style="padding: 12px; border: 1px solid #e9ecef;">%s</td>
						</tr>
					</table>
					<hr style="border: none; border-top: 1px solid #e9ecef; margin: 30px 0;">
					<p style="color: #999; font-size: 12px; text-align: center;">
						此邮件由 MeteorX 审计系统自动发送，请勿直接回复。
					</p>
				</div>
			</div>
		`,
			alert.RuleName,
			alert.Username,
			alert.Action,
			s.getRiskLevelColor(alert.RiskLevel),
			s.getRiskLevelLabel(alert.RiskLevel),
			alert.Message,
			alert.CreatedAt.Format("2006-01-02 15:04:05"),
		)

		if err := s.emailer.Send(target, subject, body); err != nil {
			fmt.Printf("[Email Alert Error] Failed to send to %s: %v\n", target, err)
		} else {
			fmt.Printf("[Email Alert] Sent to %s successfully\n", target)
		}
	}
}

func (s *AlertService) getRiskLevelColor(level string) string {
	switch strings.ToLower(level) {
	case "critical":
		return "#ff4d4f"
	case "high":
		return "#f56c6c"
	case "medium":
		return "#faad14"
	case "low":
		return "#52c41a"
	default:
		return "#8c8c8c"
	}
}

func (s *AlertService) getRiskLevelLabel(level string) string {
	switch strings.ToLower(level) {
	case "critical":
		return "严重"
	case "high":
		return "高"
	case "medium":
		return "中"
	case "low":
		return "低"
	default:
		return level
	}
}

func (s *AlertService) sendDingTalkNotification(targets []string, alert *model.AuditAlert) {
	for _, target := range targets {
		if !strings.HasPrefix(target, "http") {
			continue
		}
		
		payload := map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"title": "审计告警通知",
				"text": fmt.Sprintf(`# 审计告警通知

**规则名称**: %s
**触发用户**: %s
**操作类型**: %s
**风险等级**: <font color=%s>%s</font>
**告警消息**: %s
**触发时间**: %s

> 此消息由 MeteorX 审计系统自动发送`,
					alert.RuleName,
					alert.Username,
					alert.Action,
					s.getRiskLevelColor(alert.RiskLevel),
					s.getRiskLevelLabel(alert.RiskLevel),
					alert.Message,
					alert.CreatedAt.Format("2006-01-02 15:04:05"),
				),
			},
		}
		
		if err := s.sendWebhookRequest(target, payload); err != nil {
			fmt.Printf("[DingTalk Alert Error] Failed to send to %s: %v\n", target, err)
		} else {
			fmt.Printf("[DingTalk Alert] Sent to %s successfully\n", target)
		}
	}
}

func (s *AlertService) sendWeChatNotification(targets []string, alert *model.AuditAlert) {
	for _, target := range targets {
		if !strings.HasPrefix(target, "http") {
			continue
		}
		
		payload := map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]string{
				"content": fmt.Sprintf(`# 审计告警通知
> **规则名称**: %s
> **触发用户**: %s
> **操作类型**: %s
> **风险等级**: <font color="warning">%s</font>
> **告警消息**: %s
> **触发时间**: %s

> 此消息由 MeteorX 审计系统自动发送`,
					alert.RuleName,
					alert.Username,
					alert.Action,
					s.getRiskLevelLabel(alert.RiskLevel),
					alert.Message,
					alert.CreatedAt.Format("2006-01-02 15:04:05"),
				),
			},
		}
		
		if err := s.sendWebhookRequest(target, payload); err != nil {
			fmt.Printf("[WeChat Alert Error] Failed to send to %s: %v\n", target, err)
		} else {
			fmt.Printf("[WeChat Alert] Sent to %s successfully\n", target)
		}
	}
}

func (s *AlertService) sendWebhookNotification(targets []string, alert *model.AuditAlert) {
	for _, target := range targets {
		if !strings.HasPrefix(target, "http") {
			continue
		}
		
		payload := map[string]interface{}{
			"rule_name":   alert.RuleName,
			"username":    alert.Username,
			"action":      alert.Action,
			"risk_level":  alert.RiskLevel,
			"message":     alert.Message,
			"created_at":  alert.CreatedAt.Format("2006-01-02 15:04:05"),
			"alert_id":    alert.ID,
			"rule_id":     alert.RuleID,
			"audit_log_id": alert.AuditLogID,
		}
		
		if err := s.sendWebhookRequest(target, payload); err != nil {
			fmt.Printf("[Webhook Alert Error] Failed to send to %s: %v\n", target, err)
		} else {
			fmt.Printf("[Webhook Alert] Sent to %s successfully\n", target)
		}
	}
}

func (s *AlertService) sendWebhookRequest(url string, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	
	return nil
}

// GetAlertStats 获取告警统计数据
func (s *AlertService) GetAlertStats(ctx context.Context, days int) (*model.AlertStats, error) {
	if days <= 0 {
		days = 7 // 默认查看最近7天
	}
	
	return s.alertRepo.GetAlertStats(ctx, days)
}