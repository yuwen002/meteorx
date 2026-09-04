package repository

import (
	"context"
	"encoding/json"
	"time"

	"meteorx/internal/modules/audit/model"
	"meteorx/pkg/ulid"

	"gorm.io/gorm"
)

// AlertRulePO 告警规则数据库模型
type AlertRulePO struct {
	ID              string    `gorm:"primaryKey;size:26;comment:规则ID"`
	Name            string    `gorm:"uniqueIndex;size:100;comment:规则名称"`
	Description     string    `gorm:"size:500;comment:规则描述"`
	Enabled         bool      `gorm:"default:true;comment:是否启用"`
	TriggerType     string    `gorm:"index;size:20;comment:触发类型"`
	TriggerValue    string    `gorm:"size:100;comment:触发值"`
	NotifyChannels  string    `gorm:"size:500;comment:通知渠道(JSON)"`
	NotifyTargets   string    `gorm:"type:text;comment:通知目标(JSON)"`
	NotifyTemplate  string    `gorm:"type:text;comment:通知模板"`
	CooldownMinutes int       `gorm:"default:30;comment:冷却时间(分钟)"`
	CreatedAt       time.Time `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime;comment:更新时间"`
}

func (AlertRulePO) TableName() string {
	return "audit_alert_rules"
}

// AuditAlertPO 审计告警记录数据库模型
type AuditAlertPO struct {
	ID         string    `gorm:"primaryKey;size:26;comment:告警ID"`
	RuleID     string    `gorm:"index;size:26;comment:规则ID"`
	RuleName   string    `gorm:"size:100;comment:规则名称"`
	AuditLogID string    `gorm:"index;size:26;comment:审计日志ID"`
	UserID     string    `gorm:"index;size:26;comment:触发用户ID"`
	Username   string    `gorm:"index;size:50;comment:触发用户名"`
	RiskLevel  string    `gorm:"index;size:20;comment:风险等级"`
	Action     string    `gorm:"index;size:20;comment:操作类型"`
	Message    string    `gorm:"type:text;comment:告警消息"`
	Notified   bool      `gorm:"default:false;comment:是否已通知"`
	NotifyTime time.Time `gorm:"comment:通知时间"`
	CreatedAt  time.Time `gorm:"autoCreateTime;comment:创建时间"`
}

func (AuditAlertPO) TableName() string {
	return "audit_alerts"
}

type alertRuleRepository struct {
	db *gorm.DB
}

func NewAlertRuleRepository(db *gorm.DB) AlertRuleRepository {
	return &alertRuleRepository{db: db}
}

func (r *alertRuleRepository) Create(ctx context.Context, rule *model.AlertRule) error {
	po := alertRuleFromDomain(rule)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *alertRuleRepository) GetByID(ctx context.Context, id string) (*model.AlertRule, error) {
	var po AlertRulePO
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *alertRuleRepository) Update(ctx context.Context, rule *model.AlertRule) error {
	po := alertRuleFromDomain(rule)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *alertRuleRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&AlertRulePO{}, "id = ?", id).Error
}

func (r *alertRuleRepository) List(ctx context.Context) ([]*model.AlertRule, error) {
	var pos []AlertRulePO
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&pos).Error; err != nil {
		return nil, err
	}

	rules := make([]*model.AlertRule, len(pos))
	for i, po := range pos {
		rules[i] = po.toDomain()
	}
	return rules, nil
}

func (r *alertRuleRepository) GetEnabledRules(ctx context.Context) ([]*model.AlertRule, error) {
	var pos []AlertRulePO
	if err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&pos).Error; err != nil {
		return nil, err
	}

	rules := make([]*model.AlertRule, len(pos))
	for i, po := range pos {
		rules[i] = po.toDomain()
	}
	return rules, nil
}

func (r *alertRuleRepository) CreateAlert(ctx context.Context, alert *model.AuditAlert) error {
	po := auditAlertFromDomain(alert)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *alertRuleRepository) ListAlerts(ctx context.Context, page, pageSize int, ruleID, userID, riskLevel string) ([]*model.AuditAlert, int64, error) {
	db := r.db.WithContext(ctx).Model(&AuditAlertPO{})

	if ruleID != "" {
		db = db.Where("rule_id = ?", ruleID)
	}
	if userID != "" {
		db = db.Where("user_id = ?", userID)
	}
	if riskLevel != "" {
		db = db.Where("risk_level = ?", riskLevel)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var pos []AuditAlertPO
	if err := db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	alerts := make([]*model.AuditAlert, len(pos))
	for i, po := range pos {
		alerts[i] = po.toDomain()
	}
	return alerts, total, nil
}

func (r *alertRuleRepository) IsInCooldown(ctx context.Context, ruleID string, cooldownMinutes int) (bool, error) {
	var count int64
	cooldownTime := time.Now().Add(-time.Duration(cooldownMinutes) * time.Minute)
	err := r.db.WithContext(ctx).Model(&AuditAlertPO{}).
		Where("rule_id = ? AND created_at > ?", ruleID, cooldownTime).
		Count(&count).Error
	return count > 0, err
}

func (po AlertRulePO) toDomain() *model.AlertRule {
	return &model.AlertRule{
		ID:              po.ID,
		Name:            po.Name,
		Description:     po.Description,
		Enabled:         po.Enabled,
		TriggerType:     po.TriggerType,
		TriggerValue:    po.TriggerValue,
		NotifyChannels:  po.NotifyChannels,
		NotifyTargets:   po.NotifyTargets,
		NotifyTemplate:  po.NotifyTemplate,
		CooldownMinutes: po.CooldownMinutes,
		CreatedAt:       po.CreatedAt,
		UpdatedAt:       po.UpdatedAt,
	}
}

func alertRuleFromDomain(rule *model.AlertRule) *AlertRulePO {
	return &AlertRulePO{
		ID:              rule.ID,
		Name:            rule.Name,
		Description:     rule.Description,
		Enabled:         rule.Enabled,
		TriggerType:     rule.TriggerType,
		TriggerValue:    rule.TriggerValue,
		NotifyChannels:  rule.NotifyChannels,
		NotifyTargets:   rule.NotifyTargets,
		NotifyTemplate:  rule.NotifyTemplate,
		CooldownMinutes: rule.CooldownMinutes,
		CreatedAt:       rule.CreatedAt,
		UpdatedAt:       rule.UpdatedAt,
	}
}

func (po AuditAlertPO) toDomain() *model.AuditAlert {
	return &model.AuditAlert{
		ID:         po.ID,
		RuleID:     po.RuleID,
		RuleName:   po.RuleName,
		AuditLogID: po.AuditLogID,
		UserID:     po.UserID,
		Username:   po.Username,
		RiskLevel:  po.RiskLevel,
		Action:     po.Action,
		Message:    po.Message,
		Notified:   po.Notified,
		NotifyTime: po.NotifyTime,
		CreatedAt:  po.CreatedAt,
	}
}

func auditAlertFromDomain(alert *model.AuditAlert) *AuditAlertPO {
	return &AuditAlertPO{
		ID:         alert.ID,
		RuleID:     alert.RuleID,
		RuleName:   alert.RuleName,
		AuditLogID: alert.AuditLogID,
		UserID:     alert.UserID,
		Username:   alert.Username,
		RiskLevel:  alert.RiskLevel,
		Action:     alert.Action,
		Message:    alert.Message,
		Notified:   alert.Notified,
		NotifyTime: alert.NotifyTime,
		CreatedAt:  alert.CreatedAt,
	}
}

// 辅助函数：序列化/反序列化 JSON
func serializeJSON(v interface{}) string {
	if v == nil {
		return "[]"
	}
	data, _ := json.Marshal(v)
	return string(data)
}

func deserializeJSON(data string, v interface{}) error {
	if data == "" {
		return nil
	}
	return json.Unmarshal([]byte(data), v)
}

// GetAlertStats 获取告警统计数据
func (r *alertRuleRepository) GetAlertStats(ctx context.Context, days int) (*model.AlertStats, error) {
	stats := &model.AlertStats{
		RiskLevelStats: make(map[string]int64),
		RuleStats:      make(map[string]int64),
	}

	// 总告警数
	r.db.WithContext(ctx).Model(&AuditAlertPO{}).Count(&stats.TotalAlerts)

	// 今日告警数
	today := time.Now().Truncate(24 * time.Hour)
	r.db.WithContext(ctx).Model(&AuditAlertPO{}).Where("created_at >= ?", today).Count(&stats.TodayAlerts)

	// 已通知数
	r.db.WithContext(ctx).Model(&AuditAlertPO{}).Where("notified = ?", true).Count(&stats.NotifiedCount)

	// 未通知数
	r.db.WithContext(ctx).Model(&AuditAlertPO{}).Where("notified = ?", false).Count(&stats.PendingCount)

	// 按风险等级统计
	var riskLevelStats []struct {
		RiskLevel string
		Count     int64
	}
	r.db.WithContext(ctx).Model(&AuditAlertPO{}).
		Select("risk_level, COUNT(*) as count").
		Group("risk_level").
		Scan(&riskLevelStats)

	for _, item := range riskLevelStats {
		stats.RiskLevelStats[item.RiskLevel] = item.Count
	}

	// 按规则统计
	var ruleStats []struct {
		RuleID   string
		RuleName string
		Count    int64
	}
	r.db.WithContext(ctx).Model(&AuditAlertPO{}).
		Select("rule_id, rule_name, COUNT(*) as count").
		Group("rule_id, rule_name").
		Order("count DESC").
		Limit(10).
		Scan(&ruleStats)

	for _, item := range ruleStats {
		stats.RuleStats[item.RuleID] = item.Count
		stats.TopRules = append(stats.TopRules, model.RuleCount{
			RuleID:   item.RuleID,
			RuleName: item.RuleName,
			Count:    item.Count,
		})
	}

	// 告警趋势（最近 N 天）
	if days <= 0 {
		days = 7
	}

	startDate := time.Now().AddDate(0, 0, -days+1).Truncate(24 * time.Hour)

	var trendStats []struct {
		Date    string
		Count   int64
		Success int64
		Failure int64
	}

	r.db.WithContext(ctx).Model(&AuditAlertPO{}).
		Select(`DATE(created_at) as date, 
				COUNT(*) as count,
				SUM(CASE WHEN risk_level IN ('low', 'medium') THEN 1 ELSE 0 END) as success,
				SUM(CASE WHEN risk_level IN ('high', 'critical') THEN 1 ELSE 0 END) as failure`).
		Where("created_at >= ?", startDate).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&trendStats)

	for _, item := range trendStats {
		stats.Trend = append(stats.Trend, model.TrendPoint{
			Date:    item.Date,
			Count:   item.Count,
			Success: item.Success,
			Failure: item.Failure,
		})
	}

	return stats, nil
}

// 初始化默认告警规则
func InitDefaultAlertRules(db *gorm.DB) error {
	var count int64
	db.Model(&AlertRulePO{}).Count(&count)
	if count > 0 {
		return nil // 已有规则，不重复初始化
	}

	defaultRules := []AlertRulePO{
		{
			ID:              ulid.Generate(),
			Name:            "严重风险操作告警",
			Description:     "当发生严重风险操作时立即告警",
			Enabled:         true,
			TriggerType:     model.TriggerTypeRiskLevel,
			TriggerValue:    model.RiskCritical,
			NotifyChannels:  serializeJSON([]string{"email", "dingtalk"}),
			NotifyTargets:   serializeJSON([]string{"admin@example.com"}),
			NotifyTemplate:  "【严重告警】用户 {{.Username}} 执行了 {{.Action}} 操作，风险等级：{{.RiskLevel}}",
			CooldownMinutes: 15,
		},
		{
			ID:              ulid.Generate(),
			Name:            "高风险操作告警",
			Description:     "当发生高风险操作时告警",
			Enabled:         true,
			TriggerType:     model.TriggerTypeRiskLevel,
			TriggerValue:    model.RiskHigh,
			NotifyChannels:  serializeJSON([]string{"email"}),
			NotifyTargets:   serializeJSON([]string{"admin@example.com"}),
			NotifyTemplate:  "【高风险告警】用户 {{.Username}} 执行了 {{.Action}} 操作，风险等级：{{.RiskLevel}}",
			CooldownMinutes: 30,
		},
		{
			ID:              ulid.Generate(),
			Name:            "删除操作告警",
			Description:     "当发生删除操作时告警",
			Enabled:         true,
			TriggerType:     model.TriggerTypeAction,
			TriggerValue:    model.ActionTypeDelete,
			NotifyChannels:  serializeJSON([]string{"email", "dingtalk"}),
			NotifyTargets:   serializeJSON([]string{"admin@example.com"}),
			NotifyTemplate:  "【删除告警】用户 {{.Username}} 执行了删除操作",
			CooldownMinutes: 30,
		},
	}

	for _, rule := range defaultRules {
		if err := db.Create(&rule).Error; err != nil {
			return err
		}
	}

	return nil
}
