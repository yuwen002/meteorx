// Package emailer 提供 SMTP 邮件发送功能，支持纯文本、HTML 及模板邮件，用于邀请/验证/通知等场景。
package emailer

import (
	"fmt"
	"net/smtp"
)

// Emailer SMTP 邮件发送器，封装连接参数与 HTML 模板邮件的发送。
type Emailer struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
}

// NewEmailer 创建邮件发送器实例。
func NewEmailer(host string, port int, username, password, from, fromName string) *Emailer {
	return &Emailer{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		FromName: fromName,
	}
}

// Send 以 HTML 格式向收件人发送一封邮件。
func (e *Emailer) Send(to, subject, body string) error {
	if e.Host == "" {
		return fmt.Errorf("SMTP host not configured")
	}

	from := e.From
	if e.FromName != "" {
		from = fmt.Sprintf("%s <%s>", e.FromName, e.From)
	}

	mime := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n"
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n%s\r\n\r\n%s", from, to, subject, mime, body)

	auth := smtp.PlainAuth("", e.Username, e.Password, e.Host)

	addr := fmt.Sprintf("%s:%d", e.Host, e.Port)

	return smtp.SendMail(addr, auth, e.From, []string{to}, []byte(msg))
}

// SendResetPasswordEmail 发送密码重置邮件。
func (e *Emailer) SendResetPasswordEmail(to, resetLink, username string) error {
	subject := "【MeteorX】密码重置请求"
	body := fmt.Sprintf(`
		<div style="max-width: 600px; margin: 0 auto; padding: 20px; font-family: Arial, sans-serif;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; text-align: center;">MeteorX 多租户平台</h1>
			</div>
			<div style="background: #f8f9fa; padding: 30px; border-radius: 0 0 10px 10px; border: 1px solid #e9ecef; border-top: none;">
				<h2 style="color: #333; margin-top: 0;">密码重置请求</h2>
				<p style="color: #666; line-height: 1.6;">
					您好 <strong>%s</strong>，
				</p>
				<p style="color: #666; line-height: 1.6;">
					我们收到了您的密码重置请求。请点击下方链接设置新密码：
				</p>
				<p style="text-align: center; margin: 30px 0;">
					<a href="%s" style="display: inline-block; padding: 15px 40px; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; text-decoration: none; border-radius: 5px; font-weight: bold;">
						重置密码
					</a>
				</p>
				<p style="color: #999; font-size: 14px; line-height: 1.6;">
					<strong>注意：</strong>此链接将在 30 分钟内有效。如果您没有请求重置密码，请忽略此邮件。
				</p>
				<p style="color: #999; font-size: 14px; line-height: 1.6;">
					如果链接无法点击，请复制以下链接到浏览器中访问：<br/>
					<a href="%s" style="color: #667eea; word-break: break-all;">%s</a>
				</p>
				<hr style="border: none; border-top: 1px solid #e9ecef; margin: 30px 0;">
				<p style="color: #999; font-size: 12px; text-align: center;">
					此邮件由 MeteorX 系统自动发送，请勿直接回复。
				</p>
			</div>
		</div>
	`, username, resetLink, resetLink, resetLink)

	return e.Send(to, subject, body)
}

// SendVerificationEmail 发送带验证码的邮箱验证邮件。
func (e *Emailer) SendVerificationEmail(to, verificationCode, username string) error {
	subject := "【MeteorX】邮箱验证"
	body := fmt.Sprintf(`
		<div style="max-width: 600px; margin: 0 auto; padding: 20px; font-family: Arial, sans-serif;">
			<div style="background: linear-gradient(135deg, #11998e 0%%, #38ef7d 100%%); padding: 30px; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; text-align: center;">MeteorX 多租户平台</h1>
			</div>
			<div style="background: #f8f9fa; padding: 30px; border-radius: 0 0 10px 10px; border: 1px solid #e9ecef; border-top: none;">
				<h2 style="color: #333; margin-top: 0;">邮箱验证码</h2>
				<p style="color: #666; line-height: 1.6;">
					您好 <strong>%s</strong>，
				</p>
				<p style="color: #666; line-height: 1.6;">
					您的验证码是：
				</p>
				<p style="text-align: center; font-size: 36px; font-weight: bold; color: #11998e; margin: 20px 0; letter-spacing: 5px;">
					%s
				</p>
				<p style="color: #999; font-size: 14px;">
					此验证码将在 10 分钟内有效。
				</p>
				<hr style="border: none; border-top: 1px solid #e9ecef; margin: 30px 0;">
				<p style="color: #999; font-size: 12px; text-align: center;">
					此邮件由 MeteorX 系统自动发送，请勿直接回复。
				</p>
			</div>
		</div>
	`, username, verificationCode)

	return e.Send(to, subject, body)
}

// SendEmailVerificationLink 发送带验证链接的邮箱验证邮件。
func (e *Emailer) SendEmailVerificationLink(to, verifyLink, username string) error {
	subject := "【MeteorX】验证您的邮箱地址"
	body := fmt.Sprintf(`
		<div style="max-width: 600px; margin: 0 auto; padding: 20px; font-family: Arial, sans-serif;">
			<div style="background: linear-gradient(135deg, #11998e 0%%, #38ef7d 100%%); padding: 30px; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; text-align: center;">MeteorX 多租户平台</h1>
			</div>
			<div style="background: #f8f9fa; padding: 30px; border-radius: 0 0 10px 10px; border: 1px solid #e9ecef; border-top: none;">
				<h2 style="color: #333; margin-top: 0;">验证您的邮箱地址</h2>
				<p style="color: #666; line-height: 1.6;">
					您好 <strong>%s</strong>，
				</p>
				<p style="color: #666; line-height: 1.6;">
					请点击下方链接验证您的邮箱地址：
				</p>
				<p style="text-align: center; margin: 30px 0;">
					<a href="%s" style="display: inline-block; padding: 15px 40px; background: linear-gradient(135deg, #11998e 0%%, #38ef7d 100%%); color: white; text-decoration: none; border-radius: 5px; font-weight: bold;">
						验证邮箱
					</a>
				</p>
				<p style="color: #999; font-size: 14px; line-height: 1.6;">
					<strong>注意：</strong>此链接将在 24 小时内有效。如果您没有注册 MeteorX 账号，请忽略此邮件。
				</p>
				<p style="color: #999; font-size: 14px; line-height: 1.6;">
					如果链接无法点击，请复制以下链接到浏览器中访问：<br/>
					<a href="%s" style="color: #11998e; word-break: break-all;">%s</a>
				</p>
				<hr style="border: none; border-top: 1px solid #e9ecef; margin: 30px 0;">
				<p style="color: #999; font-size: 12px; text-align: center;">
					此邮件由 MeteorX 系统自动发送，请勿直接回复。
				</p>
			</div>
		</div>
	`, username, verifyLink, verifyLink, verifyLink)

	return e.Send(to, subject, body)
}

// SendInvitationEmail 发送加入租户的邀请邮件。
func (e *Emailer) SendInvitationEmail(to, inviteLink, inviterName, tenantName string) error {
	subject := fmt.Sprintf("【MeteorX】%s 邀请您加入 %s", inviterName, tenantName)
	body := fmt.Sprintf(`
		<div style="max-width: 600px; margin: 0 auto; padding: 20px; font-family: Arial, sans-serif;">
			<div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; border-radius: 10px 10px 0 0;">
				<h1 style="color: white; margin: 0; text-align: center;">MeteorX 多租户平台</h1>
			</div>
			<div style="background: #f8f9fa; padding: 30px; border-radius: 0 0 10px 10px; border: 1px solid #e9ecef; border-top: none;">
				<h2 style="color: #333; margin-top: 0;">加入团队邀请</h2>
				<p style="color: #666; line-height: 1.6;">
					您好，
				</p>
				<p style="color: #666; line-height: 1.6;">
					<strong>%s</strong> 邀请您加入租户 <strong>%s</strong>。
				</p>
				<p style="text-align: center; margin: 30px 0;">
					<a href="%s" style="display: inline-block; padding: 15px 40px; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; text-decoration: none; border-radius: 5px; font-weight: bold;">
						接受邀请
					</a>
				</p>
				<p style="color: #999; font-size: 14px; line-height: 1.6;">
					<strong>注意：</strong>此邀请链接将在 7 天内有效。如果您不认识邀请人，请忽略此邮件。
				</p>
				<p style="color: #999; font-size: 14px; line-height: 1.6;">
					如果链接无法点击，请复制以下链接到浏览器中访问：<br/>
					<a href="%s" style="color: #667eea; word-break: break-all;">%s</a>
				</p>
				<hr style="border: none; border-top: 1px solid #e9ecef; margin: 30px 0;">
				<p style="color: #999; font-size: 12px; text-align: center;">
					此邮件由 MeteorX 系统自动发送，请勿直接回复。
				</p>
			</div>
		</div>
	`, inviterName, tenantName, inviteLink, inviteLink, inviteLink)

	return e.Send(to, subject, body)
}
