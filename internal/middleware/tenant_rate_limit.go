// Package middleware 租户 API 限流中间件
// 基于套餐配额（ApiRateLimit，单位：请求数/分钟）对每个租户进行滑动分钟窗口限流。
package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"meteorx/internal/cache"
	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/pkg/logger"
)

// QuotaRateProvider 提供租户 API 速率上限（每分钟请求数）。
// 返回 -1 表示不限制；返回 error 时中间件按 fail-open 放行。
type QuotaRateProvider interface {
	GetTenantApiRate(ctx context.Context, tenantID string) (int, error)
}

// rateCacheTTL 租户速率配额的本地缓存时长，避免每请求都查库。
const rateCacheTTL = 5 * time.Minute

// rateCacheEntry 单个租户的速率缓存项。
type rateCacheEntry struct {
	rate      int
	expiresAt time.Time
}

// rateCache 带 TTL 的租户速率缓存（简单并发安全）。
type rateCache struct {
	mu    sync.RWMutex
	items map[string]rateCacheEntry
}

func newRateCache() *rateCache {
	return &rateCache{items: make(map[string]rateCacheEntry)}
}

// get 读取未过期的缓存速率，命中且有效时返回 (rate, true)。
func (c *rateCache) get(tenantID string) (int, bool) {
	c.mu.RLock()
	entry, ok := c.items[tenantID]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return 0, false
	}
	return entry.rate, true
}

// set 写入速率缓存。
func (c *rateCache) set(tenantID string, rate int) {
	c.mu.Lock()
	c.items[tenantID] = rateCacheEntry{rate: rate, expiresAt: time.Now().Add(rateCacheTTL)}
	c.mu.Unlock()
}

// TenantRateLimit 租户级 API 限流中间件。
// 以“自然分钟”为窗口，key 为 tenant:rate:{tenantID}:{minute}，超限返回 429。
// 无租户上下文、速率不限(-1)、Redis 不可用或配额查询失败均按 fail-open 放行。
func TenantRateLimit(provider QuotaRateProvider, rdb *cache.Redis) func(http.Handler) http.Handler {
	rateStore := newRateCache()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			tenantID := contextx.GetTenantID(ctx)
			if tenantID == "" {
				// 无租户上下文（如未认证或平台侧请求），不限流
				next.ServeHTTP(w, r)
				return
			}

			rate, ok := rateStore.get(tenantID)
			if !ok {
				var err error
				rate, err = provider.GetTenantApiRate(ctx, tenantID)
				if err != nil {
					// 配额查询失败：fail-open，放行且不写缓存（下次重试）
					logger.Warn("tenant rate limit: failed to get api rate", "tenant_id", tenantID, "error", err)
					next.ServeHTTP(w, r)
					return
				}
				rateStore.set(tenantID, rate)
			}

			if rate < 0 {
				// -1 表示不限制
				next.ServeHTTP(w, r)
				return
			}

			if rdb == nil || !rdb.IsAvailable() {
				// Redis 不可用：fail-open
				next.ServeHTTP(w, r)
				return
			}

			now := time.Now()
			minute := now.Unix() / 60
			key := "tenant:rate:" + tenantID + ":" + strconv.FormatInt(minute, 10)

			count, err := rdb.Incr(ctx, key)
			if err != nil {
				// Redis 计数失败：fail-open
				logger.Warn("tenant rate limit: incr failed", "tenant_id", tenantID, "error", err)
				next.ServeHTTP(w, r)
				return
			}
			if count == 1 {
				// 首次创建该分钟窗口，设置 60s 过期
				if err := rdb.Expire(ctx, key, time.Minute); err != nil {
					logger.Warn("tenant rate limit: set expire failed", "tenant_id", tenantID, "error", err)
				}
			}

			if count > int64(rate) {
				retryAfter := 60 - now.Second()
				if retryAfter <= 0 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				response.Fail(w, http.StatusTooManyRequests, "请求过于频繁，已达到套餐 API 速率上限，请稍后再试")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
