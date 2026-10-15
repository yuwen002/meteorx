package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ================= SecurityHeaders =================

func assertBaseHeaders(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", rr.Header().Get("X-Content-Type-Options"))
	}
	if rr.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Fatalf("X-Frame-Options = %q", rr.Header().Get("X-Frame-Options"))
	}
	if rr.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Fatalf("Referrer-Policy = %q", rr.Header().Get("Referrer-Policy"))
	}
}

func TestSecurityHeaders_BaseHeadersAlwaysSet(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	rr := httptest.NewRecorder()
	SecurityHeaders(next).ServeHTTP(rr, req)

	assertBaseHeaders(t, rr)
	// 纯 HTTP 且无 X-Forwarded-Proto 时不应下发 HSTS
	if rr.Header().Get("Strict-Transport-Security") != "" {
		t.Fatalf("HTTP 请求不应有 HSTS，实际 %q", rr.Header().Get("Strict-Transport-Security"))
	}
}

func TestSecurityHeaders_NoHSTSWhenBehindProxy(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 仅有 X-Forwarded-Proto=https 但本进程未直接终结 TLS：HSTS 应交由边缘层，Go 不发以免重复头
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	SecurityHeaders(next).ServeHTTP(rr, req)

	assertBaseHeaders(t, rr)
	if rr.Header().Get("Strict-Transport-Security") != "" {
		t.Fatalf("被反代挡在前面时 Go 不应下发 HSTS，实际 %q", rr.Header().Get("Strict-Transport-Security"))
	}
}

func TestSecurityHeaders_HSTSWhenTLS(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	req.TLS = &tls.ConnectionState{}
	rr := httptest.NewRecorder()
	SecurityHeaders(next).ServeHTTP(rr, req)

	assertBaseHeaders(t, rr)
	if rr.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("TLS 直连时应下发 HSTS")
	}
}

func TestSecurityHeaders_PassesThroughStatusCode(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	rr := httptest.NewRecorder()
	SecurityHeaders(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d", rr.Code)
	}
}
