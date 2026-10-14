package storage

import (
	"strings"
	"testing"
)

func TestBuildObjectKey_NoPrefix(t *testing.T) {
	key := buildObjectKey("", "Report.PDF")
	if !strings.HasSuffix(key, ".pdf") {
		t.Fatalf("expected lowercased extension preserved, got %q", key)
	}
	if strings.Contains(key, "/") || key == ".pdf" || len(key) <= len(".pdf") {
		t.Fatalf("key should be a flat unique name with ext, got %q", key)
	}
}

func TestBuildObjectKey_WithPrefix(t *testing.T) {
	key := buildObjectKey("/tenant/a/", "photo.jpg")
	if !strings.HasPrefix(key, "tenant/a/") {
		t.Fatalf("prefix should be normalized (no leading/trailing slash), got %q", key)
	}
	if !strings.HasSuffix(key, ".jpg") {
		t.Fatalf("extension lost, got %q", key)
	}
	// 前缀 + 唯一名，中间不应出现重复斜杠
	if strings.Contains(key, "//") {
		t.Fatalf("key contains duplicated slash: %q", key)
	}
}

func TestBuildObjectKey_NoExtension(t *testing.T) {
	key := buildObjectKey("", "README")
	if strings.HasSuffix(key, ".") {
		t.Fatalf("file without extension must not gain a trailing dot: %q", key)
	}
	if key == "" {
		t.Fatal("key must not be empty")
	}
}

func TestNormalizeTTL(t *testing.T) {
	if got := normalizeTTL(0); got != defaultPresignTTL {
		t.Fatalf("zero ttl should fall back to default %d, got %d", defaultPresignTTL, got)
	}
	if got := normalizeTTL(-5); got != defaultPresignTTL {
		t.Fatalf("negative ttl should fall back to default, got %d", got)
	}
	if got := normalizeTTL(120); got != 120 {
		t.Fatalf("positive ttl should be preserved, got %d", got)
	}
}

func TestJoinBaseURL(t *testing.T) {
	if got := joinBaseURL("https://cdn.example.com/", "/a/b.png"); got != "https://cdn.example.com/a/b.png" {
		t.Fatalf("unexpected join result: %q", got)
	}
	if got := joinBaseURL("https://cdn.example.com", "a/b.png"); got != "https://cdn.example.com/a/b.png" {
		t.Fatalf("unexpected join result: %q", got)
	}
}
