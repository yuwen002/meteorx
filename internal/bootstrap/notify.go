package bootstrap

import (
	"meteorx/internal/config"
	"meteorx/internal/notify"
	"meteorx/internal/pkg/emailer"
	"meteorx/internal/ws"
	"meteorx/pkg/logger"
)

// initNotifyManager 初始化多渠道通知管理器
// 根据配置自动注册可用的通知渠道：
//   - email: 如果 SMTP 已配置
//   - websocket: 如果 WebSocket 已启用
//   - webhook: 如果 Webhook URL 已配置
func initNotifyManager(cfg *config.Config) {
	// 构建邮件发送器（如果已配置）
	var e *emailer.Emailer
	if cfg.Email.Enabled {
		e = emailer.NewEmailer(
			cfg.Email.Host,
			cfg.Email.Port,
			cfg.Email.Username,
			cfg.Email.Password,
			cfg.Email.From,
			cfg.Email.FromName,
		)
	}

	// 获取 WebSocket Hub（如果已初始化）
	var hub *ws.Hub
	if cfg.WS.Enabled {
		hub = ws.GetHub()
	}

	// 创建通知管理器
	manager := notify.NewManager(e, cfg.Email.From, cfg.Email.FromName, hub)

	// 注册 Webhook 渠道（如果已配置）
	if cfg.Notify.Webhook.Enabled && cfg.Notify.Webhook.URL != "" {
		kind := parseWebhookKind(cfg.Notify.Webhook.Kind)
		manager.AddWebhook("webhook-"+cfg.Notify.Webhook.Kind, notify.WebhookOption{
			Kind:   kind,
			URL:    cfg.Notify.Webhook.URL,
			Secret: cfg.Notify.Webhook.Secret,
		})
		logger.Infof("[Notify] Webhook channel registered: %s -> %s", cfg.Notify.Webhook.Kind, cfg.Notify.Webhook.URL)
	}

	// 设置全局管理器
	notify.SetGlobalManager(manager)

	logger.Infof("[Notify] Notification manager initialized with %d channels", manager.ChannelCount())
}

// parseWebhookKind 解析 Webhook 类型字符串
func parseWebhookKind(kind string) notify.WebhookKind {
	switch kind {
	case "dingtalk":
		return notify.WebhookDingTalk
	case "wechat":
		return notify.WebhookWeChat
	case "feishu":
		return notify.WebhookFeishu
	default:
		return notify.WebhookGeneric
	}
}