package service

import (
	"context"
	"errors"
	"fmt"
	"meteorx/internal/config"
	"meteorx/pkg/security"
	"time"

	"meteorx/internal/cache"
	"meteorx/internal/common/jwt"
	"meteorx/internal/modules/auth/dto"
	rbacRepo "meteorx/internal/modules/rbac/repository"
	"meteorx/internal/modules/user/model"
	"meteorx/internal/modules/user/repository"
	"meteorx/internal/pkg/emailer"
	"meteorx/pkg/crypto"
	"meteorx/pkg/idgen"
)

const (
	passwordResetPrefix = "password:reset:"
	resetTokenExpire    = 30 * time.Minute
	// emailVerifyPrefix 邮箱验证令牌的 Redis 键前缀
	emailVerifyPrefix = "email:verify:"
	// emailVerifyExpire 邮箱验证令牌有效期（24 小时）
	emailVerifyExpire = 24 * time.Hour
)

const (
	tokenBlacklistPrefix = "token:blacklist:"
)

var (
	ErrEmailNotConfigured = errors.New("email service not configured")
	ErrInvalidResetToken  = errors.New("invalid or expired token")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyVerified = errors.New("email already verified")
)

// LoginError 登录错误（包含安全信息）
type LoginError struct {
	Message           string
	RemainingAttempts int
	Locked            bool
	LockoutDuration   int64
}

func (e *LoginError) Error() string {
	return e.Message
}

// AuthService 认证服务，处理注册、登录、登出、密码重置等业务逻辑
type AuthService struct {
	userRepo           repository.UserRepository
	roleRepo           rbacRepo.RoleRepository
	userRoleRepo       rbacRepo.UserRoleRepository
	rolePermissionRepo rbacRepo.RolePermissionRepository
	tokenHelper        *jwt.TokenHelper
	redis              *cache.Redis
	securityCfg        config.SecurityConfig
	lockout            *security.LoginLockout
	emailer            *emailer.Emailer
	clientBaseURL      string
	emailEnabled       bool
}

// NewAuthService 创建认证服务实例
func NewAuthService(ur repository.UserRepository, rr rbacRepo.RoleRepository, urr rbacRepo.UserRoleRepository, rpr rbacRepo.RolePermissionRepository, th *jwt.TokenHelper, redis *cache.Redis, securityCfg config.SecurityConfig, emailCfg config.EmailConfig, clientCfg config.ClientConfig) *AuthService {
	var em *emailer.Emailer
	if emailCfg.Enabled {
		em = emailer.NewEmailer(emailCfg.Host, emailCfg.Port, emailCfg.Username, emailCfg.Password, emailCfg.From, emailCfg.FromName)
	}

	return &AuthService{
		userRepo:           ur,
		roleRepo:           rr,
		userRoleRepo:       urr,
		rolePermissionRepo: rpr,
		tokenHelper:        th,
		redis:              redis,
		securityCfg:        securityCfg,
		lockout:            security.NewLoginLockout(redis, securityCfg.LoginLockout),
		emailer:            em,
		clientBaseURL:      clientCfg.BaseURL,
		emailEnabled:       emailCfg.Enabled,
	}
}

// Register 用户注册，校验密码策略并分配默认角色
func (s *AuthService) Register(ctx context.Context, req dto.RegisterUserReq) (*model.User, error) {
	// 校验密码策略
	if err := security.ValidatePassword(req.Password, s.securityCfg.PasswordPolicy); err != nil {
		return nil, err
	}

	// 默认角色为 tenant_admin，校验其是否存在且启用
	defaultRole := "tenant_admin"
	role, err := s.roleRepo.GetByCode(ctx, "", defaultRole)
	if err != nil {
		return nil, errors.New("默认注册角色未配置，请联系管理员初始化角色")
	}
	if role.Status == 0 {
		return nil, errors.New("默认注册角色已禁用，请联系管理员")
	}

	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &model.User{
		ID:        idgen.NewUUID(),
		TenantID:  req.TenantID,
		Username:  req.Username,
		Password:  hashedPassword,
		Nickname:  req.Nickname,
		Roles:     []string{defaultRole},
		Status:    1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// 创建用户后分配角色
	if err := s.userRoleRepo.AssignRoles(ctx, user.ID, []string{role.ID}); err != nil {
		return nil, err
	}

	return user, nil
}

// LoginResult 登录结果（包含安全信息）
type LoginResult struct {
	User              *model.User
	Roles             []string
	Permissions       []string
	Token             string
	RemainingAttempts int
	Locked            bool
	LockoutDuration   int64
	Error             error
}

// Login 用户登录，包含锁定检查、密码验证、角色/权限收集和 token 生成
func (s *AuthService) Login(ctx context.Context, req dto.LoginReq) (*model.User, []string, []string, string, error) {
	lockoutKey := req.Username + ":" + req.TenantID

	// 检查账号是否被锁定
	locked, err := s.lockout.IsLocked(ctx, lockoutKey)
	if err != nil {
		return nil, nil, nil, "", errors.New("登录安全检查失败")
	}
	if locked {
		duration := s.lockout.GetLockoutDuration(ctx, lockoutKey)
		return nil, nil, nil, "", &LoginError{
			Message:         "账号已被锁定，请稍后再试",
			Locked:          true,
			LockoutDuration: duration,
		}
	}

	user, err := s.userRepo.GetByUsername(ctx, req.TenantID, req.Username)
	if err != nil {
		_ = s.lockout.RecordFailedAttempt(ctx, lockoutKey)
		remaining := s.lockout.GetRemainingAttempts(ctx, lockoutKey)
		return nil, nil, nil, "", &LoginError{
			Message:           "用户名或密码错误",
			RemainingAttempts: remaining,
		}
	}

	if !crypto.CheckPassword(req.Password, user.Password) {
		_ = s.lockout.RecordFailedAttempt(ctx, lockoutKey)
		remaining := s.lockout.GetRemainingAttempts(ctx, lockoutKey)
		return nil, nil, nil, "", &LoginError{
			Message:           "用户名或密码错误",
			RemainingAttempts: remaining,
		}
	}

	// 登录成功，清除失败计数
	_ = s.lockout.RecordSuccessAttempt(ctx, lockoutKey)

	if user.Status == 0 {
		return nil, nil, nil, "", errors.New("账号已被禁用")
	}

	// 查询用户关联的角色编码列表
	roleCodes, err := s.userRoleRepo.GetRoleCodesByUserID(ctx, user.ID)
	if err != nil {
		return nil, nil, nil, "", errors.New("failed to query user roles")
	}

	// 如果是系统管理员且没有角色，兜底返回 superadmin
	if user.IsMaster && len(roleCodes) == 0 {
		roleCodes = []string{"superadmin"}
	}

	user.Roles = roleCodes

	// 查询用户关联的所有权限码（通过角色 -> 权限关联表）
	permissionCodes := make([]string, 0)
	if user.IsMaster {
		// 系统管理员兜底：返回空数组，前端通过 is_master 标志判断拥有所有权限
	} else {
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
						permissionCodes = append(permissionCodes, c)
					}
				}
			}
		}
	}

	token, err := s.tokenHelper.GenerateToken(user.ID, user.TenantID, roleCodes)
	if err != nil {
		return nil, nil, nil, "", errors.New("failed to generate token")
	}

	return user, roleCodes, permissionCodes, token, nil
}

// Logout 用户登出，将 token 加入黑名单
// 返回 userID、username 和 tenantID 供审计日志使用。
func (s *AuthService) Logout(ctx context.Context, tokenString string) (userID, username, tenantID string, err error) {
	// 解析 token
	claims, err := s.tokenHelper.ParseToken(tokenString)
	if err != nil {
		return "", "", "", nil
	}
	userID = claims.UserID
	tenantID = claims.TenantID

	// 查用户名
	if userID != "" {
		if u, lookupErr := s.userRepo.GetByID(ctx, userID); lookupErr == nil && u != nil {
			username = u.Username
		}
	}

	if s.redis == nil || !s.redis.IsAvailable() {
		return userID, username, tenantID, nil
	}

	// 计算 token 剩余有效期
	now := time.Now()
	var expiration time.Duration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.After(now) {
		expiration = claims.ExpiresAt.Time.Sub(now)
	} else {
		return userID, username, tenantID, nil
	}

	// 将 token 加入黑名单，有效期与 token 剩余有效期相同
	key := fmt.Sprintf("%s%s", tokenBlacklistPrefix, tokenString)
	e := s.redis.Set(ctx, key, "1", expiration)
	if errors.Is(e, cache.ErrRedisUnavailable) {
		return userID, username, tenantID, nil
	}
	return userID, username, tenantID, e
}

// IsTokenBlacklisted 检查 token 是否在黑名单中
func (s *AuthService) IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error) {
	if s.redis == nil || !s.redis.IsAvailable() {
		return false, nil
	}

	key := fmt.Sprintf("%s%s", tokenBlacklistPrefix, tokenString)
	result, err := s.redis.Exists(ctx, key)
	if errors.Is(err, cache.ErrRedisUnavailable) {
		return false, nil
	}
	return result, err
}

// ForgotPassword 发起密码重置，生成重置令牌并通过邮件发送
func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil
	}
	if user == nil {
		return nil
	}

	if !s.emailEnabled {
		return ErrEmailNotConfigured
	}

	if s.redis == nil || !s.redis.IsAvailable() {
		return errors.New("reset token storage unavailable")
	}

	token := idgen.NewUUID()

	key := fmt.Sprintf("%s%s", passwordResetPrefix, token)
	if err := s.redis.Set(ctx, key, user.ID, resetTokenExpire); err != nil {
		if errors.Is(err, cache.ErrRedisUnavailable) {
			return errors.New("reset token storage unavailable")
		}
		return err
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.clientBaseURL, token)

	return s.emailer.SendResetPasswordEmail(user.Email, resetLink, user.Username)
}

// ResetPassword 通过重置令牌设置新密码
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if s.redis == nil || !s.redis.IsAvailable() {
		return errors.New("redis not initialized")
	}

	key := fmt.Sprintf("%s%s", passwordResetPrefix, token)
	userID, err := s.redis.Get(ctx, key)
	if errors.Is(err, cache.ErrRedisUnavailable) {
		return errors.New("reset token storage unavailable")
	}
	if err != nil || userID == "" {
		return ErrInvalidResetToken
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if user == nil {
		return ErrUserNotFound
	}

	if err := security.ValidatePassword(newPassword, s.securityCfg.PasswordPolicy); err != nil {
		return err
	}

	hashedPassword, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	if err := s.redis.Delete(ctx, key); err != nil && !errors.Is(err, cache.ErrRedisUnavailable) {
		return err
	}

	return nil
}

// SendEmailVerification 发送邮箱验证链接
func (s *AuthService) SendEmailVerification(ctx context.Context, email string) error {
	if !s.emailEnabled {
		return ErrEmailNotConfigured
	}

	if s.redis == nil || !s.redis.IsAvailable() {
		return errors.New("verification token storage unavailable")
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil
	}
	if user == nil {
		return nil
	}

	if user.EmailVerified {
		return ErrEmailAlreadyVerified
	}

	token := idgen.NewUUID()
	key := fmt.Sprintf("%s%s", emailVerifyPrefix, token)
	if err := s.redis.Set(ctx, key, user.ID, emailVerifyExpire); err != nil {
		if errors.Is(err, cache.ErrRedisUnavailable) {
			return errors.New("verification token storage unavailable")
		}
		return err
	}

	verifyLink := fmt.Sprintf("%s/verify-email?token=%s", s.clientBaseURL, token)
	return s.emailer.SendEmailVerificationLink(email, verifyLink, user.Nickname)
}

// VerifyEmail 通过验证令牌确认邮箱
func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	if s.redis == nil || !s.redis.IsAvailable() {
		return errors.New("redis not initialized")
	}

	key := fmt.Sprintf("%s%s", emailVerifyPrefix, token)
	userID, err := s.redis.Get(ctx, key)
	if errors.Is(err, cache.ErrRedisUnavailable) {
		return errors.New("verification token storage unavailable")
	}
	if err != nil || userID == "" {
		return ErrInvalidResetToken
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}
	if user == nil {
		return ErrUserNotFound
	}

	if err := s.userRepo.UpdateEmailVerified(ctx, userID, true); err != nil {
		return err
	}

	if err := s.redis.Delete(ctx, key); err != nil && !errors.Is(err, cache.ErrRedisUnavailable) {
		return err
	}

	return nil
}