package exporter

import (
	"context"
	"fmt"
	"time"

	"meteorx/internal/modules/export/model"

	"gorm.io/gorm"
)

// wikiDocumentExporter 导出租户下的 Wiki 文档元信息（不含正文内容）。
type wikiDocumentExporter struct{ db *gorm.DB }

// NewWikiDocumentExporter 创建 Wiki 文档导出器
func NewWikiDocumentExporter(db *gorm.DB) Exporter { return &wikiDocumentExporter{db: db} }

// Resource 返回资源标识
func (e *wikiDocumentExporter) Resource() string { return model.ResourceWikiDocument }

// wikiDocRow 承载 wiki_documents 表导出所需列。
type wikiDocRow struct {
	ID            string     `gorm:"column:id"`
	NodeID        string     `gorm:"column:node_id"`
	Format        *string    `gorm:"column:format"`
	CurrentVer    int        `gorm:"column:current_ver"`
	ViewCount     int64      `gorm:"column:view_count"`
	PublishStatus *string    `gorm:"column:publish_status"`
	LastEditedBy  *string    `gorm:"column:last_edited_by"`
	PublishedAt   *time.Time `gorm:"column:published_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

// Stream 分批流式写出 Wiki 文档数据。
func (e *wikiDocumentExporter) Stream(ctx context.Context, tenantID string, filters map[string]any, sink RowSink) (int64, error) {
	header := []string{"ID", "节点ID", "格式", "当前版本", "浏览数", "发布状态", "最后编辑人", "发布时间", "创建时间", "更新时间"}
	if err := sink.WriteHeader(header); err != nil {
		return 0, err
	}

	var total int64
	q := e.db.WithContext(ctx).Table("wiki_documents").
		Select("id, node_id, format, current_ver, view_count, publish_status, last_edited_by, published_at, created_at, updated_at").
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
	if s := filterString(filters, "publishStatus"); s != "" {
		q = q.Where("publish_status = ?", s)
	}

	var batch []wikiDocRow
	err := q.FindInBatches(&batch, exportBatchSize, func(res *gorm.DB, _ int) error {
		if res.Error != nil {
			return res.Error
		}
		records := make([][]string, 0, len(batch))
		for _, r := range batch {
			records = append(records, []string{
				r.ID, r.NodeID, fmtStr(r.Format), fmt.Sprintf("%d", r.CurrentVer), fmt.Sprintf("%d", r.ViewCount),
				fmtStr(r.PublishStatus), fmtStr(r.LastEditedBy), fmtNullTime(r.PublishedAt),
				fmtTime(r.CreatedAt), fmtTime(r.UpdatedAt),
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
