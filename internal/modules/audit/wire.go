package audit

import (
	"context"

	"gorm.io/gorm"

	"meteorx/internal/modules/audit/repository"
	"meteorx/internal/modules/audit/service"
	userrepo "meteorx/internal/modules/user/repository"
	"meteorx/internal/pkg/emailer"
)

// WireAlertEvaluation 把告警评估链接入审计日志写入路径：以 db 构造告警仓储与
// AlertService，注入给定 AuditService，此后每条/每批日志落库成功即评估规则、
// 命中则产出告警并按渠道类型送达。
// 接入前 EvaluateAndAlert 在生产里没有任何调用方——规则 CRUD、告警记录、通知
// 渠道与看板统计全部就绪，唯独没有触发者，整条告警链是永远不会执行的死代码。
// email 可为 nil（邮件直发渠道自动跳过，其余渠道不受影响）。
func WireAlertEvaluation(auditSvc *service.AuditService, db *gorm.DB, email *emailer.Emailer) {
	alertSvc := service.NewAlertService(repository.NewAlertRuleRepository(db), email)
	// 告警的邮件收件人（规则 targets 里的邮箱）同样应能在收件箱离线看到：
	// 用 user 仓储把邮箱解析为用户 ID。未注册账号的外部邮箱会被跳过，
	// 告警正文不会因此广播给任何人——收件箱只写「邮箱本来就对应谁」的那些人。
	alertSvc.SetRecipientDirectory(emailUserDirectory{repo: userrepo.NewUserRepository(db)})
	auditSvc.SetAlertEvaluator(alertSvc)
}

// emailUserDirectory 用 user 仓储实现「邮箱 → 用户 ID」窄接口。
type emailUserDirectory struct {
	repo userrepo.UserRepository
}

func (d emailUserDirectory) UserIDByEmail(ctx context.Context, email string) (string, error) {
	u, err := d.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	return u.ID, nil
}
