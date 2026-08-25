package repository

import (
	"context"
	"meteorx/internal/modules/notification/model"
	"time"

	"gorm.io/gorm"
)

// AnnouncementPO 公告数据库模型
type AnnouncementPO struct {
	ID             string         `gorm:"primaryKey;size:26;comment:公告ID"`
	Title          string         `gorm:"index;size:200;comment:公告标题"`
	Content        string         `gorm:"type:text;comment:公告内容"`
	Scope          string         `gorm:"index;size:20;comment:范围"`
	TargetTenantID string         `gorm:"index;size:26;comment:目标租户ID"`
	Status         int            `gorm:"index;comment:状态"`
	PublisherID    string         `gorm:"size:26;comment:发布人ID"`
	PublishAt      *time.Time     `gorm:"comment:发布时间"`
	ExpireAt       *time.Time     `gorm:"comment:过期时间"`
	CreatedAt      time.Time      `gorm:"autoCreateTime;comment:创建时间"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime;comment:更新时间"`
	DeletedAt      gorm.DeletedAt `gorm:"index;comment:删除时间"`
}

func (AnnouncementPO) TableName() string {
	return "announcements"
}

// toDomain 转换为领域模型
func (po AnnouncementPO) toDomain() *model.Announcement {
	return &model.Announcement{
		ID:             po.ID,
		Title:          po.Title,
		Content:        po.Content,
		Scope:          po.Scope,
		TargetTenantID: po.TargetTenantID,
		Status:         po.Status,
		PublisherID:    po.PublisherID,
		PublishAt:      po.PublishAt,
		ExpireAt:       po.ExpireAt,
		CreatedAt:      po.CreatedAt,
		UpdatedAt:      po.UpdatedAt,
	}
}

// fromDomain 从领域模型转换
func announcementFromDomain(a *model.Announcement) *AnnouncementPO {
	return &AnnouncementPO{
		ID:             a.ID,
		Title:          a.Title,
		Content:        a.Content,
		Scope:          a.Scope,
		TargetTenantID: a.TargetTenantID,
		Status:         a.Status,
		PublisherID:    a.PublisherID,
		PublishAt:      a.PublishAt,
		ExpireAt:       a.ExpireAt,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}

// announcementRepository 公告仓库实现
type announcementRepository struct {
	db *gorm.DB
}

// NewAnnouncementRepository 创建公告仓库
func NewAnnouncementRepository(db *gorm.DB) AnnouncementRepository {
	return &announcementRepository{db: db}
}

// AutoMigrate 自动迁移表结构
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&AnnouncementPO{})
}

func (r *announcementRepository) Create(ctx context.Context, a *model.Announcement) error {
	po := announcementFromDomain(a)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *announcementRepository) Update(ctx context.Context, a *model.Announcement) error {
	po := announcementFromDomain(a)
	return r.db.WithContext(ctx).Model(&AnnouncementPO{}).
		Where("id = ?", a.ID).
		Select("title", "content", "scope", "target_tenant_id", "status", "publish_at", "expire_at").
		Updates(po).Error
}

func (r *announcementRepository) GetByID(ctx context.Context, id string) (*model.Announcement, error) {
	var po AnnouncementPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *announcementRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&AnnouncementPO{}, "id = ?", id).Error
}

func (r *announcementRepository) List(ctx context.Context, query *AnnouncementQuery) ([]*model.Announcement, int64, error) {
	db := r.db.WithContext(ctx).Model(&AnnouncementPO{})

	if query.Keyword != "" {
		db = db.Where("title LIKE ?", "%"+query.Keyword+"%")
	}
	// Status: -1 表示全部，其余按状态过滤
	if query.Status >= 0 {
		db = db.Where("status = ?", query.Status)
	}
	if query.Scope != "" {
		db = db.Where("scope = ?", query.Scope)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var pos []AnnouncementPO
	if err := db.Order("created_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*model.Announcement, len(pos))
	for i, po := range pos {
		items[i] = po.toDomain()
	}
	return items, total, nil
}

func (r *announcementRepository) ListForTenant(ctx context.Context, tenantID string, page, pageSize int) ([]*model.Announcement, int64, error) {
	now := time.Now()
	db := r.db.WithContext(ctx).Model(&AnnouncementPO{}).
		Where("status = ?", model.AnnouncementStatusPublished).
		Where("(scope = ? AND target_tenant_id = ?) OR scope = ?", model.AnnouncementScopeTenant, tenantID, model.AnnouncementScopeAll).
		Where("expire_at IS NULL OR expire_at > ?", now)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var pos []AnnouncementPO
	if err := db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	items := make([]*model.Announcement, len(pos))
	for i, po := range pos {
		items[i] = po.toDomain()
	}
	return items, total, nil
}