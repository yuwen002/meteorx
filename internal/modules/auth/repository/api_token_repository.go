package repository

import (
	"context"

	"meteorx/internal/modules/auth/model"

	"gorm.io/gorm"
)

type APITokenRepository interface {
	Create(ctx context.Context, token *model.APIToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.APIToken, error)
	ListByUserID(ctx context.Context, userID string) ([]*model.APIToken, error)
	RevokeByID(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*model.APIToken, error)
	UpdateLastUsedAt(ctx context.Context, id string) error
}

type apiTokenRepository struct {
	db *gorm.DB
}

func NewAPITokenRepository(db *gorm.DB) APITokenRepository {
	return &apiTokenRepository{db: db}
}

func (r *apiTokenRepository) Create(ctx context.Context, token *model.APIToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *apiTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.APIToken, error) {
	var token model.APIToken
	if err := r.db.WithContext(ctx).Where("token_hash = ? AND revoked_at IS NULL", tokenHash).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *apiTokenRepository) ListByUserID(ctx context.Context, userID string) ([]*model.APIToken, error) {
	var tokens []*model.APIToken
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func (r *apiTokenRepository) RevokeByID(ctx context.Context, id string) error {
	now := gorm.Expr("NOW()")
	return r.db.WithContext(ctx).Model(&model.APIToken{}).Where("id = ? AND revoked_at IS NULL", id).Update("revoked_at", now).Error
}

func (r *apiTokenRepository) GetByID(ctx context.Context, id string) (*model.APIToken, error) {
	var token model.APIToken
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *apiTokenRepository) UpdateLastUsedAt(ctx context.Context, id string) error {
	now := gorm.Expr("NOW()")
	return r.db.WithContext(ctx).Model(&model.APIToken{}).Where("id = ?", id).Update("last_used_at", now).Error
}