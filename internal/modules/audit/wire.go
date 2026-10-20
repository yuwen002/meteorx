package audit

import (
	"gorm.io/gorm"

	"meteorx/internal/modules/audit/repository"
	"meteorx/internal/modules/audit/service"
	"meteorx/internal/pkg/emailer"
)

// WireAlertEvaluation 把告警评估链接入审计日志写入路径：以 db 构造告警仓储与
// AlertService，注入给定 AuditService，此后每条/每批日志落库成功即评估规则、
// 命中则产出告警并按渠道类型送达。
// 接入前 EvaluateAndAlert 在生产里没有任何调用方——规则 CRUD、告警记录、通知
// 渠道与看板统计全部就绪，唯独没有触发者，整条告警链是永远不会执行的死代码。
// email 可为 nil（邮件直发渠道自动跳过，其余渠道不受影响）。
func WireAlertEvaluation(auditSvc *service.AuditService, db *gorm.DB, email *emailer.Emailer) {
	auditSvc.SetAlertEvaluator(
		service.NewAlertService(repository.NewAlertRuleRepository(db), email))
}
