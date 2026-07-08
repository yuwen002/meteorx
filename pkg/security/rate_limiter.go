package security

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"meteorx/internal/cache"
	"meteorx/internal/config"
)

const rateLimitPrefix = "rate_limit:"

// RateLimiter 基于 Redis 的滑动窗口限流器
type RateLimiter struct {
	redis  *cache.Redis
	config config.RateLimitConfig
}

// NewRateLimiter 创建限流器
func NewRateLimiter(redis *cache.Redis, cfg config.RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		redis:  redis,
		config: cfg,
	}
}

// getKey 获取限流 Redis key
func (r *RateLimiter) getKey(identifier string) string {
	return fmt.Sprintf("%s%s:%d", rateLimitPrefix, identifier, time.Now().Unix()/int64(r.config.Window.Seconds()))
}

// Allow 检查是否允许请求通过
func (r *RateLimiter) Allow(ctx context.Context, identifier string) (bool, error) {
	if !r.config.Enabled {
		return true, nil
	}

	if r.redis == nil {
		return true, nil
	}

	key := r.getKey(identifier)

	// 获取当前窗口的请求数
	val, err := r.redis.Get(ctx, key)
	var count int64 = 0
	if err == nil && val != "" {
		count, _ = strconv.ParseInt(val, 10, 64)
	}

	// 检查是否超过限制
	if int(count) >= r.config.Requests {
		return false, nil
	}

	// 增加计数
	count++
	if err := r.redis.Set(ctx, key, strconv.FormatInt(count, 10), r.config.Window); err != nil {
		return false, err
	}

	return true, nil
}

// GetClientIP 获取客户端真实 IP
func GetClientIP(r *http.Request) string {
	// 优先从 X-Forwarded-For 获取
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}

	// 从 X-Real-IP 获取
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// 从 RemoteAddr 获取
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}