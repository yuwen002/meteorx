package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"meteorx/internal/common/jwt"
	"meteorx/internal/config"
	"meteorx/internal/modules/oauth/dto"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	"meteorx/internal/modules/user/model"
	"meteorx/internal/modules/user/repository"
	"meteorx/pkg/idgen"
)

var (
	ErrOAuthProviderDisabled = errors.New("OAuth provider not enabled")
	ErrOAuthExchangeFailed   = errors.New("failed to exchange authorization code")
	ErrOAuthUserInfoFailed   = errors.New("failed to get user info from provider")
	ErrOAuthInvalidState     = errors.New("invalid OAuth state")
)

// OAuthService OAuth2 认证服务
type OAuthService struct {
	cfg                config.OAuthConfig
	userRepo           repository.UserRepository
	roleRepo           rbacRepo.RoleRepository
	userRoleRepo       rbacRepo.UserRoleRepository
	rolePermissionRepo rbacRepo.RolePermissionRepository
	tokenHelper        *jwt.TokenHelper
}

// NewOAuthService 创建 OAuth2 服务
func NewOAuthService(
	cfg config.OAuthConfig,
	userRepo repository.UserRepository,
	roleRepo rbacRepo.RoleRepository,
	userRoleRepo rbacRepo.UserRoleRepository,
	rolePermissionRepo rbacRepo.RolePermissionRepository,
	tokenHelper *jwt.TokenHelper,
) *OAuthService {
	return &OAuthService{
		cfg:                cfg,
		userRepo:           userRepo,
		roleRepo:           roleRepo,
		userRoleRepo:       userRoleRepo,
		rolePermissionRepo: rolePermissionRepo,
		tokenHelper:        tokenHelper,
	}
}

// GetRedirectURL 获取 OAuth2 授权跳转链接
func (s *OAuthService) GetRedirectURL(provider string) (string, error) {
	switch provider {
	case "google":
		if !s.cfg.Google.Enabled {
			return "", ErrOAuthProviderDisabled
		}
		return s.buildGoogleRedirectURL(), nil
	case "github":
		if !s.cfg.GitHub.Enabled {
			return "", ErrOAuthProviderDisabled
		}
		return s.buildGitHubRedirectURL(), nil
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

// Login 通过 OAuth2 授权码登录
func (s *OAuthService) Login(ctx context.Context, provider, code string) (*model.User, []string, []string, string, bool, error) {
	var userInfo *dto.OAuthUserInfo
	var err error

	switch provider {
	case "google":
		userInfo, err = s.exchangeGoogleCode(ctx, code)
	case "github":
		userInfo, err = s.exchangeGitHubCode(ctx, code)
	default:
		return nil, nil, nil, "", false, fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return nil, nil, nil, "", false, err
	}

	// 查找或创建用户
	user, isNew, err := s.findOrCreateUser(ctx, userInfo)
	if err != nil {
		return nil, nil, nil, "", false, err
	}

	// 获取用户角色编码
	roleCodes, err := s.userRoleRepo.GetRoleCodesByUserID(ctx, user.ID)
	if err != nil {
		roleCodes = []string{}
	}
	user.Roles = roleCodes

	// 获取权限列表（通过角色ID查询权限码）
	permCodes := make([]string, 0)
	roleIDs, err := s.userRoleRepo.GetRoleIDsByUserID(ctx, user.ID)
	if err == nil {
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
	}

	token, err := s.tokenHelper.GenerateToken(user.ID, user.TenantID, roleCodes)
	if err != nil {
		return nil, nil, nil, "", false, err
	}

	return user, roleCodes, permCodes, token, isNew, nil
}

// findOrCreateUser 查找已关联的 OAuth 用户，或创建新用户
func (s *OAuthService) findOrCreateUser(ctx context.Context, info *dto.OAuthUserInfo) (*model.User, bool, error) {
	// 先尝试通过邮箱查找用户
	user, err := s.userRepo.GetByEmail(ctx, info.Email)
	if err == nil && user != nil {
		// 已存在用户，直接返回
		return user, false, nil
	}

	// 创建新用户
	userID := idgen.New()
	now := idgen.New() // 用于生成默认用户名

	newUser := &model.User{
		ID:       userID,
		TenantID: "default", // 默认租户
		Username: fmt.Sprintf("%s_%s", info.Provider, strings.ToLower(info.Email[:min(8, len(info.Email))])),
		Password: "", // OAuth 用户无需密码
		Nickname: info.Name,
		Email:    info.Email,
		Status:   1,
		IsMaster: false,
	}

	// 处理用户名冲突
	exists, _ := s.userRepo.UsernameExists(ctx, newUser.Username)
	if exists {
		newUser.Username = fmt.Sprintf("%s_%s", info.Provider, now)
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		return nil, false, fmt.Errorf("failed to create user: %w", err)
	}

	// 分配默认角色
	defaultRole := "tenant_user"
	role, err := s.roleRepo.GetByCode(ctx, "", defaultRole)
	if err == nil && role != nil {
		_ = s.userRoleRepo.AssignRoles(ctx, newUser.ID, []string{role.ID})
	}

	return newUser, true, nil
}

// buildGoogleRedirectURL 构建 Google OAuth2 跳转链接
func (s *OAuthService) buildGoogleRedirectURL() string {
	params := url.Values{}
	params.Set("client_id", s.cfg.Google.ClientID)
	params.Set("redirect_uri", s.cfg.Google.RedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("access_type", "online")
	params.Set("prompt", "select_account")

	return fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?%s", params.Encode())
}

// buildGitHubRedirectURL 构建 GitHub OAuth2 跳转链接
func (s *OAuthService) buildGitHubRedirectURL() string {
	params := url.Values{}
	params.Set("client_id", s.cfg.GitHub.ClientID)
	params.Set("redirect_uri", s.cfg.GitHub.RedirectURL)
	params.Set("scope", "read:user user:email")

	return fmt.Sprintf("https://github.com/login/oauth/authorize?%s", params.Encode())
}

// exchangeGoogleCode 用授权码换取 Google 用户信息
func (s *OAuthService) exchangeGoogleCode(ctx context.Context, code string) (*dto.OAuthUserInfo, error) {
	// 交换 token
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

	// 获取用户信息
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

// exchangeGitHubCode 用授权码换取 GitHub 用户信息
func (s *OAuthService) exchangeGitHubCode(ctx context.Context, code string) (*dto.OAuthUserInfo, error) {
	// 交换 token
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

	// 获取用户信息
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
		ID      int    `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Login   string `json:"login"`
		Avatar  string `json:"avatar_url"`
	}
	if err := json.Unmarshal(userBody, &githubUser); err != nil {
		return nil, ErrOAuthUserInfoFailed
	}

	// GitHub 可能不返回 email，需要额外请求
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

// fetchGitHubPrimaryEmail 获取 GitHub 用户的主邮箱
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