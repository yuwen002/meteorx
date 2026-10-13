package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"meteorx/internal/modules/tenant/model"
)

// stubStatusProvider 测试用的租户状态提供者，可配置状态、错误与调用计数。
type stubStatusProvider struct {
	status int
	err    error
	calls  atomic.Int64
}

func (s *stubStatusProvider) GetStatus(_ context.Context, _ string) (int, error) {
	s.calls.Add(1)
	return s.status, s.err
}

func TestTenantStatusGate_EnabledPass(t *testing.T) {
	mw := TenantStatusGate(&stubStatusProvider{status: model.StatusEnabled})
	handler := mw(okHandler())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithTenant("t-on"))
	if rec.Code != http.StatusOK {
		t.Fatalf("enabled tenant should pass, got %d", rec.Code)
	}
}

func TestTenantStatusGate_DisabledBlocked(t *testing.T) {
	mw := TenantStatusGate(&stubStatusProvider{status: model.StatusDisabled})
	handler := mw(okHandler())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithTenant("t-off"))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("disabled tenant should get 403, got %d", rec.Code)
	}
}

func TestTenantStatusGate_NoTenantIDPass(t *testing.T) {
	p := &stubStatusProvider{status: model.StatusDisabled}
	mw := TenantStatusGate(p)
	handler := mw(okHandler())

	// 无租户上下文：直接放行且不触发查询
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, requestWithTenant(""))
	if rec.Code != http.StatusOK {
		t.Fatalf("request without tenant should pass, got %d", rec.Code)
	}
	if p.calls.Load() != 0 {
		t.Fatalf("provider should not be called without tenant id, calls=%d", p.calls.Load())
	}
}

func TestTenantStatusGate_ProviderErrorFailOpen(t *testing.T) {
	mw := TenantStatusGate(&stubStatusProvider{err: errors.New("db down")})
	handler := mw(okHandler())

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, requestWithTenant("t-err"))
		if rec.Code != http.StatusOK {
			t.Fatalf("provider error should fail-open, got %d", rec.Code)
		}
	}
}

func TestTenantStatusGate_CachesStatus(t *testing.T) {
	p := &stubStatusProvider{status: model.StatusEnabled}
	mw := TenantStatusGate(p)
	handler := mw(okHandler())

	// 同一租户多次请求仅查询一次（本地 TTL 缓存生效）
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, requestWithTenant("t-cache"))
		if rec.Code != http.StatusOK {
			t.Fatalf("cached enabled tenant should pass, got %d", rec.Code)
		}
	}
	if got := p.calls.Load(); got != 1 {
		t.Fatalf("status should be cached, expect 1 provider call, got %d", got)
	}
}

func TestTenantStatusGate_ProviderErrorNotCached(t *testing.T) {
	p := &stubStatusProvider{err: errors.New("db down")}
	mw := TenantStatusGate(p)
	handler := mw(okHandler())

	// 查询失败不写缓存，每次请求都重新尝试
	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, requestWithTenant("t-nocache"))
		if rec.Code != http.StatusOK {
			t.Fatalf("fail-open should pass, got %d", rec.Code)
		}
	}
	if got := p.calls.Load(); got != 3 {
		t.Fatalf("failed lookups should not be cached, expect 3 calls, got %d", got)
	}
}
