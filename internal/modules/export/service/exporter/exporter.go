// Package exporter 定义数据导出的资源导出器接口与注册中心。
// 每种可导出资源（user/task/audit_log/wiki_document）实现 Exporter 接口，
// 由 worker 池按 ExportJob.Resource 分派到对应实现。
package exporter

import (
	"context"
	"encoding/csv"
	"errors"
	"sync"
)

// Exporter 单类资源的导出器。
// Stream 应将 CSV 表头与数据行按顺序写入 w，返回实际写出的数据行数（不含表头）。
// 实现方应分页读取，避免一次性加载全量数据到内存；每写若干行调用 w.Flush() 落盘。
type Exporter interface {
	// Resource 返回该导出器支持的资源类型标识（对应 model.ResourceXxx 常量）
	Resource() string
	// Stream 将数据以 CSV 流式写入 w；filters 为创建任务时提交的筛选条件；
	// tenantID 为数据归属租户，master 用户可能跨租户导出时由上层传入具体目标租户。
	Stream(ctx context.Context, tenantID string, filters map[string]any, w *csv.Writer) (rows int64, err error)
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
