// Package exporter 定义数据导出的资源导出器接口与注册中心。
// 每种可导出资源（user/task/audit_log/wiki_document）实现 Exporter 接口，
// 由 worker 池按 ExportJob.Resource 分派到对应实现。
package exporter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"meteorx/internal/modules/export/model"
)

// RowSink 抽象「一行行数据」的序列化目标，将「数据生产」（各 Exporter）与
// 「文件格式」（CSV/XLSX/JSON/PDF）解耦。Exporter 只负责产出表头与字符串行，
// 由具体的 RowSink 决定如何落盘，从而一套数据可导出为多种格式。
type RowSink interface {
	// WriteHeader 写出表头行（列名）
	WriteHeader(columns []string) error
	// WriteRows 批量写出若干数据行，返回实际写出的行数
	WriteRows(rows [][]string) (int64, error)
	// Close 刷新并收尾（如写入文件尾、关闭 workbook）；必须调用
	Close() error
}

// Exporter 单类资源的导出器。
// Stream 将表头与数据行通过 sink 写出，返回实际写出的数据行数（不含表头）。
// 实现方应分页读取，避免一次性加载全量数据到内存；每批通过 sink.WriteRows 落盘。
type Exporter interface {
	// Resource 返回该导出器支持的资源类型标识（对应 model.ResourceXxx 常量）
	Resource() string
	// Stream 将数据通过 sink 写出；filters 为创建任务时提交的筛选条件；
	// tenantID 为数据归属租户，master 用户可能跨租户导出时由上层传入具体目标租户。
	Stream(ctx context.Context, tenantID string, filters map[string]any, sink RowSink) (rows int64, err error)
}

// NewSink 按格式构造对应的 RowSink。CSV 不写 BOM（由调用方负责）；其余格式直写。
// 不支持的格式或 PDF 缺少 CJK 字体时返回 error。
func NewSink(format string, dst io.Writer) (RowSink, error) {
	switch format {
	case model.FormatXLSX:
		return NewXLSXSink(dst)
	case model.FormatJSON:
		return NewJSONSink(dst), nil
	case model.FormatPDF:
		return NewPDFSink(dst)
	case model.FormatCSV, "":
		return NewCSVSink(dst), nil
	default:
		return nil, fmt.Errorf("不支持的导出格式: %s", format)
	}
}

// Registry 导出器注册中心，按 resource 分派
type Registry struct {
	mu        sync.RWMutex
	exporters map[string]Exporter
}

// NewRegistry 创建空的注册中心
func NewRegistry() *Registry {
	return &Registry{exporters: make(map[string]Exporter)}
}

// Register 注册一个导出器；同资源重复注册会覆盖前者
func (r *Registry) Register(e Exporter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.exporters[e.Resource()] = e
}

// Get 按资源类型查询导出器
func (r *Registry) Get(resource string) (Exporter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.exporters[resource]
	return e, ok
}

// ErrExporterNotFound 未注册的资源类型
var ErrExporterNotFound = errors.New("exporter not registered for resource")

// ErrNoRows 无数据可导出（可选：某些 exporter 用它区分空结果与真实错误）
var ErrNoRows = errors.New("no rows to export")
