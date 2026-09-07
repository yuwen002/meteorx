package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"meteorx/internal/common/contextx"
	"meteorx/internal/config"
	"meteorx/pkg/security"
)

// ================= RequestID =================

func TestRequestID_KeepsIncomingHeader(t *testing.T) {
	var got string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Request-ID", "client-rid-123")
	rr := httptest.NewRecorder()
	RequestIDMiddleware(next).ServeHTTP(rr, req)

	if rr.Header().Get("X-Request-ID") != "client-rid-123" {
		t.Fatalf("response header = %q", rr.Header().Get("X-Request-ID"))
	}
	if got != "client-rid-123" {
		t.Fatalf("ctx request id = %q", got)
	}
}

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	var got string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rr := httptest.NewRecorder()
	RequestIDMiddleware(next).ServeHTTP(rr, req)

	if len(got) != 32 {
		t.Fatalf("生成 request id 应为 32 位 hex，实际 %q", got)
	}
	if rr.Header().Get("X-Request-ID") != got {
		t.Fatalf("响应头与 ctx 不一致")
	}
}

func TestGetRequestID_EmptyWithoutMiddleware(t *testing.T) {
	if v := GetRequestID(context.Background()); v != "" {
		t.Fatalf("应返回空串，实际 %q", v)
	}
}

// ================= GlobalErrorHandler =================

func TestGlobalErrorHandler_RecoversPanic(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rr := httptest.NewRecorder()
	chain := RequestIDMiddleware(GlobalErrorHandler(next))
	chain.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"code":"INTERNAL_ERROR"`) || !strings.Contains(body, "Internal server error") {
		t.Fatalf("body = %s", body)
	}
	// panic 时的 request id 已注入
	if !strings.Contains(body, `"request_id":"`) {
		t.Fatalf("body 缺少 request_id: %s", body)
	}
}

func TestGlobalErrorHandler_PassThrough(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rr := httptest.NewRecorder()
	GlobalErrorHandler(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d", rr.Code)
	}
}

// ================= RequiresMasterAdmin =================

func withRoles(roles []string) context.Context {
	return contextx.SetVars(context.Background(), "t-1", "u-1", roles)
}

func TestRequiresMasterAdmin_AllowsSuperadmin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/x", nil)
	req = req.WithContext(withRoles([]string{"member", "superadmin"}))
	rr := httptest.NewRecorder()
	RequiresMasterAdmin()(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestRequiresMasterAdmin_ForbidsNormalUser(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cases := [][]string{nil, {"member"}, {"admin"}}
	for _, roles := range cases {
		req := httptest.NewRequest(http.MethodGet, "/admin/x", nil)
		req = req.WithContext(withRoles(roles))
		rr := httptest.NewRecorder()
		RequiresMasterAdmin()(next).ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("roles=%v status = %d", roles, rr.Code)
		}
		if !strings.Contains(rr.Body.String(), "仅限平台超级管理员") {
			t.Fatalf("roles=%v body = %s", roles, rr.Body.String())
		}
	}
}

// ================= RateLimit =================

func rateLimitCfg() config.RateLimitConfig {
	return config.RateLimitConfig{
		Enabled:  true,
		Requests: 1,
		Window:   10 * time.Second,
		BurstSize: 0,
	}
}

// RateLimit 计数依赖可用 Redis；单测环境无 Redis 时为 fail-open（放行），
// 429 分支语义（计数超限 → 拒绝）由 pkg/security rate_limiter 的职责覆盖。
func TestRateLimitMiddleware_FailOpenWithoutRedis(t *testing.T) {
	limiter := security.NewRateLimiter(nil, rateLimitCfg())
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		RateLimitMiddleware(limiter)(next).ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("第 %d 次 status = %d", i+1, rr.Code)
		}
	}
}

func TestRateLimitWithUser_FailOpenWithoutRedis(t *testing.T) {
	limiter := security.NewRateLimiter(nil, rateLimitCfg())
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mk := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("username=alice&password=x"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req
	}

	rr1 := httptest.NewRecorder()
	RateLimitWithUserMiddleware(limiter)(next).ServeHTTP(rr1, mk())
	if rr1.Code != http.StatusOK {
		t.Fatalf("首次 status = %d", rr1.Code)
	}
}

func TestRateLimitMiddleware_Disabled_AlwaysAllows(t *testing.T) {
	limiter := security.NewRateLimiter(nil, config.RateLimitConfig{Enabled: false})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		RateLimitMiddleware(limiter)(next).ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("禁用后第 %d 次 status = %d", i+1, rr.Code)
		}
	}
}
