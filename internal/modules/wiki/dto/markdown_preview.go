package dto

// MarkdownPreviewReq Markdown 实时预览请求（编辑器分栏预览使用）
type MarkdownPreviewReq struct {
	Content string `json:"content"`
	Format  string `json:"format,omitempty"`
}
