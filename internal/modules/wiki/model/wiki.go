package model

import "time"

type WikiSpace struct {
	ID          string     `gorm:"primaryKey"`
	TenantID    string     `gorm:"index;not null"`
	Name        string     `gorm:"not null;size:200"`
	Description string     `gorm:"size:500"`
	Icon        string     `gorm:"size:255"`
	Visibility  int        `gorm:"not null;default:1"`
	CreatedBy   string     `gorm:"index"`
	CreatedAt   time.Time  `gorm:"index"`
	UpdatedAt   time.Time
	DeletedAt   *time.Time `gorm:"index"`
}

const (
	VisibilityPrivate  = 1
	VisibilityTenant   = 2
	VisibilityPublic   = 3
)

func (WikiSpace) TableName() string {
	return "wiki_spaces"
}

type WikiNode struct {
	ID         string     `gorm:"primaryKey"`
	SpaceID    string     `gorm:"index;not null"`
	ParentID   string     `gorm:"index;default:''"`
	Type       string     `gorm:"not null;size:20"`
	Title      string     `gorm:"not null;size:500"`
	Icon       string     `gorm:"size:255"`
	Sort       int        `gorm:"default:0"`
	OwnerID    string     `gorm:"index"`
	Status     int        `gorm:"not null;default:1"`
	CreatedAt  time.Time  `gorm:"index"`
	UpdatedAt  time.Time
	DeletedAt  *time.Time `gorm:"index"`
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
	ID            string     `gorm:"primaryKey"`
	NodeID        string     `gorm:"uniqueIndex;not null"`
	Content       string     `gorm:"type:mediumtext"`
	ContentHTML   string     `gorm:"type:mediumtext"`
	Format        string     `gorm:"not null;default:'markdown';size:50"`
	CurrentVer    int        `gorm:"not null;default:1"`
	ViewCount     int64      `gorm:"not null;default:0"`
	LastEditedBy  string     `gorm:"index"`
	LastEditedAt  *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

func (Document) TableName() string {
	return "wiki_documents"
}

type DocumentRevision struct {
	ID          string    `gorm:"primaryKey"`
	DocumentID  string    `gorm:"index;not null"`
	Version     int       `gorm:"not null"`
	Content     string    `gorm:"type:mediumtext"`
	ContentHTML string    `gorm:"type:mediumtext"`
	Summary     string    `gorm:"size:500"`
	EditedBy    string    `gorm:"index"`
	CreatedAt   time.Time `gorm:"index"`
}

func (DocumentRevision) TableName() string {
	return "wiki_document_revisions"
}

type WikiSpaceMember struct {
	ID        string    `gorm:"primaryKey"`
	SpaceID   string    `gorm:"index;not null"`
	UserID    string    `gorm:"index;not null"`
	Role      string    `gorm:"not null;size:50"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

const (
	SpaceRoleOwner   = "owner"
	SpaceRoleAdmin   = "admin"
	SpaceRoleEditor  = "editor"
	SpaceRoleViewer  = "viewer"
)

func (WikiSpaceMember) TableName() string {
	return "wiki_space_members"
}

type WikiNodePermission struct {
	ID         string    `gorm:"primaryKey"`
	NodeID     string    `gorm:"index;not null"`
	UserID     string    `gorm:"index;not null"`
	Permission string    `gorm:"not null;size:20"`
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