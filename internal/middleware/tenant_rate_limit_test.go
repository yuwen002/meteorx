package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"meteorx/internal/cache"
	"meteorx/internal/common/contextx"

	"github.com/alicebob/miniredis/v2"
)

// stubRateProvider 测试用的速率配额提供者。
type stubRateProvider struct {
	rate int
	err  error
}

func (s stubRateProvider) GetTenantApiRate(_ context.Context, _ string) (int, error) {
	return s.rate, s.err
}

// newRedisForTest 基于 miniredis 构造可用的 cache.Redis。
func newRedisForTest(t *testing.T) (*cache.Redis, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	rdb, err := cache.NewRedis(mr.Addr(), "", 0)
	if err != nil {
		t.Fatalf("connect miniredis: %v", err)
	}
	return rdb, mr
}

// requestWithTenant 构造带租户上下文的请求。
func requestWithTenant(tenantID string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	ctx := contextx.SetVars(r.Context(), tenantID, "user-1", nil)
	return r.WithContext(ctx)
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestTenantRateLimit_Unlimited(t *testing.T) {
	rdb, mr := newRedisForTest(t)
	defer mr.Close()
	defer func() { _ = rdb.Close() }()

	mw := TenantRateLimit(stubRateProvider{rate: -1}, rdb)
	handler := mw(okHandler())

	for i := 0; i < 100; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, requestWithTenant("t-unlimited"))
		if rec.Code != http.StatusOK {
			t.Fatalf("unlimited rate should always pass, got %d", rec.Code)
		}
	}
}

func TestTenantRateLimit_ExceedsLimit(t *testing.T) {
	rdb, mr := newRedisForTest(t)
	defer mr.Close()
	defer func() { _ = rdb.Close() }()

	mw := TenantRateLimit(stubRateProvider{rate: 2}, rdb)
	handler := mw(okHandler())

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, requestWithTenant("t-limit"))
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d within limit should pass, got %d", i+1, rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithTenant("t-limit"))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("3rd request should be limited (429), got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatalf("429 response should include Retry-After header")
	}
}

func TestTenantRateLimit_NoTenantID(t *testing.T) {
	rdb, mr := newRedisForTest(t)
	defer mr.Close()
	defer func() { _ = rdb.Close() }()

	mw := TenantRateLimit(stubRateProvider{rate: 1}, rdb)
	handler := mw(okHandler())

	// 无租户上下文：直接放行，不受限流影响
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, requestWithTenant(""))
		if rec.Code != http.StatusOK {
			t.Fatalf("request without tenant should pass, got %d", rec.Code)
		}
	}
}

func TestTenantRateLimit_ProviderErrorFailOpen(t *testing.T) {
	rdb, mr := newRedisForTest(t)
	defer mr.Close()
	defer func() { _ = rdb.Close() }()

	mw := TenantRateLimit(stubRateProvider{err: errors.New("db down")}, rdb)
	handler := mw(okHandler())

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, requestWithTenant("t-err"))
		if rec.Code != http.StatusOK {
			t.Fatalf("provider error should fail-open, got %d", rec.Code)
		}
	}
}

func TestTenantRateLimit_RedisUnavailableFailOpen(t *testing.T) {
	// Redis 不可用（nil client）时应 fail-open 放行
	rdb := &cache.Redis{}
	mw := TenantRateLimit(stubRateProvider{rate: 1}, rdb)
	handler := mw(okHandler())

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, requestWithTenant("t-noredis"))
		if rec.Code != http.StatusOK {
			t.Fatalf("redis unavailable should fail-open, got %d", rec.Code)
		}
	}
}
