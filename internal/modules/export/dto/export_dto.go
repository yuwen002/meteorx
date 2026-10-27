// Package dto 定义数据导出模块的请求与响应结构。
package dto

import (
	"time"

	"meteorx/internal/modules/export/model"
)

// CreateExportReq 创建导出任务请求
type CreateExportReq struct {
	Resource string `json:"resource" validate:"required,oneof=user task audit_log wiki_document" label:"资源类型"`
	Format   string `json:"format,omitempty" validate:"omitempty,oneof=csv xlsx json pdf" label:"导出格式"`
	// TenantID 仅系统管理员生效：为其它租户创建导出任务时显式声明数据归属租户
	TenantID string         `json:"tenant_id,omitempty" validate:"omitempty,max=26" label:"目标租户"`
	Filters  map[string]any `json:"filters,omitempty" validate:"max=32" label:"筛选条件"`
}

// ExportJobResp 导出任务响应
type ExportJobResp struct {
	ID          string  `json:"id"`
	TenantID    string  `json:"tenant_id"`
	UserID      string  `json:"user_id"`
	Resource    string  `json:"resource"`
	Format      string  `json:"format"`
	Status      string  `json:"status"`
	Rows        int64   `json:"rows"`
	ErrorMsg    string  `json:"error_msg,omitempty"`
	CreatedAt   string  `json:"created_at"`
	StartedAt   *string `json:"started_at,omitempty"`
	CompletedAt *string `json:"completed_at,omitempty"`
	ExpiresAt   string  `json:"expires_at"`
}

// ListExportReq 列表查询请求
type ListExportReq struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	Status   string `form:"status"`
	Resource string `form:"resource"`
}

// ToResp 将领域模型转换为响应 DTO
func ToResp(m *model.ExportJob) *ExportJobResp {
	resp := &ExportJobResp{
		ID:        m.ID,
		TenantID:  m.TenantID,
		UserID:    m.UserID,
		Resource:  m.Resource,
		Format:    m.Format,
		Status:    m.Status,
		Rows:      m.Rows,
		ErrorMsg:  m.ErrorMsg,
		CreatedAt: m.CreatedAt.Format(time.RFC3339),
		ExpiresAt: m.ExpiresAt.Format(time.RFC3339),
	}
	if m.StartedAt != nil {
		s := m.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &s
	}
	if m.CompletedAt != nil {
		s := m.CompletedAt.Format(time.RFC3339)
		resp.CompletedAt = &s
	}
	return resp
}
