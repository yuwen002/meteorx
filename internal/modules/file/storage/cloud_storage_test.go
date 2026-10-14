package storage

import (
	"context"
	"strings"
	"testing"

	"meteorx/internal/config"
)

func TestS3Storage_PresignAndURL(t *testing.T) {
	// S3 使用独立的合法配置：端点留空走 AWS 默认端点（可离线签名），避免与 OSS 端点格式混用
	s3Cfg := func() config.FileConfig {
		cfg := config.FileConfig{StorageType: "s3"}
		cfg.Cloud.AccessKey = "test-access-key"
		cfg.Cloud.SecretKey = "test-secret-key"
		cfg.Cloud.Bucket = "test-bucket"
		cfg.Cloud.Region = "us-east-1"
		return cfg
	}
	s, err := NewS3Storage(context.TODO(), s3Cfg())
	if err != nil {
		t.Fatal(err)
	}
	t.Run("presign produces signed url offline", func(t *testing.T) {
		raw, err := s.PresignURL(context.TODO(), "dir/abc.png", 120)
		if err != nil {
			t.Fatal(err)
		}
		for _, frag := range []string{"X-Amz-Signature", "X-Amz-Expires=120", "dir/abc.png"} {
			if !strings.Contains(raw, frag) {
				t.Fatalf("presigned url missing %q: %s", frag, raw)
			}
		}
	})
	t.Run("geturl defaults to presigned when private", func(t *testing.T) {
		raw := s.GetURL("dir/abc.png")
		if !strings.Contains(raw, "X-Amz-Signature") {
			t.Fatalf("private GetURL should return presigned url, got %q", raw)
		}
	})
	t.Run("geturl uses public base when configured", func(t *testing.T) {
		cfg := s3Cfg()
		cfg.Cloud.PublicBaseURL = "https://cdn.example.com/"
		pub, err := NewS3Storage(context.TODO(), cfg)
		if err != nil {
			t.Fatal(err)
		}
		if got := pub.GetURL("dir/abc.png"); got != "https://cdn.example.com/dir/abc.png" {
			t.Fatalf("unexpected public url: %q", got)
		}
	})
}

func TestOSSStorage_PresignAndURL(t *testing.T) {
	s, err := NewOSSStorage(fullCloudCfg("oss"))
	if err != nil {
		t.Fatal(err)
	}
	t.Run("presign produces signed url offline", func(t *testing.T) {
		raw, err := s.PresignURL(context.TODO(), "dir/abc.png", 120)
		if err != nil {
			t.Fatal(err)
		}
		for _, frag := range []string{"Signature", "Expires", "abc.png"} {
			if !strings.Contains(raw, frag) {
				t.Fatalf("presigned url missing %q: %s", frag, raw)
			}
		}
	})
	t.Run("geturl defaults to presigned when private", func(t *testing.T) {
		raw := s.GetURL("dir/abc.png")
		if !strings.Contains(raw, "Signature") {
			t.Fatalf("private GetURL should return presigned url, got %q", raw)
		}
	})
	t.Run("geturl uses public base when configured", func(t *testing.T) {
		cfg := fullCloudCfg("oss")
		cfg.Cloud.PublicBaseURL = "https://cdn.example.com"
		pub, err := NewOSSStorage(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if got := pub.GetURL("dir/abc.png"); got != "https://cdn.example.com/dir/abc.png" {
			t.Fatalf("unexpected public url: %q", got)
		}
	})
}

// TestNewS3Storage_CredentialModes 校验凭证三种形态：
// 均填写→静态；均留空→默认凭证链（可离线构造）；仅填其一→报错。
func TestNewS3Storage_CredentialModes(t *testing.T) {
	base := func() config.FileConfig {
		cfg := config.FileConfig{StorageType: "s3"}
		cfg.Cloud.Bucket = "test-bucket"
		cfg.Cloud.Region = "us-east-1"
		return cfg
	}
	t.Run("both empty uses default chain", func(t *testing.T) {
		s, err := NewS3Storage(context.TODO(), base())
		if err != nil {
			t.Fatalf("empty creds should fall back to default credential chain, got error: %v", err)
		}
		if s == nil {
			t.Fatal("expected non-nil *S3Storage")
		}
	})
	t.Run("only access key errors", func(t *testing.T) {
		cfg := base()
		cfg.Cloud.AccessKey = "only-ak"
		if _, err := NewS3Storage(context.TODO(), cfg); err == nil {
			t.Fatal("expected error when only one credential is set")
		}
	})
	t.Run("only secret key errors", func(t *testing.T) {
		cfg := base()
		cfg.Cloud.SecretKey = "only-sk"
		if _, err := NewS3Storage(context.TODO(), cfg); err == nil {
			t.Fatal("expected error when only one credential is set")
		}
	})
}

// 编译期确保两类云驱动完整实现 Storage 接口。
var (
	_ Storage = (*S3Storage)(nil)
	_ Storage = (*OSSStorage)(nil)
	_ Storage = (*LocalStorage)(nil)
)
