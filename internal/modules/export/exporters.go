package export

import (
	"gorm.io/gorm"

	"meteorx/internal/modules/export/service/exporter"
)

// registerExporters 将各类资源导出器注册到 registry。
// worker 池按 ExportJob.Resource 从 registry 分派到对应 Exporter 实现。
// 具体的 user/task/audit_log/wiki_document 导出器在后续接入时于此 Register。
func registerExporters(r *exporter.Registry, db *gorm.DB) {
	// TODO(commit 3): r.Register(exporter.NewUserExporter(db)) 等
}
