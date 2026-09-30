// Package repository 定义 OAuth 模块的数据访问接口和 GORM 实现。
package repository

import (
	"context"
	"meteorx/internal/modules/oauth/model"
)

// OAuthAccountRepository 第三方 OAuth 账号绑定数据访问接口。
type OAuthAccountRepository interface {
	Create(ctx context.Context, account *model.OAuthAccount) error
	GetByProviderAndProviderID(ctx context.Context, provider, providerID string) (*model.OAuthAccount, error)
	ListByUserID(ctx context.Context, userID string) ([]*model.OAuthAccount, error)
	DeleteByID(ctx context.Context, id string) error
	GetByUserIDAndProvider(ctx context.Context, userID, provider string) (*model.OAuthAccount, error)
}
