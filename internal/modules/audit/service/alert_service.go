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
	"meteorx/internal/notify"
	"meteorx/internal/pkg/emailer"
	"meteorx/internal/ws"
	"meteorx/pkg/idgen"
	"meteorx/pkg/logger"
)

// AlertService 告警规则服务，负责规则 CRUD、审计日志匹配评估与多渠道告警通知。
type AlertService struct {
	alertRepo repository.AlertRuleRepository
	emailer   *emailer.Emailer
	// recipients 把规则配置的邮箱解析为用户 ID，使邮件收件人同样能在收件箱
	// 离线看到告警；未装配时不落收件箱，其余渠道不受影响。
	recipients AlertRecipientDirectory
}

// AlertRecipientDirectory 邮箱 → 用户 ID 的窄依赖（与 TenantDisabler /
// MemberDirectory 同一惯例）：告警只需这一个能力，测试可注入桩，
// 也不必向 alert service 暴露 user 仓储的胖接口。
type AlertRecipientDirectory interface {
	UserIDByEmail(ctx context.Context, email string) (string, error)
}

// NewAlertService 创建告警规则服务实例。
func NewAlertService(alertRepo repository.AlertRuleRepository, emailer *emailer.Emailer) *AlertService {
	return &AlertService{
		alertRepo: alertRepo,
		emailer:   emailer,
	}
}

// SetRecipientDirectory 装配「邮箱 → 用户 ID」解析能力。
// 装配后，规则 targets 里的邮箱若属于某个账号，该账号也会收到一条 audit 分类的站内信。
func (s *AlertService) SetRecipientDirectory(d AlertRecipientDirectory) {
	s.recipients = d
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

// GetRule 获取告警规则详情
func (s *AlertService) GetRule(ctx context.Context, id string) (*model.AlertRule, error) {
	return s.alertRepo.GetByID(ctx, id)
}

// GetAlert 获取告警记录详情
func (s *AlertService) GetAlert(ctx context.Context, id string) (*model.AuditAlert, error) {
	return s.alertRepo.GetAlertByID(ctx, id)
}

// ListRules 获取所有告警规则
func (s *AlertService) ListRules(ctx context.Context) ([]*model.AlertRule, error) {
	return s.alertRepo.List(ctx)
}

// ListAlerts 分页查询告警记录
func (s *AlertService) ListAlerts(ctx context.Context, page, pageSize int, ruleID, userID, riskLevel string) ([]*model.AuditAlert, int64, error) {
	return s.alertRepo.ListAlerts(ctx, page, pageSize, ruleID, userID, riskLevel)
}

// EvaluateAndAlert 评估单条审计日志并触发告警（实现 AuditEvaluator 的写入路径钩子）。
func (s *AlertService) EvaluateAndAlert(ctx context.Context, log *model.AuditLog) error {
	rules, err := s.alertRepo.GetEnabledRules(ctx)
	if err != nil {
		return err
	}
	s.evaluateWithRules(ctx, rules, log)
	return nil
}

// EvaluateAuditLogs 批量评估：整批只查一次启用规则，逐条匹配与冷却判定。
// 审计中间件每 5s/每 100 条 flush 一次，逐条查规则会使每轮 flush 多出上百次
// SELECT；匹配不命中是纯内存比较，冷却仍按每条产出时刻逐条生效。
func (s *AlertService) EvaluateAuditLogs(ctx context.Context, logs []*model.AuditLog) {
	if len(logs) == 0 {
		return
	}
	rules, err := s.alertRepo.GetEnabledRules(ctx)
	if err != nil {
		logger.Errorf("获取启用告警规则失败（本轮 %d 条日志未评估）: %v", len(logs), err)
		return
	}
	if len(rules) == 0 {
		return
	}
	for _, log := range logs {
		s.evaluateWithRules(ctx, rules, log)
	}
}

// evaluateWithRules 在给定规则集下对单条日志做匹配、冷却判定与告警产出/发送。
func (s *AlertService) evaluateWithRules(ctx context.Context, rules []*model.AlertRule, log *model.AuditLog) {
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
			// 首插失败不能裸 continue 静默吞掉：告警已不会产生，后续送达/回写都无依托，
			// 至少留一行日志供事后查证（冷却仍会拦住本轮重复）
			logger.Errorf("写入告警记录失败 rule=%s log=%s: %v", rule.ID, log.ID, err)
			continue
		}

		s.sendNotifications(ctx, rule, alert)
	}
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

	// 按收件人类型拆分：邮箱只进邮件渠道、URL 只进 webhook 类渠道。规则 targets 常混装
	// 两者，旧写法把整份 targets 同时喂给直发渠道与全局管理器，导致邮件对 URL 发信失败、
	// WS 拿邮箱静默 miss（与到期提醒/注销通知同一病灶）。
	emails := filterEmails(targets)

	// delivered 只要任一直发渠道真正发成即为 true
	delivered := false
	for _, channel := range channels {
		switch channel {
		case model.NotifyChannelEmail:
			delivered = s.sendEmailNotification(emails, alert) || delivered
		case model.NotifyChannelDingTalk:
			delivered = s.sendDingTalkNotification(targets, alert) || delivered
		case model.NotifyChannelWeChat:
			delivered = s.sendWeChatNotification(targets, alert) || delivered
		case model.NotifyChannelWebhook:
			delivered = s.sendWebhookNotification(targets, alert) || delivered
		}
	}

	// WebSocket 实时推送告警通知（alert.UserID 本就是用户 ID，口径正确；
	// 无人在线时静默 miss 不计入送达，与上面邮箱/URL 混用是两个问题）
	ws.SendAlert(alert.UserID, map[string]interface{}{
		"id":         alert.ID,
		"rule_name":  alert.RuleName,
		"risk_level": alert.RiskLevel,
		"action":     alert.Action,
		"message":    alert.Message,
		"username":   alert.Username,
		"time":       alert.CreatedAt.UnixMilli(),
	})

	// 通过多渠道通知管理器发送告警通知（补充渠道）：只喂邮箱，结果参与送达判定。
	// 收件箱收件人由规则邮箱反查用户 ID 得到——邮箱只在能收信时有意义，
	// 收件箱让同一批人重新登录后仍能看到同一条告警（不参与送达销账）。
	mgr := notify.GetGlobalManager()
	if mgr != nil {
		for _, r := range mgr.NotifyAlert(ctx, notify.AlertNotice{
			ID:       alert.ID,
			RuleName: rule.Name,
			Content:  alert.Message,
			Priority: notify.PriorityUrgent,
			Emails:   emails,
			UserIDs:  s.resolveInboxUserIDs(ctx, emails),
		}) {
			if r.Success {
				delivered = true
				break
			}
		}
	}

	// 送达才回写：此处无重驱机制（冷却按 created_at 判定、不重发），标记了也不重试，
	// 因此「未送达不标记」的意义在于不谎报——否则看板 notified_count 恒等于告警总数、
	// 故障被登记成「已处理」。回写本身失败也要留痕（旧写法 `_ =` 连错误都丢）。
	if !delivered {
		logger.Warnf("告警 %s 未送达任何渠道 rule=%s（告警已落库，需人工确认通知渠道）", alert.ID, rule.ID)
		return
	}
	now := time.Now()
	alert.Notified = true
	alert.NotifyTime = now
	if err := s.alertRepo.MarkAlertNotified(ctx, alert.ID, now); err != nil {
		logger.Errorf("告警 %s 送达回写 notified 失败: %v", alert.ID, err)
	}
}

// filterEmails 从混合收件人中只保留邮箱（与直发邮件渠道的 "@" 判定同口径）。
func filterEmails(targets []string) []string {
	emails := make([]string, 0, len(targets))
	for _, t := range targets {
		if strings.Contains(t, "@") {
			emails = append(emails, t)
		}
	}
	return emails
}

// resolveInboxUserIDs 把规则配置的邮箱解析为收件箱收件人（用户 ID）。
// 邮箱未必属于任何账号（外部值班邮箱、已删除用户），解析失败的直接跳过：
// 告警已落库、邮件渠道不受影响，不为「收件箱少一条」阻断主流程或重试。
func (s *AlertService) resolveInboxUserIDs(ctx context.Context, emails []string) []string {
	if s.recipients == nil || len(emails) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(emails))
	ids := make([]string, 0, len(emails))
	for _, e := range emails {
		uid, err := s.recipients.UserIDByEmail(ctx, e)
		if err != nil || uid == "" {
			continue
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		ids = append(ids, uid)
	}
	return ids
}

func (s *AlertService) sendEmailNotification(targets []string, alert *model.AuditAlert) bool {
	if s.emailer == nil {
		return false
	}

	sent := false
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
			logger.Errorf("[Email Alert] Failed to send to %s: %v", target, err)
		} else {
			logger.Infof("[Email Alert] Sent to %s successfully", target)
			sent = true
		}
	}
	return sent
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

func (s *AlertService) sendDingTalkNotification(targets []string, alert *model.AuditAlert) bool {
	sent := false
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
			logger.Errorf("[DingTalk Alert] Failed to send to %s: %v", target, err)
		} else {
			logger.Infof("[DingTalk Alert] Sent to %s successfully", target)
			sent = true
		}
	}
	return sent
}

func (s *AlertService) sendWeChatNotification(targets []string, alert *model.AuditAlert) bool {
	sent := false
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
			logger.Errorf("[WeChat Alert] Failed to send to %s: %v", target, err)
		} else {
			logger.Infof("[WeChat Alert] Sent to %s successfully", target)
			sent = true
		}
	}
	return sent
}

func (s *AlertService) sendWebhookNotification(targets []string, alert *model.AuditAlert) bool {
	sent := false
	for _, target := range targets {
		if !strings.HasPrefix(target, "http") {
			continue
		}

		payload := map[string]interface{}{
			"rule_name":    alert.RuleName,
			"username":     alert.Username,
			"action":       alert.Action,
			"risk_level":   alert.RiskLevel,
			"message":      alert.Message,
			"created_at":   alert.CreatedAt.Format("2006-01-02 15:04:05"),
			"alert_id":     alert.ID,
			"rule_id":      alert.RuleID,
			"audit_log_id": alert.AuditLogID,
		}

		if err := s.sendWebhookRequest(target, payload); err != nil {
			logger.Errorf("[Webhook Alert] Failed to send to %s: %v", target, err)
		} else {
			logger.Infof("[Webhook Alert] Sent to %s successfully", target)
			sent = true
		}
	}
	return sent
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
	defer func() { _ = resp.Body.Close() }()

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
