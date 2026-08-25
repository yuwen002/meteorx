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