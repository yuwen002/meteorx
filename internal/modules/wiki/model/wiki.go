package model

import "time"

type WikiSpace struct {
	ID          string    `gorm:"primaryKey"`
	TenantID    string    `gorm:"index;size:26;not null"`
	Name        string    `gorm:"not null;size:200"`
	Description string    `gorm:"size:500"`
	Icon        string    `gorm:"size:255"`
	Visibility  int       `gorm:"not null;default:1"`
	CreatedBy   string    `gorm:"index;size:26"`
	CreatedAt   time.Time `gorm:"index"`
	UpdatedAt   time.Time
	DeletedAt   *time.Time `gorm:"index"`
}

const (
	VisibilityPrivate = 1
	VisibilityTenant  = 2
	VisibilityPublic  = 3
)

func (WikiSpace) TableName() string {
	return "wiki_spaces"
}

type WikiNode struct {
	ID        string    `gorm:"primaryKey"`
	TenantID  string    `gorm:"index;size:26;not null"`
	SpaceID   string    `gorm:"index;size:26;not null"`
	ParentID  string    `gorm:"index;size:26;default:''"`
	Type      string    `gorm:"not null;size:20"`
	Title     string    `gorm:"not null;size:500"`
	Icon      string    `gorm:"size:255"`
	Sort      int       `gorm:"default:0"`
	OwnerID   string    `gorm:"index;size:26"`
	Status    int       `gorm:"not null;default:1"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

const (
	NodeTypeFolder   = "folder"
	NodeTypeDocument = "document"

	NodeStatusActive   = 1
	NodeStatusArchived = 2
)

func (WikiNode) TableName() string {
	return "wiki_nodes"
}

type Document struct {
	ID           string `gorm:"primaryKey"`
	TenantID     string `gorm:"index;size:26;not null"`
	NodeID       string `gorm:"size:26;uniqueIndex;not null"`
	Content      string `gorm:"type:mediumtext"`
	ContentHTML  string `gorm:"type:mediumtext"`
	Format       string `gorm:"not null;default:'markdown';size:50"`
	CurrentVer   int    `gorm:"not null;default:1"`
	ViewCount    int64  `gorm:"not null;default:0"`
	LastEditedBy string `gorm:"index;size:26"`
	LastEditedAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

func (Document) TableName() string {
	return "wiki_documents"
}

type DocumentRevision struct {
	ID          string    `gorm:"primaryKey"`
	TenantID    string    `gorm:"index;size:26;not null"`
	DocumentID  string    `gorm:"index;size:26;not null"`
	Version     int       `gorm:"not null"`
	Content     string    `gorm:"type:mediumtext"`
	ContentHTML string    `gorm:"type:mediumtext"`
	Summary     string    `gorm:"size:500"`
	EditedBy    string    `gorm:"index;size:26"`
	CreatedAt   time.Time `gorm:"index"`
}

func (DocumentRevision) TableName() string {
	return "wiki_document_revisions"
}

type WikiSpaceMember struct {
	ID        string `gorm:"primaryKey"`
	TenantID  string `gorm:"index;size:26;not null"`
	SpaceID   string `gorm:"index;size:26;not null"`
	UserID    string `gorm:"index;size:26;not null"`
	Role      string `gorm:"not null;size:50"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

const (
	SpaceRoleOwner  = "owner"
	SpaceRoleAdmin  = "admin"
	SpaceRoleEditor = "editor"
	SpaceRoleViewer = "viewer"
)

func (WikiSpaceMember) TableName() string {
	return "wiki_space_members"
}

type WikiNodePermission struct {
	ID         string `gorm:"primaryKey"`
	NodeID     string `gorm:"index;size:26;not null"`
	UserID     string `gorm:"index;size:26;not null"`
	Permission string `gorm:"not null;size:20"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

const (
	NodePermView   = "view"
	NodePermEdit   = "edit"
	NodePermDelete = "delete"
)

func (WikiNodePermission) TableName() string {
	return "wiki_node_permissions"
}

type WikiStats struct {
	TotalSpaces    int64 `gorm:"-"`
	TotalNodes     int64 `gorm:"-"`
	TotalDocuments int64 `gorm:"-"`
	TotalViews     int64 `gorm:"-"`
}

type TrashItem struct {
	ID        string    `gorm:"primaryKey"`
	TenantID  string    `gorm:"index;size:26;not null"`
	ItemType  string    `gorm:"index;size:50;not null"`
	ItemID    string    `gorm:"index;size:26;not null"`
	SpaceID   string    `gorm:"index;size:26"`
	Title     string    `gorm:"size:500"`
	DeletedBy string    `gorm:"index;size:26"`
	DeletedAt time.Time `gorm:"index"`
	ExpiresAt time.Time `gorm:"index"`
}

const (
	TrashTypeSpace    = "space"
	TrashTypeNode     = "node"
	TrashTypeDocument = "document"
)

func (TrashItem) TableName() string {
	return "wiki_trash"
}

type Attachment struct {
	ID         string    `gorm:"primaryKey"`
	TenantID   string    `gorm:"index;size:26;not null"`
	DocumentID string    `gorm:"index;size:26;not null"`
	FileID     string    `gorm:"index;size:26;not null"`
	FileName   string    `gorm:"size:500;not null"`
	FileSize   int64     `gorm:""`
	MimeType   string    `gorm:"size:100"`
	FileURL    string    `gorm:"size:1000"`
	UploadedBy string    `gorm:"index;size:26"`
	CreatedAt  time.Time `gorm:"index"`
}

func (Attachment) TableName() string {
	return "wiki_attachments"
}

type Tag struct {
	ID        string    `gorm:"primaryKey"`
	TenantID  string    `gorm:"index;size:26;not null"`
	Name      string    `gorm:"index;size:100;not null"`
	Color     string    `gorm:"size:20"`
	CreatedBy string    `gorm:"index;size:26"`
	CreatedAt time.Time `gorm:"index"`
}

func (Tag) TableName() string {
	return "wiki_tags"
}

type DocumentTag struct {
	ID         string `gorm:"primaryKey"`
	TenantID   string `gorm:"index;size:26;not null"`
	DocumentID string `gorm:"index;size:26;not null"`
	TagID      string `gorm:"index;size:26;not null"`
	CreatedAt  time.Time
	
	// 关联字段
	Tag *Tag `gorm:"foreignKey:TagID;references:ID"`
}

func (DocumentTag) TableName() string {
	return "wiki_document_tags"
}

type Comment struct {
	ID         string    `gorm:"primaryKey"`
	TenantID   string    `gorm:"index;size:26;not null"`
	DocumentID string    `gorm:"index;size:26;not null"`
	NodeID     string    `gorm:"index;size:26;not null"`
	ParentID   string    `gorm:"index;size:26;default:''"`
	Content    string    `gorm:"type:text;not null"`
	CreatedBy  string    `gorm:"index;size:26;not null"`
	MentionIDs string    `gorm:"type:text"`
	Status     int       `gorm:"default:1"`
	CreatedAt  time.Time `gorm:"index"`
	UpdatedAt  time.Time
	DeletedAt  *time.Time `gorm:"index"`
}

const (
	CommentStatusActive   = 1
	CommentStatusResolved = 2
	CommentStatusDeleted  = 3
)

func (Comment) TableName() string {
	return "wiki_comments"
}

type ShareLink struct {
	ID           string     `gorm:"primaryKey"`
	TenantID     string     `gorm:"index;size:26;not null"`
	DocumentID   string     `gorm:"index;size:26;not null"`
	NodeID       string     `gorm:"index;size:26;not null"`
	Token        string     `gorm:"uniqueIndex;size:64;not null"`
	Password     string     `gorm:"size:100"`
	ExpireAt     *time.Time `gorm:"index"`
	MaxViews     int        `gorm:"default:0"`
	ViewCount    int        `gorm:"default:0"`
	AllowDownload bool      `gorm:"default:false"`
	CreatedBy    string     `gorm:"index;size:26;not null"`
	CreatedAt    time.Time  `gorm:"index"`
}

func (ShareLink) TableName() string {
	return "wiki_share_links"
}

type DocumentTemplate struct {
	ID          string    `gorm:"primaryKey"`
	TenantID    string    `gorm:"index;size:26;not null"`
	Name        string    `gorm:"size:200;not null"`
	Description string    `gorm:"size:500"`
	Content     string    `gorm:"type:mediumtext"`
	Format      string    `gorm:"size:50;default:markdown"`
	Category    string    `gorm:"size:100"`
	IsPublic    bool      `gorm:"default:false"`
	CreatedBy   string    `gorm:"index;size:26;not null"`
	CreatedAt   time.Time `gorm:"index"`
	UpdatedAt   time.Time
}

func (DocumentTemplate) TableName() string {
	return "wiki_templates"
}

type DocumentAccessLog struct {
	ID        string    `gorm:"primaryKey"`
	TenantID  string    `gorm:"index;size:26;not null"`
	DocumentID string   `gorm:"index;size:26;not null"`
	NodeID    string    `gorm:"index;size:26;not null"`
	UserID    string    `gorm:"index;size:26"`
	Action    string    `gorm:"size:50;not null"`
	IPAddress string    `gorm:"size:50"`
	UserAgent string    `gorm:"size:500"`
	CreatedAt time.Time `gorm:"index"`
}

const (
	ActionView    = "view"
	ActionEdit    = "edit"
	ActionDownload = "download"
	ActionShare   = "share"
)

func (DocumentAccessLog) TableName() string {
	return "wiki_access_logs"
}

type DocumentSubscription struct {
	ID         string    `gorm:"primaryKey"`
	TenantID   string    `gorm:"index;size:26;not null"`
	DocumentID string    `gorm:"index;size:26;not null"`
	NodeID     string    `gorm:"index;size:26;not null"`
	UserID     string    `gorm:"index;size:26;not null"`
	NotifyType string    `gorm:"size:50;default:all"`
	CreatedAt  time.Time `gorm:"index"`
}

const (
	NotifyTypeAll     = "all"
	NotifyTypeEdit    = "edit"
	NotifyTypeComment = "comment"
)

func (DocumentSubscription) TableName() string {
	return "wiki_subscriptions"
}

type Notification struct {
	ID         string    `gorm:"primaryKey"`
	TenantID   string    `gorm:"index;size:26;not null"`
	UserID     string    `gorm:"index;size:26;not null"`
	Type       string    `gorm:"size:50;not null"`
	Title      string    `gorm:"size:500;not null"`
	Content    string    `gorm:"type:text"`
	RelatedID  string    `gorm:"size:26"`
	RelatedType string   `gorm:"size:50"`
	IsRead     bool      `gorm:"default:false"`
	CreatedAt  time.Time `gorm:"index"`
}

const (
	NotificationTypeEdit    = "document_edit"
	NotificationTypeComment = "comment"
	NotificationTypeMention = "mention"
	NotificationTypeShare   = "share"
)

func (Notification) TableName() string {
	return "wiki_notifications"
}

type EditLock struct {
	ID         string    `gorm:"primaryKey"`
	TenantID   string    `gorm:"index;size:26;not null"`
	DocumentID string    `gorm:"uniqueIndex;size:26;not null"`
	UserID     string    `gorm:"size:26;not null"`
	LockedAt   time.Time `gorm:"index"`
	ExpiresAt  time.Time `gorm:"index"`
}

func (EditLock) TableName() string {
	return "wiki_edit_locks"
}

type DocumentStats struct {
	TotalViews     int64     `json:"total_views"`
	TotalEdits     int64     `json:"total_edits"`
	TotalDownloads int64     `json:"total_downloads"`
	TotalShares    int64     `json:"total_shares"`
	UniqueViewers  int64     `json:"unique_viewers"`
	LastViewedAt   time.Time `json:"last_viewed_at"`
}