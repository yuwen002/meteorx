// Package model 定义数据导出模块的领域模型。
package model

import "time"

// 导出任务状态
const (
	ExportStatusPending   = "pending"   // 待处理
	ExportStatusRunning   = "running"   // 执行中
	ExportStatusCompleted = "completed" // 已完成
	ExportStatusFailed    = "failed"    // 失败
)

// 支持的资源类型
const (
	ResourceUser         = "user"
	ResourceTask         = "task"
	ResourceAuditLog     = "audit_log"
	ResourceWikiDocument = "wiki_document"
)

// ValidResourceType 判断资源类型是否受支持
func ValidResourceType(t string) bool {
	switch t {
	case ResourceUser, ResourceTask, ResourceAuditLog, ResourceWikiDocument:
		return true
	}
	return false
}

// ExportJob 数据导出任务领域模型
type ExportJob struct {
	ID          string
	TenantID    string
	UserID      string
	Resource    string // user / task / audit_log / wiki_document
	FiltersJSON string // 序列化的筛选条件 JSON
	Status      string // pending / running / completed / failed
	FilePath    string // 生成的 CSV 文件绝对路径
	Rows        int64  // 导出总行数
	ErrorMsg    string // 失败原因
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	ExpiresAt   time.Time // 文件过期时间，由定时任务清理
}
