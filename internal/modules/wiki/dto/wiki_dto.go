package dto

import "time"

type CreateWikiSpaceReq struct {
	Name        string `json:"name" validate:"required,min=2,max=200"`
	Description string `json:"description,omitempty" validate:"max=500"`
	Icon        string `json:"icon,omitempty" validate:"max=255"`
	Visibility  int    `json:"visibility" validate:"oneof=1 2 3"`
}

type UpdateWikiSpaceReq struct {
	Name        string `json:"name,omitempty" validate:"omitempty,min=2,max=200"`
	Description string `json:"description,omitempty" validate:"omitempty,max=500"`
	Icon        string `json:"icon,omitempty" validate:"omitempty,max=255"`
	Visibility  int    `json:"visibility,omitempty" validate:"omitempty,oneof=1 2 3"`
}

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

type CreateWikiNodeReq struct {
	SpaceID  string `json:"space_id" validate:"required"`
	ParentID string `json:"parent_id,omitempty"`
	Type     string `json:"type" validate:"required,oneof=folder document"`
	Title    string `json:"title" validate:"required,min=1,max=500"`
	Icon     string `json:"icon,omitempty"`
	Sort     int    `json:"sort,omitempty"`
	Content  string `json:"content,omitempty"`
}

type UpdateWikiNodeReq struct {
	Title    string `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Icon     string `json:"icon,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
	Sort     int    `json:"sort,omitempty"`
	Status   int    `json:"status,omitempty" validate:"omitempty,oneof=1 2"`
}

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

type WikiNodeTreeResp struct {
	WikiNodeResp
	Children []*WikiNodeTreeResp `json:"children,omitempty"`
}

type CreateDocumentReq struct {
	NodeID  string `json:"node_id" validate:"required"`
	Content string `json:"content,omitempty"`
	Format  string `json:"format,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type UpdateDocumentReq struct {
	Content string `json:"content,omitempty"`
	Format  string `json:"format,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type DocumentResp struct {
	ID           string     `json:"id"`
	NodeID       string     `json:"node_id"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	ContentHTML  string     `json:"content_html"`
	Format       string     `json:"format"`
	CurrentVer   int        `json:"current_ver"`
	ViewCount    int64      `json:"view_count"`
	LastEditedBy string     `json:"last_edited_by"`
	LastEditedAt *time.Time `json:"last_edited_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

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

type WikiSpaceMemberReq struct {
	UserID string `json:"user_id" validate:"required"`
	Role   string `json:"role" validate:"required,oneof=owner admin editor viewer"`
}

type WikiSpaceMemberResp struct {
	ID        string    `json:"id"`
	SpaceID   string    `json:"space_id"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	UserEmail string    `json:"user_email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type SetNodePermissionReq struct {
	UserID     string `json:"user_id" validate:"required"`
	Permission string `json:"permission" validate:"required,oneof=view edit delete"`
}

type NodePermissionResp struct {
	ID         string    `json:"id"`
	NodeID     string    `json:"node_id"`
	UserID     string    `json:"user_id"`
	Permission string    `json:"permission"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type WikiStatsResp struct {
	TotalSpaces    int64 `json:"total_spaces"`
	TotalNodes     int64 `json:"total_nodes"`
	TotalDocuments int64 `json:"total_documents"`
	TotalViews     int64 `json:"total_views"`
}

// Trash DTOs

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

type TrashListReq struct {
	SpaceID  string `json:"space_id"`
	ItemType string `json:"item_type"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type TrashRestoreReq struct {
	RestoreChildren bool `json:"restore_children"`
}

// Search DTOs

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

// Attachment DTOs

type CreateAttachmentReq struct {
	DocumentID string `json:"document_id" validate:"required"`
	FileID     string `json:"file_id,omitempty"`
	FileName   string `json:"file_name" validate:"required"`
	FileSize   int64  `json:"file_size"`
	MimeType   string `json:"mime_type"`
	FileURL    string `json:"file_url" validate:"required"`
}

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

type TagResp struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTagReq struct {
	Name  string `json:"name" validate:"required,max=100"`
	Color string `json:"color" validate:"max=20"`
}

type DocumentTagResp struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	Tag        TagResp   `json:"tag"`
	CreatedAt  time.Time `json:"created_at"`
}

type CommentResp struct {
	ID         string      `json:"id"`
	DocumentID string      `json:"document_id"`
	NodeID     string      `json:"node_id"`
	ParentID   string      `json:"parent_id"`
	Content    string      `json:"content"`
	CreatedBy  string      `json:"created_by"`
	UserName   string      `json:"user_name"`
	MentionIDs string      `json:"mention_ids"`
	Status     int         `json:"status"`
	Replies    []CommentResp `json:"replies,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

type CreateCommentReq struct {
	DocumentID string `json:"document_id" validate:"required"`
	// NodeID 允许省略：为空时由服务端根据 DocumentID 自动解析节点，避免前端重复传参
	NodeID     string `json:"node_id"`
	ParentID   string `json:"parent_id"`
	Content    string `json:"content" validate:"required"`
	MentionIDs string `json:"mention_ids"`
}

type UpdateCommentReq struct {
	Content string `json:"content" validate:"required"`
}

type ShareLinkReq struct {
	DocumentID    string     `json:"document_id" validate:"required"`
	Password      string     `json:"password"`
	ExpireAt      *time.Time `json:"expire_at"`
	MaxViews      int        `json:"max_views"`
	AllowDownload bool       `json:"allow_download"`
}

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

type CreateTemplateReq struct {
	Name        string `json:"name" validate:"required,max=200"`
	Description string `json:"description" validate:"max=500"`
	Content     string `json:"content" validate:"required"`
	Format      string `json:"format"`
	Category    string `json:"category" validate:"max=100"`
	IsPublic    bool   `json:"is_public"`
}

type UpdateTemplateReq struct {
	Name        string `json:"name" validate:"omitempty,max=200"`
	Description string `json:"description" validate:"omitempty,max=500"`
	Content     string `json:"content"`
	Category    string `json:"category" validate:"omitempty,max=100"`
	IsPublic    bool   `json:"is_public"`
}

type DocumentAccessLogResp struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	UserID     string    `json:"user_id"`
	UserName   string    `json:"user_name"`
	Action     string    `json:"action"`
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
}

type DocumentStatsResp struct {
	TotalViews     int64     `json:"total_views"`
	TotalEdits     int64     `json:"total_edits"`
	TotalDownloads int64     `json:"total_downloads"`
	TotalShares    int64     `json:"total_shares"`
	UniqueViewers  int64     `json:"unique_viewers"`
	LastViewedAt   time.Time `json:"last_viewed_at"`
}

type SubscriptionResp struct {
	ID         string    `json:"id"`
	DocumentID string    `json:"document_id"`
	NodeID     string    `json:"node_id"`
	UserID     string    `json:"user_id"`
	NotifyType string    `json:"notify_type"`
	CreatedAt  time.Time `json:"created_at"`
}

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

type DiffResult struct {
	OldVersion int        `json:"old_version"`
	NewVersion int        `json:"new_version"`
	Diffs      []DiffLine `json:"diffs"`
}

type DiffLine struct {
	Type     string `json:"type"` // added, removed, unchanged
	LineNum  int    `json:"line_num"`
	Content  string `json:"content"`
	OldLine  int    `json:"old_line,omitempty"`
	NewLine  int    `json:"new_line,omitempty"`
}

type BatchOperationReq struct {
	NodeIDs []string `json:"node_ids" validate:"required"`
	Action  string   `json:"action" validate:"required,oneof=move delete"`
	Target  string   `json:"target,omitempty"`
}

type ExportDocumentReq struct {
	Format string `json:"format" validate:"required,oneof=markdown pdf html"`
}

type ImportDocumentReq struct {
	ParentID string `json:"parent_id"`
	Format   string `json:"format" validate:"required,oneof=markdown html"`
}

type EditLockResp struct {
	DocumentID string    `json:"document_id"`
	UserID     string    `json:"user_id"`
	UserName   string    `json:"user_name"`
	LockedAt   time.Time `json:"locked_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	CanEdit    bool      `json:"can_edit"`
}

type AcquireEditLockReq struct {
	DocumentID string `json:"document_id" validate:"required"`
}

type ReleaseEditLockReq struct {
	DocumentID string `json:"document_id" validate:"required"`
}