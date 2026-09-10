package notify

import (
	"context"
	"fmt"
	"meteorx/internal/pkg/emailer"
	"strings"
	"time"
)

// EmailNotifier 邮件通知渠道
type EmailNotifier struct {
	emailer *emailer.Emailer
	from    string
	fromName string
}

// NewEmailNotifier 创建邮件通知渠道
// 如果 emailer 未配置（nil），则返回 nil，表示邮件渠道不可用
func NewEmailNotifier(e *emailer.Emailer, from, fromName string) *EmailNotifier {
	if e == nil {
		return nil
	}
	return &EmailNotifier{
		emailer:  e,
		from:     from,
		fromName: fromName,
	}
}

func (n *EmailNotifier) Type() ChannelType { return ChannelEmail }

func (n *EmailNotifier) Name() string { return "email" }

func (n *EmailNotifier) Send(ctx context.Context, msg *Message) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if len(msg.Recipients) == 0 {
		return fmt.Errorf("no recipients")
	}

	// 构建邮件 HTML 正文
	htmlBody := n.buildHTMLBody(msg)

	// 逐个发送（SMTP 不支持批量）
	var errs []string
	for _, to := range msg.Recipients {
		if err := n.emailer.Send(to, msg.Title, htmlBody); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", to, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("email send partial failure: %s", strings.Join(errs, "; "))
	}
	return nil
}

// buildHTMLBody 构建 HTML 邮件正文
func (n *EmailNotifier) buildHTMLBody(msg *Message) string {
	priorityLabel := map[Priority]string{
		PriorityLow:    "🟢 通知",
		PriorityNormal: "🔵 提醒",
		PriorityHigh:   "🟡 重要",
		PriorityUrgent: "🔴 紧急",
	}

	label := priorityLabel[msg.Priority]
	if label == "" {
		label = "🔵 提醒"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="margin:0;padding:0;background-color:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;">
	<div style="max-width:600px;margin:20px auto;background:#ffffff;border-radius:12px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.06);">
		<div style="background:linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);padding:32px 24px;text-align:center;">
			<h1 style="color:#ffffff;margin:0;font-size:20px;font-weight:600;">%s</h1>
		</div>
		<div style="padding:24px;">
			<h2 style="color:#1a1a2e;font-size:18px;margin:0 0 16px 0;">%s</h2>
			<div style="color:#4a4a6a;font-size:14px;line-height:1.8;white-space:pre-wrap;">%s</div>
			<hr style="border:none;border-top:1px solid #e8e8f0;margin:24px 0;">
			<p style="color:#999;font-size:12px;text-align:center;margin:0;">
				此邮件由 MeteorX 系统自动发送<br>
				发送时间：%s
			</p>
		</div>
	</div>
</body>
</html>`, label, msg.Title, escapeHTML(msg.Content), time.Now().Format("2006-01-02 15:04:05"))
}

// escapeHTML 简单转义 HTML 特殊字符
func escapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(s)
}