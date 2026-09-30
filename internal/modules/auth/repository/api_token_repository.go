// Package repository 定义认证模块的数据访问接口和 GORM 实现。
package repository

import (
	"context"

	"meteorx/internal/modules/auth/model"

	"gorm.io/gorm"
)

// APITokenRepository API 令牌数据访问接口，支持令牌创建、哈希查询、吊销与使用记录。
type APITokenRepository interface {
	Create(ctx context.Context, token *model.APIToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*model.APIToken, error)
	ListByUserID(ctx context.Context, userID string) ([]*model.APIToken, error)
	RevokeByID(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*model.APIToken, error)
	UpdateLastUsedAt(ctx context.Context, id string) error
}

// apiTokenRepository APITokenRepository 的 GORM 实现。
type apiTokenRepository struct {
	db *gorm.DB
}

// NewAPITokenRepository 创建 API 令牌仓储实例。
func NewAPITokenRepository(db *gorm.DB) APITokenRepository {
	return &apiTokenRepository{db: db}
}

// Create 新建 API 令牌记录。
func (r *apiTokenRepository) Create(ctx context.Context, token *model.APIToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// GetByTokenHash 按令牌哈希查询未吊销的 API 令牌。
func (r *apiTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*model.APIToken, error) {
	var token model.APIToken
	if err := r.db.WithContext(ctx).Where("token_hash = ? AND revoked_at IS NULL", tokenHash).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

// ListByUserID 列出指定用户的全部 API 令牌（按创建时间倒序）。
func (r *apiTokenRepository) ListByUserID(ctx context.Context, userID string) ([]*model.APIToken, error) {
	var tokens []*model.APIToken
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

// RevokeByID 吊销指定令牌，写入吊销时间（仅对未吊销的生效）。
func (r *apiTokenRepository) RevokeByID(ctx context.Context, id string) error {
	now := gorm.Expr("NOW()")
	return r.db.WithContext(ctx).Model(&model.APIToken{}).Where("id = ? AND revoked_at IS NULL", id).Update("revoked_at", now).Error
}

// GetByID 按主键查询 API 令牌。
func (r *apiTokenRepository) GetByID(ctx context.Context, id string) (*model.APIToken, error) {
	var token model.APIToken
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

// UpdateLastUsedAt 刷新令牌的最近使用时间。
func (r *apiTokenRepository) UpdateLastUsedAt(ctx context.Context, id string) error {
	now := gorm.Expr("NOW()")
	return r.db.WithContext(ctx).Model(&model.APIToken{}).Where("id = ?", id).Update("last_used_at", now).Error
}
