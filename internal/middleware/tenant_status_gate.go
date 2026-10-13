// Package middleware 租户状态门禁中间件
// 在认证之后校验租户当前状态：被平台禁用/暂停的租户，其成员的所有业务 API 请求返回 403，
// 实现「租户暂停软着陆」的运行时拦截（数据保留、可随时恢复）。
package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/common/response"
	"meteorx/internal/modules/tenant/model"
	"meteorx/pkg/logger"
)

// TenantStatusProvider 提供租户状态查询（1-启用 0-禁用/暂停），与租户仓储 GetStatus 签名对齐。
// 租户不存在或查询失败时返回 error，门禁按 fail-open 放行（与租户限流中间件口径一致，
// 避免基础设施故障把全部租户误锁死；平台侧账号即使用于不存在的租户 ID 也天然放行）。
type TenantStatusProvider interface {
	GetStatus(ctx context.Context, tenantID string) (int, error)
}

// tenantStatusCacheTTL 租户状态本地缓存时长。
// 取值较小以兼顾「高频请求避免查库」与「管理员恢复租户后尽快放行」。
const tenantStatusCacheTTL = time.Minute

// tenantStatusEntry 单个租户的状态缓存项。
type tenantStatusEntry struct {
	status    int
	expiresAt time.Time
}

// tenantStatusCache 带 TTL 的租户状态缓存（并发安全）。
type tenantStatusCache struct {
	mu    sync.RWMutex
	items map[string]tenantStatusEntry
}

func newTenantStatusCache() *tenantStatusCache {
	return &tenantStatusCache{items: make(map[string]tenantStatusEntry)}
}

// get 读取未过期的缓存状态，命中且有效时返回 (status, true)。
func (c *tenantStatusCache) get(tenantID string) (int, bool) {
	c.mu.RLock()
	entry, ok := c.items[tenantID]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return 0, false
	}
	return entry.status, true
}

// set 写入状态缓存。
func (c *tenantStatusCache) set(tenantID string, status int) {
	c.mu.Lock()
	c.items[tenantID] = tenantStatusEntry{status: status, expiresAt: time.Now().Add(tenantStatusCacheTTL)}
	c.mu.Unlock()
}

// TenantStatusGate 租户状态门禁：认证之后、业务之前的「租户是否可用」防线。
// 无租户上下文的请求（平台公开接口等）直接放行；禁用租户返回 403。
func TenantStatusGate(provider TenantStatusProvider) func(http.Handler) http.Handler {
	statusStore := newTenantStatusCache()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			tenantID := contextx.GetTenantID(ctx)
			if tenantID == "" {
				// 无租户上下文（平台侧/未绑定租户的请求），不拦截
				next.ServeHTTP(w, r)
				return
			}

			status, ok := statusStore.get(tenantID)
			if !ok {
				var err error
				status, err = provider.GetStatus(ctx, tenantID)
				if err != nil {
					// 租户不存在或查询失败：fail-open 放行，不写缓存（下次重试）
					logger.Warn("tenant status gate: failed to get tenant status", "tenant_id", tenantID, "error", err)
					next.ServeHTTP(w, r)
					return
				}
				statusStore.set(tenantID, status)
			}

			if status != model.StatusEnabled {
				response.Fail(w, http.StatusForbidden, "租户已被平台暂停，请联系管理员处理")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
