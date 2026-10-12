package export

import (
	"gorm.io/gorm"

	"meteorx/internal/modules/export/service/exporter"
)

// registerExporters 将各类资源导出器注册到 registry。
// worker 池按 ExportJob.Resource 从 registry 分派到对应 Exporter 实现。
// 新增可导出资源时，在此 Register 对应 exporter 即可。
func registerExporters(r *exporter.Registry, db *gorm.DB) {
	r.Register(exporter.NewUserExporter(db))
	r.Register(exporter.NewTaskExporter(db))
	r.Register(exporter.NewAuditExporter(db))
	r.Register(exporter.NewWikiDocumentExporter(db))
}
