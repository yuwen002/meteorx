// Package service 实现 OAuth 模块的业务逻辑，处理授权码交换、用户信息获取和账号绑定。
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"meteorx/pkg/idgen"
	"net/http"
	"net/url"
	"strings"
)

var (
	ErrOAuthProviderDisabled    = errors.New("OAuth provider not enabled")
	ErrOAuthExchangeFailed      = errors.New("failed to exchange authorization code")
	ErrOAuthUserInfoFailed      = errors.New("failed to get user info from provider")
	ErrOAuthInvalidState        = errors.New("invalid or expired OAuth state")
	ErrOAuthAccountNotFound     = errors.New("OAuth account not found")
	ErrOAuthAccountAlreadyBound = errors.New("this provider is already bound to your account")
	ErrOAuthEmailRequired       = errors.New("email is required for OAuth login")
	ErrRefreshTokenInvalid      = errors.New("invalid or expired refresh token")
)

const (
	oauthStatePrefix   = "oauth:state:"
	refreshTokenPrefix = "oauth:refresh:"
)

// OAuthService OAuth 认证服务，处理第三方登录、账号绑定和令牌刷新
type OAuthService struct {
	cfg                config.OAuthConfig
	userRepo           repository.UserRepository
	roleRepo           rbacRepo.RoleRepository
	userRoleRepo       rbacRepo.UserRoleRepository
	rolePermissionRepo rbacRepo.RolePermissionRepository
	tenantRepo         tenantRepo.TenantRepository
	oauthAccountRepo   oauthRepo.OAuthAccountRepository
	tokenHelper        *jwt.TokenHelper
	redis              *cache.Redis
}

// NewOAuthService 创建 OAuth 服务实例
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
) *OAuthService {
	return &OAuthService{
		cfg:                cfg,
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
	switch provider {
	case "google":
		if !s.cfg.Google.Enabled {
			return "", "", ErrOAuthProviderDisabled
		}
		redirectURL = s.buildGoogleRedirectURL(state)
	case "github":
		if !s.cfg.GitHub.Enabled {
			return "", "", ErrOAuthProviderDisabled
		}
		redirectURL = s.buildGitHubRedirectURL(state)
	default:
		return "", "", fmt.Errorf("unsupported provider: %s", provider)
	}

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

	switch provider {
	case "google":
		userInfo, err = s.exchangeGoogleCode(ctx, code)
	case "github":
		userInfo, err = s.exchangeGitHubCode(ctx, code)
	default:
		return nil, nil, nil, "", "", false, fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return nil, nil, nil, "", "", false, err
	}

	if userInfo.Email == "" {
		return nil, nil, nil, "", "", false, ErrOAuthEmailRequired
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
	user.Roles = roleCodes

	permCodes := s.collectPermissionCodes(ctx, user.ID, user.IsMaster)

	token, err := s.tokenHelper.GenerateToken(user.ID, user.TenantID, roleCodes)
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

	newToken, err := s.tokenHelper.GenerateToken(user.ID, user.TenantID, roleCodes)
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

	existing, _ := s.oauthAccountRepo.GetByUserIDAndProvider(ctx, userID, provider)
	if existing != nil {
		return ErrOAuthAccountAlreadyBound
	}

	var userInfo *dto.OAuthUserInfo
	var err error

	switch provider {
	case "google":
		userInfo, err = s.exchangeGoogleCode(ctx, code)
	case "github":
		userInfo, err = s.exchangeGitHubCode(ctx, code)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return err
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

	user, err := s.userRepo.GetByEmail(ctx, info.Email)
	if err == nil && user != nil {
		return user, false, nil
	}

	userID := idgen.New()
	now := idgen.New()

	newUser := &userModel.User{
		ID:       userID,
		TenantID: tenantID,
		Username: fmt.Sprintf("%s_%s", info.Provider, strings.ToLower(info.Email[:min(8, len(info.Email))])),
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

func (s *OAuthService) buildGoogleRedirectURL(state string) string {
	params := url.Values{}
	params.Set("client_id", s.cfg.Google.ClientID)
	params.Set("redirect_uri", s.cfg.Google.RedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("access_type", "online")
	params.Set("prompt", "select_account")
	params.Set("state", state)

	return fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?%s", params.Encode())
}

func (s *OAuthService) buildGitHubRedirectURL(state string) string {
	params := url.Values{}
	params.Set("client_id", s.cfg.GitHub.ClientID)
	params.Set("redirect_uri", s.cfg.GitHub.RedirectURL)
	params.Set("scope", "read:user user:email")
	params.Set("state", state)

	return fmt.Sprintf("https://github.com/login/oauth/authorize?%s", params.Encode())
}

func (s *OAuthService) exchangeGoogleCode(ctx context.Context, code string) (*dto.OAuthUserInfo, error) {
	tokenURL := "https://oauth2.googleapis.com/token"
	tokenData := url.Values{}
	tokenData.Set("code", code)
	tokenData.Set("client_id", s.cfg.Google.ClientID)
	tokenData.Set("client_secret", s.cfg.Google.ClientSecret)
	tokenData.Set("redirect_uri", s.cfg.Google.RedirectURL)
	tokenData.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(tokenURL, tokenData)
	if err != nil {
		return nil, ErrOAuthExchangeFailed
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrOAuthExchangeFailed, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, ErrOAuthExchangeFailed
	}

	userInfoURL := "https://www.googleapis.com/oauth2/v2/userinfo?alt=json"
	req, _ := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	client := &http.Client{}
	userResp, err := client.Do(req)
	if err != nil {
		return nil, ErrOAuthUserInfoFailed
	}
	defer userResp.Body.Close()

	userBody, _ := io.ReadAll(userResp.Body)
	if userResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrOAuthUserInfoFailed, string(userBody))
	}

	var googleUser struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(userBody, &googleUser); err != nil {
		return nil, ErrOAuthUserInfoFailed
	}

	return &dto.OAuthUserInfo{
		Provider:    "google",
		ProviderID:  googleUser.ID,
		Email:       googleUser.Email,
		Name:        googleUser.Name,
		AvatarURL:   googleUser.Picture,
		AccessToken: tokenResp.AccessToken,
	}, nil
}

func (s *OAuthService) exchangeGitHubCode(ctx context.Context, code string) (*dto.OAuthUserInfo, error) {
	tokenURL := "https://github.com/login/oauth/access_token"
	tokenData := url.Values{}
	tokenData.Set("code", code)
	tokenData.Set("client_id", s.cfg.GitHub.ClientID)
	tokenData.Set("client_secret", s.cfg.GitHub.ClientSecret)
	tokenData.Set("redirect_uri", s.cfg.GitHub.RedirectURL)

	req, _ := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(tokenData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, ErrOAuthExchangeFailed
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrOAuthExchangeFailed, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, ErrOAuthExchangeFailed
	}

	userInfoURL := "https://api.github.com/user"
	userReq, _ := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	userReq.Header.Set("Accept", "application/json")

	userResp, err := client.Do(userReq)
	if err != nil {
		return nil, ErrOAuthUserInfoFailed
	}
	defer userResp.Body.Close()

	userBody, _ := io.ReadAll(userResp.Body)
	if userResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrOAuthUserInfoFailed, string(userBody))
	}

	var githubUser struct {
		ID     int    `json:"id"`
		Email  string `json:"email"`
		Name   string `json:"name"`
		Login  string `json:"login"`
		Avatar string `json:"avatar_url"`
	}
	if err := json.Unmarshal(userBody, &githubUser); err != nil {
		return nil, ErrOAuthUserInfoFailed
	}

	email := githubUser.Email
	if email == "" {
		email = s.fetchGitHubPrimaryEmail(ctx, tokenResp.AccessToken)
	}

	name := githubUser.Name
	if name == "" {
		name = githubUser.Login
	}

	return &dto.OAuthUserInfo{
		Provider:    "github",
		ProviderID:  fmt.Sprintf("%d", githubUser.ID),
		Email:       email,
		Name:        name,
		AvatarURL:   githubUser.Avatar,
		AccessToken: tokenResp.AccessToken,
	}, nil
}

func (s *OAuthService) fetchGitHubPrimaryEmail(ctx context.Context, accessToken string) string {
	emailURL := "https://api.github.com/user/emails"
	req, _ := http.NewRequestWithContext(ctx, "GET", emailURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return ""
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	if len(emails) > 0 {
		return emails[0].Email
	}
	return ""
}
