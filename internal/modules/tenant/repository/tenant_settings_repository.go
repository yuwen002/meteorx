package repository

import (
	"context"
	"errors"
	"time"

	"meteorx/internal/modules/tenant/model"

	"gorm.io/gorm"
)

// TenantSettingsPO 租户设置持久化对象，映射 tenant_settings 表（每个租户一条）。
type TenantSettingsPO struct {
	ID           string    `gorm:"primaryKey;size:26;comment:'设置ID'"`
	TenantID     string    `gorm:"size:26;uniqueIndex;comment:'租户ID，唯一'"`
	Logo         string    `gorm:"size:500;comment:'租户Logo'"`
	Favicon      string    `gorm:"size:500;comment:'Favicon图标'"`
	PrimaryColor string    `gorm:"size:20;comment:'主题主色'"`
	Theme        string    `gorm:"size:20;default:light;comment:'主题：light/dark'"`
	Language     string    `gorm:"size:10;default:zh-CN;comment:'默认语言'"`
	Timezone     string    `gorm:"size:50;default:Asia/Shanghai;comment:'时区'"`
	Description  string    `gorm:"size:500;comment:'租户简介'"`
	WelcomeText  string    `gorm:"size:500;comment:'欢迎语'"`
	ContactName  string    `gorm:"size:50;comment:'联系人姓名'"`
	ContactEmail string    `gorm:"size:100;comment:'联系人邮箱'"`
	ContactPhone string    `gorm:"size:20;comment:'联系人电话'"`
	Address      string    `gorm:"size:255;comment:'地址'"`
	Extra        string    `gorm:"type:text;comment:'扩展字段JSON'"`
	CreatedAt    time.Time `gorm:"comment:'创建时间'"`
	UpdatedAt    time.Time `gorm:"comment:'更新时间'"`
}

// TableName 返回租户设置表名 tenant_settings。
func (TenantSettingsPO) TableName() string {
	return "tenant_settings"
}

// toDomain 将持久化对象转换为领域模型 model.TenantSettings。
func (po TenantSettingsPO) toDomain() *model.TenantSettings {
	return &model.TenantSettings{
		ID:           po.ID,
		TenantID:     po.TenantID,
		Logo:         po.Logo,
		Favicon:      po.Favicon,
		PrimaryColor: po.PrimaryColor,
		Theme:        po.Theme,
		Language:     po.Language,
		Timezone:     po.Timezone,
		Description:  po.Description,
		WelcomeText:  po.WelcomeText,
		ContactName:  po.ContactName,
		ContactEmail: po.ContactEmail,
		ContactPhone: po.ContactPhone,
		Address:      po.Address,
		Extra:        po.Extra,
		CreatedAt:    po.CreatedAt,
		UpdatedAt:    po.UpdatedAt,
	}
}

// AutoMigrateTenantSettings 自动迁移租户设置表结构。
func AutoMigrateTenantSettings(db *gorm.DB) error {
	return db.AutoMigrate(&TenantSettingsPO{})
}

// tenantSettingsRepository TenantSettingsRepository 的 GORM 实现。
type tenantSettingsRepository struct {
	db *gorm.DB
}

// NewTenantSettingsRepository 创建租户设置仓储实例。
func NewTenantSettingsRepository(db *gorm.DB) TenantSettingsRepository {
	return &tenantSettingsRepository{db: db}
}

// TenantSettingsRepository 租户设置数据访问接口。
type TenantSettingsRepository interface {
	GetByTenantID(ctx context.Context, tenantID string) (*model.TenantSettings, error)
	Create(ctx context.Context, settings *model.TenantSettings) error
	Update(ctx context.Context, settings *model.TenantSettings) error
	Upsert(ctx context.Context, settings *model.TenantSettings) error
}

// GetByTenantID 按租户ID查询设置，不存在时返回 nil（非错误）。
func (r *tenantSettingsRepository) GetByTenantID(ctx context.Context, tenantID string) (*model.TenantSettings, error) {
	var po TenantSettingsPO
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return po.toDomain(), nil
}

// Create 新建租户设置记录。
func (r *tenantSettingsRepository) Create(ctx context.Context, settings *model.TenantSettings) error {
	po := &TenantSettingsPO{
		ID:           settings.ID,
		TenantID:     settings.TenantID,
		Logo:         settings.Logo,
		Favicon:      settings.Favicon,
		PrimaryColor: settings.PrimaryColor,
		Theme:        settings.Theme,
		Language:     settings.Language,
		Timezone:     settings.Timezone,
		Description:  settings.Description,
		WelcomeText:  settings.WelcomeText,
		ContactName:  settings.ContactName,
		ContactEmail: settings.ContactEmail,
		ContactPhone: settings.ContactPhone,
		Address:      settings.Address,
		Extra:        settings.Extra,
	}
	return r.db.WithContext(ctx).Create(po).Error
}

// Update 仅更新非空字段，自动维护 updated_at。
func (r *tenantSettingsRepository) Update(ctx context.Context, settings *model.TenantSettings) error {
	updates := map[string]interface{}{}
	if settings.Logo != "" {
		updates["logo"] = settings.Logo
	}
	if settings.Favicon != "" {
		updates["favicon"] = settings.Favicon
	}
	if settings.PrimaryColor != "" {
		updates["primary_color"] = settings.PrimaryColor
	}
	if settings.Theme != "" {
		updates["theme"] = settings.Theme
	}
	if settings.Language != "" {
		updates["language"] = settings.Language
	}
	if settings.Timezone != "" {
		updates["timezone"] = settings.Timezone
	}
	if settings.Description != "" {
		updates["description"] = settings.Description
	}
	if settings.WelcomeText != "" {
		updates["welcome_text"] = settings.WelcomeText
	}
	if settings.ContactName != "" {
		updates["contact_name"] = settings.ContactName
	}
	if settings.ContactEmail != "" {
		updates["contact_email"] = settings.ContactEmail
	}
	if settings.ContactPhone != "" {
		updates["contact_phone"] = settings.ContactPhone
	}
	if settings.Address != "" {
		updates["address"] = settings.Address
	}
	if settings.Extra != "" {
		updates["extra"] = settings.Extra
	}
	updates["updated_at"] = time.Now()

	if len(updates) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Model(&TenantSettingsPO{}).Where("tenant_id = ?", settings.TenantID).Updates(updates).Error
}

// Upsert 存在则更新、不存在则创建，保证租户设置的幂等写入。
func (r *tenantSettingsRepository) Upsert(ctx context.Context, settings *model.TenantSettings) error {
	existing, err := r.GetByTenantID(ctx, settings.TenantID)
	if err != nil {
		return err
	}
	if existing == nil {
		return r.Create(ctx, settings)
	}
	return r.Update(ctx, settings)
}
