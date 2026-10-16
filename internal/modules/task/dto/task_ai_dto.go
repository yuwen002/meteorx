// Package dto 定义任务模块 AI 辅助能力的请求/响应数据传输对象。
// 这些端点均为附加能力：自然语言解析产出草稿、字段建议、描述扩写、任务摘要，
// 不直接写库，由用户确认后再走原有创建/更新流程。
package dto

// ParseTaskReq 自然语言建任务解析请求。
type ParseTaskReq struct {
	Text string `json:"text" validate:"required,max=500" label:"自然语言描述"` // 用户输入的一句/段自由文本
}

// TaskDraftResp 任务草稿响应（对齐 CreateTaskReq 字段，便于前端预填后再创建）。
// 不含 assignee/visibility 归属敏感项之外的字段仍给出建议，由用户最终确认。
type TaskDraftResp struct {
	Title      string   `json:"title"`
	Priority   string   `json:"priority"`             // low/normal/high/urgent
	DueDate    string   `json:"due_date,omitempty"`   // 解析出的截止时间（RFC3339 或 "2006-01-02 15:04:05"），无法确定为空
	Tags       []string `json:"tags"`                 // 建议标签
	Recurrence string   `json:"recurrence,omitempty"` // daily/weekly/monthly，未提及为空
}

// SuggestReq 优先级/标签建议请求。
type SuggestReq struct {
	Title       string `json:"title" validate:"required,max=200" label:"标题"`
	Description string `json:"description" validate:"max=2000" label:"描述"`
}

// SuggestResp 优先级/标签建议响应。
type SuggestResp struct {
	Priority string   `json:"priority"` // low/normal/high/urgent
	Tags     []string `json:"tags"`
	Reason   string   `json:"reason"` // 简短理由，便于用户判断是否采纳
}

// ExpandReq 描述扩写/拆解请求。
type ExpandReq struct {
	Title       string `json:"title" validate:"required,max=200" label:"标题"`
	Description string `json:"description" validate:"max=2000" label:"描述"`
}

// ExpandResp 描述扩写/拆解响应。
type ExpandResp struct {
	Description string   `json:"description"` // 扩写后的完整描述
	Steps       []string `json:"steps"`       // 拆解出的子步骤（可为空）
}

// SummaryReq 任务摘要/周报请求。Scope 指定统计口径。
type SummaryReq struct {
	Scope string `json:"scope" validate:"omitempty,oneof=today week overdue all" label:"摘要范围"` // 空默认 all
}

// SummaryResp 任务摘要响应。
type SummaryResp struct {
	Text           string   `json:"text"`            // 中文摘要正文
	Highlights     []string `json:"highlights"`      // 关键要点
	SuggestedFocus []string `json:"suggested_focus"` // 建议优先处理项
	Total          int      `json:"total"`           // 参与统计的任务总数
}

// AICapabilityResp AI 能力探测响应，供前端决定相关按钮的显隐/禁用。
type AICapabilityResp struct {
	Enabled bool   `json:"enabled"` // 是否可用（已启用且配置了凭据）
	Model   string `json:"model"`   // 当前默认模型名
}
