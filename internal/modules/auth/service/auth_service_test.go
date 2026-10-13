package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"meteorx/internal/cache"
	"meteorx/internal/common/jwt"
	"meteorx/internal/config"
	"meteorx/internal/modules/auth/dto"
	rbacModel "meteorx/internal/modules/rbac/model"
	userModel "meteorx/internal/modules/user/model"
	"meteorx/pkg/crypto"
	"meteorx/pkg/totp"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/suite"
)

// testTOTPSecret RFC 6238 标准测试密钥（Base32），用于确定性生成验证码。
const testTOTPSecret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

// AuthServiceTestSuite 认证服务测试套件
type AuthServiceTestSuite struct {
	suite.Suite
	svc      *AuthService
	userRepo *mockUserRepo
	roleRepo *mockRoleRepo
	urRepo   *mockUserRoleRepo
	rpRepo   *mockRolePermRepo
	ctx      context.Context
}

func (s *AuthServiceTestSuite) SetupTest() {
	s.userRepo = newMockUserRepo()
	s.roleRepo = newMockRoleRepo()
	s.urRepo = newMockUserRoleRepo()
	s.rpRepo = newMockRolePermRepo()

	securityCfg := config.SecurityConfig{
		LoginLockout: config.LoginLockoutConfig{Enabled: false},
		PasswordPolicy: config.PasswordPolicyConfig{
			Enabled:      true,
			MinLength:    8,
			MaxLength:    32,
			RequireDigit: true,
		},
	}

	s.svc = NewAuthService(
		s.userRepo,
		s.roleRepo,
		s.urRepo,
		s.rpRepo,
		jwt.NewTokenHelper(config.JWTConfig{Secret: "test-secret-key", Expiration: "24h", Issuer: "meteorx-test"}),
		nil, // redis 测试中置空，登录锁定功能关闭
		securityCfg,
		config.EmailConfig{Enabled: false},
		config.ClientConfig{BaseURL: "http://localhost"},
	)
	s.ctx = context.Background()
}

func (s *AuthServiceTestSuite) seedRole(code string, status int) *rbacModel.Role {
	r := &rbacModel.Role{
		ID:       "role-" + code,
		Name:     code,
		Code:     code,
		TenantID: "",
		Status:   status,
	}
	s.roleRepo.seed(r)
	return r
}

func (s *AuthServiceTestSuite) seedUser(username, password string, isMaster bool, status int) *userModel.User {
	hash, err := crypto.HashPassword(password)
	s.Require().NoError(err)
	u := &userModel.User{
		ID:       "user-" + username,
		TenantID: "tenant-001",
		Username: username,
		Password: hash,
		Email:    username + "@example.com",
		Status:   status,
		IsMaster: isMaster,
	}
	s.userRepo.seed(u)
	return u
}

func TestAuthServiceSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceTestSuite))
}

// ---------- Register 注册 ----------

func (s *AuthServiceTestSuite) TestRegisterSuccess() {
	role := s.seedRole("tenant_admin", 1)

	user, err := s.svc.Register(s.ctx, dto.RegisterUserReq{
		TenantID: "tenant-001",
		Username: "alice",
		Password: "Passw0rd123!",
		Nickname: "Alice",
		Email:    "alice@example.com",
	})

	s.NoError(err)
	s.Require().NotNil(user)
	s.Equal("alice", user.Username)
	s.Equal(1, user.Status)
	s.Equal([]string{"tenant_admin"}, user.Roles)
	// 密码必须被哈希存储
	s.NotEqual("Passw0rd123!", user.Password)
	s.True(crypto.CheckPassword("Passw0rd123!", user.Password))
	// 已分配默认角色
	s.Contains(s.urRepo.assigned, user.ID)
	s.Equal([]string{role.ID}, s.urRepo.userRoles[user.ID])
}

func (s *AuthServiceTestSuite) TestRegisterWeakPassword() {
	s.seedRole("tenant_admin", 1)

	_, err := s.svc.Register(s.ctx, dto.RegisterUserReq{
		TenantID: "tenant-001",
		Username: "alice",
		Password: "short",
		Nickname: "Alice",
		Email:    "alice@example.com",
	})

	s.Error(err)
	// 弱密码不应创建用户
	s.Empty(s.userRepo.users)
	s.Empty(s.urRepo.assigned)
}

func (s *AuthServiceTestSuite) TestRegisterDefaultRoleMissing() {
	_, err := s.svc.Register(s.ctx, dto.RegisterUserReq{
		TenantID: "tenant-001",
		Username: "alice",
		Password: "Passw0rd123!",
		Nickname: "Alice",
		Email:    "alice@example.com",
	})

	s.Error(err)
	s.Contains(err.Error(), "默认注册角色未配置")
	s.Empty(s.userRepo.users)
}

func (s *AuthServiceTestSuite) TestRegisterDefaultRoleDisabled() {
	s.seedRole("tenant_admin", 0)

	_, err := s.svc.Register(s.ctx, dto.RegisterUserReq{
		TenantID: "tenant-001",
		Username: "alice",
		Password: "Passw0rd123!",
		Nickname: "Alice",
		Email:    "alice@example.com",
	})

	s.Error(err)
	s.Contains(err.Error(), "默认注册角色已禁用")
	s.Empty(s.userRepo.users)
}

func (s *AuthServiceTestSuite) TestRegisterUserCreateFailure() {
	s.seedRole("tenant_admin", 1)
	s.userRepo.createErr = errors.New("db down")

	_, err := s.svc.Register(s.ctx, dto.RegisterUserReq{
		TenantID: "tenant-001",
		Username: "alice",
		Password: "Passw0rd123!",
		Nickname: "Alice",
		Email:    "alice@example.com",
	})

	s.Error(err)
	s.Empty(s.urRepo.assigned)
}

// ---------- Login 登录 ----------

func (s *AuthServiceTestSuite) TestLoginSuccess() {
	s.seedUser("alice", "Passw0rd123!", false, 1)
	s.urRepo.seedCodes("user-alice", []string{"editor"})

	u, roles, perms, token, err := s.svc.Login(s.ctx, dto.LoginReq{
		TenantID: "tenant-001",
		Username: "alice",
		Password: "Passw0rd123!",
	})

	s.NoError(err)
	s.Require().NotNil(u)
	s.Equal([]string{"editor"}, roles)
	s.NotEmpty(token)
	s.Empty(perms)
}

// TestLoginSuccess_UpdatesLastLogin 登录成功后应回写最后登录时间。
func (s *AuthServiceTestSuite) TestLoginSuccess_UpdatesLastLogin() {
	seeded := s.seedUser("alice", "Passw0rd123!", false, 1)
	s.urRepo.seedCodes("user-alice", []string{"editor"})
	// 初始无登录时间
	s.Nil(seeded.LastLoginAt)

	u, _, _, _, err := s.svc.Login(s.ctx, dto.LoginReq{
		TenantID: "tenant-001",
		Username: "alice",
		Password: "Passw0rd123!",
	})

	s.NoError(err)
	s.Require().NotNil(u)
	// 服务应对登录用户调用一次 UpdateLastLogin，并将 LastLoginAt 写入
	s.Equal([]string{"user-alice"}, s.userRepo.lastLoginIDs)
	s.NotNil(s.userRepo.users["user-alice"].LastLoginAt)
}

// TestLoginFailure_DoesNotUpdateLastLogin 密码错误时不应回写登录时间。
func (s *AuthServiceTestSuite) TestLoginFailure_DoesNotUpdateLastLogin() {
	s.seedUser("alice", "Passw0rd123!", false, 1)

	_, _, _, _, err := s.svc.Login(s.ctx, dto.LoginReq{
		TenantID: "tenant-001",
		Username: "alice",
		Password: "wrong-password",
	})

	s.Error(err)
	s.Empty(s.userRepo.lastLoginIDs)
}

func (s *AuthServiceTestSuite) TestLoginMasterFallbackRole() {
	s.seedUser("root", "Passw0rd123!", true, 1)

	u, roles, _, token, err := s.svc.Login(s.ctx, dto.LoginReq{
		TenantID: "",
		Username: "root",
		Password: "Passw0rd123!",
	})

	s.NoError(err)
	s.Require().NotNil(u)
	s.Equal([]string{"superadmin"}, roles)
	s.NotEmpty(token)
}

// TestLoginMasterWithAssignedRolesKeepsSuperadmin 主账号被显式分配其他角色后，
// JWT 角色仍需包含 superadmin（与前端 is_master 口径一致，锁死回归）。
func (s *AuthServiceTestSuite) TestLoginMasterWithAssignedRolesKeepsSuperadmin() {
	s.seedUser("root2", "Passw0rd123!", true, 1)
	s.urRepo.seedCodes("user-root2", []string{"editor"})

	_, roles, _, _, err := s.svc.Login(s.ctx, dto.LoginReq{
		TenantID: "",
		Username: "root2",
		Password: "Passw0rd123!",
	})

	s.NoError(err)
	s.Equal([]string{"superadmin", "editor"}, roles)
}

func (s *AuthServiceTestSuite) TestLoginWrongPassword() {
	s.seedUser("alice", "Passw0rd123!", false, 1)

	_, _, _, _, err := s.svc.Login(s.ctx, dto.LoginReq{
		TenantID: "tenant-001",
		Username: "alice",
		Password: "WrongPass123!",
	})

	var loginErr *LoginError
	s.Error(err)
	s.True(errors.As(err, &loginErr))
	s.Equal("用户名或密码错误", loginErr.Message)
}

func (s *AuthServiceTestSuite) TestLoginUserNotFound() {
	_, _, _, _, err := s.svc.Login(s.ctx, dto.LoginReq{
		TenantID: "tenant-001",
		Username: "ghost",
		Password: "Passw0rd123!",
	})

	var loginErr *LoginError
	s.Error(err)
	s.True(errors.As(err, &loginErr))
	s.Equal("用户名或密码错误", loginErr.Message)
}

func (s *AuthServiceTestSuite) TestLoginDisabledAccount() {
	s.seedUser("alice", "Passw0rd123!", false, 0)

	_, _, _, _, err := s.svc.Login(s.ctx, dto.LoginReq{
		TenantID: "tenant-001",
		Username: "alice",
		Password: "Passw0rd123!",
	})

	s.Error(err)
	s.Contains(err.Error(), "账号已被禁用")
}

// stubStatusProvider 登录租户状态拦截测试用的可编程状态提供者。
type stubStatusProvider struct {
	status int
	err    error
}

func (p stubStatusProvider) GetStatus(_ context.Context, _ string) (int, error) {
	return p.status, p.err
}

// TestLoginTenantPaused 租户被平台暂停时拒绝签发 token（软着陆拦截）。
func (s *AuthServiceTestSuite) TestLoginTenantPaused() {
	s.seedUser("alice", "Passw0rd123!", false, 1)
	s.svc.tenantStatus = stubStatusProvider{status: 0}

	_, _, _, token, err := s.svc.Login(s.ctx, dto.LoginReq{
		TenantID: "tenant-001",
		Username: "alice",
		Password: "Passw0rd123!",
	})

	s.Error(err)
	s.Empty(token)
	s.Contains(err.Error(), "租户已被平台暂停")
}

// TestLoginTenantEnabled 注入状态依赖且租户启用时正常登录。
func (s *AuthServiceTestSuite) TestLoginTenantEnabled() {
	s.seedUser("alice", "Passw0rd123!", false, 1)
	s.urRepo.seedCodes("user-alice", []string{"editor"})
	s.svc.tenantStatus = stubStatusProvider{status: 1}

	_, _, _, token, err := s.svc.Login(s.ctx, dto.LoginReq{
		TenantID: "tenant-001",
		Username: "alice",
		Password: "Passw0rd123!",
	})

	s.NoError(err)
	s.NotEmpty(token)
}

// TestLoginTenantStatusErrorFailOpen 状态查询失败（如租户不存在）时 fail-open 放行。
func (s *AuthServiceTestSuite) TestLoginTenantStatusErrorFailOpen() {
	s.seedUser("alice", "Passw0rd123!", false, 1)
	s.urRepo.seedCodes("user-alice", []string{"editor"})
	s.svc.tenantStatus = stubStatusProvider{err: errors.New("record not found")}

	_, _, _, token, err := s.svc.Login(s.ctx, dto.LoginReq{
		TenantID: "tenant-001",
		Username: "alice",
		Password: "Passw0rd123!",
	})

	s.NoError(err)
	s.NotEmpty(token)
}

// TestNewAuthServiceWithTenantStatusOption 验证函数式选项正确注入依赖。
func (s *AuthServiceTestSuite) TestNewAuthServiceWithTenantStatusOption() {
	svc := NewAuthService(
		s.userRepo, s.roleRepo, s.urRepo, s.rpRepo,
		jwt.NewTokenHelper(config.JWTConfig{Secret: "test-secret-key", Expiration: "24h", Issuer: "meteorx-test"}),
		nil,
		config.SecurityConfig{LoginLockout: config.LoginLockoutConfig{Enabled: false}},
		config.EmailConfig{Enabled: false},
		config.ClientConfig{},
		WithTenantStatusProvider(stubStatusProvider{status: 1}),
	)
	s.NotNil(svc.tenantStatus)
}

// ---------- Logout / Token 黑名单 ----------

func (s *AuthServiceTestSuite) TestLogoutRedisUnavailable() {
	// redis 为空时应静默降级成功
	_, _, _, err := s.svc.Logout(s.ctx, "some-token")
	s.NoError(err)
}

func (s *AuthServiceTestSuite) TestIsTokenBlacklistedRedisUnavailable() {
	ok, err := s.svc.IsTokenBlacklisted(s.ctx, "some-token")
	s.NoError(err)
	s.False(ok)
}

// ---------- 密码重置 ----------

func (s *AuthServiceTestSuite) TestForgotPasswordUserNotFound() {
	// 用户不存在时不暴露信息，静默成功
	err := s.svc.ForgotPassword(s.ctx, "nobody@example.com")
	s.NoError(err)
}

func (s *AuthServiceTestSuite) TestForgotPasswordEmailNotConfigured() {
	s.seedUser("alice", "Passw0rd123!", false, 1)

	err := s.svc.ForgotPassword(s.ctx, "alice@example.com")
	s.Error(err)
	s.Contains(err.Error(), "email service not configured")
}

func (s *AuthServiceTestSuite) TestResetPasswordRedisUnavailable() {
	err := s.svc.ResetPassword(s.ctx, "token", "NewPass123!")
	s.Error(err)
}

// ---------- 双因素认证 (2FA / TOTP) ----------

// seedTOTPUser 播种一个已启用 2FA 且绑定标准密钥的用户。
func (s *AuthServiceTestSuite) seedTOTPUser(username string) *userModel.User {
	hash, err := crypto.HashPassword("Passw0rd123!")
	s.Require().NoError(err)
	u := &userModel.User{
		ID:          "user-" + username,
		TenantID:    "tenant-001",
		Username:    username,
		Password:    hash,
		Email:       username + "@example.com",
		Status:      1,
		TOTPSecret:  testTOTPSecret,
		TOTPEnabled: true,
	}
	s.userRepo.seed(u)
	s.urRepo.seedCodes(u.ID, []string{"editor"})
	return u
}

// TestLogin2FA_MissingOTP 已启用 2FA 但未提供验证码时，返回 Require2FA 提示且不签发 token。
func (s *AuthServiceTestSuite) TestLogin2FA_MissingOTP() {
	s.seedTOTPUser("alice")

	_, _, _, token, err := s.svc.Login(s.ctx, dto.LoginReq{
		TenantID: "tenant-001", Username: "alice", Password: "Passw0rd123!",
	})

	s.Error(err)
	s.Empty(token)
	var loginErr *LoginError
	s.True(errors.As(err, &loginErr))
	s.True(loginErr.Require2FA)
}

// TestLogin2FA_WrongOTP 验证码错误时拒绝登录并标记 Require2FA。
func (s *AuthServiceTestSuite) TestLogin2FA_WrongOTP() {
	s.seedTOTPUser("alice")

	// 构造一个与当前窗口不一致的错误码
	cur, err := totp.Now(testTOTPSecret)
	s.Require().NoError(err)
	wrong := cur
	for wrong == cur {
		wrong = "000000"
		if cur == "000000" {
			wrong = "111111"
		}
	}

	_, _, _, token, err := s.svc.Login(s.ctx, dto.LoginReq{
		TenantID: "tenant-001", Username: "alice", Password: "Passw0rd123!", OTP: wrong,
	})

	s.Error(err)
	s.Empty(token)
	var loginErr *LoginError
	s.True(errors.As(err, &loginErr))
	s.Contains(loginErr.Message, "双因素验证码错误")
}

// TestLogin2FA_CorrectOTP 提供当前有效验证码时登录成功并签发 token。
func (s *AuthServiceTestSuite) TestLogin2FA_CorrectOTP() {
	s.seedTOTPUser("alice")

	code, err := totp.Now(testTOTPSecret)
	s.Require().NoError(err)

	_, _, _, token, err := s.svc.Login(s.ctx, dto.LoginReq{
		TenantID: "tenant-001", Username: "alice", Password: "Passw0rd123!", OTP: code,
	})

	s.NoError(err)
	s.NotEmpty(token)
}

// TestDisable2FA_NotEnabled 未启用时关闭返回 Err2FANotEnabled。
func (s *AuthServiceTestSuite) TestDisable2FA_NotEnabled() {
	s.seedUser("alice", "Passw0rd123!", false, 1)

	err := s.svc.Disable2FA(s.ctx, "user-alice", "000000")
	s.True(errors.Is(err, Err2FANotEnabled))
}

// TestDisable2FA_InvalidCode 验证码错误时不解除绑定。
func (s *AuthServiceTestSuite) TestDisable2FA_InvalidCode() {
	u := s.seedTOTPUser("alice")

	err := s.svc.Disable2FA(s.ctx, u.ID, "000001")
	s.True(errors.Is(err, Err2FAInvalidCode))
	s.True(u.TOTPEnabled)
}

// TestDisable2FA_Success 有效验证码解除绑定，清空密钥并置为未启用。
func (s *AuthServiceTestSuite) TestDisable2FA_Success() {
	u := s.seedTOTPUser("alice")
	code, err := totp.Now(testTOTPSecret)
	s.Require().NoError(err)

	err = s.svc.Disable2FA(s.ctx, u.ID, code)
	s.NoError(err)
	s.False(u.TOTPEnabled)
	s.Empty(u.TOTPSecret)
}

// withMiniredis 基于 miniredis 为套件服务注入可用 Redis。
func (s *AuthServiceTestSuite) withMiniredis() *miniredis.Miniredis {
	mr, err := miniredis.Run()
	s.Require().NoError(err)
	rdb, err := cache.NewRedis(mr.Addr(), "", 0)
	s.Require().NoError(err)
	s.svc.redis = rdb
	s.T().Cleanup(func() { _ = rdb.Close(); mr.Close() })
	return mr
}

// TestSetup2FA_GeneratesSecret 未启用用户发起开启：返回密钥与 otpauth URI，并在 Redis 暂存待绑定密钥。
func (s *AuthServiceTestSuite) TestSetup2FA_GeneratesSecret() {
	s.withMiniredis()
	s.seedUser("alice", "Passw0rd123!", false, 1)

	resp, err := s.svc.Setup2FA(s.ctx, "user-alice")
	s.Require().NoError(err)
	s.NotEmpty(resp.Secret)
	s.Contains(resp.OTPAuthURL, "otpauth://totp/")
	s.Contains(resp.OTPAuthURL, resp.Secret)
	// 待绑定密钥已写入 Redis
	stored, err := s.svc.redis.Get(s.ctx, totpPendingPrefix+"user-alice")
	s.NoError(err)
	s.Equal(resp.Secret, stored)
}

// TestSetup2FA_AlreadyEnabled 已启用用户重复开启返回 Err2FAAlreadyEnabled。
func (s *AuthServiceTestSuite) TestSetup2FA_AlreadyEnabled() {
	s.withMiniredis()
	u := s.seedTOTPUser("alice")

	_, err := s.svc.Setup2FA(s.ctx, u.ID)
	s.True(errors.Is(err, Err2FAAlreadyEnabled))
}

// TestEnable2FA_Success 使用暂存密钥生成的有效验证码确认启用，密钥持久化并清除待绑定缓存。
func (s *AuthServiceTestSuite) TestEnable2FA_Success() {
	mr := s.withMiniredis()
	s.seedUser("alice", "Passw0rd123!", false, 1)

	resp, err := s.svc.Setup2FA(s.ctx, "user-alice")
	s.Require().NoError(err)
	code, err := totp.At(resp.Secret, time.Now())
	s.Require().NoError(err)

	err = s.svc.Enable2FA(s.ctx, "user-alice", code)
	s.NoError(err)
	u := s.userRepo.users["user-alice"]
	s.True(u.TOTPEnabled)
	s.Equal(resp.Secret, u.TOTPSecret)
	// 待绑定缓存已清除
	s.False(mr.Exists(totpPendingPrefix + "user-alice"))
}

// TestEnable2FA_InvalidCode 暂存存在但验证码错误时不启用。
func (s *AuthServiceTestSuite) TestEnable2FA_InvalidCode() {
	s.withMiniredis()
	s.seedUser("alice", "Passw0rd123!", false, 1)

	_, err := s.svc.Setup2FA(s.ctx, "user-alice")
	s.Require().NoError(err)

	err = s.svc.Enable2FA(s.ctx, "user-alice", "000001")
	s.True(errors.Is(err, Err2FAInvalidCode))
	s.False(s.userRepo.users["user-alice"].TOTPEnabled)
}

// TestEnable2FA_SetupExpired 无待绑定密钥（过期/未发起）时返回 Err2FASetupExpired。
func (s *AuthServiceTestSuite) TestEnable2FA_SetupExpired() {
	s.withMiniredis()
	s.seedUser("alice", "Passw0rd123!", false, 1)

	err := s.svc.Enable2FA(s.ctx, "user-alice", "123456")
	s.True(errors.Is(err, Err2FASetupExpired))
}