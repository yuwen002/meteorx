package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"meteorx/internal/common/signedurl"
)

const uploadsTestSecret = "test-secret-abcdefghijklmnopqrstuvwxyz-1234567890"

func TestSignedUploadsHandler(t *testing.T) {
	dir := t.TempDir()
	const content = "hello signed uploads"
	if err := os.WriteFile(filepath.Join(dir, "demo.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	handler := SignedUploadsHandler(dir, uploadsTestSecret)

	t.Run("rejects directory listing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/uploads/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for directory listing, got %d", rr.Code)
		}
	})

	t.Run("rejects unsigned request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/uploads/demo.txt", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 without signature, got %d", rr.Code)
		}
	})

	t.Run("serves file with valid signature", func(t *testing.T) {
		expires := signedurl.Expires(time.Minute)
		sig := signedurl.Sign("demo.txt", uploadsTestSecret, expires)
		req := httptest.NewRequest(http.MethodGet,
			"/uploads/demo.txt?"+signedurl.QueryExpires+"="+strconv.FormatInt(expires, 10)+
				"&"+signedurl.QuerySig+"="+sig, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 with valid signature, got %d", rr.Code)
		}
		if rr.Body.String() != content {
			t.Fatalf("unexpected body: %q", rr.Body.String())
		}
	})

	t.Run("rejects expired signature", func(t *testing.T) {
		expires := time.Now().Add(-time.Minute).Unix()
		sig := signedurl.Sign("demo.txt", uploadsTestSecret, expires)
		req := httptest.NewRequest(http.MethodGet,
			"/uploads/demo.txt?"+signedurl.QueryExpires+"="+strconv.FormatInt(expires, 10)+
				"&"+signedurl.QuerySig+"="+sig, nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for expired signature, got %d", rr.Code)
		}
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		for _, p := range []string{"/uploads/../demo.txt", "/uploads/a/b.txt", "/uploads/..%2Fdemo.txt"} {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			if rr.Code == http.StatusOK {
				t.Fatalf("expected non-200 for traversal path %q, got %d", p, rr.Code)
			}
		}
	})

	t.Run("open access when secret empty", func(t *testing.T) {
		openHandler := SignedUploadsHandler(dir, "")
		req := httptest.NewRequest(http.MethodGet, "/uploads/demo.txt", nil)
		rr := httptest.NewRecorder()
		openHandler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 without secret, got %d", rr.Code)
		}
	})
}
