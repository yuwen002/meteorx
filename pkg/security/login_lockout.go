package security

import (
	"context"
	"fmt"
	"meteorx/internal/cache"
	"meteorx/internal/config"
	"strconv"
)

const (
	loginAttemptPrefix = "login:attempts:"
	loginLockPrefix    = "login:lock:"
)

// LoginLockout 登录失败锁定管理器
type LoginLockout struct {
	redis  *cache.Redis
	config config.LoginLockoutConfig
}

// NewLoginLockout 创建登录锁定管理器
func NewLoginLockout(redis *cache.Redis, cfg config.LoginLockoutConfig) *LoginLockout {
	return &LoginLockout{
		redis:  redis,
		config: cfg,
	}
}

// getAttemptKey 获取尝试次数的 Redis key
func (l *LoginLockout) getAttemptKey(identifier string) string {
	return fmt.Sprintf("%s%s", loginAttemptPrefix, identifier)
}

// getLockKey 获取锁定状态的 Redis key
func (l *LoginLockout) getLockKey(identifier string) string {
	return fmt.Sprintf("%s%s", loginLockPrefix, identifier)
}

// IsLocked 检查账号是否被锁定
func (l *LoginLockout) IsLocked(ctx context.Context, identifier string) (bool, error) {
	if !l.config.Enabled {
		return false, nil
	}

	if l.redis == nil || !l.redis.IsAvailable() {
		return false, nil
	}

	locked, err := l.redis.Exists(ctx, l.getLockKey(identifier))
	if err == cache.ErrRedisUnavailable {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return locked, nil
}

// RecordFailedAttempt 记录一次登录失败
func (l *LoginLockout) RecordFailedAttempt(ctx context.Context, identifier string) error {
	if !l.config.Enabled || l.redis == nil || !l.redis.IsAvailable() {
		return nil
	}

	key := l.getAttemptKey(identifier)

	// 获取当前失败次数
	val, err := l.redis.Get(ctx, key)
	var attempts int64 = 0
	if err == nil && val != "" {
		attempts, _ = strconv.ParseInt(val, 10, 64)
	}

	attempts++

	// 如果达到最大尝试次数，锁定账号
	if int(attempts) >= l.config.MaxAttempts {
		lockKey := l.getLockKey(identifier)
		if err := l.redis.Set(ctx, lockKey, "1", l.config.LockoutDuration); err != nil && err != cache.ErrRedisUnavailable {
			return err
		}
		// 删除尝试计数
		_ = l.redis.Delete(ctx, key)
		return nil
	}

	// 更新尝试次数，设置过期时间
	if err := l.redis.Set(ctx, key, strconv.FormatInt(attempts, 10), l.config.ResetAfter); err != nil && err != cache.ErrRedisUnavailable {
		return err
	}

	return nil
}

// RecordSuccessAttempt 记录登录成功，清除失败计数
func (l *LoginLockout) RecordSuccessAttempt(ctx context.Context, identifier string) error {
	if !l.config.Enabled || l.redis == nil || !l.redis.IsAvailable() {
		return nil
	}

	key := l.getAttemptKey(identifier)
	err := l.redis.Delete(ctx, key)
	if err == cache.ErrRedisUnavailable {
		return nil
	}
	return err
}

// GetRemainingAttempts 获取剩余可尝试次数
func (l *LoginLockout) GetRemainingAttempts(ctx context.Context, identifier string) int {
	if !l.config.Enabled || l.redis == nil || !l.redis.IsAvailable() {
		return l.config.MaxAttempts
	}

	val, err := l.redis.Get(ctx, l.getAttemptKey(identifier))
	if err == cache.ErrRedisUnavailable || err != nil || val == "" {
		return l.config.MaxAttempts
	}

	attempts, _ := strconv.ParseInt(val, 10, 64)
	remaining := l.config.MaxAttempts - int(attempts)
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// GetLockoutDuration 获取剩余锁定时间（秒）
func (l *LoginLockout) GetLockoutDuration(ctx context.Context, identifier string) int64 {
	if !l.config.Enabled {
		return 0
	}

	// 这里简化处理，返回配置的锁定时长
	// 实际生产环境可以使用 Redis TTL 命令获取精确剩余时间
	return int64(l.config.LockoutDuration.Seconds())
}