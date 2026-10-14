package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"meteorx/internal/cache"
	"meteorx/internal/common/contextx"
	"meteorx/internal/config"
	"meteorx/internal/modules/auth/dto"
	"meteorx/internal/modules/auth/model"
	"meteorx/internal/modules/auth/repository"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	userRepo "meteorx/internal/modules/user/repository"
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

// APITokenService API Token 服务，处理长期令牌的创建、验证、撤销等业务逻辑
type APITokenService struct {
	apiTokenRepo   repository.APITokenRepository
	userRepo       userRepo.UserRepository
	userRoleRepo   rbacRepo.UserRoleRepository
	rolePermRepo   rbacRepo.RolePermissionRepository
	redis          *cache.Redis
	authCfg        config.AuthConfig
}

// NewAPITokenService 创建 API Token 服务实例
func NewAPITokenService(
	apiTokenRepo repository.APITokenRepository,
	userRepo userRepo.UserRepository,
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

// Create 创建 API Token，返回明文令牌（仅此一次可见）
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
		ID:           idgen.New(),
		Name:         req.Name,
		TokenHash:    tokenHash,
		UserID:       userID,
		TenantID:     tenantID,
		AllowedPaths: serializeList(req.AllowedPaths),
		Scopes:       serializeList(req.Scopes),
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
	}

	if err := s.apiTokenRepo.Create(ctx, t); err != nil {
		if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "duplicate") {
			return nil, ErrAPITokenNameExists
		}
		return nil, err
	}

	s.cacheToken(ctx, tokenHash, userID, tenantID, req.AllowedPaths, req.Scopes, expiresAt)

	resp := &dto.CreateAPITokenResp{
		ID:           t.ID,
		Name:         t.Name,
		Token:        plainToken,
		AllowedPaths: req.AllowedPaths,
		Scopes:       req.Scopes,
		CreatedAt:    t.CreatedAt,
	}
	if expiresAt != nil {
		s := expiresAt.Format(time.RFC3339)
		resp.ExpiresAt = &s
	}
	return resp, nil
}

// ListByUserID 列出用户的所有 API Token
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

// Revoke 撤销 API Token，使其立即失效
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

// Validate 验证 API Token 有效性，优先查缓存，缓存未命中则查数据库。
// 返回 *contextx.TokenPrincipal：
//   - 令牌未配 scope 时 Scoped=false，沿用属主完整 RBAC 权限（向后兼容）；
//   - 令牌配非空 scope 时 Scoped=true，Permissions 为「属主权限 ∩ scopes」（属主为超管时即 scopes），
//     作为下游权限中间件的严格上限。
func (s *APITokenService) Validate(ctx context.Context, tokenString string) (*contextx.TokenPrincipal, error) {
	if !strings.HasPrefix(tokenString, apiTokenPrefix) {
		return nil, ErrAPITokenInvalid
	}

	tokenHash := hashAPIToken(tokenString)

	if p, ok := s.lookupCache(ctx, tokenHash); ok {
		return p, nil
	}

	t, err := s.apiTokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrAPITokenInvalid
	}

	if t.RevokedAt != nil {
		return nil, ErrAPITokenRevoked
	}
	if t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now()) {
		return nil, ErrAPITokenExpired
	}

	parsedPaths := parseList(t.AllowedPaths)
	parsedScopes := parseList(t.Scopes)
	_ = s.apiTokenRepo.UpdateLastUsedAt(ctx, t.ID)

	principal := &contextx.TokenPrincipal{
		UserID:       t.UserID,
		TenantID:     t.TenantID,
		AllowedPaths: parsedPaths,
	}
	if len(parsedScopes) > 0 {
		principal.Scoped = true
		principal.Permissions = s.resolveScopedPermissions(ctx, t.UserID, parsedScopes)
	}

	s.cacheToken(ctx, tokenHash, t.UserID, t.TenantID, parsedPaths, parsedScopes, t.ExpiresAt)
	return principal, nil
}

// resolveScopedPermissions 计算令牌 scope 与属主实际权限的交集。
// 属主为 superadmin 时其权限天然是全集，有效集合即 scopes；否则取 scopes ∩ 属主权限码，
// 确保令牌永远不能越出属主权限边界。
func (s *APITokenService) resolveScopedPermissions(ctx context.Context, userID string, scopes []string) []string {
	roleCodes, err := s.userRoleRepo.GetRoleCodesByUserID(ctx, userID)
	if err == nil {
		for _, rc := range roleCodes {
			if rc == "superadmin" {
				return scopes
			}
		}
	}

	ownerCodes, err := s.GetUserPermissionCodes(ctx, userID)
	if err != nil {
		// 无法解析属主权限时保守返回空集合（拒绝一切），避免越权
		return nil
	}
	ownerSet := make(map[string]struct{}, len(ownerCodes))
	for _, c := range ownerCodes {
		ownerSet[c] = struct{}{}
	}
	var effective []string
	for _, sc := range scopes {
		if _, ok := ownerSet[sc]; ok {
			effective = append(effective, sc)
		}
	}
	return effective
}

// GetUserPermissionCodes 合并用户全部角色持有的权限码（去重），用于 scope 收敛计算。
func (s *APITokenService) GetUserPermissionCodes(ctx context.Context, userID string) ([]string, error) {
	roleIDs, err := s.userRoleRepo.GetRoleIDsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		return []string{}, nil
	}

	codeSet := make(map[string]bool)
	for _, roleID := range roleIDs {
		codes, err := s.rolePermRepo.GetPermissionCodesByRoleID(ctx, roleID)
		if err != nil {
			continue
		}
		for _, code := range codes {
			codeSet[code] = true
		}
	}

	result := make([]string, 0, len(codeSet))
	for code := range codeSet {
		result = append(result, code)
	}
	return result, nil
}

// GetUserRoles 获取用户的角色编码列表
func (s *APITokenService) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	return s.userRoleRepo.GetRoleCodesByUserID(ctx, userID)
}

// countActiveByUserID 统计用户未撤销的 API Token 数量
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

// parseExpiresIn 解析过期时间字符串，校验不超过最大允许 TTL
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

// apiTokenCacheValue API Token 缓存的 JSON 结构（避免旧 “冒号分隔” 格式对含特殊字符 JSON 的脆弱拆分）。
// Scopes 为空表示无 scope（继承模式）；非空时 Permissions 为已计算的收敛集合。
type apiTokenCacheValue struct {
	UserID       string   `json:"user_id"`
	TenantID     string   `json:"tenant_id"`
	AllowedPaths []string `json:"allowed_paths,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	Permissions  []string `json:"permissions,omitempty"`
}

// cacheToken 将 token 信息缓存到 Redis，TTL 取缓存默认值与剩余有效期的较小值。
// 当 scopes 非空时，同步缓存已收敛的权限集合，避免每次命中都回查属主权限。
func (s *APITokenService) cacheToken(ctx context.Context, tokenHash, userID, tenantID string, allowedPaths, scopes []string, expiresAt *time.Time) {
	if s.redis == nil || !s.redis.IsAvailable() {
		return
	}
	key := apiTokenCachePrefix + tokenHash
	val := apiTokenCacheValue{UserID: userID, TenantID: tenantID, AllowedPaths: allowedPaths, Scopes: scopes}
	if len(scopes) > 0 {
		val.Permissions = s.resolveScopedPermissions(ctx, userID, scopes)
	}
	data, err := json.Marshal(val)
	if err != nil {
		return
	}

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
	_ = s.redis.Set(ctx, key, string(data), ttl)
}

// lookupCache 从 Redis 缓存中查找 token 对应的身份与授权信息，命中且合法时返回。
func (s *APITokenService) lookupCache(ctx context.Context, tokenHash string) (*contextx.TokenPrincipal, bool) {
	if s.redis == nil || !s.redis.IsAvailable() {
		return nil, false
	}
	key := apiTokenCachePrefix + tokenHash
	val, err := s.redis.Get(ctx, key)
	if err != nil {
		return nil, false
	}
	var cv apiTokenCacheValue
	if err := json.Unmarshal([]byte(val), &cv); err != nil || cv.UserID == "" {
		return nil, false
	}
	principal := &contextx.TokenPrincipal{
		UserID:       cv.UserID,
		TenantID:     cv.TenantID,
		AllowedPaths: cv.AllowedPaths,
		Scoped:       len(cv.Scopes) > 0,
		Permissions:  cv.Permissions,
	}
	return principal, true
}

// invalidateCache 使 Redis 中的 token 缓存失效
func (s *APITokenService) invalidateCache(ctx context.Context, tokenHash string) {
	if s.redis == nil || !s.redis.IsAvailable() {
		return
	}
	key := apiTokenCachePrefix + tokenHash
	_ = s.redis.Delete(ctx, key)
}

// generateAPIToken 生成随机 API Token 明文和 SHA-256 哈希值
func generateAPIToken() (plainText, hash string, err error) {
	b := make([]byte, apiTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	plainText = apiTokenPrefix + base64.RawURLEncoding.EncodeToString(b)
	hash = hashAPIToken(plainText)
	return plainText, hash, nil
}

// hashAPIToken 对 token 进行 SHA-256 哈希
func hashAPIToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.StdEncoding.EncodeToString(h[:])
}

// toAPITokenResp 将领域模型转换为 DTO 响应
func toAPITokenResp(t *model.APIToken) *dto.APITokenResp {
	resp := &dto.APITokenResp{
		ID:           t.ID,
		Name:         t.Name,
		AllowedPaths: parseList(t.AllowedPaths),
		Scopes:       parseList(t.Scopes),
		CreatedAt:    t.CreatedAt,
		Revoked:      t.RevokedAt != nil,
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

// serializeList 将字符串列表序列化为 JSON 字符串，空列表返回空字符串
func serializeList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	b, _ := json.Marshal(items)
	return string(b)
}

// parseList 将 JSON 字符串反序列化为字符串列表
func parseList(s string) []string {
	if s == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(s), &items); err != nil {
		return nil
	}
	return items
}