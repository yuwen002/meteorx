// Package security 提供安全相关工具：登录锁定、限流等
package security

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"meteorx/internal/cache"
	"meteorx/internal/config"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
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

// getKey 生成限流 Redis key，包含时间窗口信息
func (r *RateLimiter) getKey(identifier string) string {
	return fmt.Sprintf("%s%s:%d", rateLimitPrefix, identifier, time.Now().Unix()/int64(r.config.Window.Seconds()))
}

// Allow 检查请求是否允许通过，基于 Redis 滑动窗口计数
// Redis 不可用时默认放行，避免因基础设施故障阻塞正常流量
func (r *RateLimiter) Allow(ctx context.Context, identifier string) (bool, error) {
	if !r.config.Enabled {
		return true, nil
	}

	if r.redis == nil || !r.redis.IsAvailable() {
		return true, nil
	}

	key := r.getKey(identifier)

	// 获取当前窗口的请求数
	val, err := r.redis.Get(ctx, key)
	// Redis 不可用时放行请求，避免因基础设施故障阻塞正常流量
	if errors.Is(err, cache.ErrRedisUnavailable) {
		return true, nil
	}
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
	// Redis 不可用时静默忽略
	if err := r.redis.Set(ctx, key, strconv.FormatInt(count, 10), r.config.Window); err != nil && !errors.Is(err, cache.ErrRedisUnavailable) {
		return false, err
	}

	return true, nil
}

// GetClientIP 获取客户端真实 IP（防伪造）。
// 优先取 chi ClientIPFromXFF 中间件基于可信代理 CIDR 从 XFF 链解析后写入 context 的结果
// （右起逐段跳过可信段，首个不可信地址即客户端，伪造头部无法冒充）；
// 未经该中间件时回退 RemoteAddr，绝不直接信任 X-Forwarded-For / X-Real-IP 裸头。
func GetClientIP(r *http.Request) string {
	if ip := chimiddleware.GetClientIP(r.Context()); ip != "" {
		return ip
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}