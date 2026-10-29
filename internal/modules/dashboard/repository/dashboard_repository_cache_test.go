package repository

import (
	"context"
	"testing"
	"time"

	"meteorx/internal/cache"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

// newCacheRedis 基于 miniredis 构造可用的 cache.Redis。
func newCacheRedis(t *testing.T) (*cache.Redis, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	rdb, err := cache.NewRedis(mr.Addr(), "", 0)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = rdb.Close()
		mr.Close()
	})
	return rdb, mr
}

// TestGetTenantOverview_CacheHitAndExpiry 验证租户总览的短缓存生命周期：
// 首查回源并写缓存 → 删除底表数据后二查仍读缓存（证明命中，不重复聚合）→
// 快进过 TTL 后三查回源，得到删除后的零值（证明过期后失效重取）。
func TestGetTenantOverview_CacheHitAndExpiry(t *testing.T) {
	db := newOverviewTestDB(t)
	seedOverviewData(t, db)
	rdb, mr := newCacheRedis(t)

	repo := NewDashboardRepository(db, WithCache(rdb))
	ctx := context.Background()

	// 1) 首查回源
	ov1, err := repo.GetTenantOverview(ctx, "tA")
	require.NoError(t, err)
	require.Equal(t, int64(3), ov1.Member.Total)
	require.Equal(t, int64(4), ov1.Task.Total)

	// 缓存键确已写入且带 TTL
	key := tenantOverviewCacheKey("tA")
	require.True(t, mr.Exists(key), "expected cache key written after first call")
	require.Greater(t, mr.TTL(key), time.Duration(0))

	// 2) 删除底表全部数据：若走回源将得到零值
	for _, tbl := range []string{"users", "files", "tasks", "wiki_spaces", "wiki_documents"} {
		require.NoError(t, db.Exec("DELETE FROM "+tbl).Error)
	}

	ov2, err := repo.GetTenantOverview(ctx, "tA")
	require.NoError(t, err)
	require.Equal(t, int64(3), ov2.Member.Total, "second call should be served from cache, not recomputed")
	require.Equal(t, int64(4), ov2.Task.Total, "second call should be served from cache, not recomputed")

	// 3) 快进过 TTL 触发过期，三查回源 → 命中已删空的数据，得零值
	mr.FastForward(dashboardCacheTTL + time.Second)
	ov3, err := repo.GetTenantOverview(ctx, "tA")
	require.NoError(t, err)
	require.Equal(t, int64(0), ov3.Member.Total, "after expiry the loader should recompute against emptied tables")
	require.Equal(t, int64(0), ov3.Task.Total)
}

// TestGetTenantOverview_NoCacheFailOpen 验证未注入 Redis（nil）时缓存路径整体降级，结果仍正确。
func TestGetTenantOverview_NoCacheFailOpen(t *testing.T) {
	db := newOverviewTestDB(t)
	seedOverviewData(t, db)

	// WithCache(nil)：rdb 为 nil，cacheGet/cacheSet 全部短路，直接走并行回源
	repo := NewDashboardRepository(db, WithCache(nil))
	ov, err := repo.GetTenantOverview(context.Background(), "tA")
	require.NoError(t, err)
	require.Equal(t, int64(3), ov.Member.Total)
	require.Equal(t, int64(3000), ov.StorageUsed)
}

// TestCacheHelpers_RoundTrip 直接校验 cacheGet/cacheSet 的序列化回环与降级判定。
func TestCacheHelpers_RoundTrip(t *testing.T) {
	rdb, mr := newCacheRedis(t)
	ctx := context.Background()

	type payload struct {
		N int    `json:"n"`
		S string `json:"s"`
	}

	// 未写入前 → miss
	_, ok := cacheGet[payload](ctx, rdb, "dash:test:missing")
	require.False(t, ok)

	// 写入后 → hit 且值一致
	in := payload{N: 42, S: "hello"}
	cacheSet(ctx, rdb, "dash:test:hit", in, dashboardCacheTTL)
	got, ok := cacheGet[payload](ctx, rdb, "dash:test:hit")
	require.True(t, ok)
	require.Equal(t, in, got)

	// TTL 非正 → cacheSet 跳过写入
	cacheSet(ctx, rdb, "dash:test:skip", in, 0)
	require.False(t, mr.Exists("dash:test:skip"), "cacheSet with non-positive TTL must not write")

	// nil rdb → cacheGet 直接 miss，不 panic
	_, ok = cacheGet[payload](ctx, nil, "dash:test:any")
	require.False(t, ok)
}
