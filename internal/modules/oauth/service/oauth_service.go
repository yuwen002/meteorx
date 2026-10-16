// Package service 实现 OAuth 模块的业务逻辑，处理授权码交换、用户信息获取和账号绑定。
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"meteorx/internal/cache"
	"meteorx/internal/common/jwt"
	"meteorx/internal/config"
	"meteorx/internal/modules/oauth/dto"
	"meteorx/internal/modules/oauth/model"
	oauthRepo "meteorx/internal/modules/oauth/repository"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	tenantRepo "meteorx/internal/modules/tenant/repository"
	userModel "meteorx/internal/modules/user/model"
	"meteorx/internal/modules/user/repository"
	"meteorx/pkg/crypto"
	"meteorx/pkg/idgen"
)

var (
	ErrOAuthProviderDisabled    = errors.New("OAuth provider not enabled")
	ErrOAuthExchangeFailed      = errors.New("failed to exchange authorization code")
	ErrOAuthUserInfoFailed      = errors.New("failed to get user info from provider")
	ErrOAuthInvalidState        = errors.New("invalid or expired OAuth state")
	ErrOAuthAccountNotFound     = errors.New("OAuth account not found")
	ErrOAuthAccountAlreadyBound = errors.New("this provider is already bound to your account")
	ErrRefreshTokenInvalid      = errors.New("invalid or expired refresh token")
)

const (
	oauthStatePrefix   = "oauth:state:"
	refreshTokenPrefix = "oauth:refresh:"
)

// OAuthService OAuth 认证服务，处理第三方登录、账号绑定和令牌刷新
type OAuthService struct {
	cfg                config.OAuthConfig
	registry           *ProviderRegistry
	providerRepo       oauthRepo.OAuthProviderRepository
	cipher             *crypto.SecretCipher
	userRepo           repository.UserRepository
	roleRepo           rbacRepo.RoleRepository
	userRoleRepo       rbacRepo.UserRoleRepository
	rolePermissionRepo rbacRepo.RolePermissionRepository
	tenantRepo         tenantRepo.TenantRepository
	oauthAccountRepo   oauthRepo.OAuthAccountRepository
	tokenHelper        *jwt.TokenHelper
	redis              *cache.Redis
}

// NewOAuthService 创建 OAuth 服务实例。
// providerRepo 与 cipher 可为 nil：此时退回 config.yaml 静态配置（便于测试与降级运行）。
func NewOAuthService(
	cfg config.OAuthConfig,
	userRepo repository.UserRepository,
	roleRepo rbacRepo.RoleRepository,
	userRoleRepo rbacRepo.UserRoleRepository,
	rolePermissionRepo rbacRepo.RolePermissionRepository,
	tenantRepo tenantRepo.TenantRepository,
	oauthAccountRepo oauthRepo.OAuthAccountRepository,
	tokenHelper *jwt.TokenHelper,
	redis *cache.Redis,
	providerRepo oauthRepo.OAuthProviderRepository,
	cipher *crypto.SecretCipher,
) *OAuthService {
	return &OAuthService{
		cfg:                cfg,
		registry:           NewProviderRegistry(cfg),
		providerRepo:       providerRepo,
		cipher:             cipher,
		userRepo:           userRepo,
		roleRepo:           roleRepo,
		userRoleRepo:       userRoleRepo,
		rolePermissionRepo: rolePermissionRepo,
		tenantRepo:         tenantRepo,
		oauthAccountRepo:   oauthAccountRepo,
		tokenHelper:        tokenHelper,
		redis:              redis,
	}
}

// generateState 生成随机 CSRF state 令牌
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// saveState 将 OAuth state 令牌存入 Redis，用于 CSRF 防护
func (s *OAuthService) saveState(ctx context.Context, state string) error {
	if s.redis == nil || !s.redis.IsAvailable() {
		return nil
	}
	key := oauthStatePrefix + state
	return s.redis.Set(ctx, key, "1", s.cfg.GetStateTTL())
}

// validateAndConsumeState 验证并消费 OAuth state 令牌，一次性使用
func (s *OAuthService) validateAndConsumeState(ctx context.Context, state string) error {
	if s.redis == nil || !s.redis.IsAvailable() {
		return nil
	}
	key := oauthStatePrefix + state
	val, err := s.redis.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cache.ErrRedisUnavailable) {
			return nil
		}
		return ErrOAuthInvalidState
	}
	if val == "" {
		return ErrOAuthInvalidState
	}
	_ = s.redis.Delete(ctx, key)
	return nil
}

// storeRefreshToken 将 refresh token 存入 Redis，关联用户和租户
func (s *OAuthService) storeRefreshToken(ctx context.Context, refreshToken, userID, tenantID string) error {
	if s.redis == nil || !s.redis.IsAvailable() {
		return nil
	}
	key := refreshTokenPrefix + refreshToken
	data := fmt.Sprintf("%s:%s", userID, tenantID)
	return s.redis.Set(ctx, key, data, s.cfg.GetRefreshTokenTTL())
}

// consumeRefreshToken 消费 refresh token，一次性使用，返回关联的用户和租户
func (s *OAuthService) consumeRefreshToken(ctx context.Context, refreshToken string) (userID, tenantID string, err error) {
	if s.redis == nil || !s.redis.IsAvailable() {
		return "", "", ErrRefreshTokenInvalid
	}
	key := refreshTokenPrefix + refreshToken
	val, err := s.redis.Get(ctx, key)
	if err != nil || val == "" {
		return "", "", ErrRefreshTokenInvalid
	}
	parts := strings.SplitN(val, ":", 2)
	if len(parts) != 2 {
		return "", "", ErrRefreshTokenInvalid
	}
	_ = s.redis.Delete(ctx, key)
	return parts[0], parts[1], nil
}

// generateRefreshToken 生成新的 refresh token
func (s *OAuthService) generateRefreshToken() string {
	return idgen.NewUUID()
}

// GetRedirectURL 获取 OAuth 提供商的授权重定向 URL
func (s *OAuthService) GetRedirectURL(ctx context.Context, provider string) (string, string, error) {
	state, err := generateState()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	var redirectURL string
	p, ok := s.registry.Get(provider)
	if !ok {
		return "", "", fmt.Errorf("unsupported provider: %s", provider)
	}
	if !p.IsEnabled() {
		return "", "", ErrOAuthProviderDisabled
	}
	redirectURL = p.RedirectURL(state)

	if err := s.saveState(ctx, state); err != nil {
		return "", "", fmt.Errorf("failed to save state: %w", err)
	}

	return redirectURL, state, nil
}

// Login OAuth 登录流程：验证 state → 交换 code → 查找或创建用户 → 生成令牌
func (s *OAuthService) Login(ctx context.Context, provider, code, state, tenantID string) (*userModel.User, []string, []string, string, string, bool, error) {
	if err := s.validateAndConsumeState(ctx, state); err != nil {
		return nil, nil, nil, "", "", false, err
	}

	var userInfo *dto.OAuthUserInfo
	var err error

	p, ok := s.registry.Get(provider)
	if !ok {
		return nil, nil, nil, "", "", false, fmt.Errorf("unsupported provider: %s", provider)
	}
	// 与 GetRedirectURL 口径一致：未启用（未配置/被关闭）的 provider 拒绝登录，
	// 避免用任意 code 触发服务端向第三方发起出站请求
	if !p.IsEnabled() {
		return nil, nil, nil, "", "", false, ErrOAuthProviderDisabled
	}
	userInfo, err = p.Exchange(ctx, code)

	if err != nil {
		return nil, nil, nil, "", "", false, err
	}
	// 拒绝空 ProviderID：否则会命中 oauth_accounts 中 provider_id='' 的历史记录，导致身份误匹配
	if userInfo.ProviderID == "" {
		return nil, nil, nil, "", "", false, ErrOAuthUserInfoFailed
	}

	user, isNew, err := s.findOrCreateUser(ctx, userInfo, tenantID)
	if err != nil {
		return nil, nil, nil, "", "", false, err
	}

	if isNew || s.needsOAuthAccountBinding(ctx, user.ID, userInfo.Provider) {
		s.bindOAuthAccount(ctx, user.ID, userInfo)
	}

	roleCodes, err := s.userRoleRepo.GetRoleCodesByUserID(ctx, user.ID)
	if err != nil {
		roleCodes = []string{}
	}
	// 主账号始终保证 JWT 携带 superadmin 角色，与 is_master 口径一致
	roleCodes = jwt.EnsureSuperadminRole(user.IsMaster, roleCodes)
	user.Roles = roleCodes

	permCodes := s.collectPermissionCodes(ctx, user.ID, user.IsMaster)

	token, err := s.tokenHelper.GenerateTokenEx(user.ID, user.TenantID, roleCodes, user.Email == "")
	if err != nil {
		return nil, nil, nil, "", "", false, err
	}

	refreshToken := s.generateRefreshToken()
	_ = s.storeRefreshToken(ctx, refreshToken, user.ID, user.TenantID)

	return user, roleCodes, permCodes, token, refreshToken, isNew, nil
}

// RefreshToken 使用 refresh token 换取新的 access token 和 refresh token
func (s *OAuthService) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	userID, _, err := s.consumeRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", "", err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return "", "", ErrRefreshTokenInvalid
	}
	if user.Status != 1 {
		return "", "", errors.New("user is disabled")
	}

	roleCodes, err := s.userRoleRepo.GetRoleCodesByUserID(ctx, user.ID)
	if err != nil {
		roleCodes = []string{}
	}
	// 刷新时同样补齐 superadmin，避免主账号刷新后丢失超管角色
	roleCodes = jwt.EnsureSuperadminRole(user.IsMaster, roleCodes)

	// 未补全邮箱的用户刷新时保持 email_pending 标记，防止刷新一次即绕过补全约束
	newToken, err := s.tokenHelper.GenerateTokenEx(user.ID, user.TenantID, roleCodes, user.Email == "")
	if err != nil {
		return "", "", err
	}

	newRefreshToken := s.generateRefreshToken()
	_ = s.storeRefreshToken(ctx, newRefreshToken, user.ID, user.TenantID)

	return newToken, newRefreshToken, nil
}

// GetTenantList 获取所有活跃租户列表，供 OAuth 登录时选择
func (s *OAuthService) GetTenantList(ctx context.Context) ([]dto.TenantOption, error) {
	tenants, err := s.tenantRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}

	options := make([]dto.TenantOption, 0, len(tenants))
	for _, t := range tenants {
		options = append(options, dto.TenantOption{
			ID:   t.ID,
			Name: t.Name,
		})
	}

	return options, nil
}

// ListOAuthAccounts 列出用户已绑定的所有 OAuth 账号
func (s *OAuthService) ListOAuthAccounts(ctx context.Context, userID string) ([]dto.OAuthAccountResp, error) {
	accounts, err := s.oauthAccountRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.OAuthAccountResp, 0, len(accounts))
	for _, a := range accounts {
		result = append(result, dto.OAuthAccountResp{
			ID:        a.ID,
			Provider:  a.Provider,
			Email:     a.Email,
			CreatedAt: a.CreatedAt,
		})
	}
	return result, nil
}

// UnbindOAuth 解绑指定 OAuth 提供商
func (s *OAuthService) UnbindOAuth(ctx context.Context, userID, provider string) error {
	account, err := s.oauthAccountRepo.GetByUserIDAndProvider(ctx, userID, provider)
	if err != nil {
		return ErrOAuthAccountNotFound
	}
	return s.oauthAccountRepo.DeleteByID(ctx, account.ID)
}

// BindOAuth 绑定 OAuth 提供商到已有用户
func (s *OAuthService) BindOAuth(ctx context.Context, userID, provider, code, state string) error {
	if err := s.validateAndConsumeState(ctx, state); err != nil {
		return err
	}

	// 先校验 provider 可用性（与 Login 口径一致），再访问仓储，
	// 避免未启用/不支持的 provider 白白触发仓储查询与出站交换
	p, ok := s.registry.Get(provider)
	if !ok {
		return fmt.Errorf("unsupported provider: %s", provider)
	}
	if !p.IsEnabled() {
		return ErrOAuthProviderDisabled
	}

	existing, _ := s.oauthAccountRepo.GetByUserIDAndProvider(ctx, userID, provider)
	if existing != nil {
		return ErrOAuthAccountAlreadyBound
	}

	userInfo, err := p.Exchange(ctx, code)
	if err != nil {
		return err
	}
	// 拒绝空 ProviderID，避免绑定到 provider_id='' 的脏数据账号
	if userInfo.ProviderID == "" {
		return ErrOAuthUserInfoFailed
	}

	existingBinding, _ := s.oauthAccountRepo.GetByProviderAndProviderID(ctx, provider, userInfo.ProviderID)
	if existingBinding != nil && existingBinding.UserID != userID {
		return errors.New("this provider account is already bound to another user")
	}

	s.bindOAuthAccount(ctx, userID, userInfo)
	return nil
}

// needsOAuthAccountBinding 检查用户是否需要绑定指定 OAuth 提供商
func (s *OAuthService) needsOAuthAccountBinding(ctx context.Context, userID, provider string) bool {
	_, err := s.oauthAccountRepo.GetByUserIDAndProvider(ctx, userID, provider)
	return err != nil
}

// bindOAuthAccount 绑定 OAuth 账号到用户，若已存在则跳过
func (s *OAuthService) bindOAuthAccount(ctx context.Context, userID string, info *dto.OAuthUserInfo) {
	existing, _ := s.oauthAccountRepo.GetByProviderAndProviderID(ctx, info.Provider, info.ProviderID)
	if existing != nil {
		return
	}

	account := &model.OAuthAccount{
		ID:          idgen.New(),
		UserID:      userID,
		Provider:    info.Provider,
		ProviderID:  info.ProviderID,
		Email:       info.Email,
		AccessToken: info.AccessToken,
	}
	_ = s.oauthAccountRepo.Create(ctx, account)
}

// collectPermissionCodes 收集用户的所有权限码，系统管理员返回空数组（前端通过 is_master 判断）
func (s *OAuthService) collectPermissionCodes(ctx context.Context, userID string, isMaster bool) []string {
	permCodes := make([]string, 0)
	if isMaster {
		return permCodes
	}
	roleIDs, err := s.userRoleRepo.GetRoleIDsByUserID(ctx, userID)
	if err != nil {
		return permCodes
	}
	seen := make(map[string]bool)
	for _, roleID := range roleIDs {
		codes, err := s.rolePermissionRepo.GetPermissionCodesByRoleID(ctx, roleID)
		if err != nil {
			continue
		}
		for _, c := range codes {
			if !seen[c] {
				seen[c] = true
				permCodes = append(permCodes, c)
			}
		}
	}
	return permCodes
}

// findOrCreateUser 查找已有 OAuth 账号对应用户，不存在则创建新用户并分配默认角色
func (s *OAuthService) findOrCreateUser(ctx context.Context, info *dto.OAuthUserInfo, tenantID string) (*userModel.User, bool, error) {
	existingAccount, err := s.oauthAccountRepo.GetByProviderAndProviderID(ctx, info.Provider, info.ProviderID)
	if err == nil && existingAccount != nil {
		user, err := s.userRepo.GetByID(ctx, existingAccount.UserID)
		if err == nil && user != nil {
			return user, false, nil
		}
	}

	if tenantID == "" {
		return nil, false, errors.New("tenant_id is required")
	}

	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, false, fmt.Errorf("invalid tenant_id: %w", err)
	}
	if tenant == nil || tenant.Status != 1 {
		return nil, false, errors.New("tenant not found or disabled")
	}

	// 仅在有邮箱时才按邮箱匹配已有账号：空串会命中任意 email='' 的用户，
	// 造成微信/QQ 等无邮箱 provider 的登录串号（越权登录他人账号）
	if info.Email != "" {
		user, err := s.userRepo.GetByEmail(ctx, info.Email)
		if err == nil && user != nil {
			return user, false, nil
		}
	}

	userID := idgen.New()
	now := idgen.New()

	// 用户名派生：优先邮箱前缀，无邮箱时用 ProviderID 前缀，避免生成 "wechat_" 这类退化名
	suffix := info.Email
	if suffix == "" {
		suffix = info.ProviderID
	}
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}

	newUser := &userModel.User{
		ID:       userID,
		TenantID: tenantID,
		Username: fmt.Sprintf("%s_%s", info.Provider, strings.ToLower(suffix)),
		Password: "",
		Nickname: info.Name,
		Email:    info.Email,
		Status:   1,
		IsMaster: false,
	}

	exists, _ := s.userRepo.UsernameExists(ctx, newUser.Username)
	if exists {
		newUser.Username = fmt.Sprintf("%s_%s", info.Provider, now)
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, false, fmt.Errorf("failed to create user: %w", err)
	}

	defaultRole := "tenant_user"
	role, err := s.roleRepo.GetByCode(ctx, tenantID, defaultRole)
	if err == nil && role != nil {
		_ = s.userRoleRepo.AssignRoles(ctx, newUser.ID, []string{role.ID})
	}

	return newUser, true, nil
}

// ListProviders 返回当前已启用（配置且开启）的第三方登录提供商列表，供前端渲染按钮。
func (s *OAuthService) ListProviders(ctx context.Context) []dto.ProviderMeta {
	metas := s.registry.Enabled()
	out := make([]dto.ProviderMeta, 0, len(metas))
	for _, m := range metas {
		out = append(out, dto.ProviderMeta{ID: m.ID, Label: m.Label})
	}
	return out
}

// CompleteEmail 第三方登录未返回邮箱时，补全邮箱并换发正式令牌。
// 调用方需携带登录时签发的（含 email_pending 声明）临时令牌于 Authorization 头。
func (s *OAuthService) CompleteEmail(ctx context.Context, bearerToken, email string) (string, string, error) {
	claims, err := s.tokenHelper.ParseToken(bearerToken)
	if err != nil || !claims.EmailPending {
		return "", "", errors.New("无效的邮箱补全令牌或已过期")
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil || user == nil {
		return "", "", errors.New("用户不存在")
	}

	if existing, _ := s.userRepo.GetByEmail(ctx, email); existing != nil && existing.ID != user.ID {
		return "", "", errors.New("该邮箱已被其他账号使用")
	}

	user.Email = email
	if err := s.userRepo.Update(ctx, user); err != nil {
		return "", "", err
	}

	roleCodes, _ := s.userRoleRepo.GetRoleCodesByUserID(ctx, user.ID)
	roleCodes = jwt.EnsureSuperadminRole(user.IsMaster, roleCodes)

	newToken, err := s.tokenHelper.GenerateToken(user.ID, user.TenantID, roleCodes)
	if err != nil {
		return "", "", err
	}
	newRefresh := s.generateRefreshToken()
	_ = s.storeRefreshToken(ctx, newRefresh, user.ID, user.TenantID)

	return newToken, newRefresh, nil
}
