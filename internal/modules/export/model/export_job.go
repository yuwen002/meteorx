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

// 支持的导出文件格式（创建任务时不传或传空默认 CSV）
const (
	FormatCSV  = "csv"
	FormatXLSX = "xlsx"
	FormatJSON = "json"
	FormatPDF  = "pdf"

	// DefaultFormat 未显式指定格式时的默认格式
	DefaultFormat = FormatCSV
)

// ValidFormat 判断导出格式是否受支持
func ValidFormat(f string) bool {
	switch f {
	case FormatCSV, FormatXLSX, FormatJSON, FormatPDF:
		return true
	}
	return false
}

// FormatExt 返回格式对应的文件扩展名（不含点号），未知/空回落 csv
func FormatExt(f string) string {
	switch f {
	case FormatXLSX:
		return "xlsx"
	case FormatJSON:
		return "json"
	case FormatPDF:
		return "pdf"
	default:
		return "csv"
	}
}

// FormatContentType 返回格式对应的 HTTP Content-Type；未知/空回落 csv
func FormatContentType(f string) string {
	switch f {
	case FormatXLSX:
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case FormatJSON:
		return "application/json; charset=utf-8"
	case FormatPDF:
		return "application/pdf"
	default:
		return "text/csv; charset=utf-8"
	}
}

// ExportJob 数据导出任务领域模型
type ExportJob struct {
	ID          string
	TenantID    string
	UserID      string
	Resource    string // user / task / audit_log / wiki_document
	Format      string // csv / xlsx / json / pdf，空表示 csv
	FiltersJSON string // 序列化的筛选条件 JSON
	Status      string // pending / running / completed / failed
	FilePath    string // 生成的导出文件绝对路径
	Rows        int64  // 导出总行数
	ErrorMsg    string // 失败原因
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	ExpiresAt   time.Time // 文件过期时间，由定时任务清理
	// LeaseExpiresAt 执行租约到期时间：worker 抢占时写入、执行期间持续续租。
	// 进程被 kill -9/OOM 时没有任何收尾机会，租约到期即说明持有者已死，任务可被回收重驱。
	LeaseExpiresAt *time.Time
	// LeaseToken 本次执行的归属令牌（每次抢占随机生成）：回收重驱后，旧执行的迟到回写
	// 因令牌不再匹配而自然空转，不会覆写新执行的租约与终态。
	LeaseToken string
	Attempts   int // 已被抢占执行的次数，用于限制崩溃重驱上限，避免必然撞死进程的任务无限循环
}
