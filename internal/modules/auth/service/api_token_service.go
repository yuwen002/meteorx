package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/base64url"
	"errors"
	"fmt"
	"strings"
	"time"

	"meteorx/internal/cache"
	"meteorx/internal/config"
	"meteorx/internal/modules/auth/dto"
	"meteorx/internal/modules/auth/model"
	"meteorx/internal/modules/auth/repository"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	"meteorx/internal/modules/user/repository"
	"meteorx/pkg/idgen"
)

var (
	ErrAPITokenNotFound    = errors.New("api token not found")
	ErrAPITokenExpired     = errors.New("api token has expired")
	ErrAPITokenRevoked     = errors.New("api token has been revoked")
	ErrAPITokenInvalid     = errors.New("invalid api token")
	ErrAPITokenNameExists  = errors.New("a token with this name already exists")
	ErrAPITokenMaxExceeded = errors.New("maximum number of api tokens exceeded")
)

const (
	apiTokenPrefix     = "mxat_"
	apiTokenCachePrefix = "api_token:"
	apiTokenBytes      = 32
	maxTokensPerUser   = 10
	apiTokenCacheTTL   = 1 * time.Hour
)

type APITokenService struct {
	apiTokenRepo   repository.APITokenRepository
	userRepo       repository.UserRepository
	userRoleRepo   rbacRepo.UserRoleRepository
	rolePermRepo   rbacRepo.RolePermissionRepository
	redis          *cache.Redis
	authCfg        config.AuthConfig
}

func NewAPITokenService(
	apiTokenRepo repository.APITokenRepository,
	userRepo repository.UserRepository,
	userRoleRepo rbacRepo.UserRoleRepository,
	rolePermRepo rbacRepo.RolePermissionRepository,
	redis *cache.Redis,
	authCfg config.AuthConfig,
) *APITokenService {
	return &APITokenService{
		apiTokenRepo:   apiTokenRepo,
		userRepo:       userRepo,
		userRoleRepo:   userRoleRepo,
		rolePermRepo:   rolePermRepo,
		redis:          redis,
		authCfg:        authCfg,
	}
}

func (s *APITokenService) Create(ctx context.Context, userID, tenantID string, req dto.CreateAPITokenReq) (*dto.CreateAPITokenResp, error) {
	count, err := s.countActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if count >= maxTokensPerUser {
		return nil, ErrAPITokenMaxExceeded
	}

	plainToken, tokenHash, err := generateAPIToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	expiresAt, err := s.parseExpiresIn(req.ExpiresIn)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	t := &model.APIToken{
		ID:        idgen.New(),
		Name:      req.Name,
		TokenHash: tokenHash,
		UserID:    userID,
		TenantID:  tenantID,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}

	if err := s.apiTokenRepo.Create(ctx, t); err != nil {
		if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "duplicate") {
			return nil, ErrAPITokenNameExists
		}
		return nil, err
	}

	s.cacheToken(ctx, tokenHash, userID, tenantID, expiresAt)

	resp := &dto.CreateAPITokenResp{
		ID:        t.ID,
		Name:      t.Name,
		Token:     plainToken,
		CreatedAt: t.CreatedAt,
	}
	if expiresAt != nil {
		s := expiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &s
	}
	return resp, nil
}

func (s *APITokenService) ListByUserID(ctx context.Context, userID string) ([]*dto.APITokenResp, error) {
	tokens, err := s.apiTokenRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*dto.APITokenResp, len(tokens))
	for i, t := range tokens {
		result[i] = toAPITokenResp(t)
	}
	return result, nil
}

func (s *APITokenService) Revoke(ctx context.Context, userID, tokenID string) error {
	t, err := s.apiTokenRepo.GetByID(ctx, tokenID)
	if err != nil {
		return ErrAPITokenNotFound
	}
	if t.UserID != userID {
		return ErrAPITokenNotFound
	}
	if t.RevokedAt != nil {
		return ErrAPITokenRevoked
	}

	if err := s.apiTokenRepo.RevokeByID(ctx, tokenID); err != nil {
		return err
	}

	s.invalidateCache(ctx, t.TokenHash)
	return nil
}

func (s *APITokenService) Validate(ctx context.Context, tokenString string) (userID, tenantID string, err error) {
	if !strings.HasPrefix(tokenString, apiTokenPrefix) {
		return "", "", ErrAPITokenInvalid
	}

	tokenHash := hashAPIToken(tokenString)

	userID, tenantID, err = s.lookupCache(ctx, tokenHash)
	if err == nil && userID != "" {
		return userID, tenantID, nil
	}

	t, err := s.apiTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return "", "", ErrAPITokenInvalid
	}

	if t.RevokedAt != nil {
		return "", "", ErrAPITokenRevoked
	}
	if t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now()) {
		return "", "", ErrAPITokenExpired
	}

	_ = s.apiTokenRepo.UpdateLastUsedAt(ctx, t.ID)
	s.cacheToken(ctx, tokenHash, t.UserID, t.TenantID, t.ExpiresAt)

	return t.UserID, t.TenantID, nil
}

func (s *APITokenService) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	return s.userRoleRepo.GetRoleCodesByUserID(ctx, userID)
}

func (s *APITokenService) countActiveByUserID(ctx context.Context, userID string) (int, error) {
	tokens, err := s.apiTokenRepo.ListByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, t := range tokens {
		if t.RevokedAt == nil {
			count++
		}
	}
	return count, nil
}

func (s *APITokenService) parseExpiresIn(expiresIn string) (*time.Time, error) {
	maxTTL := s.authCfg.GetAPITokenMaxTTL()

	if expiresIn == "" {
		expiresAt := time.Now().Add(maxTTL)
		return &expiresAt, nil
	}

	d, err := time.ParseDuration(expiresIn)
	if err != nil {
		return nil, fmt.Errorf("invalid expires_in format: %w", err)
	}
	if d <= 0 {
		return nil, fmt.Errorf("expires_in must be positive")
	}
	if d > maxTTL {
		return nil, fmt.Errorf("expires_in exceeds maximum allowed (%s)", maxTTL)
	}

	expiresAt := time.Now().Add(d)
	return &expiresAt, nil
}

func (s *APITokenService) cacheToken(ctx context.Context, tokenHash, userID, tenantID string, expiresAt *time.Time) {
	if s.redis == nil || !s.redis.IsAvailable() {
		return
	}
	key := apiTokenCachePrefix + tokenHash
	data := fmt.Sprintf("%s:%s", userID, tenantID)

	ttl := apiTokenCacheTTL
	if expiresAt != nil {
		remaining := time.Until(*expiresAt)
		if remaining < ttl {
			ttl = remaining
		}
	}
	if ttl <= 0 {
		return
	}
	_ = s.redis.Set(ctx, key, data, ttl)
}

func (s *APITokenService) lookupCache(ctx context.Context, tokenHash string) (userID, tenantID string, err error) {
	if s.redis == nil || !s.redis.IsAvailable() {
		return "", "", fmt.Errorf("cache unavailable")
	}
	key := apiTokenCachePrefix + tokenHash
	val, err := s.redis.Get(ctx, key)
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(val, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid cache value")
	}
	return parts[0], parts[1], nil
}

func (s *APITokenService) invalidateCache(ctx context.Context, tokenHash string) {
	if s.redis == nil || !s.redis.IsAvailable() {
		return
	}
	key := apiTokenCachePrefix + tokenHash
	_ = s.redis.Delete(ctx, key)
}

func generateAPIToken() (plainText, hash string, err error) {
	b := make([]byte, apiTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	plainText = apiTokenPrefix + base64url.EncodeToString(b)
	hash = hashAPIToken(plainText)
	return plainText, hash, nil
}

func hashAPIToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.StdEncoding.EncodeToString(h[:])
}

func toAPITokenResp(t *model.APIToken) *dto.APITokenResp {
	resp := &dto.APITokenResp{
		ID:        t.ID,
		Name:      t.Name,
		CreatedAt: t.CreatedAt,
		Revoked:   t.RevokedAt != nil,
	}
	if t.LastUsedAt != nil {
		s := t.LastUsedAt.Format(time.RFC3339)
		resp.LastUsedAt = &s
	}
	if t.ExpiresAt != nil {
		s := t.ExpiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &s
	}
	return resp
}