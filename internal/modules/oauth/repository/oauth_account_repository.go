package repository

import (
	"context"

	"meteorx/internal/modules/oauth/model"

	"gorm.io/gorm"
)

type oauthAccountRepository struct {
	db *gorm.DB
}

func NewOAuthAccountRepository(db *gorm.DB) OAuthAccountRepository {
	return &oauthAccountRepository{db: db}
}

func (r *oauthAccountRepository) Create(ctx context.Context, account *model.OAuthAccount) error {
	return r.db.WithContext(ctx).Create(account).Error
}

func (r *oauthAccountRepository) GetByProviderAndProviderID(ctx context.Context, provider, providerID string) (*model.OAuthAccount, error) {
	var account model.OAuthAccount
	if err := r.db.WithContext(ctx).Where("provider = ? AND provider_id = ?", provider, providerID).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *oauthAccountRepository) ListByUserID(ctx context.Context, userID string) ([]*model.OAuthAccount, error) {
	var accounts []*model.OAuthAccount
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *oauthAccountRepository) DeleteByID(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&model.OAuthAccount{}, "id = ?", id).Error
}

func (r *oauthAccountRepository) GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*model.OAuthAccount, error) {
	var account model.OAuthAccount
	if err := r.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, provider).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}