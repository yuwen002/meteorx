// Package service 实现认证模块的业务逻辑，包括登录/登出、Token 签发刷新及 API Token 管理。
package service

import (
	"context"
	"errors"
	"fmt"
	"meteorx/internal/config"
	"meteorx/pkg/security"
	"meteorx/pkg/totp"
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

const (
	// totpPendingPrefix 双因素认证“待绑定”密钥在 Redis 中的键前缀（Setup 生成但未确认启用前暂存）。
	totpPendingPrefix = "2fa:pending:"
	// totpPendingTTL 待绑定密钥有效期，超时未确认则需重新发起 Setup。
	totpPendingTTL = 15 * time.Minute
	// totpIssuer otpauth URI 中的发行方名称。
	totpIssuer = "MeteorX"
)

var (
	ErrEmailNotConfigured   = errors.New("email service not configured")
	ErrInvalidResetToken    = errors.New("invalid or expired token")
	ErrUserNotFound         = errors.New("user not found")
	ErrEmailAlreadyVerified = errors.New("email already verified")
	ErrEmailNotOwned        = errors.New("email does not belong to current user")
	// 双因素认证相关错误
	Err2FAAlreadyEnabled     = errors.New("two-factor authentication is already enabled")
	Err2FANotEnabled         = errors.New("two-factor authentication is not enabled")
	Err2FAInvalidCode        = errors.New("invalid two-factor code")
	Err2FASetupExpired       = errors.New("two-factor setup expired, please start over")
	Err2FAStorageUnavailable = errors.New("two-factor verification storage unavailable")
)

// LoginError 登录错误（包含安全信息）
type LoginError struct {
	Message           string
	RemainingAttempts int
	Locked            bool
	LockoutDuration   int64
	Require2FA        bool
}

// Error 实现 error 接口，返回登录错误描述信息。
func (e *LoginError) Error() string {
	return e.Message
}

// TenantStatusProvider 租户状态查询能力（由租户仓储提供），用于登录时拦截被平台暂停的租户。
// 未注入时跳过校验，保持向后兼容；查询失败（如租户不存在）按 fail-open 放行。
type TenantStatusProvider interface {
	GetStatus(ctx context.Context, tenantID string) (int, error)
}

// AuthServiceOption 服务层可变参数选项，用于按需注入可选依赖。
type AuthServiceOption func(*AuthService)

// WithTenantStatusProvider 注入租户状态查询依赖（登录软着陆拦截）。
func WithTenantStatusProvider(p TenantStatusProvider) AuthServiceOption {
	return func(s *AuthService) { s.tenantStatus = p }
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
	tenantStatus       TenantStatusProvider
}

// NewAuthService 创建认证服务实例
func NewAuthService(ur repository.UserRepository, rr rbacRepo.RoleRepository, urr rbacRepo.UserRoleRepository, rpr rbacRepo.RolePermissionRepository, th *jwt.TokenHelper, redis *cache.Redis, securityCfg config.SecurityConfig, emailCfg config.EmailConfig, clientCfg config.ClientConfig, opts ...AuthServiceOption) *AuthService {
	var em *emailer.Emailer
	if emailCfg.Enabled {
		em = emailer.NewEmailer(emailCfg.Host, emailCfg.Port, emailCfg.Username, emailCfg.Password, emailCfg.From, emailCfg.FromName)
	}

	svc := &AuthService{
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
	for _, opt := range opts {
		opt(svc)
	}
	return svc
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
		Email:     req.Email,
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

	// 租户软着陆拦截：租户被平台暂停/禁用时拒绝签发 token（查询失败 fail-open，未注入时跳过）
	if s.tenantStatus != nil && user.TenantID != "" {
		status, serr := s.tenantStatus.GetStatus(ctx, user.TenantID)
		if serr == nil && status == 0 {
			return nil, nil, nil, "", errors.New("所属租户已被平台暂停，请联系管理员处理")
		}
	}

	// 双因素认证校验：账号已启用 TOTP 时，密码正确后需进一步校验动态验证码。
	if user.TOTPEnabled {
		if req.OTP == "" {
			// 未提供验证码：提示前端弹出验证码输入框（不消耗锁定计数，密码已正确）
			return nil, nil, nil, "", &LoginError{Message: "请输入双因素验证码", Require2FA: true}
		}
		ok, verr := totp.Validate(user.TOTPSecret, req.OTP)
		if verr != nil || !ok {
			// 验证码错误：计入失败次数，抵御 6 位口令暴力猜测
			_ = s.lockout.RecordFailedAttempt(ctx, lockoutKey)
			remaining := s.lockout.GetRemainingAttempts(ctx, lockoutKey)
			return nil, nil, nil, "", &LoginError{
				Message:           "双因素验证码错误",
				RemainingAttempts: remaining,
				Require2FA:        true,
			}
		}
	}

	// 回写最后登录时间（best-effort，失败不阻断登录）
	now := time.Now()
	user.LastLoginAt = &now
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	// 查询用户关联的角色编码列表
	roleCodes, err := s.userRoleRepo.GetRoleCodesByUserID(ctx, user.ID)
	if err != nil {
		return nil, nil, nil, "", errors.New("failed to query user roles")
	}

	// 主账号始终保证 JWT 携带 superadmin 角色（无论是否被显式分配了其他角色），
	// 使 RequiresMasterAdmin 等基于角色的判定与 is_master 口径一致。
	roleCodes = jwt.EnsureSuperadminRole(user.IsMaster, roleCodes)

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
	if claims.ExpiresAt != nil && claims.ExpiresAt.After(now) {
		expiration = claims.ExpiresAt.Sub(now)
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
// currentUserID 用于验证邮箱归属，只能验证当前登录用户自己的邮箱。
func (s *AuthService) SendEmailVerification(ctx context.Context, email string, currentUserID string) error {
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

	if user.ID != currentUserID {
		return ErrEmailNotOwned
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

// Setup2FA 开启双因素认证第一步：生成随机密钥并暂存于 Redis（尚未生效），
// 返回密钥与 otpauth URI 供前端生成二维码，用户需用验证器 App 扫码后调用 Enable2FA 确认。
func (s *AuthService) Setup2FA(ctx context.Context, userID string) (*dto.Setup2FAResp, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}
	if user.TOTPEnabled {
		return nil, Err2FAAlreadyEnabled
	}

	secret, err := totp.GenerateSecret()
	if err != nil {
		return nil, err
	}

	if s.redis == nil || !s.redis.IsAvailable() {
		return nil, Err2FAStorageUnavailable
	}
	key := totpPendingPrefix + userID
	if err := s.redis.Set(ctx, key, secret, totpPendingTTL); err != nil {
		if errors.Is(err, cache.ErrRedisUnavailable) {
			return nil, Err2FAStorageUnavailable
		}
		return nil, err
	}

	account := user.Email
	if account == "" {
		account = user.Username
	}
	return &dto.Setup2FAResp{Secret: secret, OTPAuthURL: totp.OTPAuthURL(secret, totpIssuer, account)}, nil
}

// Enable2FA 确认绑定双因素认证：校验验证器口令，通过后持久化密钥并置为启用，清除待绑定缓存。
func (s *AuthService) Enable2FA(ctx context.Context, userID, code string) error {
	if s.redis == nil || !s.redis.IsAvailable() {
		return Err2FAStorageUnavailable
	}
	key := totpPendingPrefix + userID
	secret, err := s.redis.Get(ctx, key)
	if errors.Is(err, cache.ErrRedisUnavailable) {
		return Err2FAStorageUnavailable
	}
	if err != nil || secret == "" {
		return Err2FASetupExpired
	}

	ok, verr := totp.Validate(secret, code)
	if verr != nil {
		return Err2FAInvalidCode
	}
	if !ok {
		return Err2FAInvalidCode
	}

	if err := s.userRepo.UpdateTOTP(ctx, userID, secret, true); err != nil {
		return err
	}
	_ = s.redis.Delete(ctx, key)
	return nil
}

// Disable2FA 关闭双因素认证：需当前有效验证码，校验通过后清除密钥并置为未启用。
func (s *AuthService) Disable2FA(ctx context.Context, userID, code string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return ErrUserNotFound
	}
	if !user.TOTPEnabled {
		return Err2FANotEnabled
	}

	ok, verr := totp.Validate(user.TOTPSecret, code)
	if verr != nil || !ok {
		return Err2FAInvalidCode
	}

	return s.userRepo.UpdateTOTP(ctx, userID, "", false)
}
