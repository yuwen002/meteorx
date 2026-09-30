// Package repository 提供 OAuth 账号数据访问的 GORM 实现。
package repository

import (
	"context"

	"meteorx/internal/modules/oauth/model"

	"gorm.io/gorm"
)

// oauthAccountRepository OAuthAccountRepository 的 GORM 实现。
type oauthAccountRepository struct {
	db *gorm.DB
}

// NewOAuthAccountRepository 创建 OAuth 账号仓储实例。
func NewOAuthAccountRepository(db *gorm.DB) OAuthAccountRepository {
	return &oauthAccountRepository{db: db}
}

// Create 新建 OAuth 账号绑定记录。
func (r *oauthAccountRepository) Create(ctx context.Context, account *model.OAuthAccount) error {
	return r.db.WithContext(ctx).Create(account).Error
}

// GetByProviderAndProviderID 按第三方平台及其用户唯一ID查询 OAuth 账号。
func (r *oauthAccountRepository) GetByProviderAndProviderID(ctx context.Context, provider, providerID string) (*model.OAuthAccount, error) {
	var account model.OAuthAccount
	if err := r.db.WithContext(ctx).Where("provider = ? AND provider_id = ?", provider, providerID).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

// ListByUserID 列出指定用户已绑定的全部 OAuth 账号。
func (r *oauthAccountRepository) ListByUserID(ctx context.Context, userID string) ([]*model.OAuthAccount, error) {
	var accounts []*model.OAuthAccount
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

// DeleteByID 按主键解除（删除）OAuth 账号绑定。
func (r *oauthAccountRepository) DeleteByID(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.OAuthAccount{}, "id = ?", id).Error
}

// GetByUserIDAndProvider 按用户与平台查询已绑定的 OAuth 账号。
func (r *oauthAccountRepository) GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*model.OAuthAccount, error) {
	var account model.OAuthAccount
	if err := r.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, provider).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}
