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
