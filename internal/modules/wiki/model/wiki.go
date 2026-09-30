// Package model 定义 Wiki 模块的领域模型，包括空间、节点、文档、版本和权限实体。
package model

import "time"

// WikiSpace Wiki 知识空间领域模型，是文档树的顶层容器。
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

// TableName 返回知识空间表名 wiki_spaces。
func (WikiSpace) TableName() string {
	return "wiki_spaces"
}

// WikiNode 空间内的树节点，可为目录或文档，支持父子层级与排序。
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

// TableName 返回节点表名 wiki_nodes。
func (WikiNode) TableName() string {
	return "wiki_nodes"
}

// Document 文档正文领域模型，关联节点并维护版本、发布状态与浏览量。
type Document struct {
	ID            string `gorm:"primaryKey"`
	TenantID      string `gorm:"index;size:26;not null"`
	NodeID        string `gorm:"size:26;uniqueIndex;not null"`
	Content       string `gorm:"type:mediumtext"`
	ContentHTML   string `gorm:"type:mediumtext"`
	Format        string `gorm:"not null;default:'markdown';size:50"`
	CurrentVer    int    `gorm:"not null;default:1"`
	ViewCount     int64  `gorm:"not null;default:0"`
	LastEditedBy  string `gorm:"index;size:26"`
	LastEditedAt  *time.Time
	PublishStatus string     `gorm:"not null;default:'draft';size:30;index"`
	PublishedAt   *time.Time `gorm:"index"`
	ReviewedBy    string     `gorm:"size:26"`
	ReviewedAt    *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

const (
	PublishStatusDraft         = "draft"
	PublishStatusPendingReview = "pending_review"
	PublishStatusPublished     = "published"
	PublishStatusArchived      = "archived"
	PublishStatusRejected      = "rejected"
)

// TableName 返回文档表名 wiki_documents。
func (Document) TableName() string {
	return "wiki_documents"
}

// DocumentRevision 文档历史版本快照，用于版本回溯与对比。
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

// TableName 返回文档版本表名 wiki_document_revisions。
func (DocumentRevision) TableName() string {
	return "wiki_document_revisions"
}

// WikiSpaceMember 知识空间成员及其角色关系。
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

// TableName 返回空间成员表名 wiki_space_members。
func (WikiSpaceMember) TableName() string {
	return "wiki_space_members"
}

// WikiNodePermission 节点粒度的用户权限授权。
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

// TableName 返回节点权限表名 wiki_node_permissions。
func (WikiNodePermission) TableName() string {
	return "wiki_node_permissions"
}

// WikiStats Wiki 模块租户级统计概览（非持久化字段）。
type WikiStats struct {
	TotalSpaces    int64 `gorm:"-"`
	TotalNodes     int64 `gorm:"-"`
	TotalDocuments int64 `gorm:"-"`
	TotalViews     int64 `gorm:"-"`
}

// TrashItem 回收站条目，记录被删除实体的类型、删除人与过期时间。
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

// TableName 返回回收站表名 wiki_trash。
func (TrashItem) TableName() string {
	return "wiki_trash"
}

// Attachment 文档附件，关联文件模块的文件记录。
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

// TableName 返回附件表名 wiki_attachments。
func (Attachment) TableName() string {
	return "wiki_attachments"
}

// Tag 文档标签定义。
type Tag struct {
	ID        string    `gorm:"primaryKey"`
	TenantID  string    `gorm:"index;size:26;not null"`
	Name      string    `gorm:"index;size:100;not null"`
	Color     string    `gorm:"size:20"`
	CreatedBy string    `gorm:"index;size:26"`
	CreatedAt time.Time `gorm:"index"`
}

// TableName 返回标签表名 wiki_tags。
func (Tag) TableName() string {
	return "wiki_tags"
}

// DocumentTag 文档与标签的多对多关联。
type DocumentTag struct {
	ID         string `gorm:"primaryKey"`
	TenantID   string `gorm:"index;size:26;not null"`
	DocumentID string `gorm:"index;size:26;not null"`
	TagID      string `gorm:"index;size:26;not null"`
	CreatedAt  time.Time

	// 关联字段
	Tag *Tag `gorm:"foreignKey:TagID;references:ID"`
}

// TableName 返回文档标签关联表名 wiki_document_tags。
func (DocumentTag) TableName() string {
	return "wiki_document_tags"
}

// Comment 文档评论，支持父子回复与软删除。
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

// TableName 返回评论表名 wiki_comments。
func (Comment) TableName() string {
	return "wiki_comments"
}

// ShareLink 文档分享链接，支持密码、过期与浏览次数限制。
type ShareLink struct {
	ID            string     `gorm:"primaryKey"`
	TenantID      string     `gorm:"index;size:26;not null"`
	DocumentID    string     `gorm:"index;size:26;not null"`
	NodeID        string     `gorm:"index;size:26;not null"`
	Token         string     `gorm:"uniqueIndex;size:64;not null"`
	Password      string     `gorm:"size:100"`
	ExpireAt      *time.Time `gorm:"index"`
	MaxViews      int        `gorm:"default:0"`
	ViewCount     int        `gorm:"default:0"`
	AllowDownload bool       `gorm:"default:false"`
	CreatedBy     string     `gorm:"index;size:26;not null"`
	CreatedAt     time.Time  `gorm:"index"`
}

// TableName 返回分享链接表名 wiki_share_links。
func (ShareLink) TableName() string {
	return "wiki_share_links"
}

// DocumentTemplate 文档模板，可为租户私有或公共。
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

// TableName 返回文档模板表名 wiki_templates。
func (DocumentTemplate) TableName() string {
	return "wiki_templates"
}

// DocumentAccessLog 文档访问行为日志，用于统计分析。
type DocumentAccessLog struct {
	ID         string    `gorm:"primaryKey"`
	TenantID   string    `gorm:"index;size:26;not null"`
	DocumentID string    `gorm:"index;size:26;not null"`
	NodeID     string    `gorm:"index;size:26;not null"`
	UserID     string    `gorm:"index;size:26"`
	Action     string    `gorm:"size:50;not null"`
	IPAddress  string    `gorm:"size:50"`
	UserAgent  string    `gorm:"size:500"`
	CreatedAt  time.Time `gorm:"index"`
}

const (
	ActionView     = "view"
	ActionEdit     = "edit"
	ActionDownload = "download"
	ActionShare    = "share"
)

// TableName 返回访问日志表名 wiki_access_logs。
func (DocumentAccessLog) TableName() string {
	return "wiki_access_logs"
}

// DocumentSubscription 用户对文档的订阅，按类型接收变更通知。
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

// TableName 返回文档订阅表名 wiki_subscriptions。
func (DocumentSubscription) TableName() string {
	return "wiki_subscriptions"
}

// Notification 站内通知消息。
type Notification struct {
	ID          string    `gorm:"primaryKey"`
	TenantID    string    `gorm:"index;size:26;not null"`
	UserID      string    `gorm:"index;size:26;not null"`
	Type        string    `gorm:"size:50;not null"`
	Title       string    `gorm:"size:500;not null"`
	Content     string    `gorm:"type:text"`
	RelatedID   string    `gorm:"size:26"`
	RelatedType string    `gorm:"size:50"`
	IsRead      bool      `gorm:"default:false"`
	CreatedAt   time.Time `gorm:"index"`
}

const (
	NotificationTypeEdit    = "document_edit"
	NotificationTypeComment = "comment"
	NotificationTypeMention = "mention"
	NotificationTypeShare   = "share"
)

// TableName 返回通知表名 wiki_notifications。
func (Notification) TableName() string {
	return "wiki_notifications"
}

// EditLock 文档编辑锁，防止多人同时编辑，默认 30 分钟过期。
type EditLock struct {
	ID         string    `gorm:"primaryKey"`
	TenantID   string    `gorm:"index;size:26;not null"`
	DocumentID string    `gorm:"uniqueIndex;size:26;not null"`
	UserID     string    `gorm:"size:26;not null"`
	LockedAt   time.Time `gorm:"index"`
	ExpiresAt  time.Time `gorm:"index"`
}

// TableName 返回编辑锁表名 wiki_edit_locks。
func (EditLock) TableName() string {
	return "wiki_edit_locks"
}

// DocumentStats 文档统计概览（非持久化，由访问日志聚合而来）。
type DocumentStats struct {
	TotalViews     int64     `json:"total_views"`
	TotalEdits     int64     `json:"total_edits"`
	TotalDownloads int64     `json:"total_downloads"`
	TotalShares    int64     `json:"total_shares"`
	UniqueViewers  int64     `json:"unique_viewers"`
	LastViewedAt   time.Time `json:"last_viewed_at"`
}

// ReviewComment 文档发布评审记录。
type ReviewComment struct {
	ID         string    `gorm:"primaryKey"`
	TenantID   string    `gorm:"index;size:26;not null"`
	DocumentID string    `gorm:"index;size:26;not null"`
	NodeID     string    `gorm:"index;size:26;not null"`
	Action     string    `gorm:"not null;size:30"`
	Content    string    `gorm:"type:text"`
	ReviewerID string    `gorm:"index;size:26;not null"`
	CreatedAt  time.Time `gorm:"index"`
}

const (
	ReviewActionSubmit  = "submit"
	ReviewActionApprove = "approve"
	ReviewActionReject  = "reject"
	ReviewActionPublish = "publish"
	ReviewActionArchive = "archive"
)

// TableName 返回评审记录表名 wiki_review_comments。
func (ReviewComment) TableName() string {
	return "wiki_review_comments"
}
