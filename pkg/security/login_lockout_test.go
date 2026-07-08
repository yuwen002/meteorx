package security

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// LoginLockoutTestSuite 登录锁定测试套件
type LoginLockoutTestSuite struct {
	suite.Suite
	lockout *LoginLockout
	ctx     context.Context
}

func (s *LoginLockoutTestSuite) SetupTest() {
	// 使用真实 Redis 或内存实现
	// 这里使用一个简单的内存实现进行测试
	cfg := config.LoginLockoutConfig{
		Enabled:         true,
		MaxAttempts:     3,
		LockoutDuration: 1 * time.Minute,
		ResetAfter:      10 * time.Minute,
	}
	s.lockout = NewLoginLockout(nil, cfg)
	s.ctx = context.Background()
}

func TestLoginLockoutSuite(t *testing.T) {
	suite.Run(t, new(LoginLockoutTestSuite))
}

func (s *LoginLockoutTestSuite) TestIsLockedDisabled() {
	cfg := config.LoginLockoutConfig{Enabled: false}
	lockout := NewLoginLockout(nil, cfg)

	locked, err := lockout.IsLocked(s.ctx, "user1")
	s.NoError(err)
	s.False(locked)
}

func (s *LoginLockoutTestSuite) TestRecordFailedAttempt() {
	// 无 Redis 时，RecordFailedAttempt 不应 panic
	err := s.lockout.RecordFailedAttempt(s.ctx, "user1")
	s.NoError(err)
}

func (s *LoginLockoutTestSuite) TestRecordSuccessAttempt() {
	err := s.lockout.RecordSuccessAttempt(s.ctx, "user1")
	s.NoError(err)
}

func (s *LoginLockoutTestSuite) TestGetRemainingAttempts() {
	// 无 Redis 时，应返回最大尝试次数
	remaining := s.lockout.GetRemainingAttempts(s.ctx, "user1")
	s.Equal(3, remaining)
}

func (s *LoginLockoutTestSuite) TestGetLockoutDuration() {
	duration := s.lockout.GetLockoutDuration(s.ctx, "user1")
	s.Equal(int64(60), duration)
}

func TestLoginLockoutConfig(t *testing.T) {
	cfg := config.LoginLockoutConfig{
		Enabled:         true,
		MaxAttempts:     5,
		LockoutDuration: 30 * time.Minute,
		ResetAfter:      24 * time.Hour,
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 5, cfg.MaxAttempts)
	assert.Equal(t, 30*time.Minute, cfg.LockoutDuration)
	assert.Equal(t, 24*time.Hour, cfg.ResetAfter)
}

func TestNewLoginLockout(t *testing.T) {
	cfg := config.LoginLockoutConfig{
		Enabled:         true,
		MaxAttempts:     3,
		LockoutDuration: 5 * time.Minute,
	}

	lockout := NewLoginLockout(nil, cfg)
	assert.NotNil(t, lockout)
	assert.Equal(t, cfg, lockout.config)
}

func TestGetKeys(t *testing.T) {
	lockout := NewLoginLockout(nil, config.LoginLockoutConfig{})

	attemptKey := lockout.getAttemptKey("user123")
	assert.Equal(t, "login:attempts:user123", attemptKey)

	lockKey := lockout.getLockKey("user123")
	assert.Equal(t, "login:lock:user123", lockKey)
}
