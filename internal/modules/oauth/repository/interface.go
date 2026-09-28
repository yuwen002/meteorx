package repository

import (
	"context"
	"meteorx/internal/modules/oauth/model"
)

type OAuthAccountRepository interface {
	Create(ctx context.Context, account *model.OAuthAccount) error
	GetByProviderAndProviderID(ctx context.Context, provider, providerID string) (*model.OAuthAccount, error)
	ListByUserID(ctx context.Context, userID string) ([]*model.OAuthAccount, error)
	DeleteByID(ctx context.Context, id string) error
	GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*model.OAuthAccount, error)
}