-- ========================================
-- 审计日志增强功能数据库迁移脚本
-- 创建时间：2026-09-03
-- 说明：添加 IP 地理位置、会话追踪、告警管理相关表和字段
-- ========================================

-- 1. 为 audit_logs 表添加新字段
ALTER TABLE `audit_logs` 
ADD COLUMN IF NOT EXISTS `session_id` VARCHAR(255) DEFAULT NULL COMMENT '会话ID' AFTER `request_id`,
ADD COLUMN IF NOT EXISTS `ip_location` VARCHAR(255) DEFAULT NULL COMMENT 'IP地理位置' AFTER `client_ip`,
ADD COLUMN IF NOT EXISTS `user_agent` VARCHAR(500) DEFAULT NULL COMMENT '用户代理' AFTER `ip_location`,
ADD COLUMN IF NOT EXISTS `device_info` VARCHAR(500) DEFAULT NULL COMMENT '设备信息' AFTER `user_agent`,
ADD COLUMN IF NOT EXISTS `trace_id` VARCHAR(255) DEFAULT NULL COMMENT '链路追踪ID' AFTER `session_id`,
ADD COLUMN IF NOT EXISTS `referer` VARCHAR(500) DEFAULT NULL COMMENT '来源页面' AFTER `trace_id`,
ADD COLUMN IF NOT EXISTS `risk_level` VARCHAR(20) DEFAULT 'low' COMMENT '风险等级（low/medium/high/critical）' AFTER `referer`,
ADD COLUMN IF NOT EXISTS `tags` JSON DEFAULT NULL COMMENT '标签（JSON数组）' AFTER `risk_level`;

-- 2. 为 audit_logs 表添加索引
CREATE INDEX IF NOT EXISTS `idx_audit_logs_session_id` ON `audit_logs` (`session_id`);
CREATE INDEX IF NOT EXISTS `idx_audit_logs_risk_level` ON `audit_logs` (`risk_level`);
CREATE INDEX IF NOT EXISTS `idx_audit_logs_ip_location` ON `audit_logs` (`ip_location`(100));
CREATE INDEX IF NOT EXISTS `idx_audit_logs_created_at` ON `audit_logs` (`created_at`);
CREATE INDEX IF NOT EXISTS `idx_audit_logs_user_id` ON `audit_logs` (`user_id`);
CREATE INDEX IF NOT EXISTS `idx_audit_logs_tenant_id` ON `audit_logs` (`tenant_id`);

-- 3. 创建审计告警规则表
CREATE TABLE IF NOT EXISTS `audit_alert_rules` (
  `id` VARCHAR(255) NOT NULL COMMENT '规则ID',
  `name` VARCHAR(255) NOT NULL COMMENT '规则名称',
  `description` TEXT DEFAULT NULL COMMENT '规则描述',
  `enabled` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用（1=启用，0=禁用）',
  `trigger_type` VARCHAR(50) NOT NULL COMMENT '触发类型（risk_level/action/user）',
  `trigger_value` VARCHAR(255) NOT NULL COMMENT '触发值（如 high/critical/delete）',
  `notify_channels` JSON NOT NULL COMMENT '通知渠道（JSON数组：["email","dingtalk","wechat"]）',
  `notify_targets` JSON NOT NULL COMMENT '通知目标（JSON数组：邮箱/手机号/webhook）',
  `notify_template` TEXT DEFAULT NULL COMMENT '通知模板',
  `cooldown_minutes` INT NOT NULL DEFAULT 30 COMMENT '冷却时间（分钟），避免重复告警',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `idx_alert_rules_enabled` (`enabled`),
  INDEX `idx_alert_rules_trigger_type` (`trigger_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='审计告警规则表';

-- 4. 创建审计告警记录表
CREATE TABLE IF NOT EXISTS `audit_alerts` (
  `id` VARCHAR(255) NOT NULL COMMENT '告警ID',
  `rule_id` VARCHAR(255) NOT NULL COMMENT '触发规则ID',
  `rule_name` VARCHAR(255) NOT NULL COMMENT '规则名称',
  `audit_log_id` VARCHAR(255) NOT NULL COMMENT '关联审计日志ID',
  `user_id` VARCHAR(255) NOT NULL COMMENT '触发用户ID',
  `username` VARCHAR(255) NOT NULL COMMENT '触发用户名',
  `risk_level` VARCHAR(20) NOT NULL COMMENT '风险等级（low/medium/high/critical）',
  `action` VARCHAR(50) NOT NULL COMMENT '操作类型',
  `message` TEXT NOT NULL COMMENT '告警消息',
  `notified` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否已通知（1=已通知，0=未通知）',
  `notify_time` DATETIME DEFAULT NULL COMMENT '通知时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  INDEX `idx_alerts_rule_id` (`rule_id`),
  INDEX `idx_alerts_user_id` (`user_id`),
  INDEX `idx_alerts_risk_level` (`risk_level`),
  INDEX `idx_alerts_audit_log_id` (`audit_log_id`),
  INDEX `idx_alerts_created_at` (`created_at`),
  INDEX `idx_alerts_notified` (`notified`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='审计告警记录表';

-- 5. 插入默认告警规则（可选）
INSERT INTO `audit_alert_rules` (`id`, `name`, `description`, `enabled`, `trigger_type`, `trigger_value`, `notify_channels`, `notify_targets`, `notify_template`, `cooldown_minutes`)
VALUES 
  ('rule-001', '高风险操作告警', '当检测到高风险操作时触发告警', 1, 'risk_level', 'high', '["email"]', '["admin@example.com"]', '【告警】用户 {{.Username}} 执行了 {{.Action}} 操作，风险等级：{{.RiskLevel}}', 30),
  ('rule-002', '严重操作告警', '当检测到严重操作时立即告警', 1, 'risk_level', 'critical', '["email", "dingtalk"]', '["admin@example.com"]', '【严重告警】用户 {{.Username}} 执行了 {{.Action}} 操作，风险等级：{{.RiskLevel}}', 15),
  ('rule-003', '删除操作告警', '当执行删除操作时触发告警', 1, 'action', 'delete', '["email"]', '["admin@example.com"]', '【告警】用户 {{.Username}} 执行了删除操作', 60)
ON DUPLICATE KEY UPDATE `name` = VALUES(`name`);

-- ========================================
-- 迁移完成
-- ========================================