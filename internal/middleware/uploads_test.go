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

	handler := SignedUploadsHandler(dir, []string{uploadsTestSecret})

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
		openHandler := SignedUploadsHandler(dir, nil)
		req := httptest.NewRequest(http.MethodGet, "/uploads/demo.txt", nil)
		rr := httptest.NewRecorder()
		openHandler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 without secret, got %d", rr.Code)
		}
	})
}

// TestSignedUploadsHandlerRotationKeyChain 验证签名密钥平滑轮换：
// 切换新密钥后，存量旧密钥签发的链接在"新旧共链"期间仍可访问，
// 从链中移除旧密钥后旧链接应被拒绝（轮换宽限期结束）。
func TestSignedUploadsHandlerRotationKeyChain(t *testing.T) {
	dir := t.TempDir()
	const content = "rotation demo"
	if err := os.WriteFile(filepath.Join(dir, "demo.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	const newKey = "rotated-new-secret-abcdefghijklmnopqrstuvwxyz-1234567890"
	oldKey := uploadsTestSecret

	signURL := func(secret string) string {
		expires := signedurl.Expires(time.Minute)
		sig := signedurl.Sign("demo.txt", secret, expires)
		return "/uploads/demo.txt?" + signedurl.QueryExpires + "=" + strconv.FormatInt(expires, 10) +
			"&" + signedurl.QuerySig + "=" + sig
	}

	t.Run("old-signed link accepted while old key kept in chain", func(t *testing.T) {
		rotated := SignedUploadsHandler(dir, []string{newKey, oldKey})
		req := httptest.NewRequest(http.MethodGet, signURL(oldKey), nil)
		rr := httptest.NewRecorder()
		rotated.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 during rotation grace period, got %d", rr.Code)
		}
	})

	t.Run("new key signs accepted link", func(t *testing.T) {
		rotated := SignedUploadsHandler(dir, []string{newKey, oldKey})
		req := httptest.NewRequest(http.MethodGet, signURL(newKey), nil)
		rr := httptest.NewRecorder()
		rotated.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for new-key link, got %d", rr.Code)
		}
	})

	t.Run("old-signed link rejected after old key removed", func(t *testing.T) {
		onlyNew := SignedUploadsHandler(dir, []string{newKey})
		req := httptest.NewRequest(http.MethodGet, signURL(oldKey), nil)
		rr := httptest.NewRecorder()
		onlyNew.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403 after key removed from chain, got %d", rr.Code)
		}
	})
}
