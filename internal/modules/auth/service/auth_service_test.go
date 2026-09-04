package service

import (
	"context"
	"errors"
	"testing"

	"meteorx/internal/common/jwt"
	"meteorx/internal/config"
	"meteorx/internal/modules/auth/dto"
	rbacModel "meteorx/internal/modules/rbac/model"
	userModel "meteorx/internal/modules/user/model"
	"meteorx/pkg/crypto"

	"github.com/stretchr/testify/suite"
)

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

// ---------- Logout / Token 黑名单 ----------

func (s *AuthServiceTestSuite) TestLogoutRedisUnavailable() {
	// redis 为空时应静默降级成功
	err := s.svc.Logout(s.ctx, "some-token")
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
