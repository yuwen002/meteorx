// Package dto 定义 Wiki 模块的请求与响应数据结构。
package dto

import "time"

// CreateWikiSpaceReq 创建知识空间请求
type CreateWikiSpaceReq struct {
	Name        string `json:"name" validate:"required,min=2,max=200"`
	Description string `json:"description,omitempty" validate:"max=500"`
	Icon        string `json:"icon,omitempty" validate:"max=255"`
	Visibility  int    `json:"visibility" validate:"oneof=1 2 3"`
}

// UpdateWikiSpaceReq 更新知识空间请求
type UpdateWikiSpaceReq struct {
	Name        string `json:"name,omitempty" validate:"omitempty,min=2,max=200"`
	Description string `json:"description,omitempty" validate:"omitempty,max=500"`
	Icon        string `json:"icon,omitempty" validate:"omitempty,max=255"`
	Visibility  int    `json:"visibility,omitempty" validate:"omitempty,oneof=1 2 3"`
}

// WikiSpaceResp 知识空间响应
type WikiSpaceResp struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Visibility  int       `json:"visibility"`
	MemberCount int64     `json:"member_count"`
	NodeCount   int64     `json:"node_count"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	MyRole      string    `json:"my_role"`
}

// CreateWikiNodeReq 创建节点请求（文件夹或文档）
type CreateWikiNodeReq struct {
	SpaceID  string `json:"space_id,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
	Type     string `json:"type" validate:"required,oneof=folder document"`
	Title    string `json:"title" validate:"required,min=1,max=500"`
	Icon     string `json:"icon,omitempty"`
	Sort     int    `json:"sort,omitempty"`
	Content  string `json:"content,omitempty"`
}

// UpdateWikiNodeReq 更新节点请求（支持移动、重命名、排序）
type UpdateWikiNodeReq struct {
	Title    string `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Icon     string `json:"icon,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
	Sort     int    `json:"sort,omitempty"`
	Status   int    `json:"status,omitempty" validate:"omitempty,oneof=1 2"`
}

// WikiNodeResp 节点响应
type WikiNodeResp struct {
	ID         string    `json:"id"`
	SpaceID    string    `json:"space_id"`
	ParentID   string    `json:"parent_id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Icon       string    `json:"icon"`
	Sort       int       `json:"sort"`
	Status     int       `json:"status"`
	OwnerID    string    `json:"owner_id"`
	ChildCount int64     `json:"child_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// WikiNodeTreeResp 节点树响应（递归嵌套子节点）
type WikiNodeTreeResp struct {
	WikiNodeResp
	Children []*WikiNodeTreeResp `json:"children,omitempty"`
}

// CreateDocumentReq 创建文档请求
type CreateDocumentReq struct {
	NodeID  string `json:"node_id" validate:"required"`
	Content string `json:"content,omitempty"`
	Format  string `json:"format,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// UpdateDocumentReq 更新文档请求
type UpdateDocumentReq struct {
	Content string `json:"content,omitempty"`
	Format  string `json:"format,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// DocumentResp 文档响应
type DocumentResp struct {
	ID            string     `json:"id"`
	NodeID        string     `json:"node_id"`
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	ContentHTML   string     `json:"content_html"`
	Format        string     `json:"format"`
	CurrentVer    int        `json:"current_ver"`
	ViewCount     int64      `json:"view_count"`
	LastEditedBy  string     `json:"last_edited_by"`
	LastEditedAt  *time.Time `json:"last_edited_at"`
	PublishStatus string     `json:"publish_status"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	ReviewedBy    string     `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// DocumentRevisionResp 文档历史版本响应
type DocumentRevisionResp struct {
	ID          string    `json:"id"`
	DocumentID  string    `json:"document_id"`
	Version     int       `json:"version"`
	Content     string    `json:"content"`
	ContentHTML string    `json:"content_html"`
	Summary     string    `json:"summary"`
	EditedBy    string    `json:"edited_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// WikiSpaceMemberReq 添加空间成员请求
type WikiSpaceMemberReq struct {
	UserID string `json:"user_id" validate:"required"`
	Role   string `json:"role" validate:"required,oneof=owner admin editor viewer"`
}

// WikiSpaceMemberResp 空间成员响应
type WikiSpaceMemberResp struct {
	ID        string    `json:"id"`
	SpaceID   string    `json:"space_id"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	UserEmail string    `json:"user_email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// SetNodePermissionReq 设置节点权限请求
type SetNodePermissionReq struct {
	UserID     string `json:"user_id" validate:"required"`
	Permission string `json:"permission" validate:"required,oneof=view edit delete"`
}

// NodePermissionResp 节点权限响应
type NodePermissionResp struct {
	ID         string    `json:"id"`
	NodeID     string    `json:"node_id"`
	UserID     string    `json:"user_id"`
	Permission string    `json:"permission"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// WikiStatsResp Wiki 统计响应
type WikiStatsResp struct {
	TotalSpaces    int64 `json:"total_spaces"`
	TotalNodes     int64 `json:"total_nodes"`
	TotalDocuments int64 `json:"total_documents"`
	TotalViews     int64 `json:"total_views"`
}

// TrashItemResp 回收站项目响应
type TrashItemResp struct {
	ID        string    `json:"id"`
	ItemType  string    `json:"item_type"`
	ItemID    string    `json:"item_id"`
	SpaceID   string    `json:"space_id"`
	Title     string    `json:"title"`
	DeletedBy string    `json:"deleted_by"`
	DeletedAt time.Time `json:"deleted_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// TrashListReq 回收站列表请求
type TrashListReq struct {
	SpaceID  string `json:"space_id"`
	ItemType string `json:"item_type"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

// TrashRestoreReq 恢复回收站项目请求
type TrashRestoreReq struct {
	RestoreChildren bool `json:"restore_children"`
}

// SearchResultResp 搜索结果响应
type SearchResultResp struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	SpaceID   string    `json:"space_id"`
	NodeID    string    `json:"node_id"`
	Snippet   string    `json:"snippet"`
	Highlight string    `json:"highlight"`
	UpdatedAt time.Time `json:"updated_at"`
	Score     float64   `json:"score"`
}

// CreateAttachmentReq 创建附件请求
type CreateAttachmentReq struct {
	DocumentID string `json:"document_id" validate:"required"`
	FileID     string `json:"file_id,omitempty"`
	FileName   string `json:"file_name" validate:"required"`
	FileSize   int64  `json:"file_size"`
	MimeType   string `json:"mime_type"`
	FileURL    string `json:"file_url" validate:"required"`
}

// AttachmentResp 附件响应
type AttachmentResp struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	FileID     string    `json:"file_id,omitempty"`
	FileName   string    `json:"file_name"`
	FileSize   int64     `json:"file_size"`
	MimeType   string    `json:"mime_type"`
	FileURL    string    `json:"file_url"`
	UploadedBy string    `json:"uploaded_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// TagResp 标签响应
type TagResp struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateTagReq 创建标签请求
type CreateTagReq struct {
	Name  string `json:"name" validate:"required,max=100"`
	Color string `json:"color" validate:"max=20"`
}

// DocumentTagResp 文档标签关联响应
type DocumentTagResp struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	Tag        TagResp   `json:"tag"`
	CreatedAt  time.Time `json:"created_at"`
}

// CommentResp 评论响应（支持嵌套回复）
type CommentResp struct {
	ID         string        `json:"id"`
	DocumentID string        `json:"document_id"`
	NodeID     string        `json:"node_id"`
	ParentID   string        `json:"parent_id"`
	Content    string        `json:"content"`
	CreatedBy  string        `json:"created_by"`
	UserName   string        `json:"user_name"`
	MentionIDs string        `json:"mention_ids"`
	Status     int           `json:"status"`
	Replies    []CommentResp `json:"replies,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

// CreateCommentReq 创建评论请求
type CreateCommentReq struct {
	DocumentID string `json:"document_id" validate:"required"`
	// NodeID 允许省略：为空时由服务端根据 DocumentID 自动解析节点，避免前端重复传参
	NodeID     string `json:"node_id"`
	ParentID   string `json:"parent_id"`
	Content    string `json:"content" validate:"required"`
	MentionIDs string `json:"mention_ids"`
}

// UpdateCommentReq 更新评论请求
type UpdateCommentReq struct {
	Content string `json:"content" validate:"required"`
}

// ShareLinkReq 创建分享链接请求
type ShareLinkReq struct {
	DocumentID    string     `json:"document_id" validate:"required"`
	Password      string     `json:"password"`
	ExpireAt      *time.Time `json:"expire_at"`
	MaxViews      int        `json:"max_views"`
	AllowDownload bool       `json:"allow_download"`
}

// ShareLinkResp 分享链接响应
type ShareLinkResp struct {
	ID            string     `json:"id"`
	DocumentID    string     `json:"document_id"`
	NodeID        string     `json:"node_id"`
	Token         string     `json:"token"`
	Password      string     `json:"password,omitempty"`
	ExpireAt      *time.Time `json:"expire_at,omitempty"`
	MaxViews      int        `json:"max_views"`
	ViewCount     int        `json:"view_count"`
	AllowDownload bool       `json:"allow_download"`
	CreatedBy     string     `json:"created_by"`
	CreatedAt     time.Time  `json:"created_at"`
	ShareURL      string     `json:"share_url"`
}

// SharedDocumentResp 公开分享落地数据：包含分享元信息与文档只读内容
type SharedDocumentResp struct {
	DocumentID    string     `json:"document_id"`
	NodeID        string     `json:"node_id"`
	Title         string     `json:"title"`
	Content       string     `json:"content"`
	ContentHTML   string     `json:"content_html"`
	Format        string     `json:"format"`
	LastEditedBy  string     `json:"last_edited_by"`
	UpdatedAt     time.Time  `json:"updated_at"`
	AllowDownload bool       `json:"allow_download"`
	ViewCount     int        `json:"view_count"`
	MaxViews      int        `json:"max_views"`
	ExpireAt      *time.Time `json:"expire_at,omitempty"`
	NeedPassword  bool       `json:"need_password"`
}

// DocumentTemplateResp 文档模板响应
type DocumentTemplateResp struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Format      string    `json:"format"`
	Category    string    `json:"category"`
	IsPublic    bool      `json:"is_public"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateTemplateReq 创建文档模板请求
type CreateTemplateReq struct {
	Name        string `json:"name" validate:"required,max=200"`
	Description string `json:"description" validate:"max=500"`
	Content     string `json:"content" validate:"required"`
	Format      string `json:"format"`
	Category    string `json:"category" validate:"max=100"`
	IsPublic    bool   `json:"is_public"`
}

// UpdateTemplateReq 更新文档模板请求
type UpdateTemplateReq struct {
	Name        string `json:"name" validate:"omitempty,max=200"`
	Description string `json:"description" validate:"omitempty,max=500"`
	Content     string `json:"content"`
	Category    string `json:"category" validate:"omitempty,max=100"`
	IsPublic    bool   `json:"is_public"`
}

// DocumentAccessLogResp 文档访问日志响应
type DocumentAccessLogResp struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	UserID     string    `json:"user_id"`
	UserName   string    `json:"user_name"`
	Action     string    `json:"action"`
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
}

// DocumentStatsResp 文档访问统计响应
type DocumentStatsResp struct {
	TotalViews     int64     `json:"total_views"`
	TotalEdits     int64     `json:"total_edits"`
	TotalDownloads int64     `json:"total_downloads"`
	TotalShares    int64     `json:"total_shares"`
	UniqueViewers  int64     `json:"unique_viewers"`
	LastViewedAt   time.Time `json:"last_viewed_at"`
}

// SubscriptionResp 文档订阅响应
type SubscriptionResp struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	NodeID     string    `json:"node_id"`
	UserID     string    `json:"user_id"`
	NotifyType string    `json:"notify_type"`
	CreatedAt  time.Time `json:"created_at"`
}

// NotificationResp 通知响应
type NotificationResp struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	RelatedID   string    `json:"related_id"`
	RelatedType string    `json:"related_type"`
	IsRead      bool      `json:"is_read"`
	CreatedAt   time.Time `json:"created_at"`
}

// DiffResult 版本对比结果
type DiffResult struct {
	OldVersion int        `json:"old_version"`
	NewVersion int        `json:"new_version"`
	Diffs      []DiffLine `json:"diffs"`
}

// DiffLine 版本差异行
type DiffLine struct {
	Type    string `json:"type"` // added, removed, unchanged
	LineNum int    `json:"line_num"`
	Content string `json:"content"`
	OldLine int    `json:"old_line,omitempty"`
	NewLine int    `json:"new_line,omitempty"`
}

// BatchOperationReq 批量操作请求（通过 action 区分移动或删除）
type BatchOperationReq struct {
	NodeIDs []string `json:"node_ids" validate:"required"`
	Action  string   `json:"action" validate:"required,oneof=move delete"`
	Target  string   `json:"target,omitempty"`
}

// ExportDocumentReq 导出文档请求
type ExportDocumentReq struct {
	Format string `json:"format" validate:"required,oneof=markdown pdf html"`
}

// ImportDocumentReq 导入文档请求
type ImportDocumentReq struct {
	ParentID string `json:"parent_id"`
	Format   string `json:"format" validate:"required,oneof=markdown html"`
}

// EditLockResp 编辑锁响应
type EditLockResp struct {
	DocumentID string    `json:"document_id"`
	UserID     string    `json:"user_id"`
	UserName   string    `json:"user_name"`
	LockedAt   time.Time `json:"locked_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	CanEdit    bool      `json:"can_edit"`
}

// AcquireEditLockReq 获取编辑锁请求
type AcquireEditLockReq struct {
	DocumentID string `json:"document_id" validate:"required"`
}

// ReleaseEditLockReq 释放编辑锁请求
type ReleaseEditLockReq struct {
	DocumentID string `json:"document_id" validate:"required"`
}

// SubmitForReviewReq 提交审核请求
type SubmitForReviewReq struct {
	Comment string `json:"comment"`
}

// ReviewActionReq 审核操作请求（通过/驳回）
type ReviewActionReq struct {
	Comment string `json:"comment"`
}

// ReviewCommentResp 审核评论响应
type ReviewCommentResp struct {
	ID           string    `json:"id"`
	DocumentID   string    `json:"document_id"`
	NodeID       string    `json:"node_id"`
	Action       string    `json:"action"`
	Content      string    `json:"content"`
	ReviewerID   string    `json:"reviewer_id"`
	ReviewerName string    `json:"reviewer_name"`
	CreatedAt    time.Time `json:"created_at"`
}

// DocumentPublishStatusResp 文档发布状态响应
type DocumentPublishStatusResp struct {
	DocumentID    string     `json:"document_id"`
	NodeID        string     `json:"node_id"`
	Title         string     `json:"title"`
	PublishStatus string     `json:"publish_status"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	ReviewedBy    string     `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
}

// ListPendingReviewsResp 待审核列表项响应
type ListPendingReviewsResp struct {
	ID            string     `json:"id"`
	DocumentID    string     `json:"document_id"`
	NodeID        string     `json:"node_id"`
	Title         string     `json:"title"`
	SpaceID       string     `json:"space_id"`
	SpaceName     string     `json:"space_name"`
	PublishStatus string     `json:"publish_status"`
	SubmittedBy   string     `json:"submitted_by"`
	SubmittedAt   *time.Time `json:"submitted_at"`
}
