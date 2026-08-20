package dto

// UploadFileReq 文件上传请求
type UploadFileReq struct {
	File interface{} `form:"file" binding:"required"` // multipart.File
}

// UploadFileResp 文件上传响应
type UploadFileResp struct {
	ID          string `json:"id"`
	FileName    string `json:"file_name"`
	OriginalName string `json:"original_name"`
	FileSize    int64  `json:"file_size"`
	MimeType    string `json:"mime_type"`
	FileType    string `json:"file_type"`
	URL         string `json:"url"`
}

// FileListReq 文件列表请求
type FileListReq struct {
	Page     int    `form:"page" binding:"min=1"`
	PageSize int    `form:"page_size" binding:"min=1,max=100"`
	FileType string `form:"file_type"` // image, document, video, audio, other
	Keyword  string `form:"keyword"`   // 文件名搜索
}

// FileResp 文件响应
type FileResp struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	UserID       string `json:"user_id"`
	FileName     string `json:"file_name"`
	OriginalName string `json:"original_name"`
	FileSize     int64  `json:"file_size"`
	MimeType     string `json:"mime_type"`
	FileType     string `json:"file_type"`
	URL          string `json:"url"`
	MD5          string `json:"md5,omitempty"`
	Status       int    `json:"status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// FileUpdateReq 文件更新请求
type FileUpdateReq struct {
	FileName string `json:"file_name" binding:"required,max=255"`
}

// BatchDeleteReq 批量删除请求
type BatchDeleteReq struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// BatchDeleteResp 批量删除响应
type BatchDeleteResp struct {
	SuccessCount int      `json:"success_count"`
	FailedCount  int      `json:"failed_count"`
	FailedIDs    []string `json:"failed_ids,omitempty"`
}
