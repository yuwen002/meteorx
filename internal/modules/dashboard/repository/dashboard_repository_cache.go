// Package repository 提供数据看板数据访问层
package repository

import (
	"context"
	"encoding/json"
	"time"

	"meteorx/internal/cache"
)

// 工作台/看板聚合结果的短缓存：这些维度是慢变的 COUNT/SUM 事实，
// 首屏高频刷新时以 Redis 短 TTL 摊薄重复全表聚合；缓存仅加速读，任何环节失败都降级为直接查库。
const (
	// dashboardCacheTTL 聚合结果缓存时长。取 30s：够摊薄看板连点/轮询，又不至于让运营数字明显滞后。
	dashboardCacheTTL = 30 * time.Second
	// cacheKeyOverview 平台运营总览缓存键（全局唯一，不含租户维度）。
	cacheKeyOverview = "dashboard:overview"
	// cacheKeyTenantOverviewPrefix 租户总览缓存键前缀，后接 tenantID。
	cacheKeyTenantOverviewPrefix = "dashboard:tenant-overview:"
)

// RepoOption 数据看板仓库的可选装配项，用于在不破坏既有构造函数签名的前提下注入依赖。
type RepoOption func(*dashboardRepository)

// WithCache 注入 Redis 缓存客户端并启用聚合结果的短 TTL 缓存。
// rdb 为 nil 或内部不可用时，缓存读写整体走降级路径（直接查库），行为与未注入一致。
func WithCache(rdb *cache.Redis) RepoOption {
	return func(r *dashboardRepository) {
		r.rdb = rdb
		r.cacheTTL = dashboardCacheTTL
	}
}

// tenantOverviewCacheKey 拼出某租户总览的缓存键。
func tenantOverviewCacheKey(tenantID string) string {
	return cacheKeyTenantOverviewPrefix + tenantID
}

// cacheGet 尝试从 Redis 读取 key 的 JSON 并反序列化到 T。
// 命中且解析成功返回 (值, true)；rdb 不可用、键缺失或解析失败一律返回 (零值, false)，由调用方回源。
func cacheGet[T any](ctx context.Context, rdb *cache.Redis, key string) (T, bool) {
	var zero T
	if rdb == nil || !rdb.IsAvailable() {
		return zero, false
	}
	raw, err := rdb.Get(ctx, key)
	if err != nil || raw == "" {
		return zero, false
	}
	var v T
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return zero, false
	}
	return v, true
}

// cacheSet 将值序列化后写入 Redis，TTL 非正（未启用缓存）时跳过。
// 缓存写入失败仅影响下次命中，不影响本次返回，故静默忽略错误。
func cacheSet(ctx context.Context, rdb *cache.Redis, key string, v any, ttl time.Duration) {
	if rdb == nil || ttl <= 0 || !rdb.IsAvailable() {
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	_ = rdb.Set(ctx, key, string(b), ttl)
}
