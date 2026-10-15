package exporter

import (
	"context"
	"time"

	"meteorx/internal/modules/export/model"

	"gorm.io/gorm"
)

// taskExporter 导出租户下的任务列表。
type taskExporter struct{ db *gorm.DB }

// NewTaskExporter 创建任务导出器
func NewTaskExporter(db *gorm.DB) Exporter { return &taskExporter{db: db} }

// Resource 返回资源标识
func (e *taskExporter) Resource() string { return model.ResourceTask }

// taskRow 承载 tasks 表导出所需列。
type taskRow struct {
	ID          string     `gorm:"column:id"`
	Title       string     `gorm:"column:title"`
	Status      string     `gorm:"column:status"`
	Priority    string     `gorm:"column:priority"`
	CreatorID   string     `gorm:"column:creator_id"`
	AssigneeID  string     `gorm:"column:assignee_id"`
	DueDate     *time.Time `gorm:"column:due_date"`
	Visibility  string     `gorm:"column:visibility"`
	CompletedAt *time.Time `gorm:"column:completed_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
}

// Stream 分批流式写出任务数据。
func (e *taskExporter) Stream(ctx context.Context, tenantID string, filters map[string]any, sink RowSink) (int64, error) {
	header := []string{"ID", "标题", "状态", "优先级", "创建人ID", "负责人ID", "截止时间", "可见范围", "完成时间", "创建时间"}
	if err := sink.WriteHeader(header); err != nil {
		return 0, err
	}

	var total int64
	q := e.db.WithContext(ctx).Table("tasks").
		Select("id, title, status, priority, creator_id, assignee_id, due_date, visibility, completed_at, created_at").
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
	if s := filterString(filters, "status"); s != "" {
		q = q.Where("status = ?", s)
	}
	if p := filterString(filters, "priority"); p != "" {
		q = q.Where("priority = ?", p)
	}

	var batch []taskRow
	err := q.FindInBatches(&batch, exportBatchSize, func(res *gorm.DB, _ int) error {
		if res.Error != nil {
			return res.Error
		}
		records := make([][]string, 0, len(batch))
		for _, r := range batch {
			records = append(records, []string{
				r.ID, r.Title, r.Status, r.Priority, r.CreatorID, r.AssigneeID,
				fmtNullTime(r.DueDate), r.Visibility, fmtNullTime(r.CompletedAt), fmtTime(r.CreatedAt),
			})
		}
		n, err := sink.WriteRows(records)
		total += n
		if err != nil {
			return err
		}
		return nil
	}).Error
	return total, err
}
