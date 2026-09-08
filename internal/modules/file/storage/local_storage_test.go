package storage

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"meteorx/internal/common/signedurl"
)

const (
	storageTestBase    = "http://localhost:8081/uploads"
	storageTestKey     = "upload-sign-key-abcdefghijklmnopqrstuvwxyz-1234567890"
	storageTestOldKey  = "old-upload-sign-key-abcdefghijklmnopqrstuvwxyz-1234567890"
	storageTestFileKey = "abc.png"
)

// parseSignedURL 解析签名 URL，返回 文件名/过期时间/签名
func parseSignedURL(t *testing.T, raw, wantName string) (string, int64, string) {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("invalid URL %q: %v", raw, err)
	}
	if u.Path != "/uploads/"+wantName {
		t.Fatalf("unexpected path: %s", u.Path)
	}
	q := u.Query()
	expiresStr := q.Get(signedurl.QueryExpires)
	sig := q.Get(signedurl.QuerySig)
	if expiresStr == "" || sig == "" {
		t.Fatalf("signed URL missing query params: %s", raw)
	}
	expires, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil {
		t.Fatalf("invalid expires in URL: %v", err)
	}
	return wantName, expires, sig
}

func TestLocalStorageGetURL_NoKey_ReturnsPlainURL(t *testing.T) {
	s := NewLocalStorage("./uploads", storageTestBase, "")
	raw := s.GetURL(storageTestFileKey)
	if raw != storageTestBase+"/"+storageTestFileKey {
		t.Fatalf("expected unsigned URL, got %q", raw)
	}
	if strings.Contains(raw, "?") {
		t.Fatalf("unsigned URL must not carry query params: %s", raw)
	}
}

func TestLocalStorageGetURL_Signed(t *testing.T) {
	s := NewLocalStorage("./uploads", storageTestBase, storageTestKey)
	name, expires, sig := parseSignedURL(t, s.GetURL(storageTestFileKey), storageTestFileKey)
	if !signedurl.VerifyAny(name, []string{storageTestKey}, expires, sig) {
		t.Fatal("URL signature should verify with current key")
	}
	// 轮换宽限期：旧密钥仍在链上时存量链接仍可验证
	if !signedurl.VerifyAny(name, []string{storageTestOldKey, storageTestKey}, expires, sig) {
		t.Fatal("URL signature should verify while rotated key kept in chain")
	}
	if signedurl.VerifyAny(name, []string{storageTestOldKey}, expires, sig) {
		t.Fatal("URL signature must not verify with unrelated key")
	}
}

func TestLocalStoragePresignURL(t *testing.T) {
	t.Run("respects explicit expiration", func(t *testing.T) {
		s := NewLocalStorage("./uploads", storageTestBase, storageTestKey)
		raw, err := s.PresignURL(nil, storageTestFileKey, 60)
		if err != nil {
			t.Fatal(err)
		}
		name, expires, sig := parseSignedURL(t, raw, storageTestFileKey)
		now := time.Now().Unix()
		if expires-now > 65 || expires-now < 55 {
			t.Fatalf("expires not within expected 60s window: %d (now %d)", expires, now)
		}
		if !signedurl.Verify(name, storageTestKey, expires, sig) {
			t.Fatal("presigned URL should verify")
		}
	})

	t.Run("defaults to DefaultTTL when expiration not given", func(t *testing.T) {
		s := NewLocalStorage("./uploads", storageTestBase, storageTestKey)
		raw, err := s.PresignURL(nil, storageTestFileKey, 0)
		if err != nil {
			t.Fatal(err)
		}
		_, expires, sig := parseSignedURL(t, raw, storageTestFileKey)
		if !signedurl.Verify(storageTestFileKey, storageTestKey, expires, sig) {
			t.Fatal("presigned URL (default TTL) should verify")
		}
	})

	t.Run("unsigned when no key configured", func(t *testing.T) {
		s := NewLocalStorage("./uploads", storageTestBase, "")
		raw, err := s.PresignURL(nil, storageTestFileKey, 60)
		if err != nil {
			t.Fatal(err)
		}
		if raw != storageTestBase+"/"+storageTestFileKey {
			t.Fatalf("expected plain URL without key, got %q", raw)
		}
	})
}
