// Package repository 提供 OAuth 账号数据访问的 GORM 实现。
package repository

import (
	"context"
	"time"

	"meteorx/internal/modules/oauth/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// oauthProviderPO OAuth 提供商运行时配置的持久化对象。
// ClientSecretEnc 存放 AES-256-GCM 密文（带 enc:v1: 前缀），Service 层负责加解密。
type oauthProviderPO struct {
	ID              string    `gorm:"primaryKey;size:32"`
	Kind            string    `gorm:"size:16;index;comment:'provider 类型'"`
	Code            string    `gorm:"size:64;uniqueIndex;comment:'provider 标识(唯一)'"`
	DisplayName     string    `gorm:"size:64;comment:'展示名'"`
	Enabled         bool      `gorm:"index;comment:'是否启用'"`
	ClientID        string    `gorm:"size:256;comment:'客户端 ID'"`
	ClientSecretEnc string    `gorm:"type:text;comment:'加密后的 client_secret'"`
	RedirectURL     string    `gorm:"size:512;comment:'授权回调地址'"`
	Tenant          string    `gorm:"size:128;comment:'Microsoft 目录租户'"`
	AuthURL         string    `gorm:"size:512;comment:'OIDC 授权端点'"`
	TokenURL        string    `gorm:"size:512;comment:'OIDC 令牌端点'"`
	UserInfoURL     string    `gorm:"size:512;comment:'OIDC 用户信息端点'"`
	Scopes          []string  `gorm:"type:text;serializer:json;comment:'OIDC 授权范围'"`
	CreatedAt       time.Time `gorm:"index"`
	UpdatedAt       time.Time
}

// TableName 显式指定表名，避免 GORM 复数化推导出意外结果
func (oauthProviderPO) TableName() string { return "oauth_providers" }

// AutoMigrateProvider 迁移 oauth_providers 表及其元数据表。
// 必须以 PO 为准：领域模型含 []string 字段，GORM 无法直接迁移。
func AutoMigrateProvider(db *gorm.DB) error {
	if err := db.AutoMigrate(&oauthProviderPO{}); err != nil {
		return err
	}
	return db.AutoMigrate(&oauthProviderMetaPO{})
}

func (po *oauthProviderPO) toDomain() *model.OAuthProvider {
	return &model.OAuthProvider{
		ID:           po.ID,
		Kind:         po.Kind,
		Code:         po.Code,
		DisplayName:  po.DisplayName,
		Enabled:      po.Enabled,
		ClientID:     po.ClientID,
		ClientSecret: po.ClientSecretEnc, // 密文原样带回，由 Service 层解密
		RedirectURL:  po.RedirectURL,
		Tenant:       po.Tenant,
		AuthURL:      po.AuthURL,
		TokenURL:     po.TokenURL,
		UserInfoURL:  po.UserInfoURL,
		Scopes:       po.Scopes,
		CreatedAt:    po.CreatedAt,
		UpdatedAt:    po.UpdatedAt,
	}
}

func oauthProviderPOFromDomain(m *model.OAuthProvider) *oauthProviderPO {
	return &oauthProviderPO{
		ID:              m.ID,
		Kind:            m.Kind,
		Code:            m.Code,
		DisplayName:     m.DisplayName,
		Enabled:         m.Enabled,
		ClientID:        m.ClientID,
		ClientSecretEnc: m.ClientSecret, // Service 层已放入密文
		RedirectURL:     m.RedirectURL,
		Tenant:          m.Tenant,
		AuthURL:         m.AuthURL,
		TokenURL:        m.TokenURL,
		UserInfoURL:     m.UserInfoURL,
		Scopes:          m.Scopes,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

// OAuthProviderRepository 第三方登录提供商运行时配置的数据访问接口。
type OAuthProviderRepository interface {
	// List 列出全部提供商配置（含禁用），按创建时间升序
	List(ctx context.Context) ([]*model.OAuthProvider, error)
	// GetByCode 按标识查询
	GetByCode(ctx context.Context, code string) (*model.OAuthProvider, error)
	// Count 统计配置条数
	Count(ctx context.Context) (int64, error)
	// Create 新建配置
	Create(ctx context.Context, provider *model.OAuthProvider) error
	// Update 按标识更新配置（Code 不可变）
	Update(ctx context.Context, provider *model.OAuthProvider) error
	// UpdateStatus 仅切换启用状态
	UpdateStatus(ctx context.Context, code string, enabled bool) error
	// DeleteByCode 按标识删除
	DeleteByCode(ctx context.Context, code string) error
	// GetMeta 读取元数据（如种子化标记），不存在返回空字符串
	GetMeta(ctx context.Context, key string) (string, error)
	// SetMeta 写入元数据
	SetMeta(ctx context.Context, key, value string) error
}

// MetaKeySeededFromConfig 种子化标记的元数据键：存在即表示已从 config.yaml 种子化过，
// 之后即使配置被管理员删光也不会再次自动种子化（避免「删除后复活」）。
const MetaKeySeededFromConfig = "seeded_from_config"

// oauthProviderMetaPO oauth_providers 的元数据表（键值对），用于跨副本记录一次性状态。
type oauthProviderMetaPO struct {
	Key       string    `gorm:"primaryKey;size:64"`
	Value     string    `gorm:"size:255"`
	UpdatedAt time.Time
}

func (oauthProviderMetaPO) TableName() string { return "oauth_provider_meta" }

type oauthProviderRepository struct {
	db *gorm.DB
}

// NewOAuthProviderRepository 创建提供商配置仓储实例。
func NewOAuthProviderRepository(db *gorm.DB) OAuthProviderRepository {
	return &oauthProviderRepository{db: db}
}

func (r *oauthProviderRepository) List(ctx context.Context) ([]*model.OAuthProvider, error) {
	var pos []*oauthProviderPO
	if err := r.db.WithContext(ctx).Order("created_at ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	out := make([]*model.OAuthProvider, 0, len(pos))
	for _, po := range pos {
		out = append(out, po.toDomain())
	}
	return out, nil
}

func (r *oauthProviderRepository) GetByCode(ctx context.Context, code string) (*model.OAuthProvider, error) {
	var po oauthProviderPO
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&po).Error; err != nil {
		return nil, err
	}
	return po.toDomain(), nil
}

func (r *oauthProviderRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&oauthProviderPO{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *oauthProviderRepository) Create(ctx context.Context, provider *model.OAuthProvider) error {
	return r.db.WithContext(ctx).Create(oauthProviderPOFromDomain(provider)).Error
}

func (r *oauthProviderRepository) Update(ctx context.Context, provider *model.OAuthProvider) error {
	// 先取出持久化对象再逐字段赋值并 Save：Save 会更新全部列（含零值），
	// 同时避免手写列名与 GORM 命名策略（如 UserInfoURL → user_info_url）不一致。
	var po oauthProviderPO
	if err := r.db.WithContext(ctx).Where("code = ?", provider.Code).First(&po).Error; err != nil {
		return err
	}
	po.Kind = provider.Kind
	po.DisplayName = provider.DisplayName
	po.Enabled = provider.Enabled
	po.ClientID = provider.ClientID
	po.ClientSecretEnc = provider.ClientSecret
	po.RedirectURL = provider.RedirectURL
	po.Tenant = provider.Tenant
	po.AuthURL = provider.AuthURL
	po.TokenURL = provider.TokenURL
	po.UserInfoURL = provider.UserInfoURL
	po.Scopes = oauthProviderPOFromDomain(provider).Scopes
	po.UpdatedAt = time.Now()

	return r.db.WithContext(ctx).Save(&po).Error
}

func (r *oauthProviderRepository) UpdateStatus(ctx context.Context, code string, enabled bool) error {
	return r.db.WithContext(ctx).
		Model(&oauthProviderPO{}).
		Where("code = ?", code).
		Updates(map[string]any{"enabled": enabled, "updated_at": time.Now()}).Error
}

func (r *oauthProviderRepository) DeleteByCode(ctx context.Context, code string) error {
	return r.db.WithContext(ctx).Where("code = ?", code).Delete(&oauthProviderPO{}).Error
}

func (r *oauthProviderRepository) GetMeta(ctx context.Context, key string) (string, error) {
	var meta oauthProviderMetaPO
	if err := r.db.WithContext(ctx).Where("`key` = ?", key).First(&meta).Error; err != nil {
		return "", err
	}
	return meta.Value, nil
}

func (r *oauthProviderRepository) SetMeta(ctx context.Context, key, value string) error {
	meta := oauthProviderMetaPO{Key: key, Value: value, UpdatedAt: time.Now()}
	// Upsert：主键冲突时更新值与时间戳
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&meta).Error
}
