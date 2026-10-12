package exporter

import (
	"context"
	"encoding/csv"
	"strconv"
	"time"

	"meteorx/internal/modules/export/model"

	"gorm.io/gorm"
)

// auditExporter 导出租户下的审计日志。
type auditExporter struct{ db *gorm.DB }

// NewAuditExporter 创建审计日志导出器
func NewAuditExporter(db *gorm.DB) Exporter { return &auditExporter{db: db} }

// Resource 返回资源标识
func (e *auditExporter) Resource() string { return model.ResourceAuditLog }

// auditRow 承载 audit_logs 表导出所需列，可空列用指针承接 NULL。
type auditRow struct {
	ID         string    `gorm:"column:id"`
	Username   *string   `gorm:"column:username"`
	UserID     *string   `gorm:"column:user_id"`
	Module     *string   `gorm:"column:module"`
	Action     *string   `gorm:"column:action"`
	Resource   *string   `gorm:"column:resource"`
	Result     *string   `gorm:"column:result"`
	RiskLevel  *string   `gorm:"column:risk_level"`
	ClientIP   *string   `gorm:"column:client_ip"`
	Method     *string   `gorm:"column:method"`
	Path       *string   `gorm:"column:path"`
	StatusCode *int      `gorm:"column:status_code"`
	Duration   *int64    `gorm:"column:duration"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

// Stream 分批流式写出审计日志 CSV。
func (e *auditExporter) Stream(ctx context.Context, tenantID string, filters map[string]any, w *csv.Writer) (int64, error) {
	header := []string{"ID", "操作用户", "用户ID", "模块", "动作", "资源", "结果", "风险等级", "客户端IP", "方法", "路径", "状态码", "耗时(ms)", "时间"}
	if err := writeHeader(w, header); err != nil {
		return 0, err
	}

	var total int64
	q := e.db.WithContext(ctx).Table("audit_logs").
		Select("id, username, user_id, module, action, resource, result, risk_level, client_ip, method, path, status_code, duration, created_at").
		Where("tenant_id = ?", tenantID)
	if s := filterString(filters, "module"); s != "" {
		q = q.Where("module = ?", s)
	}
	if s := filterString(filters, "action"); s != "" {
		q = q.Where("action = ?", s)
	}
	if s := filterString(filters, "result"); s != "" {
		q = q.Where("result = ?", s)
	}
	if s := filterString(filters, "riskLevel"); s != "" {
		q = q.Where("risk_level = ?", s)
	}
	if s := filterString(filters, "startTime"); s != "" {
		q = q.Where("created_at >= ?", s)
	}
	if s := filterString(filters, "endTime"); s != "" {
		q = q.Where("created_at <= ?", s)
	}

	var batch []auditRow
	err := q.FindInBatches(&batch, exportBatchSize, func(res *gorm.DB, _ int) error {
		if res.Error != nil {
			return res.Error
		}
		records := make([][]string, 0, len(batch))
		for _, r := range batch {
			records = append(records, []string{
				r.ID, fmtStr(r.Username), fmtStr(r.UserID), fmtStr(r.Module), fmtStr(r.Action),
				fmtStr(r.Resource), fmtStr(r.Result), fmtStr(r.RiskLevel), fmtStr(r.ClientIP),
				fmtStr(r.Method), fmtStr(r.Path), fmtInt(r.StatusCode), fmtInt64(r.Duration), fmtTime(r.CreatedAt),
			})
		}
		n, err := writeRows(w, records)
		total += n
		return err
	}).Error
	return total, err
}

// fmtInt64 格式化可空 64 位整型
func fmtInt64(v *int64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(*v, 10)
}
